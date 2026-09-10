
---

# `analysis-pipeline.md`

```md
# Analysis Pipeline

## 1. Overview

The system processes an uploaded `.eml` through a sequence of analysis stages.

```text
UPLOAD
  ↓
PARSE
  ↓
AUTHENTICATION
  ↓
RECEIVED CHAIN
  ↓
INDICATOR EXTRACTION
  ↓
ENRICHMENT
  ↓
AI INTENT ANALYSIS
  ↓
RISK + CONFIDENCE
  ↓
EVIDENCE
  ↓
GRAPH
  ↓
TIMELINE
  ↓
MAP
  ↓
REPORT

Each stage should have a clear responsibility and produce structured output.

2. Stage 1 — Upload

Input:

raw .eml file

Tasks:

Validate file
Assign email ID
Associate email with case
Preserve raw email
Record upload event

The original email must not be destroyed or silently rewritten.

3. Stage 2 — Parse

Extract:

Standard headers
Received headers
Message-ID
Return-Path
From
To
Cc
Reply-To
Subject
Date
MIME parts
Body
Attachments
URLs
Domains
IP addresses

Parsing errors should be recorded rather than silently ignored.

4. Stage 3 — Authentication

Analyze:

SPF
DKIM
DMARC
Alignment
Authentication-related header signals

Output should distinguish:

pass
fail
neutral
none
unknown

Authentication results are signals, not standalone proof of legitimacy or maliciousness.

The deterministic authentication stage reads only Authentication-Results
headers already present in the artifact. Missing values are represented as
`none`, malformed or unrecognized values as `unknown`, and every reported
value carries a source header order reference.

5. Stage 4 — Received Chain

Parse all relevant Received headers.

Reconstruct the relay chain in chronological order where possible.

Each relay should contain:

hostname
ip
timestamp
sequence
source evidence
confidence

The system should identify the earliest reliable infrastructure observed in the chain.

Do not automatically assume the earliest visible IP is the true physical origin.

Received relays preserve the source header order. Chronological sequence and
confidence are inferred from the observed header order and safely parseable
hostname, IP, and timestamp fields; malformed headers remain represented with
low confidence rather than being discarded.

6. Stage 5 — Indicator Extraction

Extract and normalize:

IPv4
IPv6
Domains
URLs
Email addresses
Attachment metadata
File hashes where applicable

Indicators should be deduplicated while preserving references to where they were observed.

7. Stage 6 — Enrichment

Possible enrichment:

IP
Geolocation
ASN
ISP
Hosting provider
Reputation
Domain
DNS
MX
Registration information where available
Reputation
URL
Domain
Redirect information where safely available
Reputation

External enrichment should be cached where practical.

Failures from external providers should not crash the entire analysis.

The first passive enrichment slice processes each deduplicated IP observed in
indicators or the Received chain. Address classification happens before a
provider call; private, loopback, link-local, multicast, unspecified,
documentation/test, and invalid values produce `not_applicable` results without
network access. A configured HTTP adapter returns provider-independent
metadata, while missing configuration returns `not_configured`. Provider
timeouts, HTTP errors, oversized or malformed responses produce `failed`
results with stable failure codes. One failed IP does not discard successful
results for other IPs. Enrichment failures make the overall analysis `partial`
when applicable; an explicit `not_configured` result is non-fatal and preserves
the existing completed analysis behavior. Parsed, authentication, AI, and risk
data remain available. Geolocation metadata is contextual and must never be treated as
proof of identity or maliciousness.

8. Stage 7 — AI Intent Analysis

Analyze the semantic content of the email.

The backend exposes a typed analyzer adapter boundary. Until an adapter is
configured, the AI assessment is explicitly `not_available`; it must not
contain a fabricated classification or confidence. Analyzer failures may be
represented as a partial analysis result.

The optional Gemini adapter reads `GEMINI_API_KEY`, `GEMINI_MODEL`,
`GEMINI_API_URL`, `GEMINI_TIMEOUT`, and `GEMINI_MAX_INPUT_CHARS` from the
server environment. An absent API key selects the explicit `not_available`
assessment. Provider, timeout, and output-validation failures return failed
adapter assessments; the existing pipeline records the overall result as
`partial` after the parser has preserved the email. Tests use local HTTP
servers and do not require provider credentials.

Gemini receives untrusted email data as evidence, never as instructions. Prompt
injection text must not be followed, URLs must not be browsed, and attachments
must not be executed. The adapter accepts only the approved classification
vocabulary and evidence references supplied by the backend. Unknown response
fields, unsupported classifications, invalid confidence values, and invented
evidence references are rejected.

Classification guidance is conservative: `benign` means no meaningful
malicious indicators were found, not guaranteed safety; `suspicious` means
concerning but inconclusive evidence; `phishing` covers credential theft,
login harvesting, or malicious-link behavior; `credential_harvesting` is the
more specific label when appropriate; `malware` requires supplied payload or
strong attachment-delivery evidence; and `fraud` or
`payment_manipulation` require financial deception evidence. Urgency alone and
authentication failure alone are not proof of maliciousness. Insufficient or
contradictory evidence should produce `unknown` or a conservative `partial`
assessment.

Before any analyzer is called, the service passes input through a typed safety
boundary. Plain text, headers, indicators, and attachment metadata have
independent bounds and the serialized input has a hard total limit of 24,000
characters. The defaults are 12,000 body characters, 100 headers, 50 values
per indicator type, and 50 attachment metadata records. Attachment bytes are not part of the analyzer input. URL values are replaced by stable
evidence slots, and URLs are never fetched or sent to the provider. Obvious
credentials and tokens are redacted from body and header text without losing
header order or evidence location. Prompt-injection-like content is treated as
untrusted data by the provider instruction and is never executed.

Provider responses are schema-validated: status and classification use the
documented vocabulary, completed responses require classification and
confidence in the range 0..1, signal counts and lengths are bounded, and every
evidence reference must identify input supplied to the analyzer. Unknown
assessment fields are rejected. Empty, malformed, contradictory, or invalid
responses produce a structured AI failure. Provider telemetry records only
provider/model, prompt version, duration, status, and stable failure code.
unavailable, failed, or low-confidence AI output never becomes a benign result;
deterministic observable evidence remains authoritative for risk scoring.

Possible signals:

Urgency
Credential harvesting
Payment manipulation
Executive impersonation
Social engineering
Suspicious requests
Phishing intent
Fraud indicators

AI output must contain:

Classification
Confidence
Supporting signals
Explanation or evidence references where possible

AI output is an assessment, not ground truth.

9. Stage 8 — Risk Engine

Combine signals from previous stages.

Example signal categories:

Authentication anomalies
Header anomalies
Relay anomalies
Suspicious domain
Suspicious URL
IP reputation
Geolocation anomalies
Social engineering signals
AI classification
Attachment indicators

Output:

risk score
risk level
verdict
confidence
contributing signals

The scoring system should be deterministic for identical inputs when using deterministic analyzer inputs.

The MVP risk rules are: SPF fail +20, DKIM fail +20, DMARC fail +20,
unknown authentication +10, extracted URL +10, extracted IP +5, executable
or double-extension attachment +25, and multiple extracted indicators +5.
AI phishing or credential-harvesting contributes up to +30, malware up to +35,
and fraud or payment-manipulation up to +30; each AI contribution is scaled by
the bounded AI confidence. Scores are clamped to 100. Levels are low 0-24,
medium 25-49, high 50-74, and critical 75-100.

Verdict precedence is malware for executable attachments or accepted malware
assessment; phishing for accepted phishing or credential-harvesting assessment
with URL or authentication support; fraud for accepted fraud or
payment-manipulation assessment with deterministic support; suspicious for
strong deterministic signals; benign only for a valid benign AI assessment
without deterministic signals; otherwise unknown. AI output remains
AI-ASSESSED and observed evidence remains separately attributed.

10. Stage 9 — Evidence

Important evidence must be preserved.

Evidence should identify:

What was observed
Where it came from
Which analyzer produced it
Whether it is observed or derived
Timestamp
Hash where applicable

Evidence must support the final assessment.

The evidence stage materializes safe provenance from parsed headers, indicators,
attachment metadata, enrichment results, AI references, and risk signals.
Evidence IDs are stable within an email and analysis result. Header order and
source locations are retained where available. Body and header snippets are
bounded and redact obvious credentials; the raw `.eml` is never returned.
Unknown AI references are ignored rather than converted into evidence. Risk
signals carry evidence IDs so the investigation workspace can trace each
conclusion to observed or derived data.

11. Stage 10 — Graph

Convert important entities and relationships into graph data only when the
relationship is supported by parsed data or an existing evidence ID. The
backend exposes `GET /api/cases/{case_id}/graph`, deriving deterministic node
and edge IDs from persisted email, analysis, enrichment, and evidence data.
Observed sender, recipient, URL, domain, IP, relay, and attachment nodes retain
`OBSERVED` provenance. Reconstructed relays and risk relationships are
`INFERRED`; provider organization/geolocation nodes are `ENRICHED`; AI nodes
are `AI-ASSESSED`. Unknown AI claims never create graph entities. Geolocation
remains an estimate derived from an IP, never a person's confirmed location.

Example:

Email
 ↓
Domain
 ↓
IP
 ↓
Infrastructure
 ↓
Geolocation

Relationships should have provenance.

## 12. Stage 11 — Timeline

Derive `GET /api/cases/{case_id}/timeline` from persisted upload, parsed,
authentication, Received-chain, indicator, enrichment, analysis, and evidence
data. Relay events preserve source header order, sequence, safely parsed
timestamps, confidence, provenance, evidence IDs, and the corresponding graph
relay node ID. Upload and analysis completion timestamps come from server-side
persisted records; no current time or sender location is invented. Timeline
ordering is timestamp ascending, then sequence, then stable event ID, with
untimestamped events after timestamped events. Missing timestamps remain null.

Observed header facts remain `OBSERVED`; reconstructed relay ordering is
`INFERRED`; provider metadata is `ENRICHED`; and any future model-derived
annotation would be `AI-ASSESSED`. The timeline is derived on demand and does
not become a second event database.

13. Stage 12 — Map

Display infrastructure locations derived from geolocation data.

Map labels should communicate uncertainty.

Example:

Estimated location
Country: India
City: Bengaluru
Confidence: Medium
Source: MaxMind

Do not present geolocation as exact physical location or confirmed actor identity.

14. Stage 13 — Report

The final report should combine:

Case information
Email summary
Authentication results
Relay timeline
Indicators
Enrichment
AI assessment
Risk assessment
Evidence
Graph relationships
Geographic findings
Limitations
15. Failure Handling

A failed analyzer must not necessarily fail the entire pipeline.

Example:

IP Geo        ✓
IP Reputation ✓
DNS           ✓
LLM           ✗

The analysis may still complete with partial results.

The final result should clearly indicate missing or unavailable analysis.

16. Confidence

Confidence should be propagated and represented explicitly.

Confidence does not mean:

"the attacker is 93% likely to be this person"

It means confidence in the specific analytical conclusion.

17. Forensic Principle

The pipeline must preserve the distinction between:

Observed
↓
Parsed
↓
Enriched
↓
Inferred
↓
AI-Assessed

The further a conclusion is from the original evidence, the more explicitly its derivation should be represented.
