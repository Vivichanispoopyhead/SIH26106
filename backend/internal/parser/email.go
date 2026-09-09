package parser

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/netip"
	"net/textproto"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"sih26106/backend/internal/domain"
)

var (
	urlPattern    = regexp.MustCompile(`(?i)https?://[^\s<>"']+`)
	domainPattern = regexp.MustCompile(`(?i)(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+(?:[a-z]{2,63})`)
	ipPattern     = regexp.MustCompile(`[0-9a-fA-F:.]+`)
)

// Parse reads an RFC 5322 message without executing or retrieving any content.
func Parse(raw []byte) (*domain.ParsedEmail, error) {
	headers, err := parseHeaders(raw)
	if err != nil {
		return nil, err
	}
	message, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("read message: %w", err)
	}

	result := &domain.ParsedEmail{
		Message:     domain.MessageMetadata{From: []string{}, To: []string{}, CC: []string{}, ReplyTo: []string{}},
		Headers:     headers,
		Indicators:  domain.Indicators{IPs: []string{}, Domains: []string{}, URLs: []string{}},
		Attachments: []domain.Attachment{},
	}
	populateMetadata(&result.Message, message.Header)
	contentType := message.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if contentType == "" {
		mediaType = "text/plain"
	} else if err != nil {
		return nil, fmt.Errorf("invalid content type: %w", err)
	}
	result.MIME.ContentType = strings.ToLower(mediaType)

	text, plainText, attachments, plain, html, err := inspectPart(message.Header, message.Body, mediaType, params)
	if err != nil {
		return nil, err
	}
	result.MIME.HasPlainText, result.MIME.HasHTML = plain, html
	result.Attachments = attachments
	result.PlainTextBody = strings.TrimSpace(plainText.String())
	result.MIME.AttachmentCount = len(attachments)
	indicatorText := strings.Builder{}
	for _, header := range headers {
		indicatorText.WriteString(header.Value)
		indicatorText.WriteByte('\n')
	}
	indicatorText.WriteString(text.String())
	result.Indicators = extractIndicators(indicatorText.String())
	return result, nil
}

func parseHeaders(raw []byte) ([]domain.Header, error) {
	separator := bytes.Index(raw, []byte("\r\n\r\n"))
	lineEnd := "\r\n"
	if separator < 0 {
		separator = bytes.Index(raw, []byte("\n\n"))
		lineEnd = "\n"
	}
	if separator < 0 {
		return nil, errors.New("message has no header/body separator")
	}
	block := string(raw[:separator])
	lines := strings.Split(block, lineEnd)
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return nil, errors.New("message has no headers")
	}
	headers := make([]domain.Header, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			if len(headers) == 0 {
				return nil, errors.New("header continuation without header")
			}
			headers[len(headers)-1].Value += " " + strings.TrimSpace(line)
			continue
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(name) == "" {
			return nil, errors.New("malformed header")
		}
		headers = append(headers, domain.Header{Name: name, Value: strings.TrimSpace(value), Order: len(headers) + 1})
	}
	return headers, nil
}

func populateMetadata(target *domain.MessageMetadata, header mail.Header) {
	target.MessageID = optional(header.Get("Message-ID"))
	target.Subject = optional(header.Get("Subject"))
	target.ReturnPath = returnPath(header.Get("Return-Path"))
	target.From = addresses(headerValues(header, "From"))
	target.To = addresses(headerValues(header, "To"))
	target.CC = addresses(headerValues(header, "Cc"))
	target.ReplyTo = addresses(headerValues(header, "Reply-To"))
	if date := strings.TrimSpace(header.Get("Date")); date != "" {
		if parsed, err := mail.ParseDate(date); err == nil {
			formatted := parsed.UTC().Format("2006-01-02T15:04:05Z07:00")
			target.Date = &formatted
		} else {
			target.Date = &date
		}
	}
}

func returnPath(value string) *string {
	if address, err := mail.ParseAddress(value); err == nil {
		return &address.Address
	}
	return optional(value)
}

func headerValues(header mail.Header, name string) []string {
	for key, values := range header {
		if strings.EqualFold(key, name) {
			return values
		}
	}
	return nil
}

func optional(value string) *string {
	if value = strings.TrimSpace(value); value != "" {
		return &value
	}
	return nil
}

func addresses(values []string) []string {
	result := make([]string, 0)
	for _, value := range values {
		parsed, err := mail.ParseAddressList(value)
		if err != nil {
			if value = strings.TrimSpace(value); value != "" {
				result = append(result, value)
			}
			continue
		}
		for _, address := range parsed {
			result = append(result, address.Address)
		}
	}
	return result
}

func inspectPart(header mail.Header, body io.Reader, mediaType string, params map[string]string) (*strings.Builder, *strings.Builder, []domain.Attachment, bool, bool, error) {
	text := &strings.Builder{}
	plainText := &strings.Builder{}
	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return nil, nil, nil, false, false, errors.New("multipart message missing boundary")
		}
		reader := multipart.NewReader(body, boundary)
		attachments := []domain.Attachment{}
		plain, html := false, false
		for {
			part, err := reader.NextPart()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return nil, nil, nil, false, false, fmt.Errorf("read MIME part: %w", err)
			}
			partType, partParams, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
			if part.Header.Get("Content-Type") == "" {
				partType = "text/plain"
			} else if err != nil {
				return nil, nil, nil, false, false, fmt.Errorf("invalid MIME part content type: %w", err)
			}
			if filename, attachment := attachmentName(part.Header); attachment {
				item, err := readAttachment(filename, partType, part.Header.Get("Content-Transfer-Encoding"), part)
				if err != nil {
					return nil, nil, nil, false, false, err
				}
				attachments = append(attachments, item)
				continue
			}
			childText, childPlainText, childAttachments, childPlain, childHTML, err := inspectPart(mail.Header(part.Header), part, strings.ToLower(partType), partParams)
			if err != nil {
				return nil, nil, nil, false, false, err
			}
			text.WriteString(childText.String())
			plainText.WriteString(childPlainText.String())
			attachments = append(attachments, childAttachments...)
			plain = plain || childPlain
			html = html || childHTML
		}
		return text, plainText, attachments, plain, html, nil
	}
	decoded, err := decodedReader(body, header.Get("Content-Transfer-Encoding"))
	if err != nil {
		return nil, nil, nil, false, false, err
	}
	if mediaType == "text/plain" || mediaType == "text/html" {
		if _, err := io.Copy(text, io.LimitReader(decoded, 50<<20)); err != nil {
			return nil, nil, nil, false, false, err
		}
		if mediaType == "text/plain" {
			plainText.WriteString(text.String())
		}
	}
	return text, plainText, []domain.Attachment{}, mediaType == "text/plain", mediaType == "text/html", nil
}

func attachmentName(header textproto.MIMEHeader) (string, bool) {
	disposition, params, err := mime.ParseMediaType(header.Get("Content-Disposition"))
	if err == nil && (strings.EqualFold(disposition, "attachment") || params["filename"] != "") {
		return params["filename"], true
	}
	_, params, err = mime.ParseMediaType(header.Get("Content-Type"))
	if err == nil && params["name"] != "" {
		return params["name"], true
	}
	return "", false
}

func readAttachment(filename, mediaType, transferEncoding string, body io.Reader) (domain.Attachment, error) {
	decoded, err := decodedReader(body, transferEncoding)
	if err != nil {
		return domain.Attachment{}, err
	}
	hash := sha256.New()
	size, err := io.Copy(hash, io.LimitReader(decoded, 50<<20+1))
	if err != nil {
		return domain.Attachment{}, err
	}
	if size > 50<<20 {
		return domain.Attachment{}, errors.New("attachment exceeds safe parsing limit")
	}
	return domain.Attachment{Filename: filename, MIMEType: strings.ToLower(mediaType), SizeBytes: size, SHA256: fmt.Sprintf("%x", hash.Sum(nil))}, nil
}

func decodedReader(reader io.Reader, transferEncoding string) (io.Reader, error) {
	switch strings.ToLower(strings.TrimSpace(transferEncoding)) {
	case "", "7bit", "8bit", "binary":
		return reader, nil
	case "base64":
		return base64.NewDecoder(base64.StdEncoding, reader), nil
	case "quoted-printable":
		return quotedprintable.NewReader(reader), nil
	default:
		return nil, fmt.Errorf("unsupported content transfer encoding %q", transferEncoding)
	}
}

func extractIndicators(content string) domain.Indicators {
	ips, domains, urls := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
	for _, candidate := range urlPattern.FindAllString(content, -1) {
		candidate = strings.TrimRight(candidate, ".,;:!?)]}\"")
		if parsed, err := url.Parse(candidate); err == nil && parsed.Hostname() != "" {
			urls[candidate] = struct{}{}
			domains[strings.ToLower(parsed.Hostname())] = struct{}{}
		}
	}
	for _, candidate := range domainPattern.FindAllString(content, -1) {
		domains[strings.ToLower(candidate)] = struct{}{}
	}
	for _, candidate := range ipPattern.FindAllString(content, -1) {
		candidate = strings.Trim(candidate, "[]()<>.,;:")
		if parsed, err := netip.ParseAddr(candidate); err == nil {
			ips[parsed.String()] = struct{}{}
		}
	}
	return domain.Indicators{IPs: sorted(ips), Domains: sorted(domains), URLs: sorted(urls)}
}

func sorted(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
