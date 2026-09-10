package report

import (
	"strings"
	"testing"
	"time"

	"sih26106/backend/internal/domain"
)

func TestBuildReportIncludesAnalysisReferencesAndLimitations(t *testing.T) {
	created := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	classification := "phishing"
	confidence := 0.92
	provider := "google"
	result := &domain.AnalysisResult{
		AnalysisID: "analysis-1", EmailID: "email-1", CaseID: "case-1", Status: "partial",
		AIAssessment: domain.AIAssessment{Status: "completed", Classification: &classification, Confidence: &confidence, Provider: &provider, SupportingSignals: []string{"credential theft"}, EvidenceReferences: []string{"url-1"}},
		Risk:         domain.RiskAssessment{Score: 80, Level: "critical", Verdict: "phishing", Confidence: &confidence, ContributingSignals: []domain.RiskSignal{{Code: "URL_PRESENT", Description: "url observed", Points: 10, Provenance: "OBSERVED", EvidenceIDs: []string{"evidence-url"}}}},
		IPEnrichment: []domain.IPEnrichment{{IPAddress: "203.0.113.7", Status: "not_configured", Provenance: "INFERRED"}},
		Evidence:     []domain.Evidence{{EvidenceID: "evidence-url", Type: "url", Value: "https://example.test/login?token=[REDACTED]", Snippet: "https://example.test/login?token=[REDACTED]", Provenance: "OBSERVED"}},
	}
	parsed := &domain.ParsedEmail{
		Message:       domain.MessageMetadata{From: []string{"sender@example.test"}, Subject: stringPointer("password=do-not-leak")},
		Indicators:    domain.Indicators{URLs: []string{"https://example.test/login?token=secret-value"}, IPs: []string{"203.0.113.7"}},
		Attachments:   []domain.Attachment{{Filename: "invoice.pdf.exe", MIMEType: "application/octet-stream", SizeBytes: 4, SHA256: "abc123"}},
		PlainTextBody: "password=secret-value and https://example.test/login?token=secret-value",
	}
	graph := &domain.Graph{Nodes: []domain.GraphNode{{Type: "ip"}, {Type: "url"}}, Edges: []domain.GraphEdge{{ID: "edge-1"}}}
	timeline := &domain.Timeline{Events: []domain.TimelineEvent{{ID: "timeline-1", Type: "relay_inferred"}}}
	value := Build(&domain.Case{ID: "case-1", Status: "created", CreatedAt: created, UpdatedAt: created}, []Input{{Email: &domain.Email{ID: "email-1", Filename: "message.eml", CreatedAt: created}, Parsed: parsed, Result: result}}, graph, timeline, created.Add(time.Hour))
	if value.Status != "partial" || value.SchemaVersion != SchemaVersion || value.Graph.NodeCount != 2 || value.Graph.EdgeCount != 1 || len(value.Timeline) != 1 {
		t.Fatalf("report = %#v", value)
	}
	if value.Analyses[0].Risk.Score != 80 || value.Analyses[0].AIAssessment.Classification == nil || *value.Analyses[0].AIAssessment.Classification != classification {
		t.Fatalf("analysis = %#v", value.Analyses[0])
	}
	subject := ""
	if value.Emails[0].Message.Subject != nil {
		subject = *value.Emails[0].Message.Subject
	}
	encoded := strings.Join(value.Emails[0].Indicators.URLs, " ") + " " + subject
	if strings.Contains(encoded, "secret-value") || strings.Contains(encoded, "raw email") {
		t.Fatalf("sensitive report content = %q", encoded)
	}
	joined := strings.Join(value.Limitations, " | ")
	for _, expected := range []string{"not ground truth", "not proof of physical location", "not automatically visited", "not configured"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("limitations %q missing %q", joined, expected)
		}
	}
}

func TestBuildReportIdentifierIsStable(t *testing.T) {
	caseValue := &domain.Case{ID: "case-1"}
	input := []Input{{Result: &domain.AnalysisResult{AnalysisID: "analysis-b"}}, {Result: &domain.AnalysisResult{AnalysisID: "analysis-a"}}}
	first := Build(caseValue, input, nil, nil, time.Unix(1, 0))
	second := Build(caseValue, input, nil, nil, time.Unix(2, 0))
	if first.ReportID != second.ReportID || first.ReportID != "report:case-1:analysis-a,analysis-b" {
		t.Fatalf("report IDs = %q, %q", first.ReportID, second.ReportID)
	}
}

func TestBuildReportRiskProfiles(t *testing.T) {
	cases := []struct {
		name    string
		score   int
		level   string
		verdict string
	}{
		{name: "safe fixture", score: 0, level: "low", verdict: "unknown"},
		{name: "moderate fixture", score: 40, level: "medium", verdict: "suspicious"},
		{name: "dangerous fixture", score: 90, level: "critical", verdict: "phishing"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result := &domain.AnalysisResult{AnalysisID: testCase.name, EmailID: "email-1", CaseID: "case-1", Status: "completed", Risk: domain.RiskAssessment{Score: testCase.score, Level: testCase.level, Verdict: testCase.verdict}}
			value := Build(&domain.Case{ID: "case-1"}, []Input{{Email: &domain.Email{ID: "email-1"}, Parsed: &domain.ParsedEmail{}, Result: result}}, nil, nil, time.Unix(1, 0))
			if value.Analyses[0].Risk.Score != testCase.score || value.Analyses[0].Risk.Level != testCase.level || value.Analyses[0].Risk.Verdict != testCase.verdict {
				t.Fatalf("risk = %#v", value.Analyses[0].Risk)
			}
		})
	}
}

func stringPointer(value string) *string { return &value }
