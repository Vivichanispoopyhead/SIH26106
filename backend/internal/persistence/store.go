package persistence

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sih26106/backend/internal/domain"
)

var (
	ErrEmailNotFound    = errors.New("email not found")
	ErrAnalysisNotFound = errors.New("analysis not found")
)

type Store interface {
	CreateUpload(context.Context, string, []byte) (*domain.Email, error)
	GetEmail(context.Context, string) (*domain.Email, error)
	CreateAnalysis(context.Context, string) (*domain.Analysis, error)
	UpdateAnalysis(context.Context, string, string, string) error
	SaveAnalysisResult(context.Context, string, *domain.AnalysisResult) error
	GetAnalysisResult(context.Context, string) (*domain.AnalysisResult, error)
	SaveParsed(context.Context, string, *domain.ParsedEmail) error
	SaveParseFailure(context.Context, string, string) error
}

type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	store := &PostgresStore{pool: pool}
	if err := store.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}
func (s *PostgresStore) Close() { s.pool.Close() }

func (s *PostgresStore) migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS cases (id TEXT PRIMARY KEY, status TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS emails (id TEXT PRIMARY KEY, case_id TEXT NOT NULL REFERENCES cases(id), filename TEXT NOT NULL, raw_content BYTEA NOT NULL, status TEXT NOT NULL, parse_failure TEXT, parsed JSONB, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS analyses (id TEXT PRIMARY KEY, email_id TEXT NOT NULL REFERENCES emails(id), case_id TEXT NOT NULL REFERENCES cases(id), status TEXT NOT NULL, failure TEXT, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL)`,
		`ALTER TABLE analyses ADD COLUMN IF NOT EXISTS result JSONB`,
		`CREATE TABLE IF NOT EXISTS audit_events (id TEXT PRIMARY KEY, case_id TEXT NOT NULL REFERENCES cases(id), event_type TEXT NOT NULL, timestamp TIMESTAMPTZ NOT NULL, actor TEXT NOT NULL, payload_hash TEXT NOT NULL, previous_hash TEXT, event_hash TEXT NOT NULL)`,
	}
	for _, statement := range statements {
		if _, err := s.pool.Exec(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostgresStore) CreateUpload(ctx context.Context, filename string, raw []byte) (*domain.Email, error) {
	now := time.Now().UTC()
	caseID, emailID := newID("case"), newID("email")
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO cases (id,status,created_at,updated_at) VALUES ($1,'created',$2,$2)`, caseID, now); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO emails (id,case_id,filename,raw_content,status,created_at,updated_at) VALUES ($1,$2,$3,$4,'uploaded',$5,$5)`, emailID, caseID, filename, raw, now); err != nil {
		return nil, err
	}
	if err = insertAudit(ctx, tx, caseID, "email_uploaded", now, raw); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &domain.Email{ID: emailID, CaseID: caseID, Filename: filename, RawContent: append([]byte(nil), raw...), Status: "uploaded", CreatedAt: now, UpdatedAt: now}, nil
}

func (s *PostgresStore) GetEmail(ctx context.Context, id string) (*domain.Email, error) {
	row := s.pool.QueryRow(ctx, `SELECT id,case_id,filename,raw_content,status,COALESCE(parse_failure,''),parsed,created_at,updated_at FROM emails WHERE id=$1`, id)
	var email domain.Email
	var parsed []byte
	if err := row.Scan(&email.ID, &email.CaseID, &email.Filename, &email.RawContent, &email.Status, &email.ParseFailure, &parsed, &email.CreatedAt, &email.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEmailNotFound
		}
		return nil, err
	}
	if len(parsed) > 0 {
		var value domain.ParsedEmail
		if err := json.Unmarshal(parsed, &value); err != nil {
			return nil, err
		}
		email.Parsed = &value
	}
	return &email, nil
}

func (s *PostgresStore) CreateAnalysis(ctx context.Context, emailID string) (*domain.Analysis, error) {
	email, err := s.GetEmail(ctx, emailID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	analysis := &domain.Analysis{ID: newID("analysis"), EmailID: email.ID, CaseID: email.CaseID, Status: "started", CreatedAt: now, UpdatedAt: now}
	_, err = s.pool.Exec(ctx, `INSERT INTO analyses (id,email_id,case_id,status,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$5)`, analysis.ID, analysis.EmailID, analysis.CaseID, analysis.Status, now)
	return analysis, err
}
func (s *PostgresStore) UpdateAnalysis(ctx context.Context, id, status, failure string) error {
	_, err := s.pool.Exec(ctx, `UPDATE analyses SET status=$2,failure=$3,updated_at=$4 WHERE id=$1`, id, status, failure, time.Now().UTC())
	return err
}
func (s *PostgresStore) SaveAnalysisResult(ctx context.Context, id string, result *domain.AnalysisResult) error {
	value, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `UPDATE analyses SET result=$2,status=$3,failure=$4,updated_at=$5 WHERE id=$1`, id, value, result.Status, failureMessage(result.Failure), time.Now().UTC())
	return err
}
func (s *PostgresStore) GetAnalysisResult(ctx context.Context, emailID string) (*domain.AnalysisResult, error) {
	row := s.pool.QueryRow(ctx, `SELECT result,status,COALESCE(failure,''),id,case_id FROM analyses WHERE email_id=$1 ORDER BY created_at DESC LIMIT 1`, emailID)
	var raw []byte
	var status, failure, analysisID, caseID string
	if err := row.Scan(&raw, &status, &failure, &analysisID, &caseID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAnalysisNotFound
		}
		return nil, err
	}
	if len(raw) > 0 {
		var result domain.AnalysisResult
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	return pendingResult(analysisID, emailID, caseID, status, failure), nil
}
func (s *PostgresStore) SaveParsed(ctx context.Context, id string, parsed *domain.ParsedEmail) error {
	value, err := json.Marshal(parsed)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `UPDATE emails SET status='parsed',parsed=$2,parse_failure=NULL,updated_at=$3 WHERE id=$1`, id, value, time.Now().UTC())
	return err
}
func (s *PostgresStore) SaveParseFailure(ctx context.Context, id, failure string) error {
	_, err := s.pool.Exec(ctx, `UPDATE emails SET status='failed',parse_failure=$2,updated_at=$3 WHERE id=$1`, id, failure, time.Now().UTC())
	return err
}

func insertAudit(ctx context.Context, tx pgx.Tx, caseID, event string, timestamp time.Time, payload []byte) error {
	payloadHash := fmt.Sprintf("%x", sha256.Sum256(payload))
	var previous *string
	if err := tx.QueryRow(ctx, `SELECT event_hash FROM audit_events WHERE case_id=$1 ORDER BY timestamp DESC LIMIT 1`, caseID).Scan(&previous); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	previousHash := ""
	if previous != nil {
		previousHash = *previous
	}
	eventHash := fmt.Sprintf("%x", sha256.Sum256([]byte(caseID+event+timestamp.Format(time.RFC3339Nano)+payloadHash+previousHash)))
	_, err := tx.Exec(ctx, `INSERT INTO audit_events (id,case_id,event_type,timestamp,actor,payload_hash,previous_hash,event_hash) VALUES ($1,$2,$3,$4,'system',$5,$6,$7)`, newID("audit"), caseID, event, timestamp, payloadHash, previous, eventHash)
	return err
}

type MemoryStore struct {
	mu       sync.RWMutex
	emails   map[string]*domain.Email
	analyses map[string]*domain.Analysis
	results  map[string]*domain.AnalysisResult
}

// NewMemoryStore is only for isolated HTTP and service tests. The application uses PostgreSQL.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{emails: map[string]*domain.Email{}, analyses: map[string]*domain.Analysis{}, results: map[string]*domain.AnalysisResult{}}
}
func (s *MemoryStore) CreateUpload(_ context.Context, filename string, raw []byte) (*domain.Email, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	email := &domain.Email{ID: newID("email"), CaseID: newID("case"), Filename: filename, RawContent: append([]byte(nil), raw...), Status: "uploaded", CreatedAt: now, UpdatedAt: now}
	s.emails[email.ID] = email
	return cloneEmail(email), nil
}
func (s *MemoryStore) GetEmail(_ context.Context, id string) (*domain.Email, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	email, ok := s.emails[id]
	if !ok {
		return nil, ErrEmailNotFound
	}
	return cloneEmail(email), nil
}
func (s *MemoryStore) CreateAnalysis(_ context.Context, emailID string) (*domain.Analysis, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	email, ok := s.emails[emailID]
	if !ok {
		return nil, ErrEmailNotFound
	}
	now := time.Now().UTC()
	analysis := &domain.Analysis{ID: newID("analysis"), EmailID: emailID, CaseID: email.CaseID, Status: "started", CreatedAt: now, UpdatedAt: now}
	s.analyses[analysis.ID] = analysis
	return analysis, nil
}
func (s *MemoryStore) UpdateAnalysis(_ context.Context, id, status, failure string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	analysis, ok := s.analyses[id]
	if !ok {
		return errors.New("analysis not found")
	}
	analysis.Status = status
	analysis.Failure = failure
	analysis.UpdatedAt = time.Now().UTC()
	return nil
}
func (s *MemoryStore) SaveAnalysisResult(_ context.Context, id string, result *domain.AnalysisResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.analyses[id]; !ok {
		return ErrAnalysisNotFound
	}
	s.results[id] = cloneAnalysisResult(result)
	return nil
}
func (s *MemoryStore) GetAnalysisResult(_ context.Context, emailID string) (*domain.AnalysisResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var latest *domain.Analysis
	for _, analysis := range s.analyses {
		if analysis.EmailID == emailID && (latest == nil || analysis.CreatedAt.After(latest.CreatedAt)) {
			latest = analysis
		}
	}
	if latest == nil {
		return nil, ErrAnalysisNotFound
	}
	if result, ok := s.results[latest.ID]; ok {
		return cloneAnalysisResult(result), nil
	}
	return pendingResult(latest.ID, latest.EmailID, latest.CaseID, latest.Status, latest.Failure), nil
}
func (s *MemoryStore) SaveParsed(_ context.Context, id string, parsed *domain.ParsedEmail) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	email, ok := s.emails[id]
	if !ok {
		return ErrEmailNotFound
	}
	email.Status = "parsed"
	email.ParseFailure = ""
	email.Parsed = parsed
	email.UpdatedAt = time.Now().UTC()
	return nil
}
func (s *MemoryStore) SaveParseFailure(_ context.Context, id, failure string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	email, ok := s.emails[id]
	if !ok {
		return ErrEmailNotFound
	}
	email.Status = "failed"
	email.ParseFailure = failure
	email.UpdatedAt = time.Now().UTC()
	return nil
}
func cloneEmail(value *domain.Email) *domain.Email {
	copy := *value
	copy.RawContent = append([]byte(nil), value.RawContent...)
	if value.Parsed != nil {
		parsed := *value.Parsed
		parsed.Headers = append([]domain.Header(nil), value.Parsed.Headers...)
		parsed.Indicators.IPs = append([]string(nil), value.Parsed.Indicators.IPs...)
		parsed.Indicators.Domains = append([]string(nil), value.Parsed.Indicators.Domains...)
		parsed.Indicators.URLs = append([]string(nil), value.Parsed.Indicators.URLs...)
		parsed.Attachments = append([]domain.Attachment(nil), value.Parsed.Attachments...)
		copy.Parsed = &parsed
	}
	return &copy
}
func cloneAnalysisResult(value *domain.AnalysisResult) *domain.AnalysisResult {
	copy := *value
	copy.AIAssessment.SupportingSignals = append([]string{}, value.AIAssessment.SupportingSignals...)
	copy.AIAssessment.EvidenceReferences = append([]string{}, value.AIAssessment.EvidenceReferences...)
	copy.ReceivedChain = append([]domain.ReceivedRelay{}, value.ReceivedChain...)
	copy.Authentication.SPF.EvidenceReferences = append([]domain.EvidenceReference{}, value.Authentication.SPF.EvidenceReferences...)
	copy.Authentication.DKIM.EvidenceReferences = append([]domain.EvidenceReference{}, value.Authentication.DKIM.EvidenceReferences...)
	copy.Authentication.DMARC.EvidenceReferences = append([]domain.EvidenceReference{}, value.Authentication.DMARC.EvidenceReferences...)
	copy.Risk.ContributingSignals = append([]domain.RiskSignal{}, value.Risk.ContributingSignals...)
	copy.Risk.EvidenceReferences = append([]domain.EvidenceReference{}, value.Risk.EvidenceReferences...)
	for index := range copy.Risk.ContributingSignals {
		copy.Risk.ContributingSignals[index].EvidenceReferences = append([]domain.EvidenceReference{}, value.Risk.ContributingSignals[index].EvidenceReferences...)
	}
	return &copy
}
func failureMessage(failure *domain.Failure) string {
	if failure == nil {
		return ""
	}
	return failure.Message
}
func pendingResult(analysisID, emailID, caseID, status, failure string) *domain.AnalysisResult {
	result := &domain.AnalysisResult{
		AnalysisID: analysisID, EmailID: emailID, CaseID: caseID, Status: status,
		AIAssessment: domain.AIAssessment{
			Status: "not_available", SupportingSignals: []string{}, EvidenceReferences: []string{}, Failure: &domain.Failure{Code: "AI_NOT_CONFIGURED", Message: "No AI analyzer is configured."},
		},
		Authentication: domain.AuthenticationResults{
			SPF:   domain.AuthenticationCheck{Status: "none", EvidenceReferences: []domain.EvidenceReference{}, Explanation: "Authentication has not been evaluated."},
			DKIM:  domain.AuthenticationCheck{Status: "none", EvidenceReferences: []domain.EvidenceReference{}, Explanation: "Authentication has not been evaluated."},
			DMARC: domain.AuthenticationCheck{Status: "none", EvidenceReferences: []domain.EvidenceReference{}, Explanation: "Authentication has not been evaluated."},
		},
		ReceivedChain: []domain.ReceivedRelay{},
		Risk:          domain.RiskAssessment{Level: "low", Verdict: "unknown", Confidence: nil, ContributingSignals: []domain.RiskSignal{}, EvidenceReferences: []domain.EvidenceReference{}},
	}
	if failure != "" {
		result.Failure = &domain.Failure{Code: "ANALYSIS_FAILED", Message: failure}
	}
	return result
}
func newID(prefix string) string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(bytes)
}
