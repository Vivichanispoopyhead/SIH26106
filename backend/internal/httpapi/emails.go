package httpapi

import (
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"sih26106/backend/internal/domain"
	"sih26106/backend/internal/email"
	"sih26106/backend/internal/persistence"
)

type emailHandler struct{ service *email.Service }

func (h emailHandler) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, email.MaxUploadSize+1<<20)
	if err := r.ParseMultipartForm(email.MaxUploadSize + 1); err != nil {
		if errors.Is(err, http.ErrNotMultipart) {
			writeError(w, http.StatusBadRequest, "MISSING_FILE", "A file field is required.")
			return
		}
		if isTooLarge(err) {
			writeError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "The uploaded file exceeds the 50 MB limit.")
			return
		}
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "The multipart request is malformed.")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "MISSING_FILE", "A file field is required.")
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, email.MaxUploadSize+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "The uploaded file could not be read.")
		return
	}
	created, err := h.service.Upload(r.Context(), header.Filename, raw)
	if err != nil {
		h.uploadError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"case_id": created.CaseID, "email_id": created.ID, "status": "uploaded"})
}

func (h emailHandler) uploadError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, email.ErrEmptyFile):
		writeError(w, http.StatusBadRequest, "EMPTY_FILE", "The uploaded file is empty.")
	case errors.Is(err, email.ErrUnsupportedType):
		writeError(w, http.StatusBadRequest, "UNSUPPORTED_FILE_TYPE", "Only .eml files are supported.")
	case errors.Is(err, email.ErrTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "The uploaded file exceeds the 50 MB limit.")
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "The upload could not be completed.")
	}
}

func (h emailHandler) startAnalysis(w http.ResponseWriter, r *http.Request) {
	analysis, err := h.service.StartAnalysis(r.Context(), chi.URLParam(r, "email_id"))
	if err != nil {
		if errors.Is(err, persistence.ErrEmailNotFound) {
			writeError(w, http.StatusNotFound, "EMAIL_NOT_FOUND", "The requested email was not found.")
			return
		}
		if errors.Is(err, email.ErrParseFailed) {
			writeError(w, http.StatusUnprocessableEntity, "EMAIL_PARSE_FAILED", "The email could not be parsed.")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "The analysis could not be completed.")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"analysis_id": analysis.ID, "email_id": analysis.EmailID, "case_id": analysis.CaseID, "status": "started"})
}

func (h emailHandler) get(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.Get(r.Context(), chi.URLParam(r, "email_id"))
	if errors.Is(err, persistence.ErrEmailNotFound) {
		writeError(w, http.StatusNotFound, "EMAIL_NOT_FOUND", "The requested email was not found.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "The email could not be retrieved.")
		return
	}
	switch value.Status {
	case "uploaded":
		writeJSON(w, http.StatusOK, map[string]string{"email_id": value.ID, "case_id": value.CaseID, "status": "uploaded", "filename": value.Filename})
	case "processing":
		writeJSON(w, http.StatusOK, map[string]string{"email_id": value.ID, "case_id": value.CaseID, "status": "processing"})
	case "parsed":
		writeParsed(w, value.ID, value.CaseID, value.Filename, value.Parsed)
	case "failed":
		writeError(w, http.StatusUnprocessableEntity, "EMAIL_PARSE_FAILED", "The email could not be parsed.")
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "The email has an unknown state.")
	}
}

func writeParsed(w http.ResponseWriter, emailID, caseID, filename string, parsed *domain.ParsedEmail) {
	if parsed == nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "The parsed email is unavailable.")
		return
	}
	writeJSON(w, http.StatusOK, struct {
		EmailID     string                 `json:"email_id"`
		CaseID      string                 `json:"case_id"`
		Status      string                 `json:"status"`
		Filename    string                 `json:"filename"`
		Message     domain.MessageMetadata `json:"message"`
		MIME        domain.MIMEMetadata    `json:"mime"`
		Headers     []domain.Header        `json:"headers"`
		Indicators  domain.Indicators      `json:"indicators"`
		Attachments []domain.Attachment    `json:"attachments"`
	}{emailID, caseID, "parsed", filename, parsed.Message, parsed.MIME, parsed.Headers, parsed.Indicators, parsed.Attachments})
}

func isTooLarge(err error) bool { var target *http.MaxBytesError; return errors.As(err, &target) }
