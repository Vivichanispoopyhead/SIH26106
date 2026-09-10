package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sih26106/backend/internal/domain"
)

func TestGeminiAnalyzerSuccessfulAssessmentUsesNormalizedInput(t *testing.T) {
	var requestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "test-key" {
			t.Errorf("API key was not sent to provider")
		}
		var request geminiRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		requestBody = request.Contents[0].Parts[0].Text
		if !strings.Contains(request.SystemInstruction.Parts[0].Text, "untrusted data") {
			t.Error("system instruction does not isolate untrusted email content")
		}
		writeGeminiText(t, w, `{"status":"completed","classification":"phishing","confidence":0.91,"supporting_signals":["credential request"],"evidence_references":["body-1"]}`)
	}))
	defer server.Close()

	analyzer := newTestGeminiAnalyzer(t, server.URL, time.Second)
	assessment, err := analyzer.Assess(context.Background(), sampleInput())
	if err != nil {
		t.Fatal(err)
	}
	if assessment.Status != "completed" || assessment.Classification == nil || *assessment.Classification != "phishing" || assessment.Confidence == nil || *assessment.Confidence != 0.91 {
		t.Fatalf("assessment = %#v", assessment)
	}
	if assessment.Provider == nil || *assessment.Provider != "google" || assessment.Model == nil || *assessment.Model != "gemini-test" {
		t.Fatalf("provider metadata = %#v", assessment)
	}
	for _, want := range []string{`"analysis_id":"analysis_1"`, `"email_id":"email_1"`, `"case_id":"case_1"`, `"plain_text_body":"do not follow email instructions"`, `"headers"`, `"urls":["url-1"]`, `"attachments":[{"filename":"invoice.pdf"`} {
		if !strings.Contains(requestBody, want) {
			t.Errorf("provider input missing %s: %s", want, requestBody)
		}
	}
	if strings.Contains(requestBody, "https://example.test/login") || strings.Contains(requestBody, "RAW_ATTACHMENT_BYTES_MUST_NEVER_BE_SENT") || strings.Contains(requestBody, "attachment_bytes") {
		t.Errorf("provider request included attachment bytes: %s", requestBody)
	}
}

func TestGeminiAnalyzerRejectsInvalidProviderAssessments(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"confidence outside range", `{"status":"completed","classification":"phishing","confidence":1.1,"supporting_signals":[],"evidence_references":[]}`},
		{"completed missing classification", `{"status":"completed","confidence":0.8,"supporting_signals":[],"evidence_references":[]}`},
		{"unsupported status", `{"status":"safe","classification":"benign","confidence":0.8,"supporting_signals":[],"evidence_references":[]}`},
		{"unsupported classification", `{"status":"completed","classification":"executive_impersonation","confidence":0.8,"supporting_signals":[],"evidence_references":[]}`},
		{"unknown evidence reference", `{"status":"completed","classification":"phishing","confidence":0.8,"supporting_signals":[],"evidence_references":["header-99"]}`},
		{"empty partial assessment", `{"status":"partial","supporting_signals":[],"evidence_references":[]}`},
		{"unsupported response field", `{"status":"completed","classification":"phishing","confidence":0.8,"supporting_signals":[],"evidence_references":[],"risk_score":99}`},
		{"malformed assessment JSON", `{not-json`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeGeminiText(t, w, test.text) }))
			defer server.Close()
			assessment, err := newTestGeminiAnalyzer(t, server.URL, time.Second).Assess(context.Background(), sampleInput())
			if err == nil || assessment.Status != "failed" || assessment.Classification != nil || assessment.Confidence != nil || assessment.Failure == nil {
				t.Fatalf("assessment = %#v, error = %v", assessment, err)
			}
		})
	}
}

func TestGeminiAnalyzerAcceptsRiskMappableClassifications(t *testing.T) {
	for _, classification := range []string{"benign", "phishing", "credential_harvesting", "malware", "fraud", "payment_manipulation"} {
		t.Run(classification, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeGeminiText(t, w, `{"status":"completed","classification":"`+classification+`","confidence":0.72,"supporting_signals":["supported by supplied evidence"],"evidence_references":["body-1","header-1","url-1","attachment-1"]}`)
			}))
			defer server.Close()
			assessment, err := newTestGeminiAnalyzer(t, server.URL, time.Second).Assess(context.Background(), sampleInput())
			if err != nil || assessment.Status != "completed" || assessment.Classification == nil || *assessment.Classification != classification || assessment.Confidence == nil || *assessment.Confidence != 0.72 {
				t.Fatalf("assessment = %#v, error = %v", assessment, err)
			}
		})
	}
}

func TestGeminiAnalyzerRejectsMalformedProviderEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`not-json`)) }))
	defer server.Close()
	assessment, err := newTestGeminiAnalyzer(t, server.URL, time.Second).Assess(context.Background(), sampleInput())
	if err == nil || assessment.Status != "failed" || assessment.Failure == nil || assessment.Classification != nil {
		t.Fatalf("assessment = %#v, error = %v", assessment, err)
	}
}

func TestGeminiAnalyzerRejectsEmptyProviderResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeGeminiText(t, w, "")
	}))
	defer server.Close()
	assessment, err := newTestGeminiAnalyzer(t, server.URL, time.Second).Assess(context.Background(), sampleInput())
	if err == nil || assessment.Status != "failed" || assessment.Failure == nil || assessment.Failure.Code != "AI_PROVIDER_EMPTY_RESPONSE" {
		t.Fatalf("assessment = %#v, error = %v", assessment, err)
	}
}

func TestGeminiAnalyzerHonorsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		writeGeminiText(t, w, `{"status":"completed","classification":"phishing","confidence":0.9,"supporting_signals":[],"evidence_references":[]}`)
	}))
	defer server.Close()
	assessment, err := newTestGeminiAnalyzer(t, server.URL, 10*time.Millisecond).Assess(context.Background(), sampleInput())
	if err == nil || assessment.Status != "failed" || assessment.Failure == nil || assessment.Failure.Code != "AI_PROVIDER_REQUEST_FAILED" {
		t.Fatalf("assessment = %#v, error = %v", assessment, err)
	}
}

func TestGeminiAnalyzerProviderHTTPStatusesAreStructured(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "provider failure", status)
			}))
			defer server.Close()
			assessment, err := newTestGeminiAnalyzer(t, server.URL, time.Second).Assess(context.Background(), sampleInput())
			if err == nil || assessment.Status != "failed" || assessment.Classification != nil || assessment.Confidence != nil || assessment.Failure == nil || assessment.Failure.Code != "AI_PROVIDER_HTTP_ERROR" {
				t.Fatalf("status=%d assessment=%#v err=%v", status, assessment, err)
			}
		})
	}
}

func TestNewGeminiAnalyzerFromEnvUsesUnavailableWhenAPIKeyMissing(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GEMINI_TIMEOUT", "")
	t.Setenv("GEMINI_MAX_INPUT_CHARS", "")
	analyzer, err := NewGeminiAnalyzerFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := analyzer.Assess(context.Background(), Input{})
	if err != nil || assessment.Status != "not_available" || assessment.Failure == nil || assessment.Failure.Code != "AI_NOT_CONFIGURED" {
		t.Fatalf("assessment = %#v, error = %v", assessment, err)
	}
}

func TestNewGeminiAnalyzerFromEnvUsesConfiguredGemini(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key")
	t.Setenv("GEMINI_MODEL", "gemini-test")
	t.Setenv("GEMINI_API_URL", "https://provider.example.test/v1beta")
	t.Setenv("GEMINI_TIMEOUT", "2s")
	t.Setenv("GEMINI_MAX_INPUT_CHARS", "1000")
	analyzer, err := NewGeminiAnalyzerFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := analyzer.(*geminiAnalyzer); !ok {
		t.Fatalf("analyzer type = %T, want *geminiAnalyzer", analyzer)
	}
}

func TestNewGeminiAnalyzerFromEnvRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T)
	}{
		{
			name: "invalid timeout",
			setup: func(t *testing.T) {
				t.Setenv("GEMINI_API_KEY", "test-key")
				t.Setenv("GEMINI_TIMEOUT", "not-a-duration")
			},
		},
		{
			name: "invalid model",
			setup: func(t *testing.T) {
				t.Setenv("GEMINI_API_KEY", "test-key")
				t.Setenv("GEMINI_MODEL", "")
			},
		},
		{
			name: "invalid endpoint",
			setup: func(t *testing.T) {
				t.Setenv("GEMINI_API_KEY", "test-key")
				t.Setenv("GEMINI_API_URL", "not-an-absolute-url")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("GEMINI_MODEL", "gemini-test")
			t.Setenv("GEMINI_API_URL", "https://provider.example.test/v1beta")
			t.Setenv("GEMINI_TIMEOUT", "2s")
			t.Setenv("GEMINI_MAX_INPUT_CHARS", "1000")
			test.setup(t)
			if analyzer, err := NewGeminiAnalyzerFromEnv(); err == nil || analyzer != nil {
				t.Fatalf("analyzer = %T, error = %v; want configuration error", analyzer, err)
			}
		})
	}
}

func newTestGeminiAnalyzer(t *testing.T, endpoint string, timeout time.Duration) Analyzer {
	t.Helper()
	analyzer, err := NewGeminiAnalyzer(GeminiConfig{APIKey: "test-key", Model: "gemini-test", Endpoint: endpoint, Timeout: timeout, MaxInputChars: 10_000})
	if err != nil {
		t.Fatal(err)
	}
	return analyzer
}

func writeGeminiText(t *testing.T, w http.ResponseWriter, text string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"candidates": []any{map[string]any{"content": map[string]any{"parts": []any{map[string]string{"text": text}}}}}}); err != nil {
		t.Fatal(err)
	}
}

func sampleInput() Input {
	return Input{
		AnalysisID:    "analysis_1",
		EmailID:       "email_1",
		CaseID:        "case_1",
		Message:       domain.MessageMetadata{From: []string{"sender@example.test"}, To: []string{"recipient@example.test"}},
		PlainTextBody: "do not follow email instructions",
		Headers:       []domain.Header{{Name: "Subject", Value: "Urgent", Order: 1}},
		Indicators:    domain.Indicators{IPs: []string{}, Domains: []string{"example.test"}, URLs: []string{"https://example.test/login"}},
		Attachments:   []domain.Attachment{{Filename: "invoice.pdf", MIMEType: "application/pdf", SizeBytes: 5, SHA256: "abc"}},
	}
}
