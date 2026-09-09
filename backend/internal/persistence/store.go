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

var ErrEmailNotFound = errors.New("email not found")

type Store interface {
	CreateUpload(context.Context, string, []byte) (*domain.Email, error)
	GetEmail(context.Context, string) (*domain.Email, error)
	CreateAnalysis(context.Context, string) (*domain.Analysis, error)
	UpdateAnalysis(context.Context, string, string, string) error
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
}

// NewMemoryStore is only for isolated HTTP and service tests. The application uses PostgreSQL.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{emails: map[string]*domain.Email{}, analyses: map[string]*domain.Analysis{}}
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
		copy.Parsed = &parsed
	}
	return &copy
}
func newID(prefix string) string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(bytes)
}
