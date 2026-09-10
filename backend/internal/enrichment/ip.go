package enrichment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"sih26106/backend/internal/domain"
)

const (
	StatusEnriched      = "enriched"
	StatusNotApplicable = "not_applicable"
	StatusFailed        = "failed"
	StatusNotConfigured = "not_configured"
	ProvenanceEnriched  = "ENRICHED"
	ProvenanceObserved  = "OBSERVED"
	ProvenanceInferred  = "INFERRED"
	defaultTimeout      = 5 * time.Second
	defaultMaxResponse  = 64 << 10
)

var errProviderNotSet = errors.New("no IP enrichment provider is configured")

type IPEnricher interface {
	Lookup(context.Context, string) (domain.IPEnrichment, error)
}

type UnavailableEnricher struct{}

func (UnavailableEnricher) Lookup(_ context.Context, ip string) (domain.IPEnrichment, error) {
	result, ok := classify(ip)
	if !ok {
		return result, nil
	}
	result.Status = StatusNotConfigured
	result.Provenance = ProvenanceInferred
	result.Failure = failure("IP_ENRICHMENT_NOT_CONFIGURED", errProviderNotSet.Error())
	return result, nil
}

// LookupIP classifies addresses before any provider call.
func LookupIP(ctx context.Context, enricher IPEnricher, ip string) (domain.IPEnrichment, error) {
	result, ok := classify(ip)
	if !ok {
		return result, nil
	}
	if enricher == nil {
		enricher = UnavailableEnricher{}
	}
	return enricher.Lookup(ctx, result.IPAddress)
}

func classify(value string) (domain.IPEnrichment, bool) {
	result := domain.IPEnrichment{IPAddress: value, Provenance: ProvenanceObserved}
	address, err := netip.ParseAddr(strings.TrimSpace(value))
	if err != nil {
		result.Status = StatusNotApplicable
		result.Failure = failure("INVALID_IP", "The observed value is not a valid IP address.")
		return result, false
	}
	address = address.Unmap()
	result.IPAddress = address.String()
	if isNonPublic(address) {
		result.Status = StatusNotApplicable
		return result, false
	}
	return result, true
}

func isNonPublic(address netip.Addr) bool {
	if address.IsLoopback() || address.IsPrivate() || address.IsLinkLocalUnicast() || address.IsMulticast() || address.IsUnspecified() {
		return true
	}
	for _, prefix := range documentationPrefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

var documentationPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"), netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("100.64.0.0/10"),
}

func failure(code, message string) *domain.Failure {
	return &domain.Failure{Code: code, Message: message}
}

type HTTPConfig struct {
	APIKey          string
	Endpoint        string
	Provider        string
	Timeout         time.Duration
	MaxResponseSize int64
	HTTPClient      *http.Client
}

func HTTPConfigFromEnv() (HTTPConfig, error) {
	timeout := defaultTimeout
	if value := strings.TrimSpace(os.Getenv("IP_ENRICHMENT_TIMEOUT")); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed <= 0 {
			return HTTPConfig{}, errors.New("IP_ENRICHMENT_TIMEOUT must be a positive duration")
		}
		timeout = parsed
	}
	maxResponse := int64(defaultMaxResponse)
	if value := strings.TrimSpace(os.Getenv("IP_ENRICHMENT_MAX_RESPONSE_BYTES")); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed <= 0 {
			return HTTPConfig{}, errors.New("IP_ENRICHMENT_MAX_RESPONSE_BYTES must be a positive integer")
		}
		maxResponse = parsed
	}
	provider := strings.TrimSpace(os.Getenv("IP_ENRICHMENT_PROVIDER"))
	if provider == "" {
		provider = "http"
	}
	return HTTPConfig{APIKey: strings.TrimSpace(os.Getenv("IP_ENRICHMENT_API_KEY")), Endpoint: strings.TrimSpace(os.Getenv("IP_ENRICHMENT_API_URL")), Provider: provider, Timeout: timeout, MaxResponseSize: maxResponse}, nil
}

func NewHTTPEnricherFromEnv() (IPEnricher, error) {
	config, err := HTTPConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return NewHTTPEnricher(config)
}

func NewHTTPEnricher(config HTTPConfig) (IPEnricher, error) {
	if strings.TrimSpace(config.APIKey) == "" {
		return UnavailableEnricher{}, nil
	}
	if config.Timeout <= 0 {
		return nil, errors.New("IP enrichment timeout must be positive")
	}
	if config.MaxResponseSize <= 0 {
		return nil, errors.New("IP enrichment maximum response size must be positive")
	}
	endpoint, err := url.Parse(config.Endpoint)
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") {
		return nil, errors.New("IP enrichment API URL must be an absolute HTTP URL")
	}
	if strings.TrimSpace(config.Provider) == "" {
		config.Provider = "http"
	}
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: config.Timeout}
	}
	return &httpEnricher{config: config, client: client}, nil
}

type httpEnricher struct {
	config HTTPConfig
	client *http.Client
}

func (e *httpEnricher) Lookup(ctx context.Context, ip string) (domain.IPEnrichment, error) {
	classified, allowed := classify(ip)
	if !allowed {
		return classified, nil
	}
	endpoint, _ := url.Parse(e.config.Endpoint)
	query := endpoint.Query()
	query.Set("ip", classified.IPAddress)
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return failedResult(classified.IPAddress, e.config.Provider, "IP_ENRICHMENT_REQUEST_FAILED", "The IP enrichment request could not be created."), err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+e.config.APIKey)
	response, err := e.client.Do(request)
	if err != nil {
		return failedResult(classified.IPAddress, e.config.Provider, "IP_ENRICHMENT_REQUEST_FAILED", "The IP enrichment provider request failed."), err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, e.config.MaxResponseSize+1))
	if err != nil {
		return failedResult(classified.IPAddress, e.config.Provider, "IP_ENRICHMENT_RESPONSE_FAILED", "The IP enrichment response could not be read."), err
	}
	if int64(len(body)) > e.config.MaxResponseSize {
		return failedResult(classified.IPAddress, e.config.Provider, "IP_ENRICHMENT_RESPONSE_TOO_LARGE", "The IP enrichment response was too large."), errors.New("IP enrichment response exceeded configured limit")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return failedResult(classified.IPAddress, e.config.Provider, "IP_ENRICHMENT_HTTP_ERROR", "The IP enrichment provider returned an error."), fmt.Errorf("IP enrichment provider returned HTTP %d", response.StatusCode)
	}
	var payload providerResponse
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return failedResult(classified.IPAddress, e.config.Provider, "IP_ENRICHMENT_MALFORMED_RESPONSE", "The IP enrichment provider returned malformed data."), err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return failedResult(classified.IPAddress, e.config.Provider, "IP_ENRICHMENT_MALFORMED_RESPONSE", "The IP enrichment provider returned malformed data."), errors.New("IP enrichment response contains trailing data")
	}
	if err := payload.validate(classified.IPAddress); err != nil {
		return failedResult(classified.IPAddress, e.config.Provider, "IP_ENRICHMENT_INVALID_RESPONSE", "The IP enrichment provider returned invalid data."), err
	}
	now := time.Now().UTC()
	provider := e.config.Provider
	return domain.IPEnrichment{IPAddress: classified.IPAddress, Status: StatusEnriched, Country: payload.Country, Region: payload.Region, City: payload.City, Latitude: payload.Latitude, Longitude: payload.Longitude, ASN: payload.ASN, Organization: payload.Organization, Provider: &provider, Confidence: payload.Confidence, RetrievedAt: &now, Provenance: ProvenanceEnriched}, nil
}

type providerResponse struct {
	IPAddress    string   `json:"ip_address"`
	Country      *string  `json:"country"`
	Region       *string  `json:"region"`
	City         *string  `json:"city"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	ASN          *string  `json:"asn"`
	Organization *string  `json:"organization"`
	Confidence   *float64 `json:"confidence"`
}

func (p providerResponse) validate(requestedIP string) error {
	if p.IPAddress == "" && p.Country == nil && p.Region == nil && p.City == nil && p.Latitude == nil && p.Longitude == nil && p.ASN == nil && p.Organization == nil && p.Confidence == nil {
		return errors.New("provider response contains no enrichment data")
	}
	if p.IPAddress != "" {
		address, err := netip.ParseAddr(p.IPAddress)
		if err != nil || address.String() != requestedIP {
			return errors.New("provider IP does not match requested IP")
		}
	}
	for _, value := range []*string{p.Country, p.Region, p.City, p.ASN, p.Organization} {
		if value != nil && len(*value) > 256 {
			return errors.New("provider field is too long")
		}
	}
	if p.Latitude != nil && (*p.Latitude < -90 || *p.Latitude > 90) {
		return errors.New("provider latitude is invalid")
	}
	if p.Longitude != nil && (*p.Longitude < -180 || *p.Longitude > 180) {
		return errors.New("provider longitude is invalid")
	}
	if p.Confidence != nil && (*p.Confidence < 0 || *p.Confidence > 1) {
		return errors.New("provider confidence is invalid")
	}
	return nil
}

func failedResult(ip, provider, code, message string) domain.IPEnrichment {
	providerValue := provider
	return domain.IPEnrichment{IPAddress: ip, Status: StatusFailed, Provider: &providerValue, Provenance: ProvenanceInferred, Failure: failure(code, message)}
}
