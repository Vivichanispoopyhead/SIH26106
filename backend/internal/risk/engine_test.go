package risk

import (
	"testing"

	"sih26106/backend/internal/domain"
)

func TestEvaluateNoSignals(t *testing.T) {
	assessment := Evaluate(domain.AnalysisResult{}, domain.Indicators{})
	if assessment.Score != 0 || assessment.Level != "low" || assessment.Verdict != "unknown" || assessment.Confidence != nil || len(assessment.ContributingSignals) != 0 {
		t.Fatalf("assessment = %#v", assessment)
	}
}

func TestEvaluateAuthenticationAndIndicatorSignals(t *testing.T) {
	result := domain.AnalysisResult{
		Authentication: domain.AuthenticationResults{
			SPF:   domain.AuthenticationCheck{Status: "fail", EvidenceReferences: []domain.EvidenceReference{{HeaderOrder: 2, HeaderName: "Authentication-Results"}}},
			DKIM:  domain.AuthenticationCheck{Status: "fail", EvidenceReferences: []domain.EvidenceReference{{HeaderOrder: 2, HeaderName: "Authentication-Results"}}},
			DMARC: domain.AuthenticationCheck{Status: "fail", EvidenceReferences: []domain.EvidenceReference{{HeaderOrder: 2, HeaderName: "Authentication-Results"}}},
		},
	}
	assessment := Evaluate(result, domain.Indicators{URLs: []string{"https://example.test"}, IPs: []string{"203.0.113.10"}})
	if assessment.Score != 80 || assessment.Level != "critical" || assessment.Verdict != "suspicious" {
		t.Fatalf("assessment = %#v", assessment)
	}
	if assessment.Confidence == nil || *assessment.Confidence <= 0 || len(assessment.EvidenceReferences) == 0 {
		t.Fatalf("confidence/evidence = %#v", assessment)
	}
}

func TestEvaluateExecutableAttachmentAndClamp(t *testing.T) {
	result := domain.AnalysisResult{
		Authentication: domain.AuthenticationResults{
			SPF: domain.AuthenticationCheck{Status: "fail"}, DKIM: domain.AuthenticationCheck{Status: "fail"}, DMARC: domain.AuthenticationCheck{Status: "fail"},
		},
	}
	assessment := Evaluate(result, domain.Indicators{URLs: []string{"https://one.test"}, IPs: []string{"203.0.113.10"}}, []domain.Attachment{{Filename: "invoice.pdf.exe"}})
	if assessment.Score != 100 || assessment.Level != "critical" || assessment.Verdict != "malware" {
		t.Fatalf("assessment = %#v", assessment)
	}
	if !hasSignal(assessment, "EXECUTABLE_ATTACHMENT") {
		t.Fatalf("signals = %#v", assessment.ContributingSignals)
	}
}

func TestEvaluateAIAndConfidenceRemainSeparate(t *testing.T) {
	classification := "phishing"
	confidence := 0.8
	result := domain.AnalysisResult{
		AIAssessment: domain.AIAssessment{Status: "completed", Classification: &classification, Confidence: &confidence, EvidenceReferences: []string{"body:1"}},
	}
	assessment := Evaluate(result, domain.Indicators{URLs: []string{"https://example.test"}})
	if assessment.Verdict != "phishing" || assessment.Score != 34 || assessment.Level != "medium" {
		t.Fatalf("assessment = %#v", assessment)
	}
	if assessment.Confidence == nil || *assessment.Confidence <= 0 || *assessment.Confidence >= 1 || *assessment.Confidence == float64(assessment.Score) {
		t.Fatalf("confidence = %#v", assessment.Confidence)
	}

	benign := "benign"
	benignConfidence := 0.95
	strongObserved := result
	strongObserved.AIAssessment = domain.AIAssessment{Status: "completed", Classification: &benign, Confidence: &benignConfidence}
	strongObserved.Authentication.SPF.Status = "fail"
	benignResult := Evaluate(strongObserved, domain.Indicators{URLs: []string{"https://example.test"}})
	if benignResult.Verdict == "benign" || benignResult.Score == 0 {
		t.Fatalf("benign AI erased observed evidence: %#v", benignResult)
	}
}

func TestEvaluateProviderFailureStillUsesDeterministicSignals(t *testing.T) {
	result := domain.AnalysisResult{
		AIAssessment:   domain.AIAssessment{Status: "failed", Failure: &domain.Failure{Code: "AI_PROVIDER_REQUEST_FAILED"}},
		Authentication: domain.AuthenticationResults{DMARC: domain.AuthenticationCheck{Status: "fail"}},
	}
	assessment := Evaluate(result, domain.Indicators{})
	if assessment.Score != 20 || assessment.Level != "low" || assessment.Verdict != "unknown" || assessment.Confidence == nil {
		t.Fatalf("assessment = %#v", assessment)
	}
}

func TestLevelBoundaries(t *testing.T) {
	tests := []struct {
		score int
		want  string
	}{
		{0, "low"}, {24, "low"}, {25, "medium"}, {49, "medium"}, {50, "high"}, {74, "high"}, {75, "critical"}, {100, "critical"},
	}
	for _, test := range tests {
		if got := levelFor(test.score); got != test.want {
			t.Errorf("levelFor(%d) = %q, want %q", test.score, got, test.want)
		}
	}
}

func hasSignal(assessment domain.RiskAssessment, code string) bool {
	for _, signal := range assessment.ContributingSignals {
		if signal.Code == code {
			return true
		}
	}
	return false
}
