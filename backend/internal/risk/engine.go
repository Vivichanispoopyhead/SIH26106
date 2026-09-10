package risk

import (
	"math"
	"path/filepath"
	"strings"

	"sih26106/backend/internal/domain"
)

type Rule struct {
	Code        string
	Description string
	Points      int
	Category    string
}

// Rules is the intentionally small, deterministic MVP rule table. AI points
// are scaled by the assessment confidence at evaluation time.
var Rules = []Rule{
	{Code: "SPF_FAIL", Description: "SPF authentication failed.", Points: 20, Category: "authentication"},
	{Code: "DKIM_FAIL", Description: "DKIM authentication failed.", Points: 20, Category: "authentication"},
	{Code: "DMARC_FAIL", Description: "DMARC authentication failed.", Points: 20, Category: "authentication"},
	{Code: "AUTHENTICATION_UNKNOWN", Description: "An authentication result was malformed or unknown.", Points: 10, Category: "authentication"},
	{Code: "URL_PRESENT", Description: "The email contains an extracted URL.", Points: 10, Category: "indicator"},
	{Code: "IP_PRESENT", Description: "The email contains an extracted IP indicator.", Points: 5, Category: "indicator"},
	{Code: "EXECUTABLE_ATTACHMENT", Description: "The email contains an executable or double-extension attachment.", Points: 25, Category: "attachment"},
	{Code: "MULTIPLE_INDICATORS", Description: "The email contains multiple bounded indicators.", Points: 5, Category: "indicator"},
	{Code: "AI_PHISHING", Description: "AI assessed phishing-related intent.", Points: 30, Category: "ai_assessed"},
	{Code: "AI_MALWARE", Description: "AI assessed malware-related intent.", Points: 35, Category: "ai_assessed"},
	{Code: "AI_FRAUD", Description: "AI assessed fraud or payment-manipulation intent.", Points: 30, Category: "ai_assessed"},
	{Code: "PUBLIC_IP_OBSERVED", Description: "A publicly routable IP address was observed.", Points: 0, Category: "indicator"},
	{Code: "PRIVATE_IP_OBSERVED", Description: "A non-public IP address was observed.", Points: 0, Category: "indicator"},
	{Code: "IP_ENRICHMENT_UNAVAILABLE", Description: "Passive IP enrichment was unavailable.", Points: 0, Category: "enrichment"},
}

var aiRules = map[string]Rule{
	"phishing":              Rules[8],
	"credential_harvesting": Rules[8],
	"malware":               Rules[9],
	"fraud":                 Rules[10],
	"payment_manipulation":  Rules[10],
}

func Evaluate(result domain.AnalysisResult, indicators domain.Indicators, attachmentSets ...[]domain.Attachment) domain.RiskAssessment {
	assessment := domain.RiskAssessment{
		Level:               "low",
		Verdict:             "unknown",
		ContributingSignals: []domain.RiskSignal{},
		EvidenceReferences:  []domain.EvidenceReference{},
	}
	deterministicCount := 0
	add := func(rule Rule, points int, provenance string, refs []domain.EvidenceReference, description string) {
		if points < 0 {
			return
		}
		if description == "" {
			description = rule.Description
		}
		signal := domain.RiskSignal{Code: rule.Code, Description: description, Points: points, Category: rule.Category, Provenance: provenance, EvidenceReferences: refs}
		assessment.ContributingSignals = append(assessment.ContributingSignals, signal)
		assessment.EvidenceReferences = appendUnique(assessment.EvidenceReferences, refs...)
		assessment.Score += points
		if provenance != "AI-ASSESSED" {
			deterministicCount++
		}
	}

	addAuthenticationSignals(result.Authentication, add)
	if len(result.ReceivedChain) > 0 {
		for _, relay := range result.ReceivedChain {
			if relay.Confidence == "low" {
				ref := domain.EvidenceReference{Source: "received_header", Value: "malformed relay", HeaderOrder: relay.SourceHeaderOrder}
				add(Rules[3], 10, "INFERRED", []domain.EvidenceReference{ref}, "A Received header could not be safely normalized.")
				break
			}
		}
	}
	if len(indicators.URLs) > 0 {
		refs := indicatorReferences("url", indicators.URLs)
		add(Rules[4], 10, "OBSERVED", refs, "The email contains an extracted URL.")
	}
	if len(indicators.IPs) > 0 {
		refs := indicatorReferences("ip", indicators.IPs)
		add(Rules[5], 5, "OBSERVED", refs, "The email contains an extracted IP indicator.")
	}
	if len(indicators.URLs)+len(indicators.IPs) >= 2 {
		refs := append(indicatorReferences("url", indicators.URLs), indicatorReferences("ip", indicators.IPs)...)
		add(Rules[7], 5, "OBSERVED", refs, "The email contains multiple extracted indicators.")
	}
	executable := false
	var attachments []domain.Attachment
	if len(attachmentSets) > 0 {
		attachments = attachmentSets[0]
	}
	for _, attachment := range attachments {
		if isExecutableAttachment(attachment.Filename) {
			executable = true
			ref := domain.EvidenceReference{Source: "attachment", Value: attachment.Filename}
			add(Rules[6], 25, "OBSERVED", []domain.EvidenceReference{ref}, "The email contains an executable or double-extension attachment.")
			break
		}
	}
	addEnrichmentSignals(result.IPEnrichment, add)

	aiClassification, aiAccepted := normalizedAIClassification(result.AIAssessment)
	if aiAccepted {
		if rule, ok := aiRules[aiClassification]; ok {
			points := int(math.Round(float64(rule.Points) * *result.AIAssessment.Confidence))
			refs := make([]domain.EvidenceReference, 0, len(result.AIAssessment.EvidenceReferences))
			for _, reference := range result.AIAssessment.EvidenceReferences {
				refs = append(refs, domain.EvidenceReference{Source: "ai_assessment", Value: reference})
			}
			add(rule, points, "AI-ASSESSED", refs, rule.Description+" Contribution scaled by AI confidence.")
		}
	}

	assessment.Score = clamp(assessment.Score, 0, 100)
	assessment.Level = levelFor(assessment.Score)
	assessment.Verdict = verdictFor(assessment.Score, aiClassification, aiAccepted, executable, deterministicCount, indicators, result.Authentication)
	assessment.Confidence = confidenceFor(assessment, result.AIAssessment, aiAccepted, deterministicCount)
	return assessment
}

func addEnrichmentSignals(values []domain.IPEnrichment, add func(Rule, int, string, []domain.EvidenceReference, string)) {
	for _, value := range values {
		ref := domain.EvidenceReference{Source: "ip", Value: value.IPAddress}
		switch value.Status {
		case "enriched":
			add(Rules[11], 0, "OBSERVED", []domain.EvidenceReference{ref}, Rules[11].Description)
		case "not_applicable":
			add(Rules[12], 0, "OBSERVED", []domain.EvidenceReference{ref}, Rules[12].Description)
		case "failed", "not_configured":
			add(Rules[13], 0, "INFERRED", []domain.EvidenceReference{ref}, Rules[13].Description)
		}
	}
}

func addAuthenticationSignals(auth domain.AuthenticationResults, add func(Rule, int, string, []domain.EvidenceReference, string)) {
	checks := []struct {
		name  string
		check domain.AuthenticationCheck
		rule  Rule
	}{
		{"SPF", auth.SPF, Rules[0]}, {"DKIM", auth.DKIM, Rules[1]}, {"DMARC", auth.DMARC, Rules[2]},
	}
	for _, item := range checks {
		refs := authenticationReferences(item.check)
		switch item.check.Status {
		case "fail":
			add(item.rule, item.rule.Points, "OBSERVED", refs, item.name+" authentication failed.")
		case "unknown":
			add(Rules[3], Rules[3].Points, "PARSED", refs, item.name+" authentication result was unknown.")
		}
	}
}

func authenticationReferences(check domain.AuthenticationCheck) []domain.EvidenceReference {
	refs := make([]domain.EvidenceReference, 0, len(check.EvidenceReferences))
	for _, ref := range check.EvidenceReferences {
		ref.Source = "authentication_results"
		ref.Value = strings.ToLower(check.Status)
		refs = append(refs, ref)
	}
	return refs
}

func indicatorReferences(kind string, values []string) []domain.EvidenceReference {
	refs := make([]domain.EvidenceReference, 0, len(values))
	for _, value := range values {
		refs = append(refs, domain.EvidenceReference{Source: kind, Value: value})
	}
	return refs
}

func isExecutableAttachment(filename string) bool {
	name := strings.ToLower(filepath.Base(filename))
	extension := filepath.Ext(name)
	for _, executable := range []string{".exe", ".scr", ".js", ".jse", ".vbs", ".vbe", ".bat", ".cmd", ".com", ".msi", ".dll", ".ps1", ".jar"} {
		if extension == executable || strings.Contains(name, executable+".") {
			return true
		}
	}
	return false
}

func normalizedAIClassification(assessment domain.AIAssessment) (string, bool) {
	if (assessment.Status != "completed" && assessment.Status != "partial") || assessment.Classification == nil || assessment.Confidence == nil || *assessment.Confidence < 0.5 || *assessment.Confidence > 1 {
		return "", false
	}
	value := strings.ToLower(strings.TrimSpace(*assessment.Classification))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value, true
}

func verdictFor(score int, classification string, aiAccepted, executable bool, deterministicCount int, indicators domain.Indicators, auth domain.AuthenticationResults) string {
	if executable || (aiAccepted && classification == "malware") {
		return "malware"
	}
	if aiAccepted && (classification == "phishing" || classification == "credential_harvesting") && (len(indicators.URLs) > 0 || authenticationFailure(auth)) {
		return "phishing"
	}
	if aiAccepted && (classification == "fraud" || classification == "payment_manipulation") && deterministicCount > 0 {
		return "fraud"
	}
	if deterministicCount > 0 && score >= 25 {
		return "suspicious"
	}
	if aiAccepted && classification == "benign" && deterministicCount == 0 {
		return "benign"
	}
	return "unknown"
}

func authenticationFailure(auth domain.AuthenticationResults) bool {
	return auth.SPF.Status == "fail" || auth.DKIM.Status == "fail" || auth.DMARC.Status == "fail" || auth.SPF.Status == "unknown" || auth.DKIM.Status == "unknown" || auth.DMARC.Status == "unknown"
}

func confidenceFor(assessment domain.RiskAssessment, aiAssessment domain.AIAssessment, aiAccepted bool, deterministicCount int) *float64 {
	if deterministicCount == 0 && !aiAccepted {
		return nil
	}
	deterministic := math.Min(0.9, 0.25+0.15*float64(deterministicCount))
	confidence := deterministic
	if deterministicCount == 0 && aiAccepted {
		confidence = *aiAssessment.Confidence
	} else if aiAccepted {
		confidence = (deterministic + *aiAssessment.Confidence) / 2
	}
	confidence = math.Min(1, math.Max(0, confidence))
	return &confidence
}

func levelFor(score int) string {
	switch {
	case score >= 75:
		return "critical"
	case score >= 50:
		return "high"
	case score >= 25:
		return "medium"
	default:
		return "low"
	}
}

func clamp(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func appendUnique(values []domain.EvidenceReference, additions ...domain.EvidenceReference) []domain.EvidenceReference {
	for _, addition := range additions {
		duplicate := false
		for _, existing := range values {
			if existing == addition {
				duplicate = true
				break
			}
		}
		if !duplicate {
			values = append(values, addition)
		}
	}
	return values
}
