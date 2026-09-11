package httpapi

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"sih26106/backend/internal/domain"
	"sih26106/backend/internal/email"
	"sih26106/backend/internal/persistence"
	reportgen "sih26106/backend/internal/report"
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

func (h emailHandler) getAnalysis(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.GetAnalysis(r.Context(), chi.URLParam(r, "email_id"))
	switch {
	case errors.Is(err, persistence.ErrEmailNotFound):
		writeError(w, http.StatusNotFound, "EMAIL_NOT_FOUND", "The requested email was not found.")
	case errors.Is(err, persistence.ErrAnalysisNotFound):
		writeError(w, http.StatusNotFound, "ANALYSIS_NOT_FOUND", "No analysis has been started for this email.")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "The analysis could not be retrieved.")
	default:
		writeJSON(w, http.StatusOK, result)
	}
}

func (h emailHandler) getEvidence(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.GetEvidence(r.Context(), chi.URLParam(r, "email_id"))
	switch {
	case errors.Is(err, persistence.ErrEmailNotFound):
		writeError(w, http.StatusNotFound, "EMAIL_NOT_FOUND", "The requested email was not found.")
	case errors.Is(err, persistence.ErrAnalysisNotFound):
		writeError(w, http.StatusNotFound, "ANALYSIS_NOT_FOUND", "No completed analysis exists for this email.")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "The evidence could not be retrieved.")
	case result.Status != "completed" && result.Status != "partial":
		writeError(w, http.StatusNotFound, "ANALYSIS_NOT_FOUND", "No completed analysis exists for this email.")
	default:
		writeJSON(w, http.StatusOK, struct {
			EmailID    string            `json:"email_id"`
			AnalysisID string            `json:"analysis_id"`
			Evidence   []domain.Evidence `json:"evidence"`
		}{result.EmailID, result.AnalysisID, result.Evidence})
	}
}

func (h emailHandler) getGraph(w http.ResponseWriter, r *http.Request) {
	graph, err := h.service.GetGraph(r.Context(), chi.URLParam(r, "case_id"))
	switch {
	case errors.Is(err, persistence.ErrCaseNotFound):
		writeError(w, http.StatusNotFound, "CASE_NOT_FOUND", "The requested case was not found.")
	case errors.Is(err, email.ErrGraphNotAvailable):
		writeError(w, http.StatusNotFound, "GRAPH_NOT_AVAILABLE", "No analyzed email graph is available for this case.")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "The graph could not be retrieved.")
	default:
		writeJSON(w, http.StatusOK, graph)
	}
}

func (h emailHandler) getTimeline(w http.ResponseWriter, r *http.Request) {
	timeline, err := h.service.GetTimeline(r.Context(), chi.URLParam(r, "case_id"))
	switch {
	case errors.Is(err, persistence.ErrCaseNotFound):
		writeError(w, http.StatusNotFound, "CASE_NOT_FOUND", "The requested case was not found.")
	case errors.Is(err, email.ErrTimelineNotAvailable):
		writeError(w, http.StatusNotFound, "TIMELINE_NOT_AVAILABLE", "No analyzed email timeline is available for this case.")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "The timeline could not be retrieved.")
	default:
		writeJSON(w, http.StatusOK, timeline)
	}
}

func (h emailHandler) createReport(w http.ResponseWriter, r *http.Request) {
	report, err := h.service.GetReport(r.Context(), chi.URLParam(r, "case_id"))
	handleReportResult(w, report, err)
}

func (h emailHandler) getReport(w http.ResponseWriter, r *http.Request) {
	report, err := h.service.GetReport(r.Context(), chi.URLParam(r, "case_id"))
	handleReportResult(w, report, err)
}

func (h emailHandler) getReportPDF(w http.ResponseWriter, r *http.Request) {
	report, err := h.service.GetReport(r.Context(), chi.URLParam(r, "case_id"))
	if err != nil {
		handleReportResult(w, nil, err)
		return
	}
	content, err := reportgen.PDF(report)
	if err != nil {
		slog.Default().Error("PDF report generation failed", "case_id", chi.URLParam(r, "case_id"), "error", err.Error())
		writeError(w, http.StatusInternalServerError, "REPORT_PDF_FAILED", "The PDF report could not be generated.")
		return
	}
	filename := safeReportFilename(report.ReportID)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`.pdf"`)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(content); err != nil {
		slog.Default().Debug("PDF report response write failed", "error", err.Error())
	}
}

func safeReportFilename(reportID string) string {
	value := strings.TrimSpace(reportID)
	value = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '_'
	}, value)
	value = strings.Trim(value, "._")
	if value == "" {
		return "forensic-report"
	}
	if len(value) > 120 {
		value = value[:120]
	}
	return value
}

func handleReportResult(w http.ResponseWriter, report any, err error) {
	switch {
	case errors.Is(err, persistence.ErrCaseNotFound):
		writeError(w, http.StatusNotFound, "CASE_NOT_FOUND", "The requested case was not found.")
	case errors.Is(err, email.ErrReportNotAvailable):
		writeError(w, http.StatusNotFound, "REPORT_NOT_AVAILABLE", "No completed or partial analysis is available for this case.")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "The report could not be generated.")
	default:
		writeJSON(w, http.StatusOK, report)
	}
}

func writeParsed(w http.ResponseWriter, emailID, caseID, filename string, parsed *domain.ParsedEmail) {
	if parsed == nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "The parsed email is unavailable.")
		return
	}
	writeJSON(w, http.StatusOK, struct {
		EmailID       string                 `json:"email_id"`
		CaseID        string                 `json:"case_id"`
		Status        string                 `json:"status"`
		Filename      string                 `json:"filename"`
		Message       domain.MessageMetadata `json:"message"`
		MIME          domain.MIMEMetadata    `json:"mime"`
		Headers       []domain.Header        `json:"headers"`
		Indicators    domain.Indicators      `json:"indicators"`
		Attachments   []domain.Attachment    `json:"attachments"`
		PlainTextBody string                 `json:"plain_text_body"`
	}{emailID, caseID, "parsed", filename, parsed.Message, parsed.MIME, parsed.Headers, parsed.Indicators, parsed.Attachments, parsed.PlainTextBody})
}

func isTooLarge(err error) bool { var target *http.MaxBytesError; return errors.As(err, &target) }
