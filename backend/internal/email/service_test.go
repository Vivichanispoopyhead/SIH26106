package email

import (
	"context"
	"testing"

	"sih26106/backend/internal/ai"
	"sih26106/backend/internal/domain"
	"sih26106/backend/internal/persistence"
)

type captureAnalyzer struct {
	input ai.Input
}

func (a *captureAnalyzer) Assess(_ context.Context, input ai.Input) (domain.AIAssessment, error) {
	a.input = input
	return domain.AIAssessment{
		Status:             "completed",
		SupportingSignals:  []string{},
		EvidenceReferences: []string{},
	}, nil
}

func TestStartAnalysisPassesNormalizedPlainTextToAnalyzer(t *testing.T) {
	analyzer := &captureAnalyzer{}
	service := NewServiceWithAnalyzer(persistence.NewMemoryStore(), analyzer)
	uploaded, err := service.Upload(context.Background(), "message.eml", []byte(
		"From: sender@example.com\r\nContent-Type: multipart/alternative; boundary=parts\r\n\r\n"+
			"--parts\r\nContent-Type: text/plain\r\n\r\nplain content\r\n"+
			"--parts\r\nContent-Type: text/html\r\n\r\n<script>alert(1)</script>\r\n"+
			"--parts--\r\n",
	))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.StartAnalysis(context.Background(), uploaded.ID); err != nil {
		t.Fatal(err)
	}
	if analyzer.input.PlainTextBody != "plain content" {
		t.Fatalf("plain text body = %q", analyzer.input.PlainTextBody)
	}
	if analyzer.input.Attachments == nil || analyzer.input.Headers == nil {
		t.Fatal("analyzer input slices must be initialized")
	}
}
