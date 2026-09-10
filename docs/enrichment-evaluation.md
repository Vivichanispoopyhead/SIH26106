# Passive IP Enrichment Evaluation Contract

## Scope

This is a design and evaluation contract for a future passive IP-enrichment
adapter. It authorizes neither live lookup nor any active action. It excludes
URL crawling, DNS/WHOIS/RDAP queries, scanning, attachment handling, autonomous
tool use, and model changes.

An enrichment record is provider-scoped `ENRICHED` context. Country, region,
city, latitude/longitude, ASN, ISP/organization, and hosting provider describe
an IP-derived estimate or provider claim. They do not identify a person, prove
sender location, prove maliciousness, or establish intent. A public IP, cloud
provider, or cloud ASN is not inherently suspicious.

## Provider result semantics

| Status | Meaning | Overall analysis effect |
| --- | --- | --- |
| `enriched` | A configured provider completed. `data_available: false` means it found no data. | Completed unless another stage is partial/failed. |
| `not_applicable` | The address is not eligible (private, loopback, link-local, or documentation space); provider did not run. | Completed; no risk implication. |
| `not_configured` | No provider was configured; provider did not run. | Completed; no risk implication. |
| `failed` | An eligible lookup was attempted and failed or timed out. | Partial when enrichment is requested; no maliciousness implication. |
| `partial` | Usable documented fields were returned, but required provider fields were missing or conflict. | Partial; retain only safe returned fields. |

Store only the provider name, retrieval timestamp, status, provider confidence
when documented, normalized documented fields, and a stable evidence ID such
as `enrichment-{provider}-{ip-index}`. Do not log or expose an opaque full
provider response. Unknown or unavailable reputation is neither clean nor
malicious.

## AI interaction policy

Enrichment is not part of the current `ai.Input`; Gemini must not be given
provider geography, ASN, hosting, or reputation data incidentally. The risk
engine consumes valid enrichment separately. Any future explicit AI-input
extension must label the values `ENRICHED`, include provider/retrieval time,
state that provider confidence is not threat confidence, prohibit nationality,
location, and actor attribution, and require `unknown` when evidence is
insufficient. Gemini never invents provider results or evidence IDs.

## Bounded risk recommendations

| Stable code | Points | Provenance | Preconditions | Forbidden interpretation |
| --- | ---: | --- | --- | --- |
| `ENRICHMENT_METADATA_AVAILABLE` | 0 | ENRICHED | Provider completed with data. | Metadata makes the IP safe or malicious. |
| `ENRICHMENT_NO_DATA` | 0 | ENRICHED | Provider completed with `data_available: false`. | No provider data means clean. |
| `ENRICHMENT_UNAVAILABLE` | 0 | ENRICHED | Provider not configured or lookup failed. | Unavailability means malicious or benign. |
| `ENRICHMENT_LOW_PROVIDER_CONFIDENCE` | 0 | ENRICHED | Provider’s documented confidence is below an approved threshold. | Low provider confidence raises threat severity. |
| `ENRICHMENT_PROVIDER_ABUSE_REPORTED` | bounded, e.g. +15 once | ENRICHED | A documented provider abuse/reputation observation, provider/time/evidence ID present, and provider confidence meets the configured threshold. | Country, ASN, cloud/hosting, or public-IP status is an abuse signal. |

The backend risk engine must clamp this enrichment contribution, preserve the
provider evidence ID, and never let it override stronger executable-attachment
or authentication evidence. Any platform interpretation beyond the provider
field is `INFERRED`, separately labeled, and needs deterministic tests.

## Local evaluation matrix

| Case | Expected status / provenance | Allowed signal | Forbidden conclusion | Confidence / analysis status |
| --- | --- | --- | --- | --- |
| Private IP | `not_applicable`, ENRICHED | None | Private IP is safe or malicious. | No enrichment confidence; completed. |
| Documentation IP | `not_applicable`, ENRICHED | None | Test/documentation address reflects real infrastructure. | No enrichment confidence; completed. |
| Ordinary public business relay | `enriched`, ENRICHED | `ENRICHMENT_METADATA_AVAILABLE` (0) | Organization or country proves legitimacy/malice. | Provider confidence is context only; completed. |
| Public cloud relay, no abuse | `enriched`, ENRICHED | metadata/no-data (0) | Cloud hosting is suspicious. | No threat-confidence increase; completed. |
| Documented provider abuse result | `enriched`, ENRICHED | one bounded `ENRICHMENT_PROVIDER_ABUSE_REPORTED` | Provider result identifies an attacker or proves intent. | Provider confidence bounds use; completed. |
| Provider returns no reputation | `enriched`, ENRICHED, `data_available: false` | `ENRICHMENT_NO_DATA` (0) | No reputation means clean. | No threat-confidence increase; completed. |
| Provider timeout | `failed`, ENRICHED | `ENRICHMENT_UNAVAILABLE` (0) | Timeout means malicious. | No enrichment confidence; partial. |
| Conflicting/incomplete provider metadata | `partial`, ENRICHED | low-confidence context (0), abuse only if independently documented | One conflicting field is authoritative. | Cap/withhold enrichment confidence; partial. |
| AI phishing, neutral enrichment | `enriched`, ENRICHED | metadata/no-data (0); AI handled separately | Neutral geography disproves phishing. | AI confidence remains separate; completed if AI valid. |
| AI unknown, provider abuse | `enriched`, ENRICHED | bounded provider-abuse signal | Abuse result proves phishing/actor identity. | Provider confidence is not AI confidence; completed. |
| Authentication fail, neutral enrichment | `enriched`, ENRICHED | metadata/no-data (0); auth handled separately | Neutral enrichment cancels authentication evidence. | Deterministic auth confidence remains; completed. |
| Authentication pass, provider abuse | `enriched`, ENRICHED | bounded provider-abuse signal | Authentication pass makes abuse result irrelevant or IP safe. | Combine bounded evidence conservatively; completed. |

These cases should become table-driven adapter/risk tests when a passive
provider is implemented. No live provider is needed for unit tests.
