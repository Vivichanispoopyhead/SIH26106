package evidence

import (
	"strings"
	"testing"

	"sih26106/backend/internal/domain"
)

func TestBuildEvidencePreservesProvenanceAndRiskLinks(t *testing.T) {
	classification := "phishing"
	confidence := 0.9
	parsed := &domain.ParsedEmail{
		PlainTextBody: "Please visit https://example.test/login?token=do-not-leak password=do-not-leak",
		Headers: []domain.Header{
			{Name: "Authentication-Results", Value: "mx; spf=fail dkim=pass dmarc=fail", Order: 7},
			{Name: "Received", Value: "from mx.example (8.8.8.8); Tue, 01 Jan 2024 00:00:00 +0000", Order: 8},
		},
		Indicators:  domain.Indicators{URLs: []string{"https://example.test/login"}, Domains: []string{"example.test"}, IPs: []string{"8.8.8.8"}},
		Attachments: []domain.Attachment{{Filename: "invoice.pdf.exe", MIMEType: "application/octet-stream", SizeBytes: 5, SHA256: "abc123"}},
	}
	order := 7
	result := &domain.AnalysisResult{
		AnalysisID: "analysis-1", EmailID: "email-1", AIAssessment: domain.AIAssessment{Status: "completed", Classification: &classification, Confidence: &confidence, EvidenceReferences: []string{"body-1", "header-7", "url-1", "ip-1", "attachment-1", "header-99"}},
		Authentication: domain.AuthenticationResults{SPF: domain.AuthenticationCheck{Status: "fail", EvidenceReferences: []domain.EvidenceReference{{HeaderOrder: order}}}, DMARC: domain.AuthenticationCheck{Status: "fail", EvidenceReferences: []domain.EvidenceReference{{HeaderOrder: order}}}},
		ReceivedChain:  []domain.ReceivedRelay{{SourceHeaderOrder: 8, IPAddress: stringPtr("8.8.8.8"), Timestamp: stringPtr("2024-01-01T00:00:00Z"), Confidence: "medium"}},
		Risk:           domain.RiskAssessment{ContributingSignals: []domain.RiskSignal{{Code: "SPF_FAIL", Provenance: Observed, EvidenceReferences: []domain.EvidenceReference{{HeaderOrder: 7, Source: "authentication_results"}}}, {Code: "AI_PHISHING", Provenance: AIAssessed, EvidenceReferences: []domain.EvidenceReference{{Source: "ai_assessment", Value: "body-1"}}}}},
	}
	items := Build("email-1", parsed, result)
	if len(items) < 8 {
		t.Fatalf("evidence count = %d, items=%#v", len(items), items)
	}
	if len(result.Risk.ContributingSignals[0].EvidenceIDs) == 0 || len(result.Risk.ContributingSignals[1].EvidenceIDs) == 0 {
		t.Fatalf("risk links = %#v", result.Risk.ContributingSignals)
	}
	for _, item := range items {
		if item.EvidenceID == "" || item.EmailID != "email-1" || item.AnalysisID != "analysis-1" {
			t.Fatalf("identity = %#v", item)
		}
		if strings.Contains(item.Snippet, "do-not-leak") {
			t.Fatalf("secret leaked in evidence: %#v", item)
		}
	}
}

func TestBuildEvidenceDropsUnknownAIReferencesAndDeduplicates(t *testing.T) {
	parsed := &domain.ParsedEmail{PlainTextBody: "safe", Indicators: domain.Indicators{URLs: []string{"https://example.test", "https://example.test"}}}
	result := &domain.AnalysisResult{AnalysisID: "analysis-2", AIAssessment: domain.AIAssessment{Status: "partial", EvidenceReferences: []string{"url-1", "url-1", "header-99"}}}
	items := Build("email-2", parsed, result)
	for _, item := range items {
		if item.Value == "header-99" {
			t.Fatalf("unknown AI reference became evidence: %#v", item)
		}
	}
	urlCount := 0
	for _, item := range items {
		if item.Type == "url" {
			urlCount++
		}
	}
	if urlCount != 1 {
		t.Fatalf("duplicate URLs were not deduplicated: %#v", items)
	}
}

func stringPtr(value string) *string { return &value }
