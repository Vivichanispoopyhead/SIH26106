package report

import (
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"sih26106/backend/internal/domain"
)

const SchemaVersion = "1.0"

type Input struct {
	Email    *domain.Email
	Parsed   *domain.ParsedEmail
	Result   *domain.AnalysisResult
	Analysis *domain.Analysis
}

func Build(caseValue *domain.Case, inputs []Input, graphValue *domain.Graph, timelineValue *domain.Timeline, generatedAt time.Time) *domain.ForensicReport {
	report := &domain.ForensicReport{
		ReportID:      reportID(caseValue.ID, inputs),
		SchemaVersion: SchemaVersion,
		Case:          domain.ReportCase{ID: caseValue.ID, Status: caseValue.Status, CreatedAt: caseValue.CreatedAt, UpdatedAt: caseValue.UpdatedAt},
		GeneratedAt:   generatedAt.UTC(),
		Status:        "completed",
		Emails:        []domain.ReportEmail{},
		Analyses:      []domain.ReportAnalysis{},
		Evidence:      []domain.Evidence{},
		Graph:         graphSummary(graphValue),
		Timeline:      []domain.TimelineEvent{},
		Limitations:   []string{},
	}
	if timelineValue != nil {
		report.Timeline = append(report.Timeline, timelineValue.Events...)
	}
	for _, input := range inputs {
		if input.Email == nil || input.Parsed == nil || input.Result == nil {
			continue
		}
		report.Emails = append(report.Emails, emailSummary(input.Email, input.Parsed))
		report.Analyses = append(report.Analyses, analysisSummary(input.Result))
		report.Evidence = append(report.Evidence, safeEvidence(input.Result.Evidence)...)
		if input.Result.Status == "partial" {
			report.Status = "partial"
			appendLimitation(&report.Limitations, "Analysis completed partially; one or more non-fatal stages were unavailable or failed.")
		}
		addLimitations(&report.Limitations, input.Result)
	}
	report.Limitations = uniqueSorted(report.Limitations)
	sort.Slice(report.Emails, func(i, j int) bool { return report.Emails[i].EmailID < report.Emails[j].EmailID })
	sort.Slice(report.Analyses, func(i, j int) bool { return report.Analyses[i].AnalysisID < report.Analyses[j].AnalysisID })
	sort.Slice(report.Evidence, func(i, j int) bool { return report.Evidence[i].EvidenceID < report.Evidence[j].EvidenceID })
	return report
}

func emailSummary(email *domain.Email, parsed *domain.ParsedEmail) domain.ReportEmail {
	return domain.ReportEmail{EmailID: email.ID, Filename: safeText(email.Filename, 256), CreatedAt: email.CreatedAt, Message: safeMessage(parsed.Message), MIME: parsed.MIME, Indicators: safeIndicators(parsed.Indicators), Attachments: safeAttachments(parsed.Attachments)}
}

func analysisSummary(result *domain.AnalysisResult) domain.ReportAnalysis {
	copy := domain.ReportAnalysis{AnalysisID: result.AnalysisID, EmailID: result.EmailID, CaseID: result.CaseID, Status: result.Status, Risk: safeRisk(result.Risk), Authentication: safeAuthentication(result.Authentication), ReceivedChain: append([]domain.ReceivedRelay{}, result.ReceivedChain...), IPEnrichment: safeEnrichment(result.IPEnrichment), AIAssessment: safeAI(result.AIAssessment)}
	for index := range copy.ReceivedChain {
		if copy.ReceivedChain[index].Hostname != nil {
			value := safeText(*copy.ReceivedChain[index].Hostname, 256)
			copy.ReceivedChain[index].Hostname = &value
		}
	}
	return copy
}

func graphSummary(value *domain.Graph) domain.ReportGraphSummary {
	result := domain.ReportGraphSummary{NodeTypes: map[string]int{}, NodeIDs: []string{}, EdgeIDs: []string{}}
	if value == nil {
		return result
	}
	result.NodeCount, result.EdgeCount = len(value.Nodes), len(value.Edges)
	for _, node := range value.Nodes {
		result.NodeTypes[node.Type]++
		result.NodeIDs = append(result.NodeIDs, safeGraphID(node.ID))
	}
	for _, edge := range value.Edges {
		result.EdgeIDs = append(result.EdgeIDs, safeGraphID(edge.ID))
	}
	result.NodeIDs = uniqueSorted(result.NodeIDs)
	result.EdgeIDs = uniqueSorted(result.EdgeIDs)
	return result
}

func safeGraphID(value string) string {
	if strings.HasPrefix(value, "url:") {
		return "url:" + safeURL(strings.TrimPrefix(value, "url:"))
	}
	return safeText(value, 512)
}

func safeMessage(value domain.MessageMetadata) domain.MessageMetadata {
	result := value
	result.MessageID, result.Subject, result.Date, result.ReturnPath = safePointer(value.MessageID, 256), safePointer(value.Subject, 512), safePointer(value.Date, 128), safePointer(value.ReturnPath, 256)
	result.From, result.To, result.CC, result.ReplyTo = safeStrings(value.From, 256), safeStrings(value.To, 256), safeStrings(value.CC, 256), safeStrings(value.ReplyTo, 256)
	return result
}

func safeIndicators(value domain.Indicators) domain.Indicators {
	result := domain.Indicators{IPs: safeStrings(value.IPs, 128), Domains: safeStrings(value.Domains, 256), URLs: []string{}}
	for _, item := range value.URLs {
		result.URLs = append(result.URLs, safeURL(item))
	}
	return result
}

func safeAttachments(values []domain.Attachment) []domain.Attachment {
	result := make([]domain.Attachment, 0, len(values))
	for _, item := range values {
		item.Filename, item.MIMEType, item.SHA256 = safeText(item.Filename, 256), safeText(item.MIMEType, 128), safeText(item.SHA256, 128)
		result = append(result, item)
	}
	return result
}

func safeRisk(value domain.RiskAssessment) domain.RiskAssessment {
	value.EvidenceReferences = safeReferences(value.EvidenceReferences)
	value.ContributingSignals = append([]domain.RiskSignal{}, value.ContributingSignals...)
	for index := range value.ContributingSignals {
		value.ContributingSignals[index].Description = safeText(value.ContributingSignals[index].Description, 512)
		value.ContributingSignals[index].EvidenceReferences = safeReferences(value.ContributingSignals[index].EvidenceReferences)
		value.ContributingSignals[index].EvidenceIDs = append([]string{}, value.ContributingSignals[index].EvidenceIDs...)
	}
	return value
}

func safeAuthentication(value domain.AuthenticationResults) domain.AuthenticationResults {
	value.SPF = safeAuthenticationCheck(value.SPF)
	value.DKIM = safeAuthenticationCheck(value.DKIM)
	value.DMARC = safeAuthenticationCheck(value.DMARC)
	return value
}

func safeAuthenticationCheck(value domain.AuthenticationCheck) domain.AuthenticationCheck {
	value.Explanation = safeText(value.Explanation, 512)
	value.EvidenceReferences = safeReferences(value.EvidenceReferences)
	return value
}

func safeReferences(values []domain.EvidenceReference) []domain.EvidenceReference {
	result := append([]domain.EvidenceReference{}, values...)
	for index := range result {
		result[index].HeaderName = safeText(result[index].HeaderName, 128)
		result[index].Source = safeText(result[index].Source, 128)
		result[index].Value = safeText(result[index].Value, 512)
	}
	return result
}

func safeEnrichment(values []domain.IPEnrichment) []domain.IPEnrichment {
	result := append([]domain.IPEnrichment{}, values...)
	for index := range result {
		result[index].Country = safePointer(result[index].Country, 128)
		result[index].Region = safePointer(result[index].Region, 128)
		result[index].City = safePointer(result[index].City, 128)
		result[index].ASN = safePointer(result[index].ASN, 128)
		result[index].Organization = safePointer(result[index].Organization, 256)
		result[index].Provider = safePointer(result[index].Provider, 128)
		result[index].Provenance = safeText(result[index].Provenance, 64)
		if result[index].Failure != nil {
			failure := *result[index].Failure
			failure.Message = safeText(failure.Message, 512)
			result[index].Failure = &failure
		}
	}
	return result
}

func safeAI(value domain.AIAssessment) domain.AIAssessment {
	value.SupportingSignals, value.EvidenceReferences = safeStrings(value.SupportingSignals, 512), safeStrings(value.EvidenceReferences, 128)
	value.Provider, value.Model = safePointer(value.Provider, 128), safePointer(value.Model, 128)
	if value.Failure != nil {
		failure := *value.Failure
		failure.Message = safeText(failure.Message, 512)
		value.Failure = &failure
	}
	return value
}

func safeEvidence(values []domain.Evidence) []domain.Evidence {
	result := make([]domain.Evidence, 0, len(values))
	for _, item := range values {
		item.Source, item.Value, item.Snippet = safeText(item.Source, 128), safeText(item.Value, 512), safeText(item.Snippet, 512)
		if item.Type == "body" {
			item.Value = "body"
			item.Snippet = "Plain-text body evidence; excerpt omitted from report."
		}
		item.SafeDisplay.Label = safeText(item.SafeDisplay.Label, 128)
		result = append(result, item)
	}
	return result
}

func addLimitations(limitations *[]string, result *domain.AnalysisResult) {
	appendLimitation(limitations, "AI output is an evaluated assessment, not ground truth.")
	appendLimitation(limitations, "Geolocation is an IP-derived estimate, not proof of physical location.")
	appendLimitation(limitations, "URLs were not automatically visited and attachments were not executed.")
	if result.AIAssessment.Status == "not_available" {
		appendLimitation(limitations, "AI assessment was unavailable; missing AI output does not imply benign behavior.")
	}
	if result.AIAssessment.Status == "failed" || result.AIAssessment.Failure != nil {
		appendLimitation(limitations, "AI provider failure was retained as a limitation; observed evidence remains authoritative.")
	}
	for _, item := range result.IPEnrichment {
		if item.Status == "not_configured" {
			appendLimitation(limitations, "IP enrichment provider was not configured; observed IP data was preserved.")
		}
		if item.Status == "failed" {
			appendLimitation(limitations, "IP enrichment failed for one or more observed addresses; this does not imply benign behavior.")
		}
	}
	if result.Failure != nil {
		appendLimitation(limitations, "A non-fatal analysis stage reported: "+safeText(result.Failure.Message, 512))
	}
}

func reportID(caseID string, inputs []Input) string {
	ids := []string{}
	for _, input := range inputs {
		if input.Result != nil {
			ids = append(ids, input.Result.AnalysisID)
		}
	}
	sort.Strings(ids)
	return "report:" + caseID + ":" + strings.Join(ids, ",")
}

var (
	privateKeyPattern = regexp.MustCompile(`(?is)-----BEGIN [^-]*PRIVATE KEY-----.*?-----END [^-]*PRIVATE KEY-----`)
	secretPattern     = regexp.MustCompile(`(?i)\b(password|passwd|pwd|token|secret|api[_-]?key|access[_-]?key)\s*[:=]\s*[^\s,;]+`)
	bearerPattern     = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+`)
)

func safeText(value string, limit int) string {
	value = privateKeyPattern.ReplaceAllString(value, "[REDACTED_PRIVATE_KEY]")
	value = secretPattern.ReplaceAllString(value, "$1=[REDACTED]")
	value = bearerPattern.ReplaceAllString(value, "Bearer [REDACTED]")
	runes := []rune(value)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return value
}

func safeURL(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return safeText(value, 512)
	}
	parsed.User = nil
	query := parsed.Query()
	for key := range query {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "token") || strings.Contains(lower, "key") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") {
			query.Set(key, "[REDACTED]")
		}
	}
	parsed.RawQuery = query.Encode()
	return safeText(parsed.String(), 512)
}

func safePointer(value *string, limit int) *string {
	if value == nil {
		return nil
	}
	item := safeText(*value, limit)
	return &item
}

func safeStrings(values []string, limit int) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, safeText(value, limit))
	}
	return result
}

func appendLimitation(values *[]string, value string) {
	for _, existing := range *values {
		if existing == value {
			return
		}
	}
	*values = append(*values, value)
}

func uniqueSorted(values []string) []string {
	result := append([]string{}, values...)
	sort.Strings(result)
	output := []string{}
	for _, value := range result {
		if value != "" && (len(output) == 0 || output[len(output)-1] != value) {
			output = append(output, value)
		}
	}
	return output
}
