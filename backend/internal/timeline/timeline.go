package timeline

import (
	"net/url"
	"sort"
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

type EmailAnalysis struct {
	Email    *domain.Email
	Parsed   *domain.ParsedEmail
	Result   *domain.AnalysisResult
	Analysis *domain.Analysis
}

func Build(caseID string, inputs []EmailAnalysis) *domain.Timeline {
	result := &domain.Timeline{CaseID: caseID, EmailIDs: []string{}, Events: []domain.TimelineEvent{}}
	for _, input := range inputs {
		if input.Email == nil || input.Parsed == nil || input.Result == nil {
			continue
		}
		result.EmailIDs = appendUnique(result.EmailIDs, input.Email.ID)
		addEmailEvents(result, input)
	}
	sort.Strings(result.EmailIDs)
	sort.SliceStable(result.Events, func(i, j int) bool {
		left, right := result.Events[i], result.Events[j]
		if left.Timestamp != nil && right.Timestamp == nil {
			return true
		}
		if left.Timestamp == nil && right.Timestamp != nil {
			return false
		}
		if left.Timestamp != nil && right.Timestamp != nil && !left.Timestamp.Equal(*right.Timestamp) {
			return left.Timestamp.Before(*right.Timestamp)
		}
		if left.Sequence != right.Sequence {
			return left.Sequence < right.Sequence
		}
		return left.ID < right.ID
	})
	return result
}

func addEmailEvents(timeline *domain.Timeline, input EmailAnalysis) {
	email, parsed, result := input.Email, input.Parsed, input.Result
	evidence := evidenceIndex(result.Evidence)
	uploadedAt := input.Email.CreatedAt
	timeline.Events = append(timeline.Events, domain.TimelineEvent{
		ID:            "timeline:" + email.ID + ":email",
		Type:          "email_received",
		Sequence:      0,
		Timestamp:     timePointer(uploadedAt),
		Title:         "Email artifact uploaded",
		Description:   "The email artifact was received by the investigation service.",
		Provenance:    Observed,
		Confidence:    "high",
		EvidenceIDs:   []string{},
		RelatedNodeID: stringPointer("email:" + email.ID),
		Metadata:      map[string]string{"timestamp_source": "upload"},
	})

	for _, relay := range result.ReceivedChain {
		ids := evidenceForHeader(evidence, relay.SourceHeaderOrder)
		headerOrder := relay.SourceHeaderOrder
		relatedNode := "relay:" + email.ID + ":" + strconv.Itoa(relay.SourceHeaderOrder)
		provenance := normalizeProvenance(relay.Provenance, Inferred)
		eventType := "relay_inferred"
		if provenance == Observed {
			eventType = "relay_observed"
		}
		timeline.Events = append(timeline.Events, domain.TimelineEvent{
			ID:                "timeline:" + email.ID + ":relay:" + strconv.Itoa(relay.Sequence),
			Type:              eventType,
			Sequence:          relay.Sequence,
			Timestamp:         parseTimestamp(relay.Timestamp),
			Title:             "Observed relay",
			Description:       "A Received header identified a relay in the message path.",
			Hostname:          relay.Hostname,
			IPAddress:         relay.IPAddress,
			SourceHeaderOrder: &headerOrder,
			Provenance:        provenance,
			Confidence:        relay.Confidence,
			EvidenceIDs:       ids,
			RelatedNodeID:     &relatedNode,
			Metadata:          map[string]string{"sequence_source": "received_header_order"},
		})
	}

	addAuthenticationEvents(timeline, email.ID, parsed.Headers, result.Authentication, evidence)
	addIndicatorEvents(timeline, email.ID, parsed.Indicators, evidence)
	addEnrichmentEvents(timeline, email.ID, result.IPEnrichment, evidence)
	if input.Analysis != nil && (result.Status == "completed" || result.Status == "partial") {
		completedAt := input.Analysis.UpdatedAt
		timeline.Events = append(timeline.Events, domain.TimelineEvent{
			ID:          "timeline:" + email.ID + ":analysis",
			Type:        "analysis_completed",
			Timestamp:   timePointer(completedAt),
			Title:       "Analysis completed",
			Description: "The persisted analysis completed with the reported status.",
			Provenance:  Observed,
			Confidence:  "high",
			EvidenceIDs: []string{},
			Metadata:    map[string]string{"status": result.Status},
		})
	}
}

func addAuthenticationEvents(events *domain.Timeline, emailID string, headers []domain.Header, auth domain.AuthenticationResults, evidence map[string]domain.Evidence) {
	checks := []struct {
		name  string
		check domain.AuthenticationCheck
	}{{"spf", auth.SPF}, {"dkim", auth.DKIM}, {"dmarc", auth.DMARC}}
	for _, item := range checks {
		if item.check.Status == "" || item.check.Status == "none" {
			continue
		}
		order := firstEvidenceOrder(item.check.EvidenceReferences)
		ids := evidenceForHeader(evidence, order)
		if order == 0 {
			order = headerOrderForAuthentication(headers, item.name)
			ids = evidenceForHeader(evidence, order)
		}
		var orderPointer *int
		if order > 0 {
			value := order
			orderPointer = &value
		}
		events.Events = append(events.Events, domain.TimelineEvent{
			ID:                "timeline:" + emailID + ":authentication:" + item.name + ":" + item.check.Status,
			Type:              "authentication_observed",
			Title:             strings.ToUpper(item.name) + " authentication observed",
			Description:       item.check.Explanation,
			SourceHeaderOrder: orderPointer,
			Provenance:        Observed,
			Confidence:        "medium",
			EvidenceIDs:       ids,
			Metadata:          map[string]string{"method": item.name, "status": item.check.Status},
		})
	}
}

func addIndicatorEvents(events *domain.Timeline, emailID string, indicators domain.Indicators, evidence map[string]domain.Evidence) {
	seen := map[string]struct{}{}
	add := func(kind, value, nodeID string) {
		key := kind + "|" + value
		if _, ok := seen[key]; ok || strings.TrimSpace(value) == "" {
			return
		}
		seen[key] = struct{}{}
		displayValue := value
		if kind == "url" {
			displayValue = safeURL(value)
		}
		events.Events = append(events.Events, domain.TimelineEvent{
			ID:            "timeline:" + emailID + ":indicator:" + kind + ":" + displayValue,
			Type:          "indicator_observed",
			Title:         "Observed " + kind,
			Description:   "An indicator was extracted from the email.",
			Provenance:    Observed,
			Confidence:    "high",
			EvidenceIDs:   evidenceForValue(evidence, kind, value),
			RelatedNodeID: stringPointer(nodeID),
			Metadata:      map[string]string{"indicator_type": kind},
		})
	}
	for _, value := range indicators.URLs {
		add("url", value, "url:"+safeURL(value))
	}
	for _, value := range indicators.Domains {
		add("domain", value, "domain:"+value)
	}
	for _, value := range indicators.IPs {
		add("ip", value, "ip:"+value)
	}
}

func addEnrichmentEvents(events *domain.Timeline, emailID string, values []domain.IPEnrichment, evidence map[string]domain.Evidence) {
	for _, value := range values {
		if value.Status != "enriched" && value.Status != "failed" {
			continue
		}
		provenance := Inferred
		if value.Status == "enriched" {
			provenance = Enriched
		}
		events.Events = append(events.Events, domain.TimelineEvent{
			ID:            "timeline:" + emailID + ":enrichment:" + value.IPAddress,
			Type:          "enrichment_completed",
			Timestamp:     value.RetrievedAt,
			Title:         "IP enrichment completed",
			Description:   "Passive IP enrichment returned metadata or a structured failure.",
			IPAddress:     stringPointer(value.IPAddress),
			Provenance:    provenance,
			Confidence:    confidenceString(value.Confidence),
			EvidenceIDs:   evidenceForValue(evidence, "ip_enrichment", value.IPAddress),
			RelatedNodeID: stringPointer("ip:" + value.IPAddress),
			Metadata:      map[string]string{"status": value.Status},
		})
	}
}

func evidenceIndex(items []domain.Evidence) map[string]domain.Evidence {
	result := make(map[string]domain.Evidence, len(items))
	for _, item := range items {
		result[item.EvidenceID] = item
	}
	return result
}

func evidenceForHeader(items map[string]domain.Evidence, order int) []string {
	if order <= 0 {
		return []string{}
	}
	ids := []string{}
	for id, item := range items {
		if item.HeaderOrder != nil && *item.HeaderOrder == order {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func evidenceForValue(items map[string]domain.Evidence, kind, value string) []string {
	ids := []string{}
	for id, item := range items {
		if item.Type == kind && (item.Value == value || (kind == "url" && safeURL(item.Value) == safeURL(value))) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func firstEvidenceOrder(refs []domain.EvidenceReference) int {
	for _, ref := range refs {
		if ref.HeaderOrder > 0 {
			return ref.HeaderOrder
		}
	}
	return 0
}

func headerOrderForAuthentication(headers []domain.Header, name string) int {
	for _, header := range headers {
		if strings.EqualFold(header.Name, "Authentication-Results") && strings.Contains(strings.ToLower(header.Value), strings.ToLower(name)+"=") {
			return header.Order
		}
	}
	return 0
}

func normalizeProvenance(value, fallback string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case Observed, Enriched, Inferred, AIAssessed:
		return strings.ToUpper(strings.TrimSpace(value))
	default:
		return fallback
	}
}

func parseTimestamp(value *string) *time.Time {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, *value)
	if err != nil {
		return nil
	}
	return &parsed
}

func timePointer(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func confidenceString(value *float64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(*value, 'f', 2, 64)
}

func safeURL(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return value
	}
	parsed.User = nil
	query := parsed.Query()
	for key := range query {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "token") || strings.Contains(lower, "key") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") {
			query.Set(key, "[REDACTED]")
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	if value != "" {
		return append(values, value)
	}
	return values
}
