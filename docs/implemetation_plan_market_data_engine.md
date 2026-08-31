
How to Use This Roadmap
Treat the roadmap as the single source of truth for what to build, where to put code, and what constitutes “done” for each phase.
Work phase‑by‑phase – only start the next phase when the current phase’s exit criteria are met.
If you need to pause, note the last completed phase and the exact file/line you left off at; you can resume by re‑running the checklist for that phase (or the next one if the phase is fully complete).
After each phase, run the verification steps listed under “Exit Criteria”. If they pass, move on; if they fail, fix the issues before proceeding.
Phase 0 – Project Setup & Baseline Extraction Contract
Goal	Prepare the repo, lock down the LLM‑extraction JSON shape, and provision a dev database.
Files to create / modify	- docker-compose.yml (ensure Postgres service exists) <br> - Makefile (add make db-up, make db-down, make migrate-up, make migrate-down) <br> - go.mod (tidy, ensure github.com/jackc/pgx/v5 present) <br> - docs/seeds/README.md (placeholder for seed CSV files)
Tasks	1. Spin up a local Postgres via docker-compose up -d postgres.<br>2. Verify the /process/job endpoint works (LLM extraction → stores a row in job_extractions).<br>3. Write a small contract test that saves a known raw JSON snippet and asserts the envelope shape (success, message, job_id, data JSONB).<br>4. Document the exact envelope format in docs/market_data_engine_api.md (under Extraction Envelope).
Deliverables	- Running Postgres instance reachable at postgres://postgres:password@localhost:5432/passo?sslmode=disable.<br>- A passing contract test (go test ./internal/processor -run TestEnvelopeShape).
Exit Criteria	1. make db-up starts a healthy Postgres container.<br>2. The contract test passes.<br>3. The extraction endpoint stores a row with the expected columns (id, job_id, provider, model, prompt_version, raw_response JSONB, created_at).
How to Resume	If stopped after step 2, re‑run docker-compose up -d and re‑run the contract test. If stopped after step 3, just re‑run the test suite.
Phase 1 – Build Core Taxonomy Tables
Goal	Create the reference tables that will hold canonical entities (skills, companies, locations, titles, families, etc.) and version them.
Files to create / modify	Migrations: <br> - backend/migrations/001_taxonomy.up.sql <br> - backend/migrations/001_taxonomy.down.sql <br> Repositories (under backend/internal/repository/): <br> - skill.go <br> - skill_category.go <br> - skill_type.go <br> - company.go <br> - location.go <br> - job_title.go <br> - job_family.go <br> - seniority_level.go <br> - employment_type.go <br> - work_mode.go <br> - taxonomy_version.go <br> Config: extend ProcessingConfig in backend/internal/config/config.go (add placeholders for taxonomy version if needed).
Tasks	1. Write the UP migration (see appendix A for full SQL).<br>2. Write the matching DOWN migration (drop tables in reverse order).<br>3. Run make migrate-up to apply the schema to the dev DB.<br>4. Implement minimal CRUD functions in each repository (Insert, GetByID, List) – only what the normalization service needs initially (mostly Insert + GetByID/GetByNormalizedName).<br>5. Seed the tables: <br> • Import Neha’s CSV files (docs/seeds/companies.csv, docs/seeds/locations.csv) using a small Go seed script (cmd/seed/seed.go) or plain psql \copy.<br> • Insert a few canonical skill rows manually (e.g., JavaScript, Python, Go, React, Vue.js) to have something to test against.<br> • Insert one taxonomy_versions row (version = '1.0.0', effective_from = now()).
Deliverables	- All taxonomy tables exist in the dev DB.<br>- Repositories can insert and retrieve rows without error.<br>- Seed data loaded (≥ 200 companies, ≥ 200 locations, ≥ 10 core skills).
Exit Criteria	1. make migrate-up runs with zero errors.<br>2. SELECT count(*) FROM companies; returns ≥ 200.<br>3. SELECT count(*) FROM locations; returns ≥ 200.<br>4. SELECT count(*) FROM skills; returns ≥ 10.<br>5. A simple Go program can call repo.CompanyInsert(ctx, …) and get back a UUID.
How to Resume	If you stopped after writing the migration files, just run make migrate-up again. If stopped after seeding, re‑run the seed script or \copy commands – they are idempotent if you add ON CONFLICT DO NOTHING on the unique keys (e.g., canonical_name).
Phase 2 – Deterministic Normalization Engine (Lookup Service)
Goal	Build a pure‑Go service that, given a raw string (skill, title, company, location, etc.), returns the canonical ID or signals “unknown” (to be sent to the review queue). No AI fallback here.
Files to create / modify	- backend/internal/normalization/lookup.go (core deterministic logic) <br> - backend/internal/normalization/service.go (high‑level Normalize* methods) <br> - backend/internal/normalization/llm_client.go – stub (we will not call it, but keep the interface for future Phase 2 if needed) <br> - Unit test file: backend/internal/normalization/lookup_test.go <br> - Update backend/internal/config/config.go to expose FUZZY_DISTANCE_LIMIT and NORMALIZATION_CONFIDENCE_THRESHOLD.
Tasks	1. Lookup struct: holds in‑memory maps loaded at startup from the DB (e.g., map[string]uuid.UUID skillAliasToID, map[string]uuid.UUID companyNormToID, etc.). Load these maps in a Load() method called from service.go.NewService.<br>2. Normalization steps per field (example for a skill): <br> a. Lower‑case & trim input.<br> b. Remove punctuation except hyphen/underscore.<> c. Exact match in alias map → return canonical ID, confidence = 1.0, match_method = exact_alias.<br> d. If not exact, generate fuzzy candidates using Levenshtein distance ≤ FUZZY_DISTANCE_LIMIT (2) against the alias map.<br> e. For each candidate, run a rule validator (e.g., same skill_type/category as the original raw token – you can store a simple rule like “if raw ends with ‘js’ and candidate is JavaScript, accept”).<br> f. If any candidate passes validation → return its ID, confidence = 0.9 (or a configurable threshold), match_method = fuzzy_validated.<br> g. If none pass → return (nil, 0.0, match_method='unknown').<br>3. Implement similar pipelines for title, company, location, seniority, employment_type, work_mode (each with its own alias map and rule set).<br>4. Write comprehensive unit tests covering: exact match, alias match, fuzzy candidate that passes validation, fuzzy candidate that fails validation, empty string, punctuation variations.<br>5. Benchmark the lookup (testing.B) – aim for ≥ 100 k lookups/sec on a dev laptop.
Deliverables	- Lookup struct with methods NormalizeSkill(raw string, evidence string) (skillID *uuid.UUID, confidence float64, method string).<br>- Corresponding methods for Title, Company, Location, Seniority, EmploymentType, WorkMode.<br>- A NormalizationService that composes the lookup and provides a high‑level NormalizeJob(extracted models.ExtractedJobData) (norm models.NormalizedJobData, unknowns []NormalizationUnknown).<br>- All unit tests passing (go test ./internal/normalization).<br>- Benchmark showing ≥ 100 k lookups/sec.
Exit Criteria	1. All unit tests pass.<br>2. Benchmark meets the latency target (≤ 2 ms per 1 k lookups).<br>3. A small manual test (e.g., normalize "JS", "Javascript", "ECMAScript" → same skill ID) works as expected.<br>4. Unknown strings (e.g., "asdfqwer") return (nil, 0.0, 'unknown').
How to Resume	If you stopped after writing lookup.go, just re‑run the unit tests (go test ./internal/normalization). If stopped after writing the service methods, re‑run the tests; the service depends only on the lookup, so you can incrementally add methods and test each.
Phase 3 – Seed Testing & Taxonomy Refinement
Goal	Run the deterministic normalization on a realistic seed of raw extractions, collect metrics, and improve the taxonomy/rules until we hit the target coverage/precision.
Files to create / modify	- A temporary seed loader script (cmd/seed/load_raw.go) that reads a JSON file of raw extractions (you can export a few rows from job_extractions or craft them manually).<br>- Update backend/internal/normalization/service.go to expose a ProcessBatch(ctx context.Context, limit int) (processed int, unknowns []NormalizationUnknown, err error) function (still deterministic, no persistence yet).
Tasks	1. Prepare 2 k–3 k raw extraction samples (copy from existing job_extractions or write a few representative JD snippets and run them through the LLM endpoint to get the raw JSON). Store them as a JSON array in docs/seeds/raw_extractions_seed.json.<br>2. In load_raw.go: read the file, for each entry parse the envelope, extract the data JSONB into an ExtractedJobData, call normSvc.NormalizeJob, and collect: <br> • How many skills/titles/etc. resolved to a canonical ID.<br> • How many went to unknown (by field).<br> • Confidence distribution.<br>3. Print a summary after the batch: overall skill coverage, precision (by manually checking a random 5 % sample), unknown rate.<br>4. Based on the report, add missing aliases to skill_aliases (or companies/locations) via SQL INSERT … ON CONFLICT DO NOTHING.<br>5. Refine any rule validators that are too strict/loose (e.g., adjust the fuzzy validation for “Node” vs. “Node.js”).<br>6. Iterate until: <br> • Skill coverage ≥ 95 % (unique raw → canonical).<br> • Skill precision ≥ 98 % (manual audit of a 100‑sample subset).<br> • Unknown rate ≤ 5 %.
Deliverables	- Seed file with raw extractions.<br> - A runnable seed‑tester (go run cmd/seed/load_raw.go).<br> - Updated taxonomy (more aliases, refined rules).
Exit Criteria	1. Running the seed tester shows skill coverage ≥ 95 % and precision ≥ 98 % (you can do a quick manual check on a sample).<br>2. Unknown rate ≤ 5 %.<br>3. No regression in unit tests.
How to Resume	If you stopped after preparing the seed file, just run the loader again. If stopped after updating taxonomy, re‑run the loader to see if metrics improved.
Phase 4 – Job‑Centric Schema & Persistence Layer
Goal	Create the tables that store the canonical job, posting, description, skills, responsibilities (stub), capabilities (stub), domains, and soft‑skills. Add transactional repository helpers.
Files to create / modify	Migrations: <br> - backend/migrations/002_job_tables.up.sql <br> - backend/migrations/002_job_tables.down.sql <br> Repositories (under backend/internal/repository/): <br> - job.go <br> - job_posting.go <br> - job_description.go <br> - job_skill.go <br> - job_skill_group.go <br> - job_capability.go (stub) <br> - job_domain.go <br> - job_soft_skill.go (optional) <br> - Update existing repository files to add InsertTx methods that accept a pgx.Tx.
Tasks	1. Write the UP migration (see Appendix B for full SQL – includes dedup_key, foreign keys, indexes, and normalization_version columns).<br>2. Write the matching DOWN migration.<br>3. Apply with make migrate-up.<br>4. For each repository, implement: <br> • Insert(ctx, …) (uuid.UUID, error) (simple wrapper that starts a tx and calls InsertTx).<br> • InsertTx(ctx tx pgx.Tx, …) (uuid.UUID, error) – the actual INSERT statement using positional parameters ($1,$2,…).<br> • Optional GetByID, ListByJobID, etc., as needed for later phases (you can stub them now).<br>5. Ensure each INSERT returns the generated UUID (RETURNING id).<br>6. Add a helper function normalizeAndPersist(ctx context.Context, tx pgx.Tx, extracted models.ExtractedJobData, norm models.NormalizedJobData, extractionID uuid.UUID) error inside service.go that uses the repositories to persist everything in a single transaction.
Deliverables	- All job‑centric tables exist in dev DB.<br>- Repository InsertTx methods work without error.<br>- The service can persist a full normalized job (including skills) in one transaction.
Exit Criteria	1. make migrate-up applies cleanly.<br>2. SELECT count(*) FROM jobs; starts at 0 (table ready).<br>3. A unit test (TestPersistJob) can: <br> • Build a fake ExtractedJobData.<br> • Call the service’s internal persist function inside a tx.<br> • Verify a row appears in jobs, at least one row in job_skills, and the source job_extractions row gets status='PROCESSED'.<br>4. No foreign‑key violations when inserting.
How to Resume	If you stopped after writing the migration files, just rerun make migrate-up. If stopped after writing a specific repository (e.g., job_skill.go), implement its InsertTx and run the persistence unit test for that table alone; then continue with the next repository.
Phase 5 – Deduplication at Ingest
Goal	Prevent duplicate job postings from inflating metrics by adding a dedup_key and using INSERT … ON CONFLICT DO NOTHING.
Files to modify	- backend/migrations/002_job_tables.up.sql (add dedup_key column and unique index).<br>- backend/internal/repository/job_posting.go (modify the Insert logic).
Tasks	1. In the UP migration, add: <br> dedup_key CHAR(64) NOT NULL, <br> UNIQUE (dedup_key) <br> and a trigger‑less generation: we will compute the key in Go before inserting (SHA‑256 of `source_job_id
Deliverables	- job_postings table has a unique dedup_key.<br>- Insert logic correctly handles new vs. duplicate postings.
Exit Criteria	1. Unit test TestJobPostingDeduplication passes.<br>2. Manual test: insert two raw extractions with identical source data → only one posting, two descriptions.
How to Resume	If you stopped after editing the migration, re‑run make migrate-up (the migration is additive; it won’t break if run again). If stopped after editing job_posting.go, just re‑run the deduplication unit test.
Phase 6 – Worker Pool with Explicit Claiming
Goal	Run a configurable pool of goroutines that claim pending extractions, invoke the normalization service, persist the result, and update the source row’s status.
Files to create / modify	- backend/internal/server/worker.go (new file) <br> - backend/internal/server/server.go (start/stop worker) <br> - Update backend/internal/config/config.go to add worker‑related fields (JOB_POLL_INTERVAL, MAX_CONCURRENT_JOBS, BATCH_SIZE, MAX_RETRY_ATTEMPTS). <br> - Add metrics initialization in backend/internal/server/metrics.go (or create it).
Tasks	1. Define constants from config (load once at startup).<br>2. Worker struct holds: <br> • db *db.DB (the wrapper that exposes Pool()).<br> • normSvc *normalization.Service.<br> • jobRepo *repository.JobRepository, jobPostingRepo *repository.JobPostingRepository, jobDescRepo *repository.JobDescriptionRepository, jobSkillRepo *repository.JobSkillRepository, etc.<br> • metrics *prometheus.<br>3. Main loop (Run(ctx context.Context)): <br> a. Build the SELECT … FOR UPDATE SKIP LOCKED query with LIMIT $batchSize.<br> b. Execute, receive rows (extractionID, rawResponse).<br> c. For each row in a goroutine (bounded by a semaphore of size MAX_CONCURRENT_JOBS): <br>  i. Parse envelope → ExtractedJobData.<br>  ii. Call normSvc.NormalizeJob → (norm, unknowns).<br>  iii. If len(unknowns) > 0: insert each unknown into entity_resolution_queue (store raw value, field_type, evidence, extractionID).<br>  iv. Persist the normalized job via the transaction helper from Phase 4 (inside a tx).<br>  v. On success: update job_extractions SET status='PROCESSED', processed_at=NOW(), processing_worker_id=NULL, processing_started_at=NULL.<br>  vi. On any error: increment retry_count (store in a hidden column we add to job_extractions – see note below); if retry_count >= MAX_RETRY_ATTEMPTS: move to failed_extractions; else set status back to PENDING (so it will be retried).<br>  vii. Emit Prometheus counters (extractions_processed_total, skills_normalized_total, entities_sent_to_review_total, new_entities_approved_total (if any unknowns later approved), failed_extractions_total).<br>  viii. Record latency histogram.<br>4. Add columns to job_extractions (if not already present) via a small migration (003_job_extractions_columns.up.sql): <br> • processing_worker_id UUID, <br> • processing_started_at TIMESTAMPTZ, <br> • retry_count INT DEFAULT 0, <br> • next_retry_at TIMESTAMPTZ (optional).<br>5. Metrics: define in metrics.go – <br> • extractionsProcessed counter.<br> • skillsNormalized counter.<br> • entitiesSentToReview counter.<br> • failedExtractions counter.<br> • normalizationLatency histogram.<br> • extractionsPending gauge (SELECT COUNT(*) FROM job_extractions WHERE status='PENDING').<br>6. Health check: add a handler in server.go (/healthz/normalization) that returns 200 if the most recent row in normalization_runs (inserted by the worker at the start/end of each batch) has finished_at within the last 5 minutes and extractionsPending gauge < 300.
Deliverables	- Worker pool running as goroutines inside the backend.<br>- Correct status transitions and retry/dead‑letter handling.<br>- Metrics exposed at /metrics.<br>- Health endpoint /healthz/normalization.
Exit Criteria	1. The worker starts without panic when the server launches (go run ./cmd/server).<br>2. Insert a raw extraction row manually (status='PENDING') and watch the worker process it: after a few seconds the row’s status becomes PROCESSED and a row appears in jobs and job_skills.<br>3. Metrics endpoint shows non‑zero counters after processing.<br>4. Health endpoint returns 200 after a successful batch.<br>5. Unit test for the worker (backend/internal/server/worker_test.go) can mock the DB and service and verify the flow.
How to Resume	If you stopped after editing worker.go, just compile and run the server – the worker will start automatically. If you stopped after adding the migration columns, re‑run make migrate-up to add them, then restart the server.
Phase 7 – Optional Synchronous Normalization (API Enrichment)
Goal	Allow the /process/job handler to optionally run normalization synchronously and return a lightweight canonical summary in the HTTP response (off by default).
Files to modify	- backend/internal/processor/handler.go <br> - backend/internal/config/config.go (add ENABLE_SYNC_NORMALIZATION bool).
Tasks	1. In config.go add a field EnableSyncNormalization bool with default false (read from env var ENABLE_SYNC_NORMALIZATION).<br>2. In handler.go, after storing the raw extraction and obtaining extractionID, check cfg.Processing.EnableSyncNormalization. If true: <br> a. Call normSvc.ProcessSingleExtraction(ctx, extractionID) (a thin wrapper around the service’s batch logic with limit = 1).<br> b. If the call returns an error, log it but do not change the HTTP response status – the extraction is still stored successfully.<br> c. Marshal the returned summary (see the example in the plan) and add it to the JSON response under a new top‑level key canonical.<br>3. Write a unit test that toggles the flag and asserts the presence/absence of the canonical field.
Deliverables	- The API behaves exactly as before when the flag is false.<br>- When the flag is true, the response includes a normalized summary (title, company, location, skills list).
Exit Criteria	1. Unit test passes for both flag values.<br>2. Manual test: set ENABLE_SYNC_NORMALIZATION=true, call /process/job with a sample JD, verify the JSON contains "canonical":{...}.<br>3. No regression in existing extraction tests.
How to Resume	If you stopped after editing the handler, just re‑run the relevant unit test (go test ./internal/processor -run TestSyncNormalization).
Phase 8 – Read‑Only Job Endpoint
Goal	Provide a way for clients (or internal services) to fetch a fully normalized job with its skills, responsibilities, domains, etc.
Files to create / modify	- backend/internal/api/handlers/job.go (new) <br> - backend/internal/api/routes.go (add the route) <br> - Repository read functions: JobGetByID, JobSkillsGetByJobID, JobCapabilitiesGetByJobID, JobDomainsGetByJobID (you can implement simple SELECT … WHERE job_id = $1).
Tasks	1. Define the response struct (mirrors the normalization output but includes arrays of skills, responsibilities, domains).<br>2. Implement JobGetByID that joins jobs → job_postings → job_descriptions (optional) and returns the core fields.<br>3. Implement separate functions to fetch related data (job_skills, job_capabilities, job_domains, job_soft_skills).<br>4. In the handler (GetJob), call those functions, assemble the response, and write JSON.<br>5. Add the route in routes.go: mux.HandleFunc("GET /api/v1/jobs/{job_id}", middleware(authMiddleware(jobHandler.GetJob))).<br>6. Write unit tests that: insert a known job via the persistence helper, then call the handler via httptest.NewRecorder and assert the JSON matches expectations.
Deliverables	- GET /api/v1/jobs/{id} returns a 200 with a JSON body containing canonical title, company, location, skills array (each with canonical name, requirement, confidence, evidence), responsibilities array, domains array, etc.<br>- The endpoint is protected by the same auth middleware as other internal routes.
Exit Criteria	1. Unit test TestGetJobHandler passes.<br>2. Manual test: after processing a known extraction via the worker (or sync flag), call the endpoint and verify you get back the expected canonical data.<br>3. 401/403 if not authenticated, 404 if job_id does not exist.
How to Resume	If you stopped after writing the handler, just re‑run the test. If stopped after adding the route, restart the server (the route is registered at startup).
Phase 9 – Admin Batch Trigger
Goal	Give operators (or cron jobs) a manual way to kick off a normalization batch on demand.
Files to modify	- backend/internal/admin/handler.go (add new method) <br> - backend/internal/api/routes.go (add route)
Tasks	1. In admin/handler.go, add HandleNormalizeBatch(w http.ResponseWriter, r *http.Request).<br> a. Parse optional query param batchSize (int, default to config’s BATCH_SIZE).<br> b. Call normSvc.ProcessPendingExtractions(r.Context(), batchSize).<br> c. Return JSON { "processed": N }.<br>2. Protect the handler with middleware(authMiddleware(adminMiddleware(adminHandler.HandleNormalizeBatch))).<br>3. Add the route in routes.go: mux.HandleFunc("POST /api/v1/admin/normalize", middleware(authMiddleware(adminMiddleware(adminHandler.HandleNormalizeBatch)))).<br>4. Write a unit test that calls the handler with a mock service and verifies the JSON response.
Deliverables	- Admin endpoint that runs a batch and returns the number of extractions processed.
Exit Criteria	1. Unit test passes.<br>2. Manual test: using curl with a valid admin token, hit the endpoint and see a JSON response with processed ≥ 0.<br>3. No errors in server logs.
How to Resume	If you stopped after editing the handler, just recompile and restart the server; the new route will be live.
Phase 10 – Responsibilities → Capabilities (Phase 2 Placeholder)
Goal	Add stub tables/columns for future capability mapping (no implementation yet).
Files to modify	- backend/migrations/002_job_tables.up.sql (add nullable capability_id to job_capabilities). <br> - backend/internal/repository/job_capability.go (add Insert stub).
Tasks	1. In the UP migration, ensure job_capabilities has capability_id UUID NULL.<br>2. Implement a minimal Insert function (you can leave it as a no‑op for now; it will be filled in Phase 2).<br>3. No DB changes are needed beyond the column addition.
Deliverables	- Table ready to hold capability links when Phase 2 work begins.
Exit Criteria	1. Migration applies cleanly.<br>2. The repository compiles (no syntax errors).<br>3. No runtime errors when the Insert stub is called (it can just return uuid.Nil, nil for now).
How to Resume	If you stopped after editing the migration, just re‑run make migrate-up. If stopped after editing the repository, run go build to ensure no compilation errors.
Phase 11 – Testing, Benchmarks & CI
Goal	Ensure correctness, performance, and that the pipeline is safe to merge.
Files to create / modify	- Unit test files for each package (*_test.go).<br> - Integration test: backend/internal/normalization/service_integration_test.go (uses testcontainers/postgres).<br> - Benchmark: backend/internal/normalization/lookup_test.go (uses testing.B).<br> - CI: ensure GitHub Actions (or whatever you use) runs go test ./... on every PR and a nightly workflow that runs the benchmarks.
Tasks	1. Write unit tests for each repository (Insert/Get).<br>2. Write unit tests for the normalization lookup (exact, alias, fuzzy+validation, unknown).<br>3. Write the service integration test: <br> a. Start a test Postgres container.<br> b. Load a few known raw extractions (from seed).<br> c. Call the service’s ProcessBatch (or start the worker in a test harness).<br> d. Assert that the expected canonical rows appear and that unknowns went to the queue.<br>4. Write benchmarks for the lookup (target ≥ 100k lookups/sec).<br>5. Add a Makefile target make test that runs go test ./... -count=1.<br>6. Add a Makefile target make bench that runs the benchmarks.
Deliverables	- All unit tests pass.<br>- Integration test passes.<br>- Benchmark meets performance target.
Exit Criteria	1. make test exits with status 0.<br>2. make bench shows ns/op within the target range.<br>3. No test failures in CI on a feature branch.
How to Resume	If you stopped after writing a specific test, just run go test ./<package> for that package. If stopped after updating the Makefile, just run make test again.
Phase 12 – Documentation & Knowledge Transfer
Goal	Produce user‑facing docs so the team knows how to use the plugin, extend the taxonomy, and operate the worker.
Files to create / modify	- docs/market_data_engine_api.md (endpoint specs, request/response examples).<br> - docs/market_data_engine_taxonomy.md (how to add a new skill/company/location via the review queue, how to version a taxonomy).<br> - Update README.md with a short “Plugin Overview” section and a link to the detailed docs.
Tasks	1. Fill in api.md with: <br> • /process/job (unchanged).<br> • Optional sync normalization flag & response shape.<br> • GET /api/v1/jobs/{id} (example JSON).<br> • POST /api/v1/admin/normalize (example).<br> • /metrics and /healthz/normalization.<br>2. In taxonomy.md explain: <br> • The review queue table (entity_resolution_queue).<br> • How an operator can approve a new skill (insert into skill_aliases with source='discovered', status='approved').<br> • How to create a new taxonomy version (insert into taxonomy_versions, then run a re‑normalization job).<br>3. Update README.md with a badge, a short description, and links to the two new docs.
Deliverables	- Clear, accessible documentation for developers and operators.
Exit Criteria	1. Docs build without errors (if you use a static site generator, ensure no broken links).<br>2. A new hire can read the docs and successfully run the plugin end‑to‑end (extract → worker → read endpoint).
How to Resume	If you stopped after editing a markdown file, just save it; the documentation is static, so no rebuild is required unless you have a site build step.
Appendix A – 001_taxonomy.up.sql (example)

-- Core reference tables
CREATE TABLE taxonomy_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version VARCHAR(20) NOT NULL UNIQUE,
    description TEXT,
    effective_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    effective_to TIMESTAMPTZ
);

CREATE TABLE skill_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    version INT NOT NULL DEFAULT 1
);

CREATE TABLE skill_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    version INT NOT NULL DEFAULT 1
);

CREATE TABLE skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    canonical_name VARCHAR(100) NOT NULL UNIQUE,
    normalized_name VARCHAR(100) NOT NULL,
    skill_type_id UUID NOT NULL REFERENCES skill_types(id),
    category_id UUID NOT NULL REFERENCES skill_categories(id),
    parent_skill_id UUID REFERENCES skills(id),
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    version INT NOT NULL DEFAULT 1,
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE skill_aliases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    skill_id UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    alias VARCHAR(100) NOT NULL,
    normalized_alias VARCHAR(100) NOT NULL,
    source VARCHAR(20) NOT NULL DEFAULT 'discovered', -- curated|discovered|llm_suggested
    confidence DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending_review', -- approved|pending_review|rejected
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (skill_id, normalized_alias)
);

CREATE TABLE companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    canonical_name VARCHAR(150) NOT NULL UNIQUE,
    normalized_name VARCHAR(150) NOT NULL,
    website VARCHAR(250),
    industry VARCHAR(100),
    size VARCHAR(50),
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100),
    country VARCHAR(100) NOT NULL,
    country_code CHAR(2),
    lat DOUBLE PRECISION,
    lon DOUBLE PRECISION,
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE job_titles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    canonical_title VARCHAR(150) NOT NULL UNIQUE,
    job_family_id UUID NOT NULL REFERENCES job_families(id),
    level_id UUID NOT NULL REFERENCES seniority_levels(id),
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE job_families (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    parent_id UUID REFERENCES job_families(id),
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE seniority_levels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    ordinal INT NOT NULL,
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE employment_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(30) NOT NULL UNIQUE,
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE work_modes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(20) NOT NULL UNIQUE,
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
(The DOWN migration simply drops the tables in reverse order.)

Appendix B – 002_job_tables.up.sql (example)

-- Job-centric tables
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    canonical_title_id UUID NOT NULL REFERENCES job_titles(id),
    company_id UUID NOT NULL REFERENCES companies(id),
    seniority_id UUID NOT NULL REFERENCES seniority_levels(id),
    job_family_id UUID NOT NULL REFERENCES job_families(id),
    normalization_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE job_postings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES locations(id),
    employment_type_id UUID NOT NULL REFERENCES employment_types(id),
    work_mode_id UUID NOT NULL REFERENCES work_modes(id),
    source VARCHAR(50) NOT NULL,
    source_job_id VARCHAR(200),
    url TEXT,
    posted_at TIMESTAMPTZ,
    scraped_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    salary_min NUMERIC(10,2),
    salary_max NUMERIC(10,2),
    currency CHAR(3),
    dedup_key CHAR(64) NOT NULL,
    UNIQUE (dedup_key),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE job_descriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_posting_id UUID NOT NULL REFERENCES job_postings(id) ON DELETE CASCADE,
    raw_text TEXT NOT NULL,
    extracted_json JSONB NOT NULL,
    parser_version VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE job_skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    skill_id UUID NOT NULL REFERENCES skills(id),
    requirement_type VARCHAR(20) NOT NULL CHECK (requirement_type IN ('required','preferred','nice_to_have')),
    requirement_weight DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    source_section VARCHAR(30),
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    evidence TEXT,
    raw_skill_name TEXT NOT NULL,
    normalization_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE job_skill_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    group_label VARCHAR(20) NOT NULL,
    operator VARCHAR(10) NOT NULL CHECK (operator IN ('ANY','ALL')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE job_capabilities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    capability_id UUID NULL REFERENCES capabilities(id), -- placeholder for Phase 2
    importance DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    evidence TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE job_domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    domain_id UUID NOT NULL REFERENCES domains(id),
    importance DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE job_soft_skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    evidence TEXT,
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Auxiliary tables (already referenced above)
CREATE TABLE entity_resolution_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    field_type VARCHAR(20) NOT NULL, -- skill|title|company|location|seniority|employment_type|work_mode
    raw_value TEXT NOT NULL,
    evidence TEXT,
    extraction_id UUID NOT NULL REFERENCES job_extractions(id),
    status VARCHAR(20) NOT NULL DEFAULT 'pending_review', -- approved|rejected
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE failed_extractions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    extraction_id UUID NOT NULL REFERENCES job_extractions(id),
    error_message TEXT NOT NULL,
    failed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    retry_count INT NOT NULL DEFAULT 0
);

CREATE TABLE normalization_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    rows_processed INT NOT NULL DEFAULT 0,
    success_count INT NOT NULL DEFAULT 0,
    failure_count INT NOT NULL DEFAULT 0,
    entities_sent_to_review INT NOT NULL DEFAULT 0,
    new_entities_approved INT NOT NULL DEFAULT 0,
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id)
);

CREATE INDEX ix_jobs_canonical_title_id ON jobs(canonical_title_id);
CREATE INDEX ix_jobs_company_id ON jobs(company_id);
CREATE INDEX ix_jobs_seniority_id ON jobs(seniority_id);
CREATE INDEX ix_jobs_job_family_id ON jobs(job_family_id);
CREATE INDEX ix_job_postings_job_id ON job_postings(job_id);
CREATE INDEX ix_job_postings_location_id ON job_postings(location_id);
CREATE INDEX ix_job_skills_job_id ON job_skills(job_id);
CREATE INDEX ix_job_skills_skill_id ON job_skills(skill_id);
CREATE INDEX ix_entity_resolution_queue_field_type ON entity_resolution_queue(field_type);
(Adjust column types/sizes as needed for your exact domain.)

How to Resume After an Interruption
Phase	What to note when you stop	Resume steps
0	Last command you ran (e.g., docker-compose up -d or go test ./internal/processor).	Re‑run that command, then continue with the next unfinished task.
1	Migration file you were writing or the seed script you were editing.	Run make migrate-up (if migration not applied) or re‑run the seed script.
2	Which lookup method you were implementing (skill, title, etc.).	Re‑run go test ./internal/normalization to see which test fails, then implement the missing method and test again.
3	Seed file preparation or the number of samples tested.	Re‑run go run cmd/seed/load_raw.go and check the output.
4	Which repository’s InsertTx you were writing.	Re‑run the persistence unit test (TestPersistJob) – it will tell you which table is missing.
5	Dedup key logic.	Run the deduplication unit test (TestJobPostingDeduplication).
6	Worker loop or metric registration.	Start the server (go run ./cmd/server) and verify the worker logs show “claimed X extractions”.
7	Sync normalization flag handling.	Run the handler unit test (TestSyncNormalization).
8	Read endpoint.	Call the endpoint manually with a known job ID (curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/jobs/<id>).
9	Admin batch endpoint.	Use curl with admin token to hit /api/v1/admin/normalize?batchSize=10.
10	Unit/integration tests or benchmarks.	Run make test or make bench and fix any failures.
11	Documentation edits.	Just save the markdown file; no build step required unless you have a static site generator (then run make docs if you added one).
TL;DR – One‑Line Summary
Follow the phases in order, verify each phase’s exit criteria before moving on, and you’ll have a production‑ready Market Data Engine plugin living inside the existing Passo backend, with zero LLM calls during normalization, a claimed‑worker pool, versioned taxonomy, deduplication, and full observability.