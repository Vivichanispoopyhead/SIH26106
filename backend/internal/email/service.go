package email

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"sih26106/backend/internal/ai"
	"sih26106/backend/internal/domain"
	"sih26106/backend/internal/enrichment"
	evidencegen "sih26106/backend/internal/evidence"
	"sih26106/backend/internal/forensics"
	"sih26106/backend/internal/graph"
	"sih26106/backend/internal/parser"
	"sih26106/backend/internal/persistence"
	"sih26106/backend/internal/risk"
)

const MaxUploadSize int64 = 50 << 20

var (
	ErrMissingFile       = errors.New("missing file")
	ErrEmptyFile         = errors.New("empty file")
	ErrUnsupportedType   = errors.New("unsupported file type")
	ErrTooLarge          = errors.New("file too large")
	ErrParseFailed       = errors.New("email parse failed")
	ErrGraphNotAvailable = errors.New("graph not available")
)

type Service struct {
	store    persistence.Store
	analyzer ai.Analyzer
	enricher enrichment.IPEnricher
}

func NewService(store persistence.Store) *Service {
	return NewServiceWithAnalyzer(store, ai.UnavailableAnalyzer{})
}

func NewServiceWithAnalyzer(store persistence.Store, analyzer ai.Analyzer) *Service {
	return NewServiceWithAnalyzerAndEnricher(store, analyzer, enrichment.UnavailableEnricher{})
}

func NewServiceWithAnalyzerAndEnricher(store persistence.Store, analyzer ai.Analyzer, enricher enrichment.IPEnricher) *Service {
	analyzer = ai.NewGuardedAnalyzer(analyzer, ai.DefaultInputPolicy())
	if enricher == nil {
		enricher = enrichment.UnavailableEnricher{}
	}
	return &Service{store: store, analyzer: analyzer, enricher: enricher}
}

func (s *Service) Upload(ctx context.Context, filename string, raw []byte) (*domain.Email, error) {
	if filename == "" {
		return nil, ErrMissingFile
	}
	if !strings.EqualFold(filepath.Ext(filename), ".eml") {
		return nil, ErrUnsupportedType
	}
	if len(raw) == 0 {
		return nil, ErrEmptyFile
	}
	if int64(len(raw)) > MaxUploadSize {
		return nil, ErrTooLarge
	}
	return s.store.CreateUpload(ctx, filepath.Base(filename), raw)
}

func (s *Service) Get(ctx context.Context, emailID string) (*domain.Email, error) {
	return s.store.GetEmail(ctx, emailID)
}

func (s *Service) GetAnalysis(ctx context.Context, emailID string) (*domain.AnalysisResult, error) {
	if _, err := s.store.GetEmail(ctx, emailID); err != nil {
		return nil, err
	}
	return s.store.GetAnalysisResult(ctx, emailID)
}

func (s *Service) GetEvidence(ctx context.Context, emailID string) (*domain.AnalysisResult, error) {
	return s.GetAnalysis(ctx, emailID)
}

func (s *Service) GetGraph(ctx context.Context, caseID string) (*domain.Graph, error) {
	emails, err := s.store.GetCaseEmails(ctx, caseID)
	if err != nil {
		return nil, err
	}
	inputs := make([]graph.EmailAnalysis, 0, len(emails))
	for _, email := range emails {
		if email.Parsed == nil {
			continue
		}
		result, err := s.store.GetAnalysisResult(ctx, email.ID)
		if errors.Is(err, persistence.ErrAnalysisNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if result.Status != "completed" && result.Status != "partial" {
			continue
		}
		inputs = append(inputs, graph.EmailAnalysis{Email: email, Parsed: email.Parsed, Result: result})
	}
	if len(inputs) == 0 {
		return nil, ErrGraphNotAvailable
	}
	return graph.Build(caseID, inputs), nil
}

func (s *Service) StartAnalysis(ctx context.Context, emailID string) (*domain.Analysis, error) {
	analysis, err := s.store.CreateAnalysis(ctx, emailID)
	if err != nil {
		return nil, err
	}
	if err = s.store.UpdateAnalysis(ctx, analysis.ID, "processing", ""); err != nil {
		return nil, err
	}
	email, err := s.store.GetEmail(ctx, emailID)
	if err != nil {
		return nil, err
	}
	parsed, err := parser.Parse(email.RawContent)
	if err != nil {
		_ = s.store.SaveParseFailure(ctx, emailID, err.Error())
		_ = s.store.UpdateAnalysis(ctx, analysis.ID, "failed", err.Error())
		return nil, ErrParseFailed
	}
	if err = s.store.SaveParsed(ctx, emailID, parsed); err != nil {
		return nil, err
	}

	auth, received := forensics.AnalyzeHeaders(parsed.Headers)
	ipResults, enrichmentFailure := s.enrichIPs(ctx, parsed.Indicators, received)
	assessment, analyzerErr := s.analyzer.Assess(ctx, ai.Input{
		AnalysisID:    analysis.ID,
		EmailID:       email.ID,
		CaseID:        email.CaseID,
		Message:       parsed.Message,
		PlainTextBody: parsed.PlainTextBody,
		Headers:       parsed.Headers,
		Indicators:    parsed.Indicators,
		Attachments:   parsed.Attachments,
	})
	result := &domain.AnalysisResult{
		AnalysisID:     analysis.ID,
		EmailID:        email.ID,
		CaseID:         email.CaseID,
		Status:         "completed",
		AIAssessment:   assessment,
		Authentication: auth,
		ReceivedChain:  received,
		IPEnrichment:   ipResults,
	}
	if enrichmentFailure != nil {
		result.Status = "partial"
		result.Failure = enrichmentFailure
	}
	if analyzerErr != nil {
		result.Status = "partial"
		failure := assessment.Failure
		if failure == nil {
			failure = &domain.Failure{Code: "AI_ANALYSIS_FAILED", Message: "The AI assessment could not be completed."}
		}
		result.AIAssessment = domain.AIAssessment{
			Status:             "failed",
			SupportingSignals:  []string{},
			EvidenceReferences: []string{},
			Failure:            failure,
		}
		result.Failure = result.AIAssessment.Failure
	}
	if result.AIAssessment.Status == "failed" && result.Failure == nil {
		result.Status = "partial"
		result.Failure = &domain.Failure{Code: "AI_ANALYSIS_FAILED", Message: "The AI assessment could not be completed."}
	}
	result.Risk = risk.Evaluate(*result, parsed.Indicators, parsed.Attachments)
	result.Evidence = evidencegen.Build(email.ID, parsed, result)
	if err = s.store.SaveAnalysisResult(ctx, analysis.ID, result); err != nil {
		return nil, err
	}
	if err = s.store.UpdateAnalysis(ctx, analysis.ID, result.Status, failureMessage(result.Failure)); err != nil {
		return nil, err
	}
	return analysis, nil
}

func (s *Service) enrichIPs(ctx context.Context, indicators domain.Indicators, received []domain.ReceivedRelay) ([]domain.IPEnrichment, *domain.Failure) {
	ips := make([]string, 0, len(indicators.IPs)+len(received))
	seen := make(map[string]struct{})
	appendIP := func(value string) {
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		ips = append(ips, value)
	}
	for _, ip := range indicators.IPs {
		appendIP(ip)
	}
	for _, relay := range received {
		if relay.IPAddress != nil {
			appendIP(*relay.IPAddress)
		}
	}
	results := make([]domain.IPEnrichment, 0, len(ips))
	var firstFailure *domain.Failure
	for _, ip := range ips {
		value, err := enrichment.LookupIP(ctx, s.enricher, ip)
		if value.IPAddress == "" {
			value.IPAddress = ip
		}
		results = append(results, value)
		if value.Status == enrichment.StatusFailed {
			if firstFailure == nil {
				if value.Failure != nil {
					firstFailure = value.Failure
				} else {
					firstFailure = &domain.Failure{Code: "IP_ENRICHMENT_FAILED", Message: "IP enrichment was unavailable."}
				}
			}
		}
		if err != nil && firstFailure == nil {
			firstFailure = &domain.Failure{Code: "IP_ENRICHMENT_FAILED", Message: "IP enrichment was unavailable."}
		}
	}
	return results, firstFailure
}

func failureMessage(failure *domain.Failure) string {
	if failure == nil {
		return ""
	}
	return failure.Message
}
