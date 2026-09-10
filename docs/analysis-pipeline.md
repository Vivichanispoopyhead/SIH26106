
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

11. Stage 10 — Graph

Convert important entities and relationships into graph data.

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

12. Stage 11 — Map

Display infrastructure locations derived from geolocation data.

Map labels should communicate uncertainty.

Example:

Estimated location
Country: India
City: Bengaluru
Confidence: Medium
Source: MaxMind

Do not present geolocation as exact physical location or confirmed actor identity.

13. Stage 12 — Report

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
14. Failure Handling

A failed analyzer must not necessarily fail the entire pipeline.

Example:

IP Geo        ✓
IP Reputation ✓
DNS           ✓
LLM           ✗

The analysis may still complete with partial results.

The final result should clearly indicate missing or unavailable analysis.

15. Confidence

Confidence should be propagated and represented explicitly.

Confidence does not mean:

"the attacker is 93% likely to be this person"

It means confidence in the specific analytical conclusion.

16. Forensic Principle

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
