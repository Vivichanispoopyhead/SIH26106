# API Contract

## 1. Purpose

This document defines the communication contract between the React frontend and Go backend for SIH26106.

The frontend and backend must implement this contract rather than independently inventing request and response formats.

The API is designed around the investigation workflow:

```text
Upload .EML
    ↓
Automatic Case Creation
    ↓
Raw Artifact Preservation
    ↓
Analysis
    ↓
Parsed Email Data
    ↓
Investigation Workspace

Changes to this document require corresponding changes in both frontend and backend.

2. API Principles
REST over HTTP.
JSON for structured request and response data.
multipart/form-data for .eml uploads.
Raw uploaded email content is never returned through normal API responses.
The original uploaded artifact must be preserved by the backend.
Upload and analysis are separate operations.
Analysis has explicit processing states.
API responses use explicit status values.
Backend responses must not expose implementation-specific internal structures.
Observed data and derived data must remain distinguishable.
Error responses use stable machine-readable error codes.
The frontend must not depend on undocumented fields.
The API must remain simple enough for the MVP while allowing future asynchronous analysis.
3. Base URL

Development API prefix:

/api

The frontend must not hardcode environment-specific hostnames throughout the application.

Use frontend environment configuration for the API host.

4. Initial MVP Workflow

The initial MVP workflow is:

User selects .eml
        ↓
POST /api/emails
        ↓
Validate upload
        ↓
Create Case automatically
        ↓
Create Email artifact
        ↓
Preserve original raw .eml
        ↓
Return case_id + email_id
        ↓
POST /api/emails/{email_id}/analysis
        ↓
Parse / analyze email
        ↓
GET /api/emails/{email_id}
        ↓
Frontend displays structured results

The MVP does not require the user to manually create a case before uploading an email.

Each initial email upload creates a new case automatically.

Future versions may support attaching multiple emails to an existing case.

5. API Scope

The API is divided into:

Implemented MVP Slice
POST /api/emails
POST /api/emails/{email_id}/analysis
GET  /api/emails/{email_id}
GET  /api/cases/{case_id}/graph
GET  /api/cases/{case_id}/timeline
POST /api/cases/{case_id}/report
GET  /api/cases/{case_id}/report
Future Investigation Endpoints
GET  /api/cases/{case_id}
GET  /api/cases/{case_id}/analysis
GET  /api/cases/{case_id}/map
GET  /api/cases/{case_id}/evidence
Future endpoints must extend the existing contract rather than breaking the initial upload model.

6. Endpoint: Upload Email
Request
POST /api/emails
Content-Type: multipart/form-data

Form field:

file=<email.eml>
6.1 Upload Validation

Accepted file extension:

.eml

Maximum file size:

50 MB

The backend must reject:

files larger than 50 MB
unsupported file extensions
missing file fields
empty files
malformed multipart requests

The MVP does not accept .msg.

6.2 Upload Semantics

The backend must perform:

Receive multipart upload
        ↓
Validate file
        ↓
Create Case
        ↓
Create Email record
        ↓
Preserve original raw artifact
        ↓
Record upload event
        ↓
Return identifiers

The upload operation must NOT:

silently discard the original artifact
return the complete raw .eml
silently begin expensive external enrichment
silently begin AI analysis
silently modify the uploaded artifact

The upload operation establishes the case and email artifact only.

Analysis is explicitly initiated by a separate request.

7. Successful Upload Response

HTTP status:

202 Accepted

Example:

{
  "case_id": "case_01J...",
  "email_id": "email_01J...",
  "status": "uploaded"
}
7.1 Response Fields
case_id

Unique identifier for the automatically created investigation case.

email_id

Unique identifier for the uploaded email artifact.

status

Initial upload state.

Possible value:

uploaded
8. Automatic Case Creation

A Case is created automatically during the initial email upload.

Minimum conceptual case information:

Case
- id
- status
- created_at
- updated_at

Initial case status:

created

A user-visible case title may be derived later from parsed email metadata.

The upload API must not require the frontend to know or provide the final case title.

9. Endpoint: Start Analysis
Request
POST /api/emails/{email_id}/analysis

No request body is required for the initial MVP.

9.1 Start Analysis Response

HTTP status:

202 Accepted

Example:

{
  "analysis_id": "analysis_01J...",
  "email_id": "email_01J...",
  "case_id": "case_01J...",
  "status": "started"
}
9.2 Analysis States

Valid analysis states:

started
processing
completed
partial
failed

Meaning:

started

The analysis request was accepted and processing has begun.

processing

The analysis is currently being executed.

completed

All required analysis stages for the current implementation completed successfully.

partial

The analysis completed with one or more non-fatal analyzer failures or unavailable enrichments.

failed

The analysis could not produce a usable result.

The analysis state describes processing state, not threat severity.

For example:

processing

must never be interpreted by the frontend as:

suspicious
10. Analysis Execution Model

The initial MVP may execute analysis synchronously inside the backend after receiving the analysis request.

However, the API must preserve the explicit analysis-state model:

started
    ↓
processing
    ↓
completed / partial / failed

This allows the implementation to become asynchronous later without requiring a fundamental frontend API redesign.

The frontend must rely on the documented status model rather than assuming analysis is always synchronous.

11. Endpoint: Get Email Analysis
GET /api/emails/{email_id}

This endpoint returns the latest known structured state of the email.

It may represent:

uploaded
processing
parsed
partial
failed

depending on the current implementation state.

11.1 Canonical Analysis Result

GET /api/emails/{email_id}/analysis

This endpoint returns the latest analysis result independently from the parsed
email artifact. It is the stable integration point for future deterministic
analyzers and AI/ML providers.

Responses:

- `200 OK` when an analysis exists
- `404 EMAIL_NOT_FOUND` when the email does not exist
- `404 ANALYSIS_NOT_FOUND` when analysis has not been started

Example:

```json
{
  "analysis_id": "analysis_01J...",
  "email_id": "email_01J...",
  "case_id": "case_01J...",
  "status": "completed",
  "ai_assessment": {
    "status": "not_available",
    "classification": null,
    "confidence": null,
    "supporting_signals": [],
    "evidence_references": [],
    "provider": null,
    "model": null,
    "failure": {
      "code": "AI_NOT_CONFIGURED",
      "message": "No AI analyzer is configured."
    }
  },
  "authentication": {
    "spf": {
      "status": "none",
      "evidence_references": [],
      "explanation": "No Authentication-Results header reported SPF."
    },
    "dkim": {
      "status": "none",
      "evidence_references": [],
      "explanation": "No Authentication-Results header reported DKIM."
    },
    "dmarc": {
      "status": "none",
      "evidence_references": [],
      "explanation": "No Authentication-Results header reported DMARC."
    }
  },
  "received_chain": [
    {
      "sequence": 1,
      "hostname": "mx.example.org",
      "ip_address": "203.0.113.10",
      "timestamp": "2026-09-08T10:20:30Z",
      "source_header_order": 4,
      "confidence": "medium",
      "provenance": "inferred"
    }
  ],
  "ip_enrichment": [
    {
      "ip_address": "203.0.113.10",
      "status": "not_applicable",
      "country": null,
      "region": null,
      "city": null,
      "latitude": null,
      "longitude": null,
      "asn": null,
      "organization": null,
      "provider": null,
      "confidence": null,
      "retrieved_at": null,
      "provenance": "OBSERVED",
      "failure": null
    }
  ],
  "risk": {
    "score": 0,
    "level": "low",
    "verdict": "unknown",
    "confidence": null,
    "contributing_signals": [],
    "evidence_references": []
  },
  "failure": null
}
```

`ai_assessment.status` is one of `not_available`, `completed`, `failed`, or
`partial`. Classification and confidence remain nullable and must not be
fabricated. AI confidence describes confidence in the assessment, not identity
confidence or physical attribution. The default backend response is
`not_available` until an explicit AI adapter is configured.

The deterministic risk assessment contains an integer score from 0 to 100,
with levels `low` (0-24), `medium` (25-49), `high` (50-74), and `critical`
(75-100). Verdicts are `benign`, `suspicious`, `phishing`, `malware`, `fraud`,
or `unknown`. Risk score and confidence are separate: confidence is nullable
and describes confidence in the final assessment, never attacker identity.
Each contributing signal contains `code`, `description`, `points`,
`category`, `provenance`, and typed evidence references. AI points are bounded
by the AI confidence and never erase stronger observed evidence. A failed AI
provider still produces a deterministic risk result from available evidence.

AI provider input is normalized at the backend boundary. Plain text, headers,
indicators, and attachment metadata have independent limits and the
serialized input has a hard total limit of 24,000 characters. The default
limits are 12,000 body characters, 100 headers with 512 characters per value,
50 values per indicator type with 256 characters per value, and 50 attachment
metadata records with 256 characters per field. Attachment bytes are never sent.
Literal URL values are replaced with stable evidence slots such as `url-1`;
the backend never fetches URLs or resolves domains. Obvious passwords, bearer
tokens, API keys, and private keys are redacted while header order and evidence
locations remain available. Provider responses accept only the documented
classification vocabulary, bounded signals, and evidence references present
in the normalized input. Invalid, empty, or contradictory responses become
structured AI failures and cannot become a benign assessment. Provider
telemetry records provider, model, prompt version, duration, status, and a
stable failure code without storing prompts, bodies, responses, or credentials.

Authentication statuses are `pass`, `fail`, `neutral`, `none`, or `unknown`.
Authentication evidence references identify the source header order and name.
No DNS or external lookup is performed by this endpoint. Each received relay
preserves its source header order; sequence and relay confidence are derived
from the observed header order and available header values.

`ip_enrichment` contains one deduplicated result for each valid IP observed in
the parsed indicators or Received chain. `status` is `enriched`,
`not_applicable`, `failed`, or `not_configured`. Private, loopback, link-local,
multicast, unspecified, documentation/test, and invalid values are
`not_applicable` and are never sent to a provider. A configured provider result
is `enriched`; provider errors are `failed`; an absent provider key is
`not_configured`. Nullable location, ASN, organization, provider, confidence,
and retrieval time fields remain null when unavailable. `retrieved_at` is an
RFC3339 timestamp. `provenance` uses `OBSERVED` for directly observed address
classification, `ENRICHED` for provider-returned metadata, and `INFERRED` for
backend-derived failure or availability state. Enrichment is contextual
metadata and geolocation is not proof of identity, ownership, or maliciousness.

### 11.2 Endpoint: Get Email Evidence

`GET /api/emails/{email_id}/evidence` returns safe, typed provenance for a
completed or partial analysis:

```json
{
  "email_id": "email_...",
  "analysis_id": "analysis_...",
  "evidence": [
    {
      "evidence_id": "evidence_...",
      "email_id": "email_...",
      "analysis_id": "analysis_...",
      "type": "authentication",
      "source": "Authentication-Results",
      "value": "spf=fail",
      "snippet": "Authentication-Results: ... spf=fail",
      "provenance": "OBSERVED",
      "header_order": 7,
      "related_signal_codes": ["SPF_FAIL"],
      "related_ai_evidence_references": [],
      "hash": null,
      "safe_display": {"label": "SPF authentication header", "redacted": true}
    }
  ]
}
```

Evidence types include `body`, `authentication`, `received`, `ip`, `domain`,
`url`, `attachment`, `ip_enrichment`, and `ai_assessment`. Provenance is one of
`OBSERVED`, `ENRICHED`, `INFERRED`, or `AI-ASSESSED`. Values and snippets are
bounded safe displays; complete raw email content, credentials, provider keys,
and private keys are never returned. Attachment hashes are returned only when
already present in parsed metadata. AI references are resolved only when they
match evidence supplied to the analyzer; unknown references are discarded.
Each risk signal in the canonical analysis also contains `evidence_ids` that
point to this list.

The endpoint returns `200 OK` for completed or partial analysis,
`404 EMAIL_NOT_FOUND` when the email does not exist, and
`404 ANALYSIS_NOT_FOUND` when analysis has not completed or does not exist.

### 11.3 Endpoint: Get Case Graph

`GET /api/cases/{case_id}/graph` derives an evidence-backed graph from the
case's persisted parsed emails, analysis results, enrichment, and evidence:

```json
{
  "case_id": "case_...",
  "email_ids": ["email_..."],
  "analysis_ids": ["analysis_..."],
  "nodes": [
    {
      "id": "ip:203.0.113.77",
      "type": "ip",
      "label": "203.0.113.77",
      "value": "203.0.113.77",
      "provenance": "OBSERVED",
      "evidence_ids": ["evidence_..."],
      "metadata": {}
    }
  ],
  "edges": [
    {
      "id": "edge:email:email_...:contains:ip:203.0.113.77",
      "source_node_id": "email:email_...",
      "target_node_id": "ip:203.0.113.77",
      "relationship": "contains",
      "provenance": "OBSERVED",
      "evidence_ids": ["evidence_..."],
      "confidence": null
    }
  ]
}
```

Node types are `email`, `sender`, `recipient`, `domain`, `url`, `ip`,
`relay`, `attachment`, `organization`, `geolocation`, `ai_assessment`, and
`risk_signal`. Relationships include `from`, `to`, `contains`,
`received_via`, `observed_ip`, `has_domain`, `associated_with`,
`located_in_estimate`, `assessed_by`, and `supported_by`. Graph IDs are
deterministic for the same case data. Nodes and edges carry only evidence IDs,
safe metadata, and bounded parsed values; raw email content and provider
secrets are never copied into the graph. Geolocation nodes are estimates
derived from an IP and are not a person's physical location or identity.

The endpoint returns `404 CASE_NOT_FOUND` when the case does not exist and
`404 GRAPH_NOT_AVAILABLE` when the case exists but has no completed or partial
analyzed email. The graph is derived on demand and is not stored separately.

### 11.4 Endpoint: Get Case Timeline

`GET /api/cases/{case_id}/timeline` derives a chronological, evidence-backed
timeline from persisted email, parsed data, analysis, enrichment, and evidence
records. Its response contains `case_id`, `email_ids`, and `events`:

```json
{
  "case_id": "case_...",
  "email_ids": ["email_..."],
  "events": [
    {
      "id": "timeline:email_...:relay:1",
      "type": "relay_inferred",
      "sequence": 1,
      "timestamp": "2026-09-10T13:45:00Z",
      "title": "Observed relay",
      "description": "A Received header identified a relay in the message path.",
      "hostname": "mx.example.org",
      "ip_address": "203.0.113.77",
      "source_header_order": 8,
      "provenance": "INFERRED",
      "confidence": "medium",
      "evidence_ids": ["evidence_..."],
      "related_node_id": "relay:email_...:8",
      "metadata": {"sequence_source": "received_header_order"}
    }
  ]
}
```

Event types are `email_received`, `relay_observed`, `relay_inferred`,
`authentication_observed`, `indicator_observed`, `enrichment_completed`, and
`analysis_completed`. Relay events preserve source header order, sequence,
hostname, IP, timestamp, confidence, provenance, evidence IDs, and the
corresponding graph relay node ID. Events are generated only from persisted
data and never contain raw email content, credentials, provider prompts, or
attachment contents.

Events with timestamps sort ascending. Timestamped events precede events with
no timestamp; equal timestamps use sequence when present, then the stable event
ID. The upload event uses the server-observed artifact creation time and is
explicitly labeled as an upload timestamp, not proof of sender location or
identity. Nullable timestamp, hostname, IP, header-order, and related-node
fields remain null when the source does not provide them. The backend never
invents timestamps.

The endpoint returns `404 CASE_NOT_FOUND` for a missing case and
`404 TIMELINE_NOT_AVAILABLE` when no completed or partial analyzed email is
available. The timeline is derived on demand rather than stored separately.

### 11.5 Endpoints: Forensic Report

`POST /api/cases/{case_id}/report` synchronously creates or refreshes the
latest structured report. `GET /api/cases/{case_id}/report` returns the report
derived from the same persisted case data. Both return `200 OK`; no request
body is required for the MVP.

The response has schema version `1.0`:

```json
{
  "report_id": "report:case_...:analysis_...",
  "schema_version": "1.0",
  "case": {
    "id": "case_...",
    "status": "created",
    "created_at": "2026-09-10T12:00:00Z",
    "updated_at": "2026-09-10T12:05:00Z"
  },
  "generated_at": "2026-09-10T12:05:01Z",
  "status": "completed",
  "emails": [],
  "analyses": [],
  "evidence": [],
  "graph": {"node_count": 0, "edge_count": 0, "node_types": {}, "node_ids": [], "edge_ids": []},
  "timeline": [],
  "limitations": []
}
```

The `analyses` entries contain risk score, level, verdict, confidence,
contributing signals, authentication, Received-chain data, IP enrichment, and
AI assessment. `emails` contain safe message metadata, indicators, attachment
metadata and hashes; the report never contains raw email bytes, plain-text
bodies, complete provider prompts, credentials, API keys, private keys, or
executable contents. The graph field is a summary of node counts, edge counts,
and node types; the graph endpoint remains the relationship API.

Report `status` is `completed` when all included analyses are completed and
`partial` when any included analysis is partial. Limitations state that AI
output is an evaluated assessment rather than ground truth, geolocation is an
IP-derived estimate rather than proof of physical location, URLs were not
automatically visited, attachments were not executed, and missing providers do
not imply benign behavior. Nested analysis data preserves `OBSERVED`,
`ENRICHED`, `INFERRED`, and `AI-ASSESSED` provenance.

The endpoint returns `404 CASE_NOT_FOUND` for an unknown case and
`404 REPORT_NOT_AVAILABLE` when no completed or partial analysis exists.
Reports are derived on demand; `report_id` is deterministic for a case and its
analysis IDs while `generated_at` changes when regenerated.

12. Upload-Only Email Response

If the email has been uploaded but analysis has not yet produced parsed data:

HTTP status:

200 OK

Example:

{
  "email_id": "email_01J...",
  "case_id": "case_01J...",
  "status": "uploaded",
  "filename": "suspicious.eml"
}
13. Processing Email Response

If analysis is currently running:

HTTP status:

200 OK

Example:

{
  "email_id": "email_01J...",
  "case_id": "case_01J...",
  "status": "processing"
}

The frontend should use this status to display the analysis/loading state.

14. Parsed Email Response

When the parsing stage has completed successfully:

HTTP status:

200 OK

Example:

{
  "email_id": "email_01J...",
  "case_id": "case_01J...",
  "status": "parsed",
  "filename": "suspicious.eml",

  "message": {
    "message_id": "<abc123@example.com>",
    "from": [
      "attacker@example.com"
    ],
    "to": [
      "victim@example.org"
    ],
    "cc": [],
    "reply_to": [
      "reply@example.net"
    ],
    "subject": "Urgent Account Notice",
    "date": "2026-09-08T10:20:30Z",
    "return_path": "bounce@example.com"
  },

  "mime": {
    "content_type": "multipart/alternative",
    "has_plain_text": true,
    "has_html": true,
    "attachment_count": 1
  },

  "headers": [
    {
      "name": "From",
      "value": "attacker@example.com",
      "order": 1
    },
    {
      "name": "Received",
      "value": "from mail.example.com by mx.example.org",
      "order": 2
    }
  ],

  "indicators": {
    "ips": [],
    "domains": [],
    "urls": []
  },

  "attachments": [
    {
      "filename": "invoice.pdf.exe",
      "mime_type": "application/octet-stream",
      "size_bytes": 145120,
      "sha256": "..."
    }
  ]
}
15. Message Metadata

The structured message object may contain:

message_id
from
to
cc
reply_to
subject
date
return_path

The API must preserve multiple addresses where applicable.

Fields absent from the source email must not be fabricated.

The backend must represent missing values consistently according to its type definitions.

16. Complete Header Representation

The API must provide the complete parsed header set.

Each header contains:

name
value
order

Example:

{
  "name": "Received",
  "value": "from mail.example.com by mx.example.org",
  "order": 4
}

Header order must preserve the order in which headers appear in the source email.

The complete header set is required for later stages including:

SPF analysis
DKIM analysis
DMARC analysis
Received-chain reconstruction
Header anomaly detection
Message-ID analysis
17. MIME Information

The mime object provides high-level structural information.

Required fields:

content_type
has_plain_text
has_html
attachment_count

The parser should preserve enough MIME information to support later forensic inspection.

Deep MIME analysis is outside the initial MVP parsing slice.

18. Indicator Extraction

The parser must extract and normalize:

IPv4
IPv6
Domains
URLs

The indicators object:

{
  "ips": [],
  "domains": [],
  "urls": []
}

Indicators should be deduplicated where appropriate while preserving source references internally.

The initial parser does not assign reputation or threat scores.

Reputation and enrichment belong to later analysis stages.

19. Attachments

The initial parser must detect attachments and return metadata.

Each attachment may include:

filename
mime_type
size_bytes
sha256

The backend must NOT:

execute attachments
execute embedded files
execute macros
launch extracted content

Attachment threat analysis belongs to later analyzer stages.

20. Raw Artifact Handling

The original .eml must be preserved by the backend.

Normal API responses must NOT return:

raw_content

or the complete raw email body as the default API representation.

The raw artifact should instead remain available to authorized backend forensic and evidence functions.

The original artifact is the source of truth for forensic reconstruction.

21. Parse Failure

If the uploaded .eml cannot be parsed:

HTTP status:

422 Unprocessable Entity

Example:

{
  "error": {
    "code": "EMAIL_PARSE_FAILED",
    "message": "The email could not be parsed."
  }
}

The original uploaded artifact MUST still be preserved.

The failed artifact must not be silently deleted.

22. Parse Failure Semantics

When parsing fails, the backend should preserve:

Case
Email record
Original raw artifact
Upload event
Parse failure information

The parsed representation may be incomplete or absent.

The failure must be visible to the frontend.

A parsing failure is not permission to discard the source evidence.

23. Partial Analysis

A non-fatal analyzer failure must not necessarily fail the entire analysis.

Example:

Email Parsing       ✓
Header Analysis     ✓
IP Extraction       ✓
IP Geolocation      ✗
AI Analysis         ✓

The final analysis may return:

status: partial

The response should clearly indicate unavailable or failed analyzer results where applicable.

The frontend must not represent missing enrichment as though the enrichment returned a negative result.

24. General Error Format

All API errors must use:

{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable explanation."
  }
}

The frontend must use code for programmatic behavior.

The frontend must NOT depend on exact human-readable message text.

25. Common Error Codes

Initial MVP error codes:

INVALID_REQUEST
MISSING_FILE
UNSUPPORTED_FILE_TYPE
FILE_TOO_LARGE
EMPTY_FILE
EMAIL_NOT_FOUND
EMAIL_PARSE_FAILED
ANALYSIS_NOT_FOUND
ANALYSIS_FAILED
INTERNAL_ERROR

Additional error codes may be introduced when a real requirement emerges.

26. HTTP Status Codes

Use standard HTTP semantics.

200 OK
201 Created
202 Accepted
400 Bad Request
404 Not Found
409 Conflict
413 Content Too Large
422 Unprocessable Entity
500 Internal Server Error

Do not use 200 to represent a failed operation.

27. Provenance Model

All future analytical fields must identify their epistemic class where applicable.

Supported classes:

OBSERVED
ENRICHED
INFERRED
AI-ASSESSED
OBSERVED

Directly present in the uploaded email or raw artifact.

Examples:

From header
Received header
URL in email body
Attachment filename
IP in header
ENRICHED

Obtained from external or local intelligence sources.

Examples:

MaxMind geolocation
DNS result
ASN
IP reputation
Domain reputation
INFERRED

Produced through deterministic rules or heuristics.

Examples:

Earliest reliable external relay
Header anomaly
Domain mismatch
AI-ASSESSED

Produced by an AI, ML, or LLM model.

Examples:

Phishing intent
Credential-harvesting intent
Social-engineering assessment

An INFERRED or AI-ASSESSED conclusion must be traceable to supporting observed evidence.

28. Risk and Confidence

Risk and confidence are separate concepts.

Example:

{
  "risk": {
    "score": 87,
    "level": "high"
  },
  "confidence": 0.93
}

Risk score represents threat severity.

Confidence represents confidence in the assessment.

Do not combine them into one percentage.

This distinction must remain consistent throughout the frontend and backend.

29. Remaining Future Investigation Endpoints

The following endpoints belong to later stages of the MVP:

Get Case
GET /api/cases/{case_id}
Get Case Analysis
GET /api/cases/{case_id}/analysis
Get Map Data
GET /api/cases/{case_id}/map
Get Evidence
GET /api/cases/{case_id}/evidence

Entity graph, forensic timeline, and structured report endpoints are already
implemented and documented in sections 11.3 through 11.5. Remaining future
endpoints must be added incrementally as their corresponding analysis stages
are implemented.

They are part of the long-term API design but are not requirements for the initial upload/parse slice.

30. Future Analysis Stages

The API is designed so that future analysis stages can be added without changing the basic upload model.

Future stages include:

SPF
DKIM
DMARC
Received-chain reconstruction
IP enrichment
Domain intelligence
URL analysis
AI intent
Risk scoring
Evidence
Graph
Map
Report

These stages should extend the existing email/case model rather than replace it.

31. Frontend / Backend Ownership
Codex-CLI

Codex owns backend implementation inside:

backend/

including:

HTTP handlers
email ingestion
parser
models
persistence
analysis execution
API responses
backend tests
Agy-CLI

Agy owns frontend implementation inside:

frontend/

including:

upload UI
analysis progress UI
parsed email display
API client
frontend state
frontend tests

Neither agent may modify the other agent's codebase without explicit user authorization.

32. Contract Change Rule

When a feature requires an API change:

Identify the required change.
Update this document.
Implement the backend side.
Implement the frontend side.
Test both sides.
Verify that previously supported requests remain compatible where possible.

Do not silently change:

endpoint paths
HTTP methods
request fields
response fields
field names
field types
error codes

without updating this document.

33. Frontend Contract Rules

The frontend must:

consume documented API fields only
handle documented status values
handle loading states
handle partial states
handle structured errors
avoid assuming optional data exists
not treat processing state as threat severity
not expose raw email content unnecessarily

Temporary frontend mocks may be used during isolated frontend development only when clearly separated from real API integration.

34. Backend Contract Rules

The backend must:

return documented structures
use stable field names
use stable data types
use documented error codes
validate all uploaded data
preserve the original artifact
avoid returning raw email content by default
report partial analyzer failures explicitly
avoid silently fabricating missing values
35. Security Rules

Uploaded email content must always be treated as untrusted input.

The backend must not:

execute attachments
execute scripts from email content
execute macros
automatically visit untrusted URLs
trust sender-provided metadata as authoritative
store API keys or secrets in source code

The frontend must not render untrusted raw email HTML directly into the application DOM.

Email HTML rendering must use an isolated, sandboxed approach as specified by the UI/UX design system.

36. MVP Constraints

Do not introduce the following for the initial upload and parsing slice:

GraphQL
gRPC
message brokers
microservices
streaming infrastructure
mandatory WebSockets
additional databases

REST + JSON + multipart upload is sufficient for the initial MVP.

The implementation should remain a modular monolith.

37. Design and Forensic Consistency

The API should provide enough structured information for the frontend to represent the forensic provenance model defined by the UI/UX specification:

OBSERVED
ENRICHED
INFERRED
AI-ASSESSED

The frontend should be able to connect analytical conclusions back to supporting evidence.

Geolocation must be treated as an estimate.

AI attribution must be treated as an assessment rather than confirmed identity.

Risk and confidence must remain separate.

38. Definition of Contract Completion

The initial API contract is considered implemented when:

.eml upload works.
Upload automatically creates a case.
Original .eml is preserved.
Upload returns case_id and email_id.
Analysis can be started separately.
Analysis state is represented explicitly.
Parsed email metadata is returned.
Complete parsed headers are returned.
MIME metadata is returned.
IP/domain/URL indicators are returned.
Attachment metadata is returned.
Parse failures return structured errors.
Failed artifacts remain preserved.
Frontend and backend use the same field names and types.
Relevant backend and frontend tests pass.
