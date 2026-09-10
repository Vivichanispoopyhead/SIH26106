package email

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"sih26106/backend/internal/ai"
	"sih26106/backend/internal/domain"
	"sih26106/backend/internal/forensics"
	"sih26106/backend/internal/parser"
	"sih26106/backend/internal/persistence"
	"sih26106/backend/internal/risk"
)

const MaxUploadSize int64 = 50 << 20

var (
	ErrMissingFile     = errors.New("missing file")
	ErrEmptyFile       = errors.New("empty file")
	ErrUnsupportedType = errors.New("unsupported file type")
	ErrTooLarge        = errors.New("file too large")
	ErrParseFailed     = errors.New("email parse failed")
)

type Service struct {
	store    persistence.Store
	analyzer ai.Analyzer
}

func NewService(store persistence.Store) *Service {
	return NewServiceWithAnalyzer(store, ai.UnavailableAnalyzer{})
}

func NewServiceWithAnalyzer(store persistence.Store, analyzer ai.Analyzer) *Service {
	return &Service{store: store, analyzer: analyzer}
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
	auth, received := forensics.AnalyzeHeaders(parsed.Headers)
	result := &domain.AnalysisResult{
		AnalysisID:     analysis.ID,
		EmailID:        email.ID,
		CaseID:         email.CaseID,
		Status:         "completed",
		AIAssessment:   assessment,
		Authentication: auth,
		ReceivedChain:  received,
	}
	if analyzerErr != nil {
		result.Status = "partial"
		result.AIAssessment = domain.AIAssessment{
			Status:             "failed",
			SupportingSignals:  []string{},
			EvidenceReferences: []string{},
			Failure:            &domain.Failure{Code: "AI_ANALYSIS_FAILED", Message: analyzerErr.Error()},
		}
		result.Failure = result.AIAssessment.Failure
	}
	result.Risk = risk.Evaluate(*result, parsed.Indicators, parsed.Attachments)
	if err = s.store.SaveAnalysisResult(ctx, analysis.ID, result); err != nil {
		return nil, err
	}
	if err = s.store.UpdateAnalysis(ctx, analysis.ID, result.Status, failureMessage(result.Failure)); err != nil {
		return nil, err
	}
	return analysis, nil
}

func failureMessage(failure *domain.Failure) string {
	if failure == nil {
		return ""
	}
	return failure.Message
}
