package email

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"sih26106/backend/internal/ai"
	"sih26106/backend/internal/domain"
	"sih26106/backend/internal/enrichment"
	"sih26106/backend/internal/persistence"
)

type validationAnalyzer struct {
	inputs []ai.Input
}

func (a *validationAnalyzer) Assess(_ context.Context, input ai.Input) (domain.AIAssessment, error) {
	a.inputs = append(a.inputs, input)
	subject := ""
	if input.Message.Subject != nil {
		subject = strings.ToLower(*input.Message.Subject)
	}
	switch {
	case strings.Contains(subject, "monthly planning"):
		return domain.AIAssessment{Status: "not_available", SupportingSignals: []string{}, EvidenceReferences: []string{}, Failure: &domain.Failure{Code: "AI_NOT_CONFIGURED", Message: "No AI analyzer is configured."}}, nil
	case strings.Contains(subject, "payment"):
		return domain.AIAssessment{Status: "failed", SupportingSignals: []string{}, EvidenceReferences: []string{}, Failure: &domain.Failure{Code: "AI_PROVIDER_HTTP_ERROR", Message: "The local validation provider failed."}}, errors.New("local validation provider failed")
	default:
		return domain.AIAssessment{Status: "completed", Classification: stringPointer("phishing"), Confidence: floatPointer(0.9), SupportingSignals: []string{"credential request"}, EvidenceReferences: []string{"url-1", "url-1", "attachment-1"}, Provider: stringPointer("validation"), Model: stringPointer("deterministic-test")}, nil
	}
}

type validationEnricher struct {
	calls []string
}

func (e *validationEnricher) Lookup(_ context.Context, ip string) (domain.IPEnrichment, error) {
	e.calls = append(e.calls, ip)
	return domain.IPEnrichment{IPAddress: ip, Status: enrichment.StatusEnriched, Provenance: enrichment.ProvenanceEnriched}, nil
}

func TestCompleteFixturePipelinePreservesCrossStageResults(t *testing.T) {
	fixtureDir := fixtureDirectory(t)
	analyzer := &validationAnalyzer{}
	enricher := &validationEnricher{}
	store := persistence.NewMemoryStore()
	service := NewServiceWithAnalyzerAndEnricher(store, analyzer, enricher)
	cases := []struct {
		name               string
		file               string
		wantStatus         string
		wantAIStatus       string
		wantClassification string
		wantVerdict        string
		wantAuthFailure    bool
		wantAttachment     bool
	}{
		{name: "safe", file: "judge-safe-business.eml", wantStatus: "completed", wantAIStatus: "not_available", wantVerdict: "unknown"},
		{name: "moderate", file: "judge-moderate-payment-request.eml", wantStatus: "partial", wantAIStatus: "failed", wantVerdict: "unknown"},
		{name: "dangerous", file: "judge-dangerous-multi-signal.eml", wantStatus: "completed", wantAIStatus: "completed", wantClassification: "phishing", wantVerdict: "malware", wantAuthFailure: true, wantAttachment: true},
	}
	for _, fixture := range cases {
		t.Run(fixture.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(fixtureDir, fixture.file))
			if errors.Is(err, os.ErrNotExist) {
				raw = []byte(fixtureFallback(fixture.file))
			} else if err != nil {
				t.Fatal(err)
			}
			uploaded, err := service.Upload(context.Background(), fixture.file, raw)
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
			if result.Status != fixture.wantStatus || result.AIAssessment.Status != fixture.wantAIStatus {
				t.Fatalf("status=%s ai=%s result=%#v", result.Status, result.AIAssessment.Status, result)
			}
			if fixture.wantClassification == "" && result.AIAssessment.Classification != nil {
				t.Fatalf("unexpected AI classification = %v", *result.AIAssessment.Classification)
			}
			if fixture.wantClassification != "" && (result.AIAssessment.Classification == nil || *result.AIAssessment.Classification != fixture.wantClassification) {
				t.Fatalf("AI classification = %#v", result.AIAssessment.Classification)
			}
			if result.Risk.Verdict != fixture.wantVerdict {
				t.Fatalf("risk verdict = %q, result=%#v", result.Risk.Verdict, result.Risk)
			}
			if fixture.wantAuthFailure && (result.Authentication.SPF.Status != "fail" || result.Authentication.DKIM.Status != "fail" || result.Authentication.DMARC.Status != "fail") {
				t.Fatalf("authentication = %#v", result.Authentication)
			}
			if fixture.wantAttachment && len(result.ReceivedChain) == 0 {
				t.Fatal("dangerous fixture lost received chain")
			}
			if fixture.wantAttachment {
				emailValue, getErr := store.GetEmail(context.Background(), uploaded.ID)
				if getErr != nil || emailValue.Parsed == nil || len(emailValue.Parsed.Attachments) != 1 || !strings.Contains(strings.ToLower(emailValue.Parsed.Attachments[0].Filename), ".exe") {
					t.Fatalf("attachments = %#v err=%v", emailValue.Parsed, getErr)
				}
			}
			assertEvidenceLinks(t, result)
			graphValue, err := service.GetGraph(context.Background(), uploaded.CaseID)
			if err != nil || len(graphValue.Nodes) == 0 || len(graphValue.Edges) == 0 {
				t.Fatalf("graph=%#v err=%v", graphValue, err)
			}
			validEvidence := make(map[string]struct{}, len(result.Evidence))
			for _, item := range result.Evidence {
				validEvidence[item.EvidenceID] = struct{}{}
			}
			for _, node := range graphValue.Nodes {
				for _, id := range node.EvidenceIDs {
					if _, ok := validEvidence[id]; !ok {
						t.Fatalf("graph node %s references missing evidence %s", node.ID, id)
					}
				}
			}
			for _, edge := range graphValue.Edges {
				for _, id := range edge.EvidenceIDs {
					if _, ok := validEvidence[id]; !ok {
						t.Fatalf("graph edge %s references missing evidence %s", edge.ID, id)
					}
				}
			}
			timelineValue, err := service.GetTimeline(context.Background(), uploaded.CaseID)
			if err != nil || len(timelineValue.Events) == 0 {
				t.Fatalf("timeline=%#v err=%v", timelineValue, err)
			}
			for _, event := range timelineValue.Events {
				for _, id := range event.EvidenceIDs {
					if _, ok := validEvidence[id]; !ok {
						t.Fatalf("timeline event %s references missing evidence %s", event.ID, id)
					}
				}
			}
			reportValue, err := service.GetReport(context.Background(), uploaded.CaseID)
			if err != nil || reportValue.Status != fixture.wantStatus || len(reportValue.Analyses) != 1 || len(reportValue.Evidence) == 0 || len(reportValue.Timeline) == 0 {
				t.Fatalf("report=%#v err=%v", reportValue, err)
			}
			if fixture.wantStatus == "partial" && len(reportValue.Limitations) == 0 {
				t.Fatal("partial report has no limitations")
			}
			encoded, err := json.Marshal(reportValue)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(encoded), string(raw)) || strings.Contains(string(encoded), "RAW_ATTACHMENT_BYTES") {
				t.Fatal("report contains raw email or attachment content")
			}
		})
	}
	if len(enricher.calls) != 0 {
		t.Fatalf("documentation IPs triggered enrichment calls: %#v", enricher.calls)
	}
	if len(analyzer.inputs) != len(cases) {
		t.Fatalf("analyzer inputs = %d, want %d", len(analyzer.inputs), len(cases))
	}
}

func assertEvidenceLinks(t *testing.T, result *domain.AnalysisResult) {
	t.Helper()
	valid := make(map[string]struct{}, len(result.Evidence))
	for _, item := range result.Evidence {
		if item.EvidenceID == "" || (item.Provenance != "OBSERVED" && item.Provenance != "ENRICHED" && item.Provenance != "INFERRED" && item.Provenance != "AI-ASSESSED") {
			t.Fatalf("invalid evidence = %#v", item)
		}
		valid[item.EvidenceID] = struct{}{}
	}
	for _, signal := range result.Risk.ContributingSignals {
		for _, id := range signal.EvidenceIDs {
			if _, ok := valid[id]; !ok {
				t.Fatalf("risk signal %s references missing evidence %s", signal.Code, id)
			}
		}
	}
	seen := map[string]struct{}{}
	for _, ref := range result.AIAssessment.EvidenceReferences {
		if _, ok := seen[ref]; ok {
			t.Fatalf("duplicate AI evidence reference persisted: %s", ref)
		}
		seen[ref] = struct{}{}
	}
}

func fixtureDirectory(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate integration test")
	}
	return filepath.Join(filepath.Dir(filename), "../../test-fixtures")
}

func fixtureFallback(name string) string {
	switch name {
	case "judge-safe-business.eml":
		return "From: Operations <operations@example.test>\r\nTo: Analyst <analyst@example.org>\r\nSubject: Monthly planning meeting\r\nDate: Thu, 10 Sep 2026 09:00:00 +0000\r\nAuthentication-Results: mx.example.org; spf=pass; dkim=pass; dmarc=pass\r\nReceived: from mail.example.test (192.0.2.10) by mx.example.org; Thu, 10 Sep 2026 09:00:00 +0000\r\nContent-Type: text/plain\r\n\r\nThe monthly planning meeting is scheduled for Friday.\r\n"
	case "judge-moderate-payment-request.eml":
		return "From: Accounts Desk <accounts@billing-example.test>\r\nTo: Finance Review <finance@example.org>\r\nSubject: Updated payment details for upcoming invoice\r\nDate: Thu, 10 Sep 2026 11:30:00 +0000\r\nAuthentication-Results: mx.example.org; spf=neutral; dkim=none; dmarc=none\r\nReceived: from relay.billing-example.test (198.51.100.24) by mx.example.org; Thu, 10 Sep 2026 11:30:00 +0000\r\nContent-Type: text/plain\r\n\r\nPlease review the updated payment instructions at https://billing-example.test/invoice-review.\r\n"
	case "judge-dangerous-multi-signal.eml":
		return "From: Executive Office <executive-office@lookalike-example.test>\r\nTo: Finance Operations <finance@example.org>\r\nSubject: URGENT: confidential transfer required today\r\nDate: Thu, 10 Sep 2026 13:45:00 +0000\r\nAuthentication-Results: mx.example.org; spf=fail; dkim=fail; dmarc=fail\r\nReceived: from unknown (203.0.113.77) by mx.example.org; Thu, 10 Sep 2026 13:45:00 +0000\r\nContent-Type: multipart/mixed; boundary=parts\r\n\r\n--parts\r\nContent-Type: text/plain\r\n\r\nProcess the confidential transfer at https://secure-transfer-example.test/approval.\r\n--parts\r\nContent-Type: application/octet-stream; name=payment-instructions.pdf.exe\r\nContent-Disposition: attachment; filename=payment-instructions.pdf.exe\r\nContent-Transfer-Encoding: base64\r\n\r\naGVsbG8=\r\n--parts--\r\n"
	default:
		return "From: sender@example.test\r\n\r\nbody\r\n"
	}
}

func stringPointer(value string) *string { return &value }

func floatPointer(value float64) *float64 { return &value }
