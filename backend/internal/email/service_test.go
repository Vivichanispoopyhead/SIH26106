package email

import (
	"context"
	"testing"
	"time"

	"sih26106/backend/internal/ai"
	"sih26106/backend/internal/domain"
	"sih26106/backend/internal/enrichment"
	"sih26106/backend/internal/persistence"
)

type captureAnalyzer struct {
	input ai.Input
}

type captureEnricher struct{ calls []string }

func (e *captureEnricher) Lookup(_ context.Context, ip string) (domain.IPEnrichment, error) {
	e.calls = append(e.calls, ip)
	if ip == "1.1.1.1" {
		return domain.IPEnrichment{IPAddress: ip, Status: enrichment.StatusFailed, Provenance: enrichment.ProvenanceInferred, Failure: &domain.Failure{Code: "IP_ENRICHMENT_HTTP_ERROR", Message: "The provider returned an error."}}, context.DeadlineExceeded
	}
	provider := "test-provider"
	now := time.Now().UTC()
	return domain.IPEnrichment{IPAddress: ip, Status: enrichment.StatusEnriched, Provider: &provider, RetrievedAt: &now, Provenance: enrichment.ProvenanceEnriched}, nil
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

func TestStartAnalysisPreservesPartialIPEnrichmentFailure(t *testing.T) {
	enricher := &captureEnricher{}
	service := NewServiceWithAnalyzerAndEnricher(persistence.NewMemoryStore(), ai.UnavailableAnalyzer{}, enricher)
	uploaded, err := service.Upload(context.Background(), "message.eml", []byte(
		"From: sender@example.com\r\n"+
			"Received: from mx.example (8.8.8.8); Tue, 01 Jan 2024 00:00:00 +0000\r\n"+
			"Received: from origin.example (1.1.1.1); Tue, 01 Jan 2024 00:01:00 +0000\r\n\r\n"+
			"body\r\n",
	))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.StartAnalysis(context.Background(), uploaded.ID); err != nil {
		t.Fatal(err)
	}
	result, err := service.GetAnalysis(context.Background(), uploaded.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "partial" || len(result.IPEnrichment) != 2 || len(enricher.calls) != 2 {
		t.Fatalf("status=%s enrichment=%#v calls=%#v", result.Status, result.IPEnrichment, enricher.calls)
	}
}
