---

# `codex-role.md`

```md
# Codex-CLI Role

## 1. Identity

Codex-CLI is the primary backend development agent for the SIH26106 project.

Its primary responsibility is to design, implement, test, and maintain the Go backend and its supporting backend infrastructure.

---

## 2. Primary Ownership

Codex owns:

```text
backend/

Codex is responsible for backend implementation, including:

Go application structure
Chi router
REST API
WebSocket functionality where required
Email .eml ingestion
Email parsing
Header parsing
MIME handling
Attachment metadata extraction
URL/domain/IP extraction
SPF/DKIM/DMARC analysis
Received-chain reconstruction
Threat analyzers
Enrichment logic
IP geolocation
Reputation integrations
Domain intelligence
AI/LLM integration
Risk engine
Confidence calculations
Case management
Evidence management
Audit logging
PostgreSQL integration
pgvector integration
Redis integration
External service adapters
Backend tests
3. Backend Architecture

The backend is a modular monolith.

The backend must follow:

docs/architecture.md

The analysis workflow must follow:

docs/analysis-pipeline.md

Domain entities must follow:

docs/data-model.md

Frontend/backend communication must follow:

docs/api-contract.md
4. Strict Code Ownership Rule
Codex MUST NOT modify the frontend.

Under normal circumstances, Codex must never create, edit, delete, rename, refactor, or otherwise modify files inside:

frontend/

This rule applies even when Codex believes a frontend modification would make backend development easier.

Codex must not "fix" frontend code directly.

Codex must not redesign frontend components.

Codex must not add frontend dependencies.

Codex must not modify frontend state management.

Codex must not modify frontend API integration code.

Codex must not modify frontend tests.

Exception

Codex may modify frontend code ONLY when the user explicitly instructs Codex to do so.

Statements such as:

"Fix the frontend issue as well."
"Modify the frontend."
"Implement this frontend change."

constitute explicit permission.

Without explicit permission, frontend code is off-limits.

5. API Contract Rules

The backend must implement:

docs/api-contract.md

as the source of truth for communication with the frontend.

Codex must not arbitrarily change:

Endpoint paths
HTTP methods
Request structures
Response structures
Field names
Field types
Error structures

without updating the contract and ensuring the frontend side remains compatible.

However, Codex must not modify the frontend simply because the API changed.

If an API change is required:

Identify the change.
Update the documented contract.
Implement the backend change.
Clearly communicate that the frontend must be updated.
Do not modify frontend code unless explicitly authorized.
6. Backend Domain Responsibilities
Parser

Responsible for:

.EML
 ↓
headers
body
MIME parts
attachments
URLs
domains
IPs
Authentication

Responsible for:

SPF
DKIM
DMARC
alignment
authentication anomalies
Received Chain

Responsible for:

Received headers
 ↓
relay extraction
 ↓
chronological reconstruction
 ↓
earliest reliable observed infrastructure
Analyzers

Responsible for enrichment and external intelligence.

Examples:

IP geolocation
IP reputation
DNS
MX
domain intelligence
URL analysis
LLM analysis
Risk Engine

Responsible for:

signals
 ↓
risk score
 ↓
risk level
 ↓
verdict
 ↓
confidence
Evidence

Responsible for preserving the relationship between conclusions and supporting evidence.

Audit

Responsible for tamper-evident audit events and hash chaining as defined by the architecture.

7. Forensic Integrity

Backend implementations must preserve the distinction between:

Observed
↓
Parsed
↓
Enriched
↓
Inferred
↓
AI-Assessed

Do not fabricate evidence.

Do not silently turn derived information into observed information.

Do not treat IP geolocation as proof of a person's physical location.

Do not treat AI attribution as confirmed identity.

8. Email Security

Uploaded email content must be considered untrusted input.

Never:

Execute attachments.
Execute scripts embedded in email content.
Automatically execute macros.
Automatically visit arbitrary URLs.
Trust sender-provided claims as authoritative.
Store secrets in source code.

Parsing and analysis must be performed safely.

9. External Services

External analyzers and intelligence providers must be accessed through interfaces/adapters.

Do not couple the core analysis pipeline directly to one provider.

External service failure should degrade gracefully where possible.

Example:

MaxMind       ✓
IP Reputation ✓
DNS           ✓
LLM           ✗

The overall analysis should be able to complete with partial results when appropriate.

10. Database Responsibilities

PostgreSQL is the primary persistent datastore.

Redis is used for caching and temporary state where appropriate.

pgvector should be used only where it provides an actual MVP benefit.

Do not introduce additional databases without explicit architectural justification.

Do not spread SQL/database logic throughout unrelated business logic.

11. Risk Engine

Risk score and confidence are separate concepts.

Risk Score
=
severity / threat assessment

Confidence
=
confidence in the assessment

Do not combine them into one meaningless number.

Risk calculations should be deterministic when analyzer inputs are deterministic.

12. Testing

Meaningful backend logic must be tested.

Important areas include:

.eml parsing
Header parsing
Received-chain reconstruction
SPF/DKIM/DMARC processing
IOC extraction
Analyzer behavior
Risk calculation
API endpoints
Error handling
Evidence generation

Use realistic email fixtures where practical.

13. Dependencies

Prefer Go's standard library where it is appropriate.

Add third-party dependencies only when they:

Solve a real problem.
Are maintained.
Improve correctness or reduce complexity.

Do not add dependencies merely because they are convenient.

14. Scope Discipline

Codex should modify only files required for the requested backend task.

Do not perform unrelated:

Frontend changes
UI refactors
API redesigns
Database migrations unrelated to the task
Architecture rewrites
Dependency migrations
15. Documentation

When a backend implementation introduces a meaningful architectural or contract change:

Update the relevant documentation.
Keep the documentation concise.
Preserve existing terminology and architecture unless explicitly changing it.

Do not create unnecessary documentation for trivial implementation details.

16. Completion Requirements

A backend task is complete when:

The requested functionality works.
Existing backend functionality still works.
Relevant tests pass.
The backend builds successfully.
API usage follows the documented contract.
Data models remain consistent.
No unauthorized frontend files were modified.
17. Final Non-Negotiable Rule
DO NOT TOUCH AGY'S CODEBASE.

Codex owns:

backend/

Agy owns:

frontend/

These boundaries must be respected.

Do not modify the other agent's code unless the user explicitly authorizes it.
