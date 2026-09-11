package report

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/phpdave11/gofpdf"
	"sih26106/backend/internal/domain"
)

// PDF renders an already sanitized canonical report. It does not access the
// store or perform enrichment, URL fetching, or attachment processing.
func PDF(value *domain.ForensicReport) ([]byte, error) {
	if value == nil {
		return nil, fmt.Errorf("report is nil")
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetCompression(false)
	pdf.SetTitle("Forensic Report "+value.ReportID, true)
	pdf.SetAuthor("SIH26106", true)
	pdf.SetMargins(16, 16, 16)
	pdf.SetAutoPageBreak(true, 16)
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 20)
	pdf.CellFormat(0, 12, "Forensic Investigation Report", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 6, "Report ID: "+value.ReportID, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, "Generated: "+value.GeneratedAt.UTC().Format("2006-01-02 15:04:05 UTC"), "", 1, "L", false, 0, "")

	section(pdf, "Case overview")
	line(pdf, "Case ID", value.Case.ID)
	line(pdf, "Case status", value.Case.Status)
	line(pdf, "Email IDs", joinEmailIDs(value.Emails))
	line(pdf, "Filenames", joinFilenames(value.Emails))
	line(pdf, "Report status", value.Status)

	section(pdf, "Executive summary")
	if len(value.Analyses) == 0 {
		text(pdf, "No completed or partial analysis is available.")
	}
	for _, analysis := range value.Analyses {
		line(pdf, "Analysis", analysis.AnalysisID)
		line(pdf, "Risk", fmt.Sprintf("%d (%s)", analysis.Risk.Score, analysis.Risk.Level))
		line(pdf, "Verdict", analysis.Risk.Verdict)
		line(pdf, "Confidence", formatConfidence(analysis.Risk.Confidence))
	}

	section(pdf, "Message metadata and authentication")
	for _, email := range value.Emails {
		line(pdf, "Sender", join(email.Message.From))
		line(pdf, "Recipients", join(email.Message.To))
		line(pdf, "Subject", pointer(email.Message.Subject))
		line(pdf, "Date", pointer(email.Message.Date))
		line(pdf, "Return-Path", pointer(email.Message.ReturnPath))
		indicatorLines(pdf, email)
	}
	for _, analysis := range value.Analyses {
		line(pdf, "SPF", authentication(analysis.Authentication.SPF))
		line(pdf, "DKIM", authentication(analysis.Authentication.DKIM))
		line(pdf, "DMARC", authentication(analysis.Authentication.DMARC))
	}

	section(pdf, "Received / relay timeline")
	for _, event := range value.Timeline {
		line(pdf, fmt.Sprintf("%d. %s", event.Sequence, event.Title), strings.Join(nonEmpty(
			formatTime(event.Timestamp), pointer(event.Hostname), pointer(event.IPAddress),
			event.Description, "provenance="+event.Provenance, "confidence="+event.Confidence,
		), " | "))
	}
	if len(value.Timeline) == 0 {
		text(pdf, "No relay timeline events were recorded.")
	}

	section(pdf, "IP enrichment")
	for _, analysis := range value.Analyses {
		for _, item := range analysis.IPEnrichment {
			line(pdf, item.IPAddress, strings.Join(nonEmpty(item.Status, pointer(item.Country), pointer(item.Region), pointer(item.City), pointer(item.ASN), pointer(item.Organization), pointer(item.Provider), "provenance="+item.Provenance), " | "))
			if item.Failure != nil {
				line(pdf, "Provider failure", item.Failure.Code+": "+item.Failure.Message)
			}
		}
	}

	section(pdf, "AI assessment")
	for _, analysis := range value.Analyses {
		ai := analysis.AIAssessment
		line(pdf, "Classification", pointer(ai.Classification))
		line(pdf, "Confidence", formatConfidence(ai.Confidence))
		line(pdf, "Supporting signals", join(ai.SupportingSignals))
		line(pdf, "Evidence references", join(ai.EvidenceReferences))
		line(pdf, "Provider / model", strings.Join(nonEmpty(pointer(ai.Provider), pointer(ai.Model)), " / "))
		if ai.Failure != nil {
			line(pdf, "Provider status", ai.Failure.Code+": "+ai.Failure.Message)
		}
	}

	section(pdf, "Evidence summary and provenance")
	for _, item := range value.Evidence {
		line(pdf, item.EvidenceID, strings.Join(nonEmpty(item.SafeDisplay.Label, item.Type, item.Source, item.Value, "provenance="+item.Provenance), " | "))
	}
	if len(value.Evidence) == 0 {
		text(pdf, "No evidence items were recorded.")
	}

	section(pdf, "Entity graph summary")
	line(pdf, "Node count", fmt.Sprintf("%d", value.Graph.NodeCount))
	line(pdf, "Edge count", fmt.Sprintf("%d", value.Graph.EdgeCount))
	line(pdf, "Node types", formatCounts(value.Graph.NodeTypes))
	line(pdf, "Key entities", join(value.Graph.NodeIDs))
	line(pdf, "Relationships", join(value.Graph.EdgeIDs))

	section(pdf, "Limitations and safety disclaimers")
	for _, limitation := range value.Limitations {
		bullet(pdf, limitation)
	}
	for _, disclaimer := range []string{
		"AI output is an evaluated assessment, not ground truth.",
		"IP geolocation is an estimate and does not prove physical location or identity.",
		"URLs were not automatically visited.",
		"Attachments were not executed.",
		"Provider failures or unavailable stages do not imply benign or malicious behavior.",
	} {
		bullet(pdf, disclaimer)
	}

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, fmt.Errorf("render PDF: %w", err)
	}
	return output.Bytes(), nil
}

func section(pdf *gofpdf.Fpdf, title string) {
	pdf.Ln(5)
	pdf.SetFont("Arial", "B", 13)
	pdf.SetTextColor(30, 70, 110)
	pdf.CellFormat(0, 8, title, "", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "", 9)
}

func line(pdf *gofpdf.Fpdf, label, value string) {
	if strings.TrimSpace(value) == "" {
		value = "—"
	}
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(38, 5, label+":", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.MultiCell(0, 5, value, "", "L", false)
}

func text(pdf *gofpdf.Fpdf, value string)   { pdf.MultiCell(0, 5, value, "", "L", false) }
func bullet(pdf *gofpdf.Fpdf, value string) { pdf.MultiCell(0, 5, "- "+value, "", "L", false) }

func indicatorLines(pdf *gofpdf.Fpdf, email domain.ReportEmail) {
	line(pdf, "URLs", join(email.Indicators.URLs))
	line(pdf, "IP addresses", join(email.Indicators.IPs))
	line(pdf, "Domains", join(email.Indicators.Domains))
	for _, attachment := range email.Attachments {
		line(pdf, "Attachment", strings.Join(nonEmpty(attachment.Filename, attachment.MIMEType, attachment.SHA256), " | "))
	}
}

func authentication(value domain.AuthenticationCheck) string {
	return strings.Join(nonEmpty(value.Status, value.Explanation, references(value.EvidenceReferences)), " | ")
}
func references(values []domain.EvidenceReference) string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, strings.Join(nonEmpty(value.HeaderName, value.Source, value.Value), ":"))
	}
	return join(result)
}
func pointer(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func formatTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
func formatConfidence(value *float64) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%.2f", *value)
}
func join(values []string) string { return strings.Join(nonEmpty(values...), ", ") }
func nonEmpty(values ...string) []string {
	result := []string{}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, value)
		}
	}
	return result
}
func formatCounts(values map[string]int) string {
	result := []string{}
	for key, count := range values {
		result = append(result, fmt.Sprintf("%s=%d", key, count))
	}
	return join(result)
}
func joinEmailIDs(values []domain.ReportEmail) string {
	result := []string{}
	for _, value := range values {
		result = append(result, value.EmailID)
	}
	return join(result)
}
func joinFilenames(values []domain.ReportEmail) string {
	result := []string{}
	for _, value := range values {
		result = append(result, value.Filename)
	}
	return join(result)
}
