package forensics

import (
	"net/mail"
	"net/netip"
	"regexp"
	"strings"

	"sih26106/backend/internal/domain"
)

var (
	authResultPattern = regexp.MustCompile(`(?i)\b(spf|dkim|dmarc)\s*=\s*([a-z]+)\b`)
	fromHostPattern   = regexp.MustCompile(`(?i)\bfrom\s+([^\s();]+)`)
	tokenPattern      = regexp.MustCompile(`[0-9A-Fa-f:.]+`)
)

// AnalyzeHeaders derives only deterministic observations from parsed headers.
// It performs no DNS, reputation, or other external lookups.
func AnalyzeHeaders(headers []domain.Header) (domain.AuthenticationResults, []domain.ReceivedRelay) {
	auth := domain.AuthenticationResults{
		SPF:   missingAuthentication("SPF"),
		DKIM:  missingAuthentication("DKIM"),
		DMARC: missingAuthentication("DMARC"),
	}
	received := make([]domain.ReceivedRelay, 0)
	receivedHeaders := make([]domain.Header, 0)
	for _, header := range headers {
		if strings.EqualFold(header.Name, "Authentication-Results") {
			applyAuthentication(&auth, header)
		}
		if strings.EqualFold(header.Name, "Received") {
			receivedHeaders = append(receivedHeaders, header)
		}
	}
	for index, header := range receivedHeaders {
		received = append(received, parseReceived(header, len(receivedHeaders)-index))
	}
	return auth, received
}

func missingAuthentication(name string) domain.AuthenticationCheck {
	return domain.AuthenticationCheck{
		Status:             "none",
		EvidenceReferences: []domain.EvidenceReference{},
		Explanation:        "No Authentication-Results header reported " + strings.ToLower(name) + ".",
	}
}

func applyAuthentication(results *domain.AuthenticationResults, header domain.Header) {
	found := map[string]bool{}
	for _, match := range authResultPattern.FindAllStringSubmatch(header.Value, -1) {
		name := strings.ToLower(match[1])
		check := checkFor(results, name)
		status := normalizeStatus(match[2])
		check.Status = status
		check.Explanation = "Authentication-Results header reports " + name + "=" + strings.ToLower(match[2]) + "."
		check.EvidenceReferences = append(check.EvidenceReferences, domain.EvidenceReference{HeaderOrder: header.Order, HeaderName: header.Name})
		found[name] = true
	}
	for _, name := range []string{"spf", "dkim", "dmarc"} {
		if !found[name] && checkFor(results, name).Status == "none" {
			check := checkFor(results, name)
			check.Status = "unknown"
			check.Explanation = "Authentication-Results header was present but did not report " + strings.ToUpper(name) + "."
			check.EvidenceReferences = []domain.EvidenceReference{{HeaderOrder: header.Order, HeaderName: header.Name}}
		}
	}
}

func checkFor(results *domain.AuthenticationResults, name string) *domain.AuthenticationCheck {
	switch name {
	case "spf":
		return &results.SPF
	case "dkim":
		return &results.DKIM
	default:
		return &results.DMARC
	}
}

func normalizeStatus(value string) string {
	switch strings.ToLower(value) {
	case "pass", "fail", "neutral", "none":
		return strings.ToLower(value)
	default:
		return "unknown"
	}
}

func parseReceived(header domain.Header, sequence int) domain.ReceivedRelay {
	relay := domain.ReceivedRelay{
		Sequence:          sequence,
		SourceHeaderOrder: header.Order,
		Confidence:        "low",
		Provenance:        "inferred",
	}
	if match := fromHostPattern.FindStringSubmatch(header.Value); len(match) == 2 {
		host := strings.Trim(match[1], "[]")
		if host != "" {
			relay.Hostname = &host
		}
	}
	for _, candidate := range tokenPattern.FindAllString(header.Value, -1) {
		address, err := netip.ParseAddr(strings.Trim(candidate, "[]"))
		if err == nil {
			value := address.String()
			relay.IPAddress = &value
			break
		}
	}
	if separator := strings.LastIndex(header.Value, ";"); separator >= 0 {
		if parsed, err := mail.ParseDate(strings.TrimSpace(header.Value[separator+1:])); err == nil {
			value := parsed.UTC().Format("2006-01-02T15:04:05Z07:00")
			relay.Timestamp = &value
		}
	}
	if relay.Hostname != nil || relay.IPAddress != nil {
		relay.Confidence = "medium"
	}
	return relay
}
