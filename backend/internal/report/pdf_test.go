package report

import (
	"strings"
	"testing"
	"time"

	"sih26106/backend/internal/domain"
)

func TestPDFContainsSafeReportContent(t *testing.T) {
	verdict := "phishing"
	secret := "API_KEY=do-not-include"
	value := &domain.ForensicReport{
		ReportID:    "report:case-1:analysis-1",
		GeneratedAt: time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
		Status:      "completed",
		Case:        domain.ReportCase{ID: "case-1", Status: "closed"},
		Emails: []domain.ReportEmail{{
			EmailID:     "email-1",
			Filename:    "message.eml",
			Message:     domain.MessageMetadata{From: []string{"sender@example.test"}, To: []string{"recipient@example.test"}},
			Indicators:  domain.Indicators{URLs: []string{"https://example.test/login"}, IPs: []string{"192.0.2.1"}, Domains: []string{"example.test"}},
			Attachments: []domain.Attachment{{Filename: "invoice.pdf", SHA256: "hash-1"}},
		}},
		Analyses: []domain.ReportAnalysis{{
			AnalysisID: "analysis-1",
			Risk:       domain.RiskAssessment{Score: 90, Level: "high", Verdict: verdict},
			AIAssessment: domain.AIAssessment{
				Status: "completed", SupportingSignals: []string{"urgent request"}, EvidenceReferences: []string{"evidence-1"},
			},
		}},
		Graph:       domain.ReportGraphSummary{NodeCount: 2, EdgeCount: 1, NodeIDs: []string{"email:email-1"}},
		Limitations: []string{"AI output is an evaluated assessment, not ground truth."},
	}

	pdf, err := PDF(value)
	if err != nil {
		t.Fatalf("PDF() error = %v", err)
	}
	if !strings.HasPrefix(string(pdf), "%PDF-") {
		t.Fatalf("output is not a PDF: %q", pdf[:min(16, len(pdf))])
	}
	output := string(pdf)
	for _, expected := range []string{
		"Forensic Investigation Report", "phishing", "Limitations and safety disclaimers",
		"AI output is an evaluated assessment, not ground truth.",
		"IP geolocation is an estimate and does not prove physical location or identity.",
		"URLs were not automatically visited.", "Attachments were not executed.",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("PDF does not contain %q", expected)
		}
	}
	for _, excluded := range []string{secret, "raw attachment contents", "Gemini prompt"} {
		if strings.Contains(output, excluded) {
			t.Errorf("PDF contains excluded content %q", excluded)
		}
	}
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
