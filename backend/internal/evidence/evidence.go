package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"sih26106/backend/internal/domain"
)

const (
	Observed   = "OBSERVED"
	Enriched   = "ENRICHED"
	Inferred   = "INFERRED"
	AIAssessed = "AI-ASSESSED"
)

var (
	authPattern       = regexp.MustCompile(`(?i)\b(spf|dkim|dmarc)\s*=\s*([a-z]+)\b`)
	secretPattern     = regexp.MustCompile(`(?i)(password|passwd|pwd|api[_ -]?key|secret|token|authorization|bearer)\s*[:=]\s*[^\s,;]+`)
	bearerPattern     = regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+/-]+=*`)
	privateKeyPattern = regexp.MustCompile(`(?s)-----BEGIN [^-]*PRIVATE KEY-----.*?-----END [^-]*PRIVATE KEY-----`)
	urlSecretPattern  = regexp.MustCompile(`(?i)([?&](?:token|api[_-]?key|secret|password|session)=)[^&\s]+`)
)

type builder struct {
	emailID, analysisID string
	items               []domain.Evidence
	byKey               map[string]int
	byReference         map[string][]string
}

// Build derives safe, typed evidence only from parsed data and the canonical
// analysis result. It never reads or returns the raw uploaded artifact.
func Build(emailID string, parsed *domain.ParsedEmail, result *domain.AnalysisResult) []domain.Evidence {
	if parsed == nil || result == nil {
		return []domain.Evidence{}
	}
	b := &builder{emailID: emailID, analysisID: result.AnalysisID, byKey: map[string]int{}, byReference: map[string][]string{}}
	if parsed.PlainTextBody != "" {
		id := b.add(domain.Evidence{Type: "body", Source: "plain_text_body", Value: "body-1", Snippet: safeText(parsed.PlainTextBody, 320), Provenance: Observed, SourceLocation: "plain_text_body", SafeDisplay: domain.EvidenceDisplay{Label: "Plain-text body excerpt", Redacted: true}})
		b.reference("body-1", id)
	}
	for index, value := range parsed.Message.From {
		id := b.add(domain.Evidence{Type: "sender", Source: "From", Value: safeText(value, 256), Snippet: safeText(value, 256), Provenance: Observed, SourceLocation: "message.from", SafeDisplay: domain.EvidenceDisplay{Label: "Sender"}})
		b.reference(fmt.Sprintf("sender-%d", index+1), id)
	}
	recipientIndex := 0
	for _, values := range [][]string{parsed.Message.To, parsed.Message.CC, parsed.Message.ReplyTo} {
		for _, value := range values {
			recipientIndex++
			id := b.add(domain.Evidence{Type: "recipient", Source: "To/Cc/Reply-To", Value: safeText(value, 256), Snippet: safeText(value, 256), Provenance: Observed, SourceLocation: "message.recipient", SafeDisplay: domain.EvidenceDisplay{Label: "Recipient"}})
			b.reference(fmt.Sprintf("recipient-%d", recipientIndex), id)
		}
	}
	b.addAuthentication(parsed.Headers, result.Authentication)
	b.addReceived(parsed.Headers, result.ReceivedChain)
	for index, value := range parsed.Indicators.URLs {
		id := b.add(domain.Evidence{Type: "url", Source: "indicator", Value: safeURL(value), Snippet: safeURL(value), Provenance: Observed, SourceLocation: "indicator.url", SafeDisplay: domain.EvidenceDisplay{Label: "Extracted URL", Redacted: safeURL(value) != value}})
		b.reference(fmt.Sprintf("url-%d", index+1), id)
	}
	for index, value := range parsed.Indicators.Domains {
		id := b.add(domain.Evidence{Type: "domain", Source: "indicator", Value: safeText(value, 256), Snippet: safeText(value, 256), Provenance: Observed, SourceLocation: "indicator.domain", SafeDisplay: domain.EvidenceDisplay{Label: "Extracted domain"}})
		b.reference(fmt.Sprintf("domain-%d", index+1), id)
	}
	for index, value := range parsed.Indicators.IPs {
		id := b.add(domain.Evidence{Type: "ip", Source: "indicator", Value: value, Snippet: value, Provenance: Observed, SourceLocation: "indicator.ip", SafeDisplay: domain.EvidenceDisplay{Label: "Extracted IP"}})
		b.reference(fmt.Sprintf("ip-%d", index+1), id)
		b.reference("ip-value:"+value, id)
	}
	for index, attachment := range parsed.Attachments {
		var hash *string
		if attachment.SHA256 != "" {
			value := attachment.SHA256
			hash = &value
		}
		id := b.add(domain.Evidence{Type: "attachment", Source: "attachment", Value: safeText(attachment.Filename, 256), Snippet: safeText(fmt.Sprintf("%s (%s, %d bytes)", attachment.Filename, attachment.MIMEType, attachment.SizeBytes), 320), Provenance: Observed, SourceLocation: "attachment.metadata", Hash: hash, SafeDisplay: domain.EvidenceDisplay{Label: "Attachment metadata"}})
		b.reference(fmt.Sprintf("attachment-%d", index+1), id)
	}
	for _, enrichment := range result.IPEnrichment {
		value := enrichment.IPAddress
		id := b.add(domain.Evidence{Type: "ip_enrichment", Source: providerName(enrichment.Provider), Value: value, Snippet: enrichmentSnippet(enrichment), Provenance: enrichment.Provenance, SourceLocation: "ip_enrichment", SafeDisplay: domain.EvidenceDisplay{Label: "Passive IP enrichment"}})
		b.reference("ip-value:"+value, id)
	}
	b.addAIEvidence(result.AIAssessment)
	b.connectRisk(result)
	return b.items
}

func (b *builder) addAuthentication(headers []domain.Header, auth domain.AuthenticationResults) {
	checks := []struct {
		name  string
		check domain.AuthenticationCheck
	}{{"spf", auth.SPF}, {"dkim", auth.DKIM}, {"dmarc", auth.DMARC}}
	for _, item := range checks {
		for _, header := range headers {
			if !strings.EqualFold(header.Name, "Authentication-Results") {
				continue
			}
			matched := false
			for _, match := range authPattern.FindAllStringSubmatch(header.Value, -1) {
				if strings.EqualFold(match[1], item.name) {
					matched = true
					b.addAuthItem(item.name, item.check.Status, header)
				}
			}
			if !matched && item.check.Status != "none" && containsHeaderOrder(item.check.EvidenceReferences, header.Order) {
				b.addAuthItem(item.name, item.check.Status, header)
			}
		}
	}
}

func (b *builder) addAuthItem(name, status string, header domain.Header) {
	order := header.Order
	code := ""
	if status == "fail" {
		code = strings.ToUpper(name) + "_FAIL"
	}
	if status == "unknown" {
		code = "AUTHENTICATION_UNKNOWN"
	}
	item := domain.Evidence{Type: "authentication", Source: header.Name, Value: name + "=" + strings.ToLower(status), Snippet: safeText(header.Name+": "+header.Value, 320), Provenance: Observed, SourceLocation: "header", HeaderOrder: &order, SafeDisplay: domain.EvidenceDisplay{Label: strings.ToUpper(name) + " authentication header", Redacted: true}}
	if code != "" {
		item.RelatedSignalCodes = []string{code}
	}
	id := b.add(item)
	b.reference("header-"+strconv.Itoa(order), id)
}

func (b *builder) addReceived(headers []domain.Header, relays []domain.ReceivedRelay) {
	for _, header := range headers {
		if !strings.EqualFold(header.Name, "Received") {
			continue
		}
		order := header.Order
		id := b.add(domain.Evidence{Type: "received", Source: header.Name, Value: "Received header", Snippet: safeText(header.Name+": "+header.Value, 320), Provenance: Observed, SourceLocation: "header", HeaderOrder: &order, SafeDisplay: domain.EvidenceDisplay{Label: "Received header", Redacted: true}})
		b.reference("header-"+strconv.Itoa(order), id)
		for _, relay := range relays {
			if relay.SourceHeaderOrder != order || relay.IPAddress == nil {
				continue
			}
			ipID := b.add(domain.Evidence{Type: "ip", Source: "Received", Value: *relay.IPAddress, Snippet: *relay.IPAddress, Provenance: Observed, SourceLocation: "Received header", HeaderOrder: &order, RelatedSignalCodes: []string{"IP_PRESENT"}, SafeDisplay: domain.EvidenceDisplay{Label: "Relay IP"}})
			b.reference("received-ip-"+strconv.Itoa(order), ipID)
			if relay.Timestamp != nil {
				if parsed, err := time.Parse(time.RFC3339, *relay.Timestamp); err == nil {
					if index := ipIDIndex(b, ipID); index >= 0 {
						b.items[index].ObservedAt = &parsed
					}
				}
			}
		}
	}
}

func (b *builder) addAIEvidence(assessment domain.AIAssessment) {
	if assessment.Status == "not_available" || assessment.Status == "failed" {
		return
	}
	refs := deduplicateStrings(assessment.EvidenceReferences)
	for _, ref := range refs {
		if ids := b.byReference[ref]; len(ids) > 0 {
			for _, id := range ids {
				b.markAI(id, ref)
			}
			id := b.add(domain.Evidence{Type: "ai_assessment", Source: providerString(assessment.Provider), Value: ref, Snippet: "AI assessment referenced " + ref + ".", Provenance: AIAssessed, RelatedAIEvidenceReferences: []string{ref}, SafeDisplay: domain.EvidenceDisplay{Label: "AI evidence reference"}})
			b.reference("ai_assessment|"+ref, id)
		}
	}
	if assessment.Classification != nil {
		b.reference("ai_classification", b.add(domain.Evidence{Type: "ai_assessment", Source: providerString(assessment.Provider), Value: *assessment.Classification, Snippet: "AI assessment classification: " + safeText(*assessment.Classification, 80), Provenance: AIAssessed, SafeDisplay: domain.EvidenceDisplay{Label: "AI assessment", Redacted: true}}))
	}
}

func (b *builder) connectRisk(result *domain.AnalysisResult) {
	for index := range result.Risk.ContributingSignals {
		signal := &result.Risk.ContributingSignals[index]
		ids := []string{}
		for _, ref := range signal.EvidenceReferences {
			ids = appendUnique(ids, b.idsForReference(ref)...)
		}
		if len(ids) == 0 && signal.Provenance == AIAssessed {
			ids = append(ids, b.firstReference("ai_classification")...)
		}
		signal.EvidenceIDs = ids
		for _, id := range ids {
			b.markSignal(id, signal.Code)
		}
	}
	for _, ref := range result.Risk.EvidenceReferences {
		for _, id := range b.idsForReference(ref) {
			b.markSignal(id, "risk")
		}
	}
}

func (b *builder) add(item domain.Evidence) string {
	key := item.Type + "|" + item.Source + "|" + item.Value + "|" + strconv.Itoa(headerOrder(item.HeaderOrder))
	if index, ok := b.byKey[key]; ok {
		b.merge(index, item)
		return b.items[index].EvidenceID
	}
	item.EmailID, item.AnalysisID = b.emailID, b.analysisID
	item.RelatedSignalCodes = deduplicateStrings(item.RelatedSignalCodes)
	item.RelatedAIEvidenceReferences = deduplicateStrings(item.RelatedAIEvidenceReferences)
	item.EvidenceID = evidenceID(b.emailID, b.analysisID, key)
	b.byKey[key] = len(b.items)
	b.items = append(b.items, item)
	return item.EvidenceID
}

func (b *builder) merge(index int, item domain.Evidence) {
	for _, code := range item.RelatedSignalCodes {
		b.items[index].RelatedSignalCodes = appendUnique(b.items[index].RelatedSignalCodes, code)
	}
	for _, ref := range item.RelatedAIEvidenceReferences {
		b.items[index].RelatedAIEvidenceReferences = appendUnique(b.items[index].RelatedAIEvidenceReferences, ref)
	}
}

func (b *builder) reference(ref, id string) {
	b.byReference[ref] = appendUnique(b.byReference[ref], id)
}
func (b *builder) markAI(id, ref string) {
	for i := range b.items {
		if b.items[i].EvidenceID == id {
			b.items[i].RelatedAIEvidenceReferences = appendUnique(b.items[i].RelatedAIEvidenceReferences, ref)
		}
	}
}
func (b *builder) markSignal(id, code string) {
	for i := range b.items {
		if b.items[i].EvidenceID == id {
			b.items[i].RelatedSignalCodes = appendUnique(b.items[i].RelatedSignalCodes, code)
		}
	}
}
func (b *builder) firstReference(ref string) []string { return b.byReference[ref] }

func (b *builder) idsForReference(ref domain.EvidenceReference) []string {
	if ref.Source == "ai_assessment" {
		return b.byReference["ai_assessment|"+ref.Value]
	}
	if ref.Source == "url" {
		for _, item := range b.items {
			if item.Type == "url" && item.Value == safeURL(ref.Value) {
				return []string{item.EvidenceID}
			}
		}
		return nil
	}
	if ref.Source == "ip" {
		return b.byReference["ip-value:"+ref.Value]
	}
	if ref.Source == "attachment" {
		for _, item := range b.items {
			if item.Type == "attachment" && item.Value == safeText(ref.Value, 256) {
				return []string{item.EvidenceID}
			}
		}
		return nil
	}
	if ref.HeaderOrder > 0 {
		return b.byReference["header-"+strconv.Itoa(ref.HeaderOrder)]
	}
	return nil
}

func providerName(provider *string) string {
	if provider == nil || *provider == "" {
		return "IP enrichment"
	}
	return *provider
}
func providerString(provider *string) string {
	if provider == nil || *provider == "" {
		return "AI"
	}
	return *provider
}
func containsHeaderOrder(refs []domain.EvidenceReference, order int) bool {
	for _, ref := range refs {
		if ref.HeaderOrder == order {
			return true
		}
	}
	return false
}
func headerOrder(order *int) int {
	if order == nil {
		return 0
	}
	return *order
}
func ipIDIndex(b *builder, id string) int {
	for i := range b.items {
		if b.items[i].EvidenceID == id {
			return i
		}
	}
	return -1
}
func evidenceID(emailID, analysisID, key string) string {
	sum := sha256.Sum256([]byte(emailID + "|" + analysisID + "|" + key))
	return "evidence_" + hex.EncodeToString(sum[:16])
}
func appendUnique(values []string, additions ...string) []string {
	for _, addition := range additions {
		found := false
		for _, value := range values {
			if value == addition {
				found = true
				break
			}
		}
		if !found && addition != "" {
			values = append(values, addition)
		}
	}
	return values
}
func deduplicateStrings(values []string) []string { return appendUnique(nil, values...) }
func safeText(value string, limit int) string {
	value = privateKeyPattern.ReplaceAllString(value, "[REDACTED_PRIVATE_KEY]")
	value = secretPattern.ReplaceAllString(value, "$1=[REDACTED]")
	value = bearerPattern.ReplaceAllString(value, "Bearer [REDACTED]")
	value = urlSecretPattern.ReplaceAllString(value, "$1[REDACTED]")
	runes := []rune(value)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return value
}
func safeURL(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return safeText(value, 256)
	}
	parsed.User = nil
	query := parsed.Query()
	for key := range query {
		if strings.Contains(strings.ToLower(key), "token") || strings.Contains(strings.ToLower(key), "key") || strings.Contains(strings.ToLower(key), "secret") || strings.Contains(strings.ToLower(key), "password") {
			query.Set(key, "[REDACTED]")
		}
	}
	parsed.RawQuery = query.Encode()
	return safeText(parsed.String(), 256)
}
func enrichmentSnippet(value domain.IPEnrichment) string {
	if value.Failure != nil {
		return value.Failure.Message
	}
	parts := []string{}
	for _, item := range []string{ptr(value.Country), ptr(value.Region), ptr(value.City), ptr(value.ASN), ptr(value.Organization)} {
		if item != "" {
			parts = append(parts, item)
		}
	}
	return safeText(strings.Join(parts, ", "), 320)
}
func ptr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
