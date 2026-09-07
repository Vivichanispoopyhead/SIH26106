
---

# `api-contract.md`

```md
# API Contract

## 1. Purpose

This document defines the communication contract between the React frontend and Go backend.

The frontend and backend must implement this contract rather than independently inventing request and response formats.

Changes to this document require corresponding changes in both implementations.

---

## 2. Base URL

Development:

```text
/api

The frontend must not hardcode environment-specific hostnames.

Use environment configuration.

3. API Style
REST over HTTP for request/response operations.
JSON for structured API data.
Multipart form-data for .eml uploads.
WebSocket may be used for long-running analysis progress.
4. Primary Endpoints
Create Case
POST /api/cases

Response:

{
  "case_id": "case_123",
  "status": "created"
}
Upload Email
POST /api/cases/{case_id}/emails
Content-Type: multipart/form-data

Input:

file=<email.eml>

Response:

{
  "email_id": "email_123",
  "case_id": "case_123",
  "status": "accepted"
}
Start Analysis
POST /api/cases/{case_id}/analysis

Response:

{
  "analysis_id": "analysis_123",
  "status": "started"
}
Get Case
GET /api/cases/{case_id}
Get Analysis
GET /api/cases/{case_id}/analysis
Get Graph
GET /api/cases/{case_id}/graph
Get Timeline
GET /api/cases/{case_id}/timeline
Get Map Data
GET /api/cases/{case_id}/map
Get Evidence
GET /api/cases/{case_id}/evidence
Generate Report
POST /api/cases/{case_id}/report
Download Report
GET /api/cases/{case_id}/report
5. Analysis Response

The primary analysis response should expose a structure similar to:

{
  "analysis_id": "analysis_123",
  "case_id": "case_123",
  "status": "completed",

  "risk": {
    "score": 87,
    "level": "high",
    "confidence": 0.93
  },

  "verdict": {
    "label": "phishing",
    "confidence": 0.93
  },

  "sender": {},
  "authentication": {},
  "received_chain": [],
  "indicators": [],
  "entities": [],
  "geolocation": {},
  "ai_analysis": {},
  "evidence": []
}

The exact schema should be expanded as implementation proceeds.

Do not add arbitrary fields independently in frontend and backend.

6. Evidence Representation

Evidence should contain provenance.

Example:

{
  "id": "evidence_123",
  "type": "received_header",
  "source": "email_header",
  "value": "...",
  "observed": true
}

Derived data should identify its origin.

Example:

{
  "type": "ip_geolocation",
  "source": "maxmind",
  "derived_from": "203.0.113.10"
}
7. Risk

Risk is represented numerically and categorically.

{
  "score": 87,
  "level": "high",
  "confidence": 0.93
}

Score and confidence are distinct.

Score = estimated threat/risk severity.
Confidence = confidence in the assessment.
8. Errors

Errors should have a predictable structure:

{
  "error": {
    "code": "INVALID_EMAIL",
    "message": "The uploaded file could not be parsed."
  }
}

Frontend code should depend on code, not on matching human-readable messages.

9. HTTP Status Codes

Use standard semantics.

Examples:

200 successful request
201 resource created
202 accepted for processing
400 invalid request
404 resource not found
409 conflict
422 semantically invalid input
500 internal server error
10. Contract Rules
Do not silently rename fields.
Do not change data types without updating this document.
Do not return frontend-specific structures from backend business logic.
Backend responses must remain deterministic in structure.
Frontend must not rely on undocumented fields.
