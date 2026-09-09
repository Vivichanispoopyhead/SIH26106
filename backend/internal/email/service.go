package email

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"sih26106/backend/internal/domain"
	"sih26106/backend/internal/parser"
	"sih26106/backend/internal/persistence"
)

const MaxUploadSize int64 = 50 << 20

var (
	ErrMissingFile     = errors.New("missing file")
	ErrEmptyFile       = errors.New("empty file")
	ErrUnsupportedType = errors.New("unsupported file type")
	ErrTooLarge        = errors.New("file too large")
	ErrParseFailed     = errors.New("email parse failed")
)

type Service struct{ store persistence.Store }

func NewService(store persistence.Store) *Service { return &Service{store: store} }

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
	if err = s.store.UpdateAnalysis(ctx, analysis.ID, "completed", ""); err != nil {
		return nil, err
	}
	return analysis, nil
}
