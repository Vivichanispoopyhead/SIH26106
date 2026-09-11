
---

# `data-model.md`

```md
# Data Model

## 1. Purpose

This document defines the conceptual entities used by the system.

It is the source of truth for the meaning and relationships of core data.

Database-specific implementation details should follow this model rather than redefining it.

---

## 2. Core Entities

### Case

Represents an investigation.

```text
Case
- id
- title
- description
- status
- created_at
- updated_at

A case may contain one or more analyzed emails.

Email

Represents an uploaded raw email.

Email
- id
- case_id
- filename
- raw_content
- message_id
- sender
- recipients
- subject
- received_at
Header

Represents an individual email header.

Header
- id
- email_id
- name
- value
- order
Relay

Represents a server or node discovered from the Received chain.

Relay
- id
- email_id
- hostname
- ip
- timestamp
- sequence
- confidence
IP Address

Represents an IP indicator.

IP
- id
- value
- version
Domain

Represents a domain indicator.

Domain
- id
- name
URL

Represents a URL extracted from an email.

URL
- id
- value
- domain
IOC

Represents an indicator of compromise.

Possible types:

ip
domain
url
email
hash
attachment
Entity

Represents an object participating in the investigation graph.

Examples:

Email
IP
Domain
URL
Person/Identity
Mail Server
Organization
Evidence

Represents information used to support an investigative conclusion.

Evidence
- id
- case_id
- email_id
- analysis_id
- type
- source
- value
- snippet
- observed_or_derived
- header_order
- related_signal_codes
- related_ai_evidence_references
- created_at
- hash
- safe_display

Graph

Represents the evidence-backed relationships for one case.

Graph
- case_id
- email_ids
- analysis_ids
- nodes
- edges

GraphNode
- id
- type
- label
- value
- provenance
- evidence_ids
- metadata

GraphEdge
- id
- source_node_id
- target_node_id
- relationship
- provenance
- evidence_ids
- confidence

Graph nodes and edges are derived from persisted analysis and evidence rather
than maintained as a second graph database copy.

Timeline

Represents a chronological view derived from persisted case data.

Timeline
- case_id
- email_ids
- events

TimelineEvent
- id
- type
- sequence
- timestamp
- title
- description
- hostname
- ip_address
- source_header_order
- provenance
- confidence
- evidence_ids
- related_node_id
- metadata

Timeline timestamps and source fields are nullable. Received-chain sequence
and source header order remain separate because chronology can be inferred from
header order. Timeline events are derived on demand instead of persisted as a
second event store.

Forensic Report

Represents a safe structured snapshot assembled from persisted case analysis.

ForensicReport
- report_id
- schema_version
- case
- generated_at
- status
- emails
- analyses
- evidence
- graph summary
- timeline
- limitations

Reports contain metadata and safe evidence displays only. Raw email bytes,
provider prompts, secrets, and executable contents are excluded. Reports are
derived on demand rather than stored in a separate reporting database.
Analysis Result

Represents output from an analyzer.

AnalysisResult
- id
- case_id
- analyzer
- status
- result
- confidence
- created_at
Risk Assessment

Represents the aggregated threat assessment.

RiskAssessment
- score
- level
- verdict
- confidence
- contributing_signals
- evidence_references

RiskSignal
- code
- description
- points
- category
- provenance
- evidence_references
IP Enrichment

Represents optional passive metadata for an observed IP address. It is not a
reputation verdict or proof of identity.

IPEnrichment
- ip_address
- status
- country / region / city
- latitude / longitude
- asn
- organization
- provider
- confidence
- retrieved_at
- provenance
- failure

Statuses are `enriched`, `not_applicable`, `failed`, and `not_configured`.
Non-public and invalid addresses are classified locally and are never sent to
external providers.
Geolocation

Represents estimated geographic information associated with infrastructure.

Geolocation
- ip
- country
- region
- city
- latitude
- longitude
- provider
- confidence

Geolocation is an estimate and must not be treated as proof of a person's physical location.

### Passive IP Enrichment (planned contract)

An IP enrichment record is separate from an observed IP indicator and contains
only provider-scoped metadata: provider name, retrieval timestamp, status,
`data_available`, provider confidence when documented, and selected normalized
fields (country, region, city, coordinates, ASN, ISP/organization, hosting
provider, and documented reputation observations). Its provenance is
`ENRICHED`; any platform conclusion derived from it is `INFERRED`.

Status is one of `enriched`, `not_applicable`, `not_configured`, `failed`, or
`partial`. `enriched` with `data_available: false` means the provider ran but
had no matching data. It is not equivalent to benign or clean. Private,
loopback, link-local, and documentation addresses are normally
`not_applicable`; absent configuration is `not_configured`; attempted provider
errors are `failed`; usable incomplete or conflicting fields are `partial`.

Country, city, ASN, ISP, hosting provider, and public-IP status are context
only. They cannot identify an actor, prove physical location, or independently
add risk points. A documented provider abuse observation may be preserved as
`ENRICHED` evidence, but it remains a provider assertion rather than an
observed fact from the email.

Audit Event

Represents a tamper-evident event in the investigation history.

AuditEvent
- id
- case_id
- event_type
- timestamp
- actor
- payload_hash
- previous_hash
- event_hash
3. Relationships
Case
 ├── Email
 │    ├── Header
 │    ├── Relay
 │    ├── URL
 │    ├── Domain
 │    ├── IOC
 │    └── Attachment
 │
 ├── AnalysisResult
 ├── RiskAssessment
 ├── Evidence
 ├── Entity
 └── AuditEvent

IP ─── Geolocation

URL ─── Domain

Entity ─── Entity
        via graph relationships
4. Evidence Provenance

Every important conclusion should be traceable to evidence.

Example:

Risk signal
    ↓
Analyzer result
    ↓
Observed indicator
    ↓
Original email/header/content
5. Observed vs Derived Data
Observed

Directly present in the email or directly obtained from a trusted source.

Examples:

Header value
URL in email body
IP in Received header
Attachment hash
Derived

Produced through analysis.

Examples:

Parsed relay
Geolocation
Domain reputation
Risk score
AI classification

The system should preserve this distinction.

6. Graph Model

Graph nodes represent entities.

Graph edges represent relationships.

Example:

Email
  │
  ├── contains → URL
  │                │
  │                └── belongs_to → Domain
  │
  └── received_from → IP
                         │
                         └── located_in → GeoLocation

The graph should represent evidence-backed relationships rather than arbitrary associations.
