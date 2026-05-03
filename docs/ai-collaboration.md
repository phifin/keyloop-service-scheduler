# AI Collaboration Narrative

## 1. Strategy

AI was used as an implementation accelerator and review partner for the Keyloop technical assessment. The goal was not to delegate ownership of correctness, but to move quickly through scaffolding, implementation, and repeated audits while keeping the final system understandable and testable.

The project was built in small steps: repository foundation, database schema, reference APIs, availability logic, appointment creation, frontend demo, and documentation. Each step had explicit constraints, such as not adding scheduling logic during scaffolding, not adding appointment creation during availability work, and treating the backend as the primary implementation layer.

## 2. Design Phase

AI helped shape the initial monorepo layout and the separation between backend, frontend, database, and documentation. The design direction was intentionally conservative:

- Go backend with chi and pgx.
- PostgreSQL as the source of truth.
- React/Vite frontend as a demo layer.
- Repository and scheduling packages separated from HTTP handlers.
- Markdown and Mermaid for design documentation.

The scheduling design was reviewed around one core rule: availability requires both a qualified technician and a service bay for the full service duration.

## 3. Implementation Phase

AI was directed to implement the project incrementally. Examples included:

- Scaffold the repository structure for Scenario A.
- Create PostgreSQL migrations and stable seed data.
- Implement read-only reference APIs.
- Implement `GET /availability` without appointment creation.
- Implement `POST /appointments` with a transactional availability re-check.
- Add appointment list, detail, and cancellation endpoints.
- Build a React demo UI for booking and appointment management.

The prompts were intentionally specific about what not to implement in each phase. This helped keep the scope controlled and avoided moving business logic into the frontend.

## 4. Verification Phase

AI was also used for targeted review passes after implementation. These audits focused on risks that matter for a scheduler:

- Whether adjacent appointments were incorrectly treated as conflicts.
- Whether the SQL overlap condition used strict `<` and `>` comparisons.
- Whether only `CONFIRMED` appointments block resources.
- Whether appointment creation re-checks availability inside a transaction.
- Whether the PostgreSQL advisory lock is transaction-scoped and acquired before resource selection.
- Whether cancellation makes appointments stop blocking availability.
- Whether the frontend clears stale availability after form changes.

Verification included backend unit tests, handler tests, integration-style repository tests, and frontend TypeScript build checks.

## 5. Examples of How AI Was Directed

Representative instructions used during the project:

- "Create the initial monorepo structure. Do not implement scheduling business logic yet."
- "Implement PostgreSQL schema, migrations, and seed data. Do not implement HTTP business endpoints yet."
- "Audit the migration files for required constraints and indexes."
- "Implement read-only reference data APIs."
- "Implement core scheduling availability logic and expose `GET /availability`. Do not implement `POST /appointments` yet."
- "Audit the current scheduling overlap logic before appointment creation."
- "Implement appointment creation with transactional availability re-check."
- "Audit transaction/advisory lock correctness."
- "Build a clean frontend demo UI."
- "Audit the frontend booking flow for stale availability and UX correctness."

These prompts kept the work organized around assessment deliverables and specific correctness risks.

## 6. How AI Output Was Reviewed

AI-generated code and documentation were reviewed against the project requirements and the implemented behavior. Key checks included:

- Reading the generated SQL migrations and repository queries.
- Verifying route registration and handler status-code mapping.
- Checking that backend errors are not exposed directly to clients.
- Confirming that the frontend API types match backend response shapes.
- Running `go test ./...` for backend coverage.
- Running `npm run build` for frontend TypeScript and production build validation.

Where correctness mattered most, the review focused on executable tests rather than visual inspection alone.

## 7. Bugs or Risks Caught During Review

Several targeted reviews helped reduce risk:

- The overlap rule was confirmed to allow adjacent appointments and block only true overlaps.
- Repository SQL was checked for `status = 'CONFIRMED'`, `start_time < new_end`, and `end_time > new_start`.
- Appointment creation was checked to ensure it does not trust a previous `GET /availability` response.
- Transaction handling was reviewed to confirm the advisory lock, availability re-check, resource selection, insert, and commit all happen inside one transaction.
- Frontend booking state was audited so changing booking inputs clears stale availability and disables confirmation.

These checks are especially important because scheduling bugs are often race-condition or boundary-condition bugs.

## 8. Final Ownership Statement

AI accelerated the project by generating scaffolding, implementation drafts, tests, review checklists, and documentation. Final responsibility remains with the developer submitting the assessment. The implementation was reviewed against the requirements, tested through backend and frontend commands, and documented with the known MVP limitations and future production improvements.
