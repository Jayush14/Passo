# Passo — AI Career Mentor Master Project Document

**Working name:** Passo (subject to change — see section 15.1 for how the name is kept swappable across the codebase)

**Status:** Planning complete, ready for implementation
**Last updated:** July 2026
**This document consolidates:** product-requirements.md, build-plan.md, build-plan-v3.md, technical-spec-v1/v2/v3.md

---

## 1. Product Vision

An AI-powered mentor app for IT professionals switching specializations (e.g. frontend → backend, QA → DevOps, support → cloud). It differs from static roadmap sites and generic AI career coaches by combining:

- A **personalized, phased roadmap** generated from the user's actual skills, target role, and available time
- An **active daily mentor loop** — nudges, behind-schedule check-ins, and re-pacing — not a tracker you have to remember to open
- A **bring-your-own-key (BYOK)** model — users supply their own free Gemini API key, so the app carries no AI cost

**Target users:** IT professionals switching specialization, career changers entering IT, self-taught learners wanting structure and accountability.

---

## 2. Competitive Positioning

| Category | Examples | Gap we fill |
|---|---|---|
| AI career coaching all-in-ones | Kickresume, TealHQ, CareerHero.io, AICareerCoach.org | Generic career coaching, not IT-specific; AI cost baked into subscription price |
| Voice/mobile mentor apps | Path AI (Lea) | General coaching, not technical skill-gap specific |
| Bootcamp/human-mentor hybrids | CareerFoundry, MentorCruise | Tied to paid courses, or human-mentor cost/availability limits |

**Our wedge:** IT-specific skill-to-skill gap analysis + proactive daily mentor loop + BYOK cost model (near-zero AI cost vs. competitors' $13–19/mo subscriptions).

---

## 3. Format & Engagement Decisions

**Format:** Progressive Web App (PWA) for v1 — installable, offline-capable tracker, push notifications on Android and iOS (16.4+). One codebase while validating the mentor loop. Native apps deferred to Phase 3, contingent on retention data.

**Engagement mechanics (v1):**
- Daily/weekly nudge notifications ("You planned 2 hrs today — still on?")
- Behind-schedule detection → proactive check-in / re-plan offer
- End-of-phase milestone check-ins
- **Streak counter** — consecutive days with at least one *meaningful* activity (a task checked, or a chat message with real content — not just opening the app). Shown quietly in the tracker header next to % progress. No tiers, no unlocks, no separate achievements page — just the number, as a plain "you've shown up N days in a row" signal.

**Deferred to Phase 2:** badges, streak tiers, rewards/points systems — validate response to the plain streak counter before layering on more gamification.

---

## 4. AI Usage & Cost Analysis

**Gemini free tier (per-user project, not shared):** roughly 10–15 RPM, 250–1,500 RPD (sources vary — verify live via AI Studio, not a hardcoded number), 250K TPM, 1M token context. Pro models are paid-only as of April 2026; free tier covers Flash/Flash-Lite.

**Our realistic per-user daily usage:** ~20–25 Gemini calls/day at peak engagement (roadmap generation occasionally, 5–20 chat messages, occasional re-plan). This is under 10% of even the conservative free-tier daily ceiling — free tier comfortably covers normal usage.

**Edge case to handle in build:** a user in an extended chat session could hit the RPM ceiling. Handle with graceful retry/backoff messaging, not a crash — noted as a required feature, not a cost concern.

**Cost to us:** ~$0 in AI spend for BYOK users. House-key subscribers (see section 4.5) carry real AI cost, offset by subscription revenue.

## 4.5 Hybrid AI Key Model

Default remains BYOK (section 5.2), but users can optionally subscribe to a paid tier that uses a house-managed key instead of their own — removes the friction of getting a Gemini key entirely, and gives us a monetization path.

```go
func resolveAPIKey(user User) (key string, err error) {
    if user.PlanTier == "subscription" {
        return decryptHouseKey(), nil // our own, server-managed
    }
    return decryptStoredUserKey(user.ID) // BYOK, default
}
```

Both paths flow through the same `AIProvider` interface — no branching in the roadmap/chat/repace handlers themselves, only in key resolution.

**Capacity note:** house-key traffic is pooled across all subscribers (unlike BYOK, where each user has isolated quota) — fine at low subscriber counts, but graceful 429 handling (already noted as required) becomes a real capacity-planning concern as the subscriber base grows, not just an edge case.

---

## 5. Finalized Product Decisions

### 5.1 Gemini key storage
Store server-side, encrypted at rest (AES-256-GCM, encryption key in server env, not DB). Decrypted only in-memory per request, never logged. Enables cross-device persistence, matching the "always-on mentor" pitch.

### 5.2 Free tier / shared key
No shared-key free tier — reintroduces the AI cost the architecture is designed to avoid. Manual onboarding requires no key. Resume-upload onboarding requires the key at upload (extraction is itself an AI call). Manual path defers the key ask until first roadmap generation.

### 5.3 Onboarding input model
Two paths, user's choice:
- **Manual fill:** structured optional fields + one open "tell your mentor more" free-text field (see section 7)
- **Resume upload:** PDF/DOCX processed via Gemini extraction, pre-fills the same fields, user reviews/edits before confirming. Raw file discarded after extraction; only skills/experience/education/certs/portfolio retained — contact PII discarded even if the model returns it. **Key-ask timing:** the Gemini key is requested *after* file selection and validation, not before — pairs the ask with an explanation tied to what the user just did, rather than a cold trust ask up front.

### 5.4 Roadmap re-planning logic (core mentor behavior)
**Step 1:** Continue this plan vs. Start fresh
- Continue: completed tasks/notes preserved as-is; only remaining incomplete tasks get re-planned
- Fresh: old roadmap archived (read-only), new onboarding → new roadmap

**Step 2 (if continuing):** Keep same deadline vs. Extend timeline
| Choice | Fixed | Recalculated |
|---|---|---|
| Keep same deadline | `target_end_date` | Required pace (likely increases) |
| Extend timeline | Pace ~similar to original | New `target_end_date` |

**Step 3:** Realism check — if keeping the deadline requires meaningfully more hours/week than the user has available, the mentor flags this explicitly before confirming (a real warning screen, not a dismissable toast) — e.g. *"To hit your original date you'd need ~15 hrs/week — that's a big jump from your original 5. Keep the deadline or extend it?"*

**Required stored fields for this to work:** `start_date` (immutable), `target_end_date` (mutable, recalculated on re-pace), `hours_per_week_available` (needed for the realism comparison).

---

## 6. Personalization Vectors (all optional, silent calibration — never quoted back at the user)

| Category | Fields |
|---|---|
| Compensation/target context | current_company, current_package, expected_company, expected_package |
| Background & signal | education_background, certifications, portfolio_url, interview_experience |
| Working style | weekly_time_pattern (daily/weekends/irregular), learning_style |
| Direction | target_domain, network_access |
| Catch-all | additional_notes (free text, manual path only — resume path gets equivalent signal from extraction) |

**Explicitly excluded, by design:** age, gender, ethnicity, disability, nationality, personality-type quizzes — no roadmap value, real bias/legal exposure for a career-adjacent tool. Not collected even as optional.

---

## 7. Onboarding Flow (final)

```
Signup → choose path:

  [Manual fill]                          [Upload resume]
  No key needed yet.                     Key requested HERE (extraction = AI call)
  Full form: required fields +           Extract -> pre-fill review screen ->
  all optional personalization           user edits/confirms
  vectors + "tell your mentor more"      (raw file discarded after extraction;
  free-text field                        only skills/experience/education/certs/
                                          portfolio retained)
         |                                        |
         +------------------ merge -----------------+
                              |
              Key requested here IF NOT
              already provided (manual path only)
                              |
                     Roadmap generation
```

---

## 8. Roadmap & Task Data Schema

```json
{
  "roadmap_id": "uuid",
  "user_id": "uuid",
  "status": "active",
  "onboarding_source": "manual",
  "current_skills": ["JavaScript", "React", "basic SQL"],
  "target_role": "Backend Engineer (Go)",
  "hours_per_week_available": 10,
  "weekly_time_pattern": "daily",
  "start_date": "2026-07-13",
  "target_end_date": "2026-12-07",
  "current_company": null,
  "current_package": null,
  "expected_company": null,
  "expected_package": null,
  "education_background": null,
  "certifications": [],
  "portfolio_url": null,
  "interview_experience": null,
  "learning_style": null,
  "target_domain": null,
  "network_access": null,
  "additional_notes": null,
  "created_at": "2026-07-13T09:00:00Z",
  "last_repaced_at": null,
  "phases": [
    {
      "phase_id": "p1",
      "title": "Phase 1 — Foundation",
      "order": 1,
      "weeks": [
        {
          "week_id": "w1",
          "tag": "W1-2",
          "title": "Go syntax & core language",
          "start_date": "2026-07-13",
          "end_date": "2026-07-26",
          "tasks": [
            {
              "task_id": "w1_0",
              "title": "Finish 'A Tour of Go' end to end",
              "description": "Work through Go's official interactive tour, covering basic syntax, structs, methods, and interfaces. Type out every exercise rather than skimming — muscle memory pays off once you hit concurrency in week 3. Budget roughly 4-5 hours across the week.",
              "resources": [
                { "label": "A Tour of Go (official)", "url": "https://go.dev/tour/" }
              ],
              "checked": false,
              "note": "",
              "subtasks": [
                { "subtask_id": "w1_0_a", "text": "Complete Basics section", "checked": false },
                { "subtask_id": "w1_0_b", "text": "Complete Methods and Interfaces section", "checked": false },
                { "subtask_id": "w1_0_c", "text": "Complete Concurrency intro section", "checked": false }
              ]
            }
          ]
        }
      ]
    }
  ]
}
```

**Design decisions:**
- `task_id` format `{week_id}_{index}` — matches the original tracker prototype UI, no rework needed there
- Every week carries resolved `start_date`/`end_date`, not a vague label — required for the realism check math
- `subtasks` populated only when the AI judges a task too large for one sitting (>~4-5 hrs or multiple distinct sub-skills) — small tasks stay flat
- `resources` included only for well-known, stable references — AI is instructed to omit rather than invent a URL

---

## 9. System Prompts (final)

### 9.1 Roadmap Generation
```
You are an IT career transition mentor. Generate a personalized, phased learning
roadmap in strict JSON matching the schema in section 8.

Required inputs:
- Current skills: {current_skills}
- Target role: {target_role}
- Hours available per week: {hours_per_week}
- Requested timeframe: {timeframe}

Optional inputs (use if provided, ignore silently if absent — never ask why missing):
- Current company/package, expected company/package
- Education background, certifications, portfolio URL, interview experience
- Weekly time pattern, learning style, target domain, network access
- Additional notes (manual path, unstructured — use judgment; if it contradicts a
  structured field, prefer the more specific/recent signal)

How to use optional inputs:
- Calibrate task difficulty/depth toward the target company/level if known
- Prioritize addressing specific weak areas mentioned in interview_experience early,
  not just at the end
- Never quote package numbers, company names, or personal notes back at the user in
  task text — use them only to silently shape the plan

Task-writing rules:
- `title`: one actionable line (existing UI). `description`: 2-4 sentences — what to
  do, why it matters for the target role, rough time budget
- `resources`: only well-known, stable, official links. Omit rather than invent a URL.
- Split into `subtasks` only if a task realistically exceeds ~4-5 hours or spans
  multiple distinct sub-skills
- Resolve start_date/end_date per week from start_date = today and the stated pace

Output ONLY valid JSON matching the schema. No markdown, no commentary, no code fences.
```

### 9.2 Mentor Chat
```
You are the user's personal mentor for their IT career transition. You have full
context on their plan — use it, don't give generic advice.

Context: target role, current phase/week, tasks completed/remaining this week,
days remaining until target_end_date.

Behavior:
- Answer as a mentor who knows their specific plan, not a generic chatbot
- Off-topic questions still get answered helpfully, related back to their plan
  where natural
- Keep responses focused and actionable — limited-time working professional, not
  someone browsing for entertainment
- Never invent progress or skills they haven't reported
```

### 9.3 Re-pacing / Realism Check
```
You are recalculating a user's remaining roadmap after a pause or re-plan request.

Inputs: completed tasks (preserved as-is), remaining tasks (to re-plan), start_date,
days_elapsed, hours_per_week available, user's choice (keep_deadline | extend_timeline),
current target_end_date.

If keep_deadline: calculate required pace to finish remaining tasks by target_end_date.
If required pace exceeds hours_per_week by more than ~30%, set a non-null "warning"
field — do not silently assume the user can do it. Only re-phase remaining tasks;
don't touch completed ones.

If extend_timeline: calculate a new target_end_date that keeps pace close to
hours_per_week_available; redistribute remaining tasks accordingly.

Output strict JSON: { "warning": string|null, "new_target_end_date": "YYYY-MM-DD",
"repaced_weeks": [...] }
```

---

## 10. Re-plan Flow (UI states)

**Revised for usability:** rather than opening straight into a decision tree, the app leads with an AI-computed recommendation ("Based on your pace, I'd suggest extending ~2 weeks, everything else stays the same") — one tap to accept, or "Customize" to expand into the full choice tree below. Reduces decision fatigue at a moment (falling behind) when the user is often already discouraged. The "Start fresh" option is labeled to make clear the current plan is archived, not deleted.

```
[Tracker] --"Re-plan"--> [AI suggestion] --Accept--> Apply
                              |
                          Customize
                              v
[Continue this plan | Start fresh (saved, not deleted)]

  Start fresh:
    confirm -> archive current roadmap -> redirect to Onboarding

  Continue this plan:
    [Keep same deadline | Extend timeline]

      Keep same deadline:
        call /roadmap/replan {keep_deadline}
        if warning present -> [Warning screen: required pace vs available hours,
          "Keep deadline anyway" | "Extend timeline instead"] (real branch, not a toast)
        else -> apply repaced_weeks -> back to Tracker

      Extend timeline:
        call /roadmap/replan {extend_timeline}
        show new target_end_date -> confirm -> apply repaced_weeks -> back to Tracker
```

---

## 11. Technical Architecture

**Frontend:** React, built as a PWA (manifest, service worker, push notification support)
**Backend:** Go (REST API)
**Database:** PostgreSQL
**AI integration:** Go server as pass-through proxy only — never uses an internally-owned key. Forwards the user's stored (decrypted) Gemini key + prompt to `generateContent`, relays the response. Required because Gemini's API can't be called directly from a browser (CORS restriction).
**Notifications:** Web Push API
**Auth:** Email/password v1; OAuth as fast-follow

### Go backend structure
```
/cmd/server/main.go
/internal/auth/          # signup, login, session/JWT
/internal/apikey/        # encrypt/decrypt Gemini key, storage
/internal/roadmap/       # generation, repace, archive
/internal/progress/      # task checks, notes
/internal/chat/          # mentor chat handler
/internal/guardrail/     # scope enforcement + rate limiting on chat, ahead of aiproxy
/internal/aiproxy/       # AIProvider interface (Gemini now, others later)
/internal/notify/        # web push subscription + send
/internal/db/            # postgres models + migrations
```

### AI proxy interface
```go
type AIProvider interface {
    Generate(ctx context.Context, apiKey string, prompt string) (string, error)
}
type GeminiProvider struct{}
func (g GeminiProvider) Generate(ctx context.Context, apiKey, prompt string) (string, error) {
    // calls generativelanguage.googleapis.com
}
```

### Core endpoints
| Method | Path | Purpose |
|---|---|---|
| POST | /auth/signup, /auth/login | Account |
| POST | /apikey | Store/rotate Gemini key |
| POST | /onboarding/resume | Upload + extract resume fields |
| POST | /roadmap/generate | First roadmap generation |
| POST | /roadmap/replan | Continue-vs-fresh, deadline choice |
| GET | /roadmap/active | Fetch current roadmap + progress |
| PATCH | /progress/:task_id | Check/uncheck, update note |
| POST | /chat | Mentor chat message |
| POST | /push/subscribe | Register push subscription |
| POST | /subscription/start | Move user to house-key plan tier |
| GET | /admin/dashboard/* | Aggregated growth/engagement/roadmap/AI/monetization/support metrics (admin-only, separate auth guard) |

### React PWA structure
```
/src/pages/Onboarding.tsx       # path choice: manual form or resume upload
/src/pages/KeySetup.tsx         # shown at resume upload OR first roadmap generation
/src/pages/Tracker.tsx          # phases/weeks/tasks + notes + subtasks + resources
/src/pages/Chat.tsx             # mentor chat
/src/pages/RePlan.tsx           # continue/fresh -> deadline choice -> realism warning
/src/components/                # shared UI, ported from original tracker prototype
/src/lib/api.ts                 # typed fetch wrapper
/public/manifest.json           # PWA manifest
/public/service-worker.js       # offline cache + push handling
```

---

## 12. Data Model (Postgres tables)

- **users**: id, email, password_hash, current_streak, last_activity_date, plan_tier (byok/subscription), is_admin, created_at
- **house_api_key**: id, encrypted_key, provider, created_at (single row/small table — our own managed key for subscription-tier users)
- **api_keys**: id, user_id, encrypted_key, provider, created_at
- **roadmaps**: id, user_id, status (active/archived), onboarding_source, all onboarding fields (section 6-7), start_date, target_end_date, phases (JSON), created_at, last_repaced_at
- **progress**: id, roadmap_id, task_id, checked, note
- **chat_messages**: id, user_id, roadmap_id, role, content, created_at
- **push_subscriptions**: id, user_id, endpoint, keys (JSON)
- **nudge_schedule**: id, user_id, cadence, last_sent_at
- **analytics_events**: id, user_id, event_type, metadata (JSON), created_at — event_type values: signup, onboarding_completed, roadmap_generated, task_checked, replan_requested, chat_message_sent, subscription_started

---

## 13. Admin Dashboard & Analytics

Underpinned by the `analytics_events` table (section 12) — the dashboard is queries on top of that event log, refreshed daily (no real-time requirement for v1).

| Section | What it shows |
|---|---|
| Growth | Signups over time, onboarding path split (manual vs. resume upload), onboarding completion rate |
| Engagement | DAU/WAU, average streak length, retention at week 4/week 8, nudge notification open rates |
| Roadmap health | Roadmaps generated, average tasks/roadmap, % re-planned (and which choice — continue vs. fresh, keep deadline vs. extend), most common target roles |
| AI reliability | Malformed-JSON retry rate on roadmap generation, Gemini error rate (429s, invalid key), BYOK vs. house-key call volume split |
| Monetization | Active subscribers, conversion rate from BYOK to subscription, churn |
| Support signals | Users with zero activity after signup, aggregated chat topic trends |

**Privacy boundary (explicit):** the dashboard shows aggregated chat topic trends only — admins cannot browse individual users' raw chat conversations.

**Access control:** separate `is_admin` flag and route guard, isolated from regular user auth/sessions — never reuse a normal user session for admin routes.

## 14. Feature Scope by Phase

**Phase 1 — MVP:** everything specified above — two onboarding paths, roadmap generation, tracker with subtasks/resources, mentor chat, nudges, streak counter, full re-plan flow with realism check, cross-device persistence, hybrid AI key model (BYOK default + house-key subscription), analytics event logging, admin dashboard.

**Phase 2 — Retention & depth:** badges, streak tiers, rewards/points systems, richer milestone check-ins, additional AI providers behind the same interface.

**Phase 3 — Expansion (contingent on retention data):** native mobile apps, resume review tool (separate from onboarding extraction), mock interview tools, job application tracker, paid tier.

---

## 15. Brand Name — Kept Swappable

**Decision:** "Passo" is adopted as the working project name, but may change later (naming isn't fully finalized — see section 14's naming meeting outcome). To make a future rename cheap rather than a repo-wide find/replace, the name is never hardcoded — it lives in one place per layer, referenced everywhere else as a variable:

| Layer | Where the name lives |
|---|---|
| Backend (Go) | Single constant, e.g. `const AppName = "Passo"` in one config file (`/internal/config/brand.go`), imported anywhere the name is needed (emails, notification text, API responses) |
| Frontend (React) | One value in an env/config file (e.g. `VITE_APP_NAME` or a `brand.ts` constant), referenced by components — never a literal string typed into JSX |
| PWA manifest | `manifest.json`'s `name`/`short_name` fields — generated from the same source at build time if possible, or updated in this one file only |
| Domain/repo | Kept separate from the brand constant — renaming the product name doesn't require renaming the repo or domain immediately |
| Docs (this file, diagrams) | Referred to as "the project name" in structural sections; only the title and this section need updating on a rename |

**Rule going forward:** any time the AI-generated code (Claude Code or otherwise) writes user-facing text containing the product name, it should reference the constant/variable, not type the literal string. This is a one-line discipline now that avoids a tedious rename later.

## 16. Current Status

✅ Product vision, positioning, and format decided
✅ Engagement/mentor model decided
✅ All 4 original open questions resolved (key storage, free tier, taxonomy, regeneration)
✅ Re-pacing logic fully specified (continue/fresh, deadline choice, realism check)
✅ AI usage/cost validated against Gemini free tier
✅ Personalization vectors finalized (included and explicitly excluded)
✅ Two onboarding paths designed (manual + resume upload)
✅ Roadmap/task JSON schema finalized
✅ All three system prompts drafted
✅ Backend and frontend architecture specified
✅ Data model finalized
✅ Plain streak counter added to Phase 1 scope (badges/tiers remain Phase 2)
✅ Hybrid AI key model decided (BYOK default + house-key subscription tier)
✅ Admin dashboard and analytics events table specified
✅ Pre-build review completed (iOS push caveat, key management, chat guardrails, deletion/export, JSON validation, deployment/monitoring — see below)
✅ Flow diagrams reviewed for usability — key-ask timing (resume path), AI-led re-plan suggestion, archive-not-delete copy, chat guardrail added to architecture
✅ Working name chosen: **Passo** — kept swappable via single-source-of-truth constants (section 15), not hardcoded anywhere

**Remaining open items from pre-build review (not yet resolved):**
- "Add to home screen" onboarding step, required for iOS push notifications to actually fire — **deferred: will solve during build, not blocking start**
- Account deletion / data export endpoints
- Real key-management plan for the master encryption key (env var alone is not final)
- Chat proxy guardrails against off-topic/abuse usage
- Explicit multi-device conflict resolution decision (proposed default: last-write-wins)
- Environments, logging, and basic monitoring plan before first deploy
- JSON validation + retry strategy for malformed AI output on roadmap generation

**Not yet done — next steps:**
1. Scaffold the Go backend repo (auth, DB migrations, AI proxy)
2. Scaffold the React PWA repo (onboarding, tracker, chat, manifest/service worker)
3. Build and test the resume-extraction endpoint against real sample resumes
4. Wire up push notifications end-to-end
5. Implement the re-plan flow UI exactly per section 10
6. First internal test run with a real roadmap generation to validate prompt output quality
