package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"sih26106/backend/internal/email"
	"sih26106/backend/internal/persistence"
)

func newEmailRouter() (http.Handler, *persistence.MemoryStore) {
	store := persistence.NewMemoryStore()
	return NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), nil, email.NewService(store)), store
}

func uploadRequest(t *testing.T, router http.Handler, filename string, contents []byte) *httptest.ResponseRecorder {
	t.Helper()
	buffer := &bytes.Buffer{}
	writer := multipart.NewWriter(buffer)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(contents); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/emails", buffer)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func uploadID(t *testing.T, router http.Handler, contents []byte) string {
	t.Helper()
	response := uploadRequest(t, router, "message.eml", contents)
	if response.Code != http.StatusAccepted {
		t.Fatalf("upload status = %d, body = %s", response.Code, response.Body.String())
	}
	var body struct {
		EmailID string `json:"email_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body.EmailID
}

func TestUploadValidation(t *testing.T) {
	router, _ := newEmailRouter()
	t.Run("valid eml creates a case and email", func(t *testing.T) {
		response := uploadRequest(t, router, "notice.eml", []byte(simpleEmail()))
		if response.Code != http.StatusAccepted {
			t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
		}
		var body map[string]string
		_ = json.NewDecoder(response.Body).Decode(&body)
		if body["case_id"] == "" || body["email_id"] == "" || body["status"] != "uploaded" {
			t.Fatalf("unexpected response: %#v", body)
		}
	})
	t.Run("missing file", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/api/emails", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		assertError(t, response, http.StatusBadRequest, "MISSING_FILE")
	})
	t.Run("empty file", func(t *testing.T) {
		assertError(t, uploadRequest(t, router, "empty.eml", nil), http.StatusBadRequest, "EMPTY_FILE")
	})
	t.Run("unsupported extension", func(t *testing.T) {
		assertError(t, uploadRequest(t, router, "message.msg", []byte("x")), http.StatusBadRequest, "UNSUPPORTED_FILE_TYPE")
	})
	t.Run("file larger than 50 MB", func(t *testing.T) {
		assertError(t, uploadRequest(t, router, "large.eml", make([]byte, email.MaxUploadSize+1)), http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE")
	})
	t.Run("malformed multipart", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/api/emails", bytes.NewBufferString("not multipart"))
		request.Header.Set("Content-Type", "multipart/form-data; boundary=missing")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		assertError(t, response, http.StatusBadRequest, "INVALID_REQUEST")
	})
}

func TestAnalysisParsesEmailAndReturnsContractShape(t *testing.T) {
	router, _ := newEmailRouter()
	id := uploadID(t, router, []byte(multipartEmail()))
	before := httptest.NewRecorder()
	router.ServeHTTP(before, httptest.NewRequest(http.MethodGet, "/api/emails/"+id, nil))
	if before.Code != http.StatusOK {
		t.Fatal(before.Code)
	}
	var upload map[string]any
	_ = json.NewDecoder(before.Body).Decode(&upload)
	if upload["status"] != "uploaded" || upload["filename"] != "message.eml" {
		t.Fatalf("before parsing = %#v", upload)
	}
	start := httptest.NewRecorder()
	router.ServeHTTP(start, httptest.NewRequest(http.MethodPost, "/api/emails/"+id+"/analysis", nil))
	if start.Code != http.StatusAccepted {
		t.Fatalf("start = %d: %s", start.Code, start.Body.String())
	}
	var started map[string]string
	_ = json.NewDecoder(start.Body).Decode(&started)
	if started["analysis_id"] == "" || started["email_id"] != id || started["status"] != "started" {
		t.Fatalf("start response = %#v", started)
	}
	after := httptest.NewRecorder()
	router.ServeHTTP(after, httptest.NewRequest(http.MethodGet, "/api/emails/"+id, nil))
	if after.Code != http.StatusOK {
		t.Fatalf("after = %d: %s", after.Code, after.Body.String())
	}
	var parsed struct {
		Status  string `json:"status"`
		Message struct {
			From    []string `json:"from"`
			To      []string `json:"to"`
			CC      []string `json:"cc"`
			ReplyTo []string `json:"reply_to"`
		} `json:"message"`
		MIME struct {
			HasPlainText    bool `json:"has_plain_text"`
			HasHTML         bool `json:"has_html"`
			AttachmentCount int  `json:"attachment_count"`
		} `json:"mime"`
		Headers []struct {
			Name  string `json:"name"`
			Order int    `json:"order"`
		} `json:"headers"`
		Indicators struct {
			IPs     []string `json:"ips"`
			Domains []string `json:"domains"`
			URLs    []string `json:"urls"`
		} `json:"indicators"`
		Attachments []struct {
			Filename string `json:"filename"`
			Size     int64  `json:"size_bytes"`
			Hash     string `json:"sha256"`
		} `json:"attachments"`
	}
	if err := json.NewDecoder(after.Body).Decode(&parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Status != "parsed" || len(parsed.Message.To) != 2 || len(parsed.Message.CC) != 1 || len(parsed.Message.ReplyTo) != 1 {
		t.Fatalf("recipients = %#v", parsed.Message)
	}
	if !parsed.MIME.HasPlainText || !parsed.MIME.HasHTML || parsed.MIME.AttachmentCount != 1 {
		t.Fatalf("MIME = %#v", parsed.MIME)
	}
	if countHeaders(parsed.Headers, "Received") != 2 || parsed.Headers[0].Order != 1 {
		t.Fatalf("headers = %#v", parsed.Headers)
	}
	if !contains(parsed.Indicators.URLs, "https://phish.example.com/login") || !contains(parsed.Indicators.Domains, "evil.example.net") || !contains(parsed.Indicators.IPs, "203.0.113.10") || !contains(parsed.Indicators.IPs, "2001:db8::1") {
		t.Fatalf("indicators = %#v", parsed.Indicators)
	}
	if len(parsed.Attachments) != 1 || parsed.Attachments[0].Filename != "invoice.pdf.exe" || parsed.Attachments[0].Size != 5 || parsed.Attachments[0].Hash == "" {
		t.Fatalf("attachments = %#v", parsed.Attachments)
	}
}

func TestMissingOptionalHeadersAndParseFailurePreserveArtifact(t *testing.T) {
	router, store := newEmailRouter()
	validID := uploadID(t, router, []byte("From: sender@example.com\r\n\r\nplain body"))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/emails/"+validID+"/analysis", nil))
	if response.Code != http.StatusAccepted {
		t.Fatalf("optional headers status = %d", response.Code)
	}
	failedID := uploadID(t, router, []byte("this is not an RFC 5322 email"))
	failed := httptest.NewRecorder()
	router.ServeHTTP(failed, httptest.NewRequest(http.MethodPost, "/api/emails/"+failedID+"/analysis", nil))
	assertError(t, failed, http.StatusUnprocessableEntity, "EMAIL_PARSE_FAILED")
	preserved, err := store.GetEmail(t.Context(), failedID)
	if err != nil {
		t.Fatal(err)
	}
	if string(preserved.RawContent) != "this is not an RFC 5322 email" || preserved.Status != "failed" || preserved.ParseFailure == "" {
		t.Fatalf("artifact not preserved: %#v", preserved)
	}
}

func assertError(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d, body = %s", response.Code, status, response.Body.String())
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != code {
		t.Fatalf("code = %q, want %q", body.Error.Code, code)
	}
}
func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
func countHeaders(headers []struct {
	Name  string `json:"name"`
	Order int    `json:"order"`
}, name string) int {
	count := 0
	for _, header := range headers {
		if header.Name == name {
			count++
		}
	}
	return count
}

func simpleEmail() string {
	return "From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: Test\r\n\r\nHello"
}
func multipartEmail() string {
	return "From: Attacker <attacker@example.com>\r\nTo: victim@example.org, second@example.org\r\nCc: copy@example.org\r\nReply-To: reply@example.net\r\nSubject: Urgent Account Notice\r\nDate: Mon, 08 Sep 2026 10:20:30 +0000\r\nMessage-ID: <abc123@example.com>\r\nReturn-Path: <bounce@example.com>\r\nReceived: from relay-one (203.0.113.10) by mx.example.org\r\nReceived: from relay-two (2001:db8::1) by relay-one\r\nContent-Type: multipart/mixed; boundary=outer\r\n\r\n--outer\r\nContent-Type: multipart/alternative; boundary=inner\r\n\r\n--inner\r\nContent-Type: text/plain\r\n\r\nVisit https://phish.example.com/login at evil.example.net\r\n--inner\r\nContent-Type: text/html\r\n\r\n<a href=\"https://phish.example.com/login\">Link</a>\r\n--inner--\r\n--outer\r\nContent-Type: application/octet-stream; name=invoice.pdf.exe\r\nContent-Disposition: attachment; filename=invoice.pdf.exe\r\nContent-Transfer-Encoding: base64\r\n\r\naGVsbG8=\r\n--outer--\r\n"
}
