
---

# `development-rules.md`

```md
# Development Rules

## 1. Primary Goal

Build a clean, functional MVP.

Prioritize:

1. Correctness
2. End-to-end functionality
3. Maintainability
4. Clear interfaces
5. Useful UX

Avoid unnecessary complexity.

---

## 2. Before Changing Code

The agent MUST:

1. Inspect the existing repository.
2. Read relevant documentation in `/docs`.
3. Inspect related files before editing.
4. Understand existing interfaces and data structures.
5. Reuse existing implementations when appropriate.

Do not immediately rewrite existing code.

---

## 3. Scope Control

Only modify files relevant to the requested task.

Do not perform unrelated refactors.

Do not introduce new libraries unless they provide a clear benefit.

Do not replace an existing implementation simply because another approach is personally preferred.

---

## 4. Architecture

The project uses a modular monolithic Go backend.

Do not introduce microservices unless explicitly requested.

Keep domain responsibilities separated.

Frontend presentation logic belongs in the frontend.

Forensic and business logic belongs in the backend.

Database-specific logic should not leak throughout the application.

External services must be accessed through clear interfaces.

---

## 5. API Contract

`api-contract.md` is the source of truth for frontend/backend communication.

Do not independently invent API formats.

When changing an API:

1. Update the contract.
2. Update backend implementation.
3. Update frontend consumers.
4. Update tests.

---

## 6. Data Model

`data-model.md` defines the meaning of domain entities.

Do not introduce conflicting definitions of entities.

Prefer explicit models over loosely structured objects.

Do not put arbitrary JSON blobs everywhere when a stable structure is appropriate.

---

## 7. Analysis Pipeline

`analysis-pipeline.md` defines the intended processing stages.

New analysis features should clearly identify which pipeline stage they belong to.

A component should not silently perform responsibilities belonging to another stage.

---

## 8. Evidence and Provenance

Forensic conclusions should be traceable.

Do not fabricate evidence.

Do not silently convert inferred information into observed information.

Do not present geolocation as proof of physical location.

Do not present AI-generated attribution as confirmed identity.

---

## 9. Security

Never:

- Execute arbitrary email attachments.
- Execute arbitrary scripts found in email content.
- Automatically visit untrusted URLs in the user's browser.
- Store secrets in source code.
- Log credentials or API keys.

Treat uploaded email content as untrusted input.

Validate and sanitize external input.

---

## 10. Error Handling

Errors must be explicit.

Do not silently swallow failures.

External service failures should degrade gracefully where possible.

Return useful structured errors to the frontend.

---

## 11. Testing

Every meaningful backend feature should have tests.

At minimum:

- Unit tests for parsing logic
- Unit tests for analysis logic
- API tests for important endpoints

Important forensic logic must be tested with realistic `.eml` examples.

---

## 12. Dependencies

Prefer standard library functionality where practical.

Add third-party dependencies only when:

- They solve a real problem.
- They are maintained.
- They reduce complexity or improve correctness.

Do not add dependencies merely for convenience.

---

## 13. Configuration

Secrets and environment-specific configuration must not be hardcoded.

Use environment variables or configuration files excluded from version control.

Provide safe example configuration through `.env.example`.

---

## 14. Logging

Logs should be useful for debugging.

Do not log:

- API keys
- Passwords
- Sensitive secrets

Be deliberate when logging raw email content because emails may contain sensitive information.

---

## 15. Git

Commits should be focused.

Avoid mixing:

```text
feature + unrelated refactor + formatting + dependency migration

in one change.

16. Agent Behaviour

Coding agents must:

Inspect before modifying.
Read and follow /docs.
Preserve existing contracts.
Reuse existing code where appropriate.
Run relevant tests and build checks.
Report failures honestly.
Avoid speculative features.
Avoid over-engineering.

Agents must not:

Rewrite the architecture without approval.
Delete working code without justification.
Invent APIs that contradict the documented contract.
Modify unrelated subsystems.
Claim tests passed when they were not run.
17. Definition of Done

A task is not complete merely because code was written.

A task is complete when:

The requested functionality exists.
Existing functionality still works.
Relevant tests pass.
The application builds.
API and data contracts remain consistent.
Documentation is updated when necessary.
18. MVP Rule

When multiple technically valid solutions exist, prefer the simplest solution that:

Works reliably.
Fits the architecture.
Can be tested.
Can be extended later.

Do not optimize for hypothetical scale.

Do not solve problems we do not currently have.
