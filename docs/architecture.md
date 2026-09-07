# System Architecture

## 1. Purpose

SIH26106 is an MVP AI-powered email threat detection, geolocation, and forensic intelligence platform.

The platform accepts raw `.eml` files and produces a structured forensic analysis including:

- Email and header parsing
- SPF/DKIM/DMARC validation
- Received-header chain reconstruction
- URL, domain, and IP extraction
- IP geolocation and reputation enrichment
- Domain intelligence
- AI-based intent and threat analysis
- Risk and confidence assessment
- Evidence preservation
- Entity and relationship graph
- Geographic visualization
- Forensic PDF report

The system is intended as an MVP demonstration and investigative aid, not as a production-grade enterprise security platform.

---

## 2. Architectural Philosophy

Priorities, in order:

1. Correctness
2. Clear separation of responsibilities
3. End-to-end functionality
4. Ease of development and debugging
5. Reasonable extensibility
6. Performance

Do not introduce architectural complexity without a demonstrated need.

The MVP uses a modular monolithic backend rather than microservices.

---

## 3. High-Level Architecture

```text
React + TypeScript Frontend
          |
     REST / WebSocket
          |
   Go Backend (Chi Router)
          |
  +-------+-------+-------+-------+
  |               |               |
Parser &       Analyzers       Risk / Case
Forensics      & Enrichment      Engine
  |               |               |
  +---------------+---------------+
                  |
             PostgreSQL
              + pgvector
                  |
                Redis
            + external APIs

4. Frontend

Technology:

React
TypeScript

Primary responsibilities:

Case creation
.eml upload
Display analysis results
Case overview
Threat and risk visualization
Received-chain timeline
Entity relationship graph
Geographic map
Evidence viewer
Forensic report generation/download
Display analysis confidence and provenance

The frontend must not implement forensic analysis logic that belongs to the backend.

5. Backend

Technology:

Go
Chi router

The backend is a modular monolith.

Suggested internal packages:

backend/
├── cmd/
├── internal/
│   ├── parser/
│   ├── authcheck/
│   ├── analyzers/
│   ├── engine/
│   ├── audit/
│   ├── case/
│   └── ...
└── ...
Parser

Responsible for:

Parsing raw .eml
Extracting headers
Extracting body/content
Extracting attachments
Extracting URLs/domains/IPs
Detecting relevant email artifacts
Authcheck

Responsible for:

SPF
DKIM
DMARC
Header authentication/alignment analysis
Analyzers

Responsible for enrichment and external intelligence such as:

IP geolocation
IP reputation
DNS
MX
Domain intelligence
URL analysis
Other approved external intelligence providers
LLM analysis

External services should be accessed through internal interfaces so implementations can be replaced later.

Engine

Responsible for:

Combining analysis signals
Producing risk score
Producing confidence
Producing verdict
Producing recommended actions where applicable
Audit

Responsible for:

Evidence events
Hashing
Chain-of-custody metadata
Tamper-evident event history
6. Data Storage
PostgreSQL

Primary persistent store for:

Cases
Emails
Indicators
Entities
Analysis results
Evidence metadata
Audit events
pgvector

Used for:

Vector representations where required
Similarity/search functionality
Future campaign correlation

Do not introduce vector search into a feature unless it provides a concrete MVP benefit.

Redis

Used for:

External analyzer caching
Short-lived analysis state
Rate limiting where required
Cached enrichment results
MaxMind

Used as the local IP geolocation source where available.

7. External Interfaces

External integrations must be isolated behind interfaces.

Examples:

URL reputation
IP reputation
Domain intelligence
DNS
LLM provider

The core forensic pipeline must not be tightly coupled to a single external provider.

8. Core Data Flow
.EML
 ↓
Parse
 ↓
Authentication Checks
 ↓
Received Chain Reconstruction
 ↓
Indicator Extraction
 ↓
External / Local Enrichment
 ↓
AI Intent Analysis
 ↓
Risk + Confidence Engine
 ↓
Evidence Recording
 ↓
Case / Entity Graph
 ↓
Map + Timeline
 ↓
Forensic Report
9. Important Architectural Constraint

Observed facts must be separated from inferred conclusions.

Examples:

Observed:

An IP appeared in a Received header.

Enriched:

The IP belongs to a hosting provider.

Inferred:

The IP is likely an originating relay.

AI assessment:

The infrastructure appears suspicious.

The system must never present an inference as a directly observed fact.

10. MVP Scope

The MVP should prioritize a complete end-to-end workflow over exhaustive feature coverage.

A feature is preferred when it makes the primary workflow demonstrably functional.

Avoid adding:

Microservices
Message brokers
Kubernetes
Complex event-driven architecture
Distributed systems infrastructure
Unnecessary abstraction layers

unless a concrete requirement emerges.

11. Architectural Change Rule

Existing architecture should be extended before being replaced.

Any significant architectural change must:

Solve a real identified problem.
Preserve existing API and data contracts where possible.
Be documented.
Avoid introducing unnecessary infrastructure.
