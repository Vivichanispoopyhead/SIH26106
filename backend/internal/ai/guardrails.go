package ai

import (
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"

	"sih26106/backend/internal/domain"
)

const PromptVersion = "email-assessment-v1"

// InputPolicy bounds each input component as well as the serialized provider
// request. The limits are intentionally conservative and contain metadata,
// never attachment bytes.
type InputPolicy struct {
	MaxTotalChars           int
	MaxBodyChars            int
	MaxHeaders              int
	MaxHeaderValueChars     int
	MaxIndicatorsPerType    int
	MaxIndicatorChars       int
	MaxAttachments          int
	MaxAttachmentFieldChars int
}

func DefaultInputPolicy() InputPolicy {
	return InputPolicy{
		MaxTotalChars:           24000,
		MaxBodyChars:            12000,
		MaxHeaders:              100,
		MaxHeaderValueChars:     512,
		MaxIndicatorsPerType:    50,
		MaxIndicatorChars:       256,
		MaxAttachments:          50,
		MaxAttachmentFieldChars: 256,
	}
}

// NormalizeInput creates the only representation that should cross an
// analyzer/provider boundary. It preserves useful structure while ensuring
// raw attachment bytes and literal URL values are never included.
func NormalizeInput(input Input, policy InputPolicy) Input {
	policy = policy.withDefaults()
	input.Message = normalizeMessage(input.Message, policy)
	input.PlainTextBody = redactAndLimit(input.PlainTextBody, policy.MaxBodyChars)
	input.Headers = normalizeHeaders(input.Headers, policy)
	input.Indicators = normalizeIndicators(input.Indicators, policy)
	input.Attachments = normalizeAttachments(input.Attachments, policy)
	return fitTotal(input, policy.MaxTotalChars)
}

type GuardedAnalyzer struct {
	delegate Analyzer
	policy   InputPolicy
}

func NewGuardedAnalyzer(delegate Analyzer, policy InputPolicy) Analyzer {
	if delegate == nil {
		delegate = UnavailableAnalyzer{}
	}
	return &GuardedAnalyzer{delegate: delegate, policy: policy.withDefaults()}
}

func (a *GuardedAnalyzer) Assess(ctx context.Context, input Input) (assessment domain.AIAssessment, err error) {
	start := time.Now()
	assessment, err = a.delegate.Assess(ctx, NormalizeInput(input, a.policy))
	assessment.SupportingSignals = deduplicateStrings(assessment.SupportingSignals)
	assessment.EvidenceReferences = deduplicateStrings(assessment.EvidenceReferences)
	provider, model := "", ""
	if assessment.Provider != nil {
		provider = *assessment.Provider
	}
	if assessment.Model != nil {
		model = *assessment.Model
	}
	status := assessment.Status
	failureCode := ""
	if assessment.Failure != nil {
		failureCode = assessment.Failure.Code
	}
	if err != nil && failureCode == "" {
		failureCode = "AI_ANALYSIS_FAILED"
	}
	slog.Info("ai assessment completed",
		slog.String("provider", provider),
		slog.String("model", model),
		slog.String("prompt_version", PromptVersion),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		slog.String("status", status),
		slog.String("failure_code", failureCode),
	)
	return assessment, err
}

func deduplicateStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (p InputPolicy) withDefaults() InputPolicy {
	d := DefaultInputPolicy()
	if p.MaxTotalChars <= 0 {
		p.MaxTotalChars = d.MaxTotalChars
	}
	if p.MaxBodyChars <= 0 {
		p.MaxBodyChars = d.MaxBodyChars
	}
	if p.MaxHeaders <= 0 {
		p.MaxHeaders = d.MaxHeaders
	}
	if p.MaxHeaderValueChars <= 0 {
		p.MaxHeaderValueChars = d.MaxHeaderValueChars
	}
	if p.MaxIndicatorsPerType <= 0 {
		p.MaxIndicatorsPerType = d.MaxIndicatorsPerType
	}
	if p.MaxIndicatorChars <= 0 {
		p.MaxIndicatorChars = d.MaxIndicatorChars
	}
	if p.MaxAttachments <= 0 {
		p.MaxAttachments = d.MaxAttachments
	}
	if p.MaxAttachmentFieldChars <= 0 {
		p.MaxAttachmentFieldChars = d.MaxAttachmentFieldChars
	}
	return p
}

func normalizeMessage(message domain.MessageMetadata, policy InputPolicy) domain.MessageMetadata {
	message.MessageID = optionalRedacted(message.MessageID, policy.MaxHeaderValueChars)
	message.Subject = optionalRedacted(message.Subject, policy.MaxHeaderValueChars)
	message.Date = optionalRedacted(message.Date, policy.MaxHeaderValueChars)
	message.ReturnPath = optionalRedacted(message.ReturnPath, policy.MaxHeaderValueChars)
	message.From = normalizeStrings(message.From, policy.MaxIndicatorsPerType, policy.MaxHeaderValueChars)
	message.To = normalizeStrings(message.To, policy.MaxIndicatorsPerType, policy.MaxHeaderValueChars)
	message.CC = normalizeStrings(message.CC, policy.MaxIndicatorsPerType, policy.MaxHeaderValueChars)
	message.ReplyTo = normalizeStrings(message.ReplyTo, policy.MaxIndicatorsPerType, policy.MaxHeaderValueChars)
	return message
}

func normalizeHeaders(headers []domain.Header, policy InputPolicy) []domain.Header {
	if len(headers) > policy.MaxHeaders {
		headers = headers[:policy.MaxHeaders]
	}
	result := make([]domain.Header, 0, len(headers))
	for _, header := range headers {
		result = append(result, domain.Header{Name: redactAndLimit(header.Name, policy.MaxHeaderValueChars), Value: redactAndLimit(header.Value, policy.MaxHeaderValueChars), Order: header.Order})
	}
	return result
}

func normalizeIndicators(indicators domain.Indicators, policy InputPolicy) domain.Indicators {
	return domain.Indicators{
		IPs:     normalizeStrings(indicators.IPs, policy.MaxIndicatorsPerType, policy.MaxIndicatorChars),
		Domains: normalizeStrings(indicators.Domains, policy.MaxIndicatorsPerType, policy.MaxIndicatorChars),
		// Keep stable URL evidence slots without sending literal URLs to a provider.
		URLs: redactedIndicatorSlots(indicators.URLs, policy.MaxIndicatorsPerType),
	}
}

func normalizeAttachments(attachments []domain.Attachment, policy InputPolicy) []domain.Attachment {
	if len(attachments) > policy.MaxAttachments {
		attachments = attachments[:policy.MaxAttachments]
	}
	result := make([]domain.Attachment, 0, len(attachments))
	for _, attachment := range attachments {
		result = append(result, domain.Attachment{
			Filename:  redactAndLimit(attachment.Filename, policy.MaxAttachmentFieldChars),
			MIMEType:  redactAndLimit(attachment.MIMEType, policy.MaxAttachmentFieldChars),
			SizeBytes: attachment.SizeBytes,
			SHA256:    redactAndLimit(attachment.SHA256, policy.MaxAttachmentFieldChars),
		})
	}
	return result
}

func fitTotal(input Input, limit int) Input {
	for attempt := 0; attempt < 20; attempt++ {
		encoded, _ := json.Marshal(input)
		if len(encoded) <= limit {
			return input
		}
		if len(input.PlainTextBody) > 0 {
			input.PlainTextBody = truncateRunes(input.PlainTextBody, len(input.PlainTextBody)/2)
			continue
		}
		if len(input.Headers) > 0 {
			input.Headers = input.Headers[:len(input.Headers)/2]
			continue
		}
		if len(input.Attachments) > 0 {
			input.Attachments = input.Attachments[:len(input.Attachments)/2]
			continue
		}
		if len(input.Indicators.URLs)+len(input.Indicators.IPs)+len(input.Indicators.Domains) > 0 {
			input.Indicators = domain.Indicators{}
			continue
		}
		input.Message = domain.MessageMetadata{}
		input.PlainTextBody = ""
		input.Headers = nil
		input.Attachments = nil
		input.AnalysisID = truncateRunes(input.AnalysisID, 32)
		input.EmailID = truncateRunes(input.EmailID, 32)
		input.CaseID = truncateRunes(input.CaseID, 32)
		break
	}
	return input
}

var (
	secretPattern     = regexp.MustCompile(`(?i)(password|passwd|pwd|api[_ -]?key|secret|token|authorization|bearer)\s*[:=]\s*[^\s,;]+`)
	bearerPattern     = regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+/-]+=*`)
	privateKeyPattern = regexp.MustCompile(`(?s)-----BEGIN [^-]*PRIVATE KEY-----.*?-----END [^-]*PRIVATE KEY-----`)
	urlPattern        = regexp.MustCompile(`(?i)\b(?:https?|ftp)://[^\s<>"']+`)
)

func redactAndLimit(value string, limit int) string {
	value = privateKeyPattern.ReplaceAllString(value, "[REDACTED_PRIVATE_KEY]")
	value = secretPattern.ReplaceAllString(value, "$1=[REDACTED]")
	value = bearerPattern.ReplaceAllString(value, "Bearer [REDACTED]")
	value = urlPattern.ReplaceAllString(value, "[REDACTED_URL]")
	return truncateRunes(value, limit)
}

func optionalRedacted(value *string, limit int) *string {
	if value == nil {
		return nil
	}
	result := redactAndLimit(*value, limit)
	return &result
}

func normalizeStrings(values []string, max, chars int) []string {
	if len(values) > max {
		values = values[:max]
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, redactAndLimit(value, chars))
	}
	return result
}

func redactedIndicatorSlots(values []string, max int) []string {
	if len(values) > max {
		values = values[:max]
	}
	result := make([]string, len(values))
	for index := range values {
		result[index] = "url-" + formatIndex(index+1)
	}
	return result
}

func formatIndex(index int) string {
	return strconv.Itoa(index)
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit])
}
