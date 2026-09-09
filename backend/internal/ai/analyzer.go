package ai

import (
	"context"

	"sih26106/backend/internal/domain"
)

// Analyzer is the adapter seam for an AI/ML implementation. Its input contains
// normalized observed data only; callers never provide raw attachments or cause
// the analyzer to execute content or retrieve URLs.
type Analyzer interface {
	Assess(context.Context, Input) (domain.AIAssessment, error)
}

type Input struct {
	AnalysisID    string
	EmailID       string
	CaseID        string
	Message       domain.MessageMetadata
	PlainTextBody string
	Headers       []domain.Header
	Indicators    domain.Indicators
	Attachments   []domain.Attachment
}

// UnavailableAnalyzer is the deterministic default until an explicit adapter
// is configured by the application.
type UnavailableAnalyzer struct{}

func (UnavailableAnalyzer) Assess(_ context.Context, _ Input) (domain.AIAssessment, error) {
	return domain.AIAssessment{
		Status:             "not_available",
		Classification:     nil,
		Confidence:         nil,
		SupportingSignals:  []string{},
		EvidenceReferences: []string{},
		Provider:           nil,
		Model:              nil,
		Failure: &domain.Failure{
			Code:    "AI_NOT_CONFIGURED",
			Message: "No AI analyzer is configured.",
		},
	}, nil
}
