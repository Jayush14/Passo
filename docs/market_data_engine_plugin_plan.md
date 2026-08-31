# Market Data Engine Plugin – Implementation Plan (Revised)

**Goal:** Build a high‑accuracy plugin that takes raw LLM‑extracted job data (already stored in `job_extractions`) and converts it into a canonical, normalized relational model (`jobs`, `job_skills`, `job_postings`, …) without additional LLM calls for the majority of rows. The plugin enforces a strict separation between **extraction** (LLM) and **canonicalization** (deterministic taxonomy + human‑in‑the‑loop review).

---  

## 1. Success Criteria  

| Metric | Target | How we measure |
|--------|--------|----------------|
| **Canonical Skill Coverage** | ≥ 95 % of distinct `skill.raw_name` map to a single `skills.canonical_name` | Sample 10 k extractions; compute % unique after normalization |
| **Canonical Skill Precision** | ≥ 98 % of mappings are correct (validated against a curated gold set) | Manual audit of 500 mapped skills |
| **Title Normalization Accuracy** | ≥ 95 % of titles resolve to the same canonical title for same job family & seniority level | Manual audit of 500 titles |
| **Location Normalization** | ≥ 98 % of city/state/country triples resolve to a known GeoNames entry | Compare to GeoNames dump |
| **Processing Latency** | ≤ 200 ms per extraction (excl. LLM call) | Benchmark normalization service |
| **Freshness** | New extractions canonicalized within 5 min of arrival | Time between `job_extractions.created_at` and `jobs.updated_at` |
| **Error Rate** | ≤ 0.5 % rows end up `status='FAILED'` after normalization | Count failed rows in `job_extractions` |
| **False Merge Rate** | ≤ 0.5 % of distinct canonical entities are incorrectly merged | Sample review of merged entities |
| **Unknown Rate** | ≤ 5 % of extracted entities remain in review queue after stabilization | Size of `entity_resolution_queue` |

---  

## 2. High‑Level Architecture (4‑Layer Flow)

```
RAW TEXT (copy‑paste JD)
        │
        ▼
┌─────────────────────┐
│ 1️⃣ LLM Extraction   │   ← already done (`/process/job`)
│    → Enveloped JSON │
└───────┬─────────────┘
        │
        ▼
┌─────────────────────┐
│ 2️⃣ Normalization   │   ← Plugin (deterministic only + review queue)
│    → Canonical IDs │
└───────┬─────────────┘
        │
        ▼
┌─────────────────────────────┐
│ 3️⃣ Canonical DB Store      │   ← Tables: jobs, job_postings,
│    (source of truth)        │   job_descriptions, job_skills, …
└───────┬─────────────────────┘
        │
        ▼
┌─────────────┐   ┌─────────────────────┐
│ API (read)  │   │ Background Worker   │
│ (GET /jobs) │   │ (continuous pull)   │
└─────────────┘   └─────────────────────┘
```

*The plugin lives in **step 2** (Normalization) and **step 3** (persist). No LLM calls are made during normalization; unresolved items go to a review queue for human curation.*

---  

## 3. Data Model – Canonical Layer  

All tables include `created_at`, `updated_at`, `source` (`LLM`, `USER_EDIT`, `IMPORT`). Primary keys are UUID (`gen_random_uuid()`).  
Each canonical entity stores the **taxonomy version** that produced the mapping.

### 3.1 Core Reference Tables  

| Table | Key Columns | Purpose |
|-------|-------------|---------|
| **companies** | `id PK`, `canonical_name`, `normalized_name`, `website`, `industry`, `size`, `taxonomy_version` | One record per employer (e.g., Cognizant). |
| **job_families** | `id PK`, `name`, `parent_id`, `taxonomy_version` | Hierarchical families (Technology → Software Engineering → Backend Engineering …). |
| **job_titles** | `id PK`, `canonical_title`, `job_family_id`, `level_id`, `taxonomy_version` | Normalized title (e.g., “Backend Engineer”). |
| **seniority_levels** | `id PK`, `name`, `ordinal`, `taxonomy_version` | Entry, Junior, Mid, Senior, Lead, Staff, Principal, Executive. |
| **locations** | `id PK`, `city`, `state`, `country`, `country_code`, `lat`, `lon`, `taxonomy_version` | Geo‑normalized place (seed from GeoNames). |
| **employment_types** | `id PK`, `name`, `taxonomy_version` | full_time, part_time, contract, temporary, internship, freelance. |
| **work_modes** | `id PK`, `name`, `taxonomy_version` | remote, hybrid, onsite. |
| **skills** | `id PK`, `canonical_name`, `normalized_name`, `skill_type`, `category_id`, `parent_skill_id`, `description`, `status`, `version`, `taxonomy_version` | Canonical skill (e.g., JavaScript). |
| **skill_categories** | `id PK`, `name`, `description`, `taxonomy_version` | e.g., programming_language, database, tool, framework. |
| **skill_types** | `id PK`, `name`, `description`, `taxonomy_version` | e.g., language, framework, platform, tool. |
| **skill_aliases** | `id PK`, `skill_id FK`, `alias`, `normalized_alias`, `source` (curated/discovered/llm_suggested), `confidence`, `status` (approved/pending_review/rejected), `taxonomy_version` | All observed spellings/variants; tracks provenance. |
| **skill_relationships** | `id PK`, `source_skill_id FK`, `target_skill_id FK`, `type` (required_for, enhances, …), `strength` (0‑1), `taxonomy_version` | Skill‑to‑skill dependencies (Go → Gin, Docker → Kubernetes). |
| **capabilities** | `id PK`, `name`, `category`, `description`, `taxonomy_version` | Higher‑order abilities (System Design, API Development, Testing, Observability). |
| **domains** | `id PK`, `name`, `parent_id`, `category`, `taxonomy_version` | Business/tech domains (AI‑powered operations → Artificial Intelligence → MLOps). |
| **taxonomy_versions** | `id PK`, `version`, `description`, `effective_from`, `effective_to` | Tracks released taxonomy snapshots. |

### 3.2 Job‑Centric Tables  

| Table | Key Columns | Purpose |
|-------|-------------|---------|
| **jobs** | `id PK`, `canonical_title_id FK`, `company_id FK`, `seniority_id FK`, `job_family_id FK` | Logical job (what the role *is*). |
| **job_postings** | `id PK`, `job_id FK`, `location_id FK`, `employment_type_id FK`, `work_mode_id FK`, `source` (LinkedIn, internal, …), `source_job_id`, `url`, `posted_at`, `scraped_at`, `salary_min`, `salary_max`, `currency`, `dedup_key` | Each appearance of a job; `dedup_key` enables deduplication. |
| **job_descriptions** | `id PK`, `job_posting_id FK`, `raw_text`, `extracted_json JSONB`, `parser_version`, `created_at` | Immutable record of what LLM saw. |
| **job_skills** | `id PK`, `job_id FK`, `skill_id FK`, `requirement_type` (required/preferred/nice_to_have), `requirement_weight` (0‑1), `source_section` (must_have, responsibilities, nice_to_have), `confidence`, `evidence`, `raw_skill_name`, `normalization_version` | Many‑to‑many link; preserves extraction metadata and which taxonomy version was used. |
| **job_skill_groups** | `id PK`, `job_id FK`, `group_label` (e.g., `group_1`), `operator` (ANY/ALL) | Encodes AND/OR logic from the LLM extraction. |
| **job_capabilities** | `id PK`, `job_id FK`, `capability_id FK`, `importance`, `evidence` | Higher‑order skills derived from responsibilities (Phase 2). |
| **job_domains** | `id PK`, `job_id FK`, `domain_id FK`, `importance` | Normalized domain knowledge. |
| **job_soft_skills** (optional) | `id PK`, `job_id FK`, `name`, `evidence`, `confidence` | Soft‑skill storage if needed later. |

### 3.3 Auxiliary / Analytics Tables  

| Table | Purpose |
|-------|---------|
| **market_profiles** | `(job_family_id, seniority_id, location_id, time_period)` → aggregated demand. |
| **market_profile_skills** | FK to market_profiles + `skill_id` + `job_pct`, `required_pct`, `nice_to_have_pct`, `avg_importance`. |
| **market_profile_capabilities** | Same as above but for capabilities. |
| **market_profile_domains** | Same as above but for domains. |
| **normalization_runs** | Log of each batch run (`started_at`, `finished_at`, `rows_processed`, `success_count`, `failure_count`, `taxonomy_version`). |
| **entity_resolution_queue** | Items that fell out of deterministic lookup and need human/AI review (`skill_id` NULL, `raw_value`, `field_type`, `evidence`, `status`, `created_at`). |
| **failed_extractions** | After max retries, rows that repeatedly fail (`extraction_id`, `error_message`, `retry_count`, `failed_at`). |

---  

## 4. Normalization Pipeline (Plugin Core)  

Runs as a background worker pool (or can be triggered on‑demand). The pipeline is **deterministic only**; unresolved items are placed in a review queue for human curation.

### 4.1 Step‑by‑Step Flow  

1. **Claim Pending Extractions** (avoid race conditions)  
   ```sql
   SELECT id, raw_response
   FROM job_extractions
   WHERE status = 'PENDING'
   ORDER BY received_at ASC
   FOR UPDATE SKIP LOCKED
   LIMIT :batchSize;
   ```
   Then update claimed rows to `status = 'PROCESSING'`, `processing_worker_id = $worker_id`, `processing_started_at = NOW()`.

2. **Parse Envelope → ExtractedJobData**  
   - Extract the `data` JSONB field.  
   - Unmarshal into `models.ExtractedJobData`.  

3. **Deterministic Pre‑Normalization** (applied to every field)  
   - **Lower‑case & trim** strings.  
   - **Remove punctuation** (except hyphens/underscores that are part of a token).  
   - **Static alias lookup** (exact match on `skill_aliases.normalized_alias`, `companies.normalized_name`, `locations` (city/state/country), `job_titles.canonical_title`).  
   - If match → use canonical ID, set `confidence = 1.0`, record `match_method = 'exact_alias'`.  
   - **Rule‑based match** (e.g., known patterns for experience, seniority, location).  
   - **Fuzzy candidate generation**: Levenshtein distance ≤ 2 against the same alias tables **only to generate candidates**; the candidate must then be validated by a human‑curated rule (e.g., same skill_type, same category) before acceptance. If not validated, item goes to review queue.  

4. **Resolution Outcome**  
   - **Found** → canonical ID, confidence = 1.0 (or rule‑based confidence).  
   - **Not found** → leave canonical ID NULL, set `confidence = 0.0`, and insert a row into `entity_resolution_queue` with the raw value, field_type, evidence, and source extraction ID.  

5. **Build NormalizedJobData** (pure Go struct)  
   - Populate all fields using canonical IDs/names obtained above (NULL if unresolved).  
   - Preserve original `evidence` and the confidence from step 3 (deterministic = 1.0, rule‑based = rule confidence).  

6. **Persist (Transactional)** – inside a single `BEGIN … COMMIT` block  
   - Insert into `jobs` (store `normalization_version` from current taxonomy). Obtain `job_id`.  
   - Insert into `job_postings` (link to `job_id`).  
   - Insert into `job_descriptions` (store raw text + extracted JSON).  
   - For each skill in `extractedJobData.Skills`:  
     - Resolve `skill_id` via alias lookup (or NULL if unresolved).  
     - Insert a row into `job_skills` (with `requirement_type`, `weight`, `source_section`, `confidence`, `evidence`, `raw_skill_name`, `normalization_version`).  
     - If skill appeared in a requirement group (`group_1`, `group_2`, …) store that info in `job_skill_groups`.  
   - Persist responsibilities → `job_capabilities` (Phase 2).  
   - Persist domain knowledge → `job_domains`.  
   - (Optional) Persist soft skills → `job_soft_skills`.  
   - Update source row:  
     ```sql
     UPDATE job_extractions
        SET status = CASE WHEN err IS NULL THEN 'PROCESSED' ELSE 'FAILED' END,
            processed_at = NOW(),
            error_message = COALESCE(err, NULL),
            processing_worker_id = NULL,
            processing_started_at = NULL
      WHERE id = $extractionID;
     ```  

7. **Emit Metrics & Logs** (structured JSON)  
   - Counters: `extractions_processed_total`, `skills_normalized_total`, `entities_sent_to_review_total`, `ai_fallback_calls_total` (should be 0), `new_entities_approved_total`.  
   - Histograms: `processing_latency_seconds`.  
   - Log each batch with counts and any errors.  

### 4.2 Concurrency & Reliability  

| Concern | Solution |
|---------|----------|
| **Parallelism** | Worker pool of `N` goroutines (`MAX_CONCURRENT_JOBS` from config). Each pulls its own batch via `FOR UPDATE SKIP LOCKED`, processes, commits. |
| **Idempotency & Claiming** | Explicit status flow: `PENDING → PROCESSING → {PROCESSED, FAILED}` with worker ID and timestamps prevents double‑processing. |
| **Retries** | On transient DB error, row left in `PROCESSING`; after `MAX_RETRY_ATTEMPTS` (checked via `retry_count`), move to `FAILED` and log to `failed_extractions`. |
| **Dead‑letter** | After max retries, move row to `failed_extractions` table for manual inspection. |
| **Graceful shutdown** | On SIGTERM, stop accepting new batches, wait for in‑flight jobs to finish (ctx cancellation). |

### 4.3 Accuracy‑Boosting Techniques (to hit > 95 % precision)  

- **Seed alias dictionaries** from existing taxonomy (skills, companies, locations, titles).  
- **Fuzzy matching as candidate generator only** – never auto‑accept; requires rule‑based validation or human review.  
- **Contextual evidence weighting** – if evidence contains “expert in X” or “strong proficiency with X”, boost rule‑based confidence for X.  
- **Confidence threshold** – only accept rule‑based suggestion if confidence ≥ 0.90 (deterministic exact matches are 1.0).  
- **Periodic bulk re‑normalization** – nightly job that re‑processes all rows with `status='PROCESSED'` using the latest taxonomy version to detect drift.  
- **Human‑in‑the‑loop review queue** – entities inserted with `status='pending_review'` are surfaced in an internal UI for curation; once approved, they flip to `status='approved'` and the corresponding alias/skill records are updated.  
- **Provenance tracking** – each normalization decision records `match_method`, `source` (curated/discovered/llm_suggested), `taxonomy_version`, and `evidence`.  

---  

## 5. Required Raw Data Volume  

| Phase | Approx. # of raw extractions needed | Reason |
|-------|-----------------------------------|--------|
| **Initial Seed** | **1 k – 3 k** diverse JD samples | Sufficient to build initial alias map and test deterministic rules. Diversity (different companies, job boards, countries, seniority, job families) matters more than raw count. |
| **Iterative Tuning** | **Additional 5 k – 15 k** over first 2‑4 weeks | Discover new variants, refine rules, curate via review queue. |
| **Production Steady‑State** | **Continuous** – plugin improves as more data flows in. | No hard upper bound; the system learns from every new extraction via the review queue. |

---  

## 6. API Changes  

### 6.1 Optional Synchronous Normalization (Immediate Feedback)  

Modify `backend/internal/processor/handler.go` after storing the raw extraction:  

```go
// OPTIONAL: run normalization synchronously for immediate feedback
summary, err := normSvc.ProcessSingleExtraction(ctx, extractedID)
if err != nil {
    log.Warnf("sync normalization failed for %s: %v", extractedID, err)
}
// Include `summary` in the JSON response if desired
```

`normSvc.ProcessSingleExtraction` executes steps 2‑6 of the pipeline on a single UUID and returns a map like:

```json
{
  "job_id": "c4f8…",
  "title": { "canonical": "Backend Engineer", "confidence": 0.99 },
  "company": { "canonical": "Cognizant", "confidence": 1.0 },
  "location": { "city": "Bangalore Urban", "state": "Karnataka", "country": "India", "confidence": 1.0 },
  "skills": [
    { "canonical": "TypeScript", "requirement": "required", "confidence": 1.0 },
    { "canonical": "React",       "requirement": "required", "confidence": 1.0 },
    { "canonical": "Vue.js",      "requirement": "nice_to_have", "confidence": 1.0 }
  ]
  // … responsibilities, domains, etc. omitted for brevity (Phase 2)
}
```

If you prefer **purely asynchronous** processing, skip step 2; the background worker will eventually canonicalize the row and clients can poll a read endpoint (`GET /jobs/{id}`) to see when it’s ready.

### 6.2 New Read Endpoint  

```
GET /api/v1/jobs/{job_id}
```

Returns the fully normalized job (including linked skills, responsibilities, domains) as JSON. Implemented via a thin repository layer that joins `jobs → job_skills → skills`, etc.

### 6.3 Admin Batch Endpoint (Optional)  

```
POST /api/v1/admin/normalize
{
  "batchSize": 500   // optional, defaults to config
}
```

Runs `normSvc.ProcessPendingExtractions` once and returns `{ processed: N }`. Useful for manual back‑fills or after a schema change.

All new endpoints are protected by the existing authentication/guardrail middleware (same as other internal routes).  

---  

## 7. Implementation Phases (Re‑ordered around Data Architecture)

| Phase | Description | Files to Create / Modify | Estimated Effort |
|-------|-------------|--------------------------|------------------|
| **0 – Freeze Extraction Contract** | Verify the current LLM JSON envelope (`ExtractedJobData`) is stable. Create a validation suite with 100–500 real examples. | No new files; just document contract. | 0.5 day |
| **1 – Build Core Taxonomy** | Create tables for `skill_categories`, `skill_types`, `skills`, `skill_aliases`, `companies`, `locations`, `job_titles`, `seniority_levels`, `job_families`, `employment_types`, `work_modes`, `taxonomy_versions`. Seed with initial data from existing taxonomy or CSV. | Migrations `001_taxonomy.up/down.sql`; repository files (`internal/repository/skill.go`, `company.go`, `location.go`, `title.go`, …). | 3 days |
| **2 – Deterministic Normalization Engine** | Build lookup service: exact/alias/rule matching, fuzzy candidate generation, review queue insertion. No AI fallback. | `internal/normalization/lookup.go`, `internal/normalization/service.go` (deterministic only), unit tests. | 4 days |
| **3 – Test Normalization on Seed Data** | Run service against 1 k–3 k raw extractions, collect metrics, inspect review queue, improve taxonomy and rules. | Update lookup rules, add missing aliases, adjust `skill_aliases.source`/`status`. | 2 days |
| **4 – Job‑Centric Schema & Persistence** | Add `jobs`, `job_postings`, `job_descriptions`, `job_skills`, `job_skill_groups`, `job_capabilities` (stub), `job_domains`, `job_soft_skills` (optional). Implement repository `InsertTx` helpers. | Migration `002_job_tables.up/down.sql`; repository files (`job.go`, `job_skill.go`, `job_responsibility.go`, …). | 4 days |
| **5 – Deduplication Layer** | Add `dedup_key` (hash of source_job_id, company, title, location, URL, description fingerprint) to `job_postings`. Implement deduplication before inserting into `jobs` (or mark duplicates). | Update `job_postings` table, modify persistence logic. | 1 day |
| **6 – Worker Pool & Claiming Mechanism** | Implement `PROCESSING` status, `FOR UPDATE SKIP LOCKED`, worker ID, retry handling, dead‑letter table. | `internal/server/worker.go`, modifications to `server.go` to start/stop worker, update `normalization_runs` table. | 2 days |
| **7 – API Enrichment (Optional Sync)** | Modify `processor/handler.go` to call the normalization service synchronously and return a canonical summary. | `internal/processor/handler.go` (add sync call), update tests. | 1 day |
| **8 – Read Endpoint** | Implement `GET /api/v1/jobs/{id}` (and optional list endpoint) with pagination/filtering. | `internal/api/handlers/job.go`, `internal/api/routes.go`, repository read functions. | 2 days |
| **9 – Admin Batch Endpoint** | `POST /api/v1/admin/normalize` (protected). | `internal/admin/handler.go`, route addition. | 1 day |
|**10 – Responsibilities → Capabilities (Phase 2)**| Build capability taxonomy, map responsibility verbs to capabilities, persist `job_capabilities`. | New taxonomy tables, normalization updates, repository updates. | 3 days |
|**11 – Market Profile Aggregation**| Create `market_profiles`, `market_profile_*`, nightly aggregation jobs. | Migrations, aggregation service. | 2 days |
|**12 – Tests & Benchmarks**| Unit tests for each layer, integration test with testcontainers Postgres, benchmark (≥ 10k rows/sec). | `*_test.go` files, `scripts/bench.sh`. | 3 days |
|**13 – Documentation**| Update `README.md` with plugin overview, API spec, deployment instructions, and seed‑data guidance. | `README.md`, optionally `docs/plugin_overview.md`. | 1 day |
|**Total**| **≈ 28 days** (≈ 5‑6 weeks part‑time) – phases can overlap where independent. | — | — |

---  

## 8. Configuration & Environment Variables  

| Variable | Description | Default |
|----------|-------------|---------|
| `JOB_POLL_INTERVAL` | How often the worker polls for new extractions (e.g., `30s`, `1m`). | `1m` |
| `MAX_CONCURRENT_JOBS` | Number of worker goroutines. | `4` |
| `BATCH_SIZE` | Number of rows fetched per poll. | `100` |
| `MAX_RETRY_ATTEMPTS` | How many times to retry a failed extraction before moving to dead‑letter. | `3` |
| `FUZZY_DISTANCE_LIMIT` | Levenshtein distance for deterministic fuzzy candidate generation. | `2` |
| `NORMALIZATION_CONFIDENCE_THRESHOLD` | Minimum confidence to accept a rule‑based suggestion (deterministic exact = 1.0). | `0.90` |
| `LLM_PROVIDER`, `LLM_MODEL`, `LLM_API_KEY` | Re‑used from existing config (no new LLM call needed for extraction). | — |
| `ENABLE_SYNC_NORMALIZATION` | If `true`, the `/process/job` handler runs normalization synchronously and returns a summary. | `false` |

All variables are loaded via the existing `config.Load()` function (extend `ProcessingConfig` struct).  

---  

## 9. Testing Strategy  

| Test Type | Scope | Tools |
|-----------|-------|-------|
| **Unit** | Lookup deterministic maps, fuzzy candidate generation, envelope parsing, repository insert helpers. | `testing` + `testify` |
| **Integration** | Spin up a test Postgres (testcontainers), insert a few raw extractions with known JSON, run the worker (or call the service directly), assert that canonical tables contain expected rows and that unresolved items go to `entity_resolution_queue`. | `testcontainers/postgres`, `go test` |
| **Benchmark** | Feed 10 k–50 k synthetic extractions through the service, measure latency and throughput. | `testing.B` + `pprof` for profiling |
| **End‑to‑end (optional)** | Hit the real `/process/job` endpoint with a plain‑text JD, wait for the worker (or call sync mode), verify the read endpoint returns the normalized job. | `net/http/httptest` |
| **Review Queue** | Insert a known “unknown” skill, ensure it lands in `entity_resolution_queue` with `status='pending_review'`. | Manual test + UI (if built) |
| **Taxonomy Versioning** | Insert a new taxonomy version, run re‑normalization, verify `normalization_version` on updated rows. | Manual test |

CI pipeline runs unit + integration tests on every PR; benchmark runs nightly.  

---  

## 10. Deployment & Observability  

* **Docker** – The plugin runs inside the same backend container; no extra image needed.  
* **Logging** – Structured JSON logs (`zap`) with fields: `extraction_id`, `step`, `status`, `latency_ms`, `error`, `match_method`, `taxonomy_version`.  
* **Metrics** – Prometheus counters (`extractions_processed_total`, `skills_normalized_total`, `entities_sent_to_review_total`, `new_entities_approved_total`) and histograms (`normalization_latency_seconds`). Expose at `/metrics` (already present in the backend).  
* **Health** – The worker updates a `normalization_runs` row on each successful batch; a simple `SELECT MAX(finished_at) FROM normalization_runs` can be used in a health check.  

---  

## 11. Risks & Mitigations  

| Risk | Impact | Mitigation |
|------|--------|------------|
| **LLM provider cost** (if AI fallback is used heavily) | Increased operational cost. | AI fallback is **removed** from the normalization path; only deterministic + rule‑based matching is used. Cost stays at extraction only. |
| **Schema drift** (adding new columns later) | Migration complexity. | Use a migration tool (goose/golang‑migrate) that supports down migrations; keep all changes backward‑compatible where possible (add nullable columns). |
| **Incorrect canonical mapping** (e.g., merging two distinct skills) | Downstream analytics become inaccurate. | Require human review for any new entity (`status='pending_review'`); maintain a skill‑relationship graph to detect odd merges; run periodic sanity‑check jobs that sample random rows and compare against a curated validation set. |
| **Worker backlog** if inflow outpaces processing. | Increased latency, possible `processed_at` lag. | Autoscale `MAX_CONCURRENT_JOBS` based on queue size (simple rule: if `SELECT COUNT(*) FROM job_extractions WHERE status='PENDING' > THRESHOLD`, increase workers); expose lag metric (`extraction_age_seconds`). |
| **Data loss due to transaction failure** | Missing job records. | All persistence is inside a transaction; on error we roll back and leave the extraction row in `PROCESSING` (or `PENDING` after timeout) so it will be retried. |
| **Insufficient diversity in seed data** | Poor alias coverage, high unknown rate. | Ensure seed data spans multiple companies, job boards, geographies, seniority levels, and job families. |

---  

## 12. Summary  

* A **self‑contained plugin** (`internal/normalization/…`) that turns raw LLM JSON into a highly accurate canonical relational model **without** using the LLM during normalization.  
* **Deterministic + rule‑based** normalization only; unresolved items go to a review queue for human curation.  
* **Configurable background worker** with explicit claiming (`FOR UPDATE SKIP LOCKED`) guarantees exactly‑once processing.  
* **Optional synchronous API enrichment** for immediate feedback.  
* **Read‑only API endpoints** to inspect the normalized data.  
* **Clear success metrics** covering coverage, precision, false merge/split rates, latency, and freshness.  
* **Guidance on required raw data volume** (≈ 1‑3 k to seed, then continuous improvement via review queue).  

Implementing the phases above will give you a robust, production‑ready data‑extraction foundation that can later be leveraged by any downstream feature (gap analysis, road‑mapping, job matching, etc.) without worrying about noisy or duplicated skill names, title variations, or location inconsistencies.  

---  

*Prepared for the Passo backend team – August 2026.*