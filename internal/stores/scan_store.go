package stores

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
	"time"

	_ "modernc.org/sqlite"

	"webtester/internal/dtos"
)

//go:embed schema.sql
var schemaFS embed.FS

type Scan struct {
	ID           string
	TargetURL    string
	OpenAPIURL   string
	ScanType     string
	Status       string
	Progress     int
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CreateScanParams struct {
	TargetURL  string
	OpenAPIURL string
	ScanType   string
	Status     string
}

type ListScansParams struct {
	Limit  int
	Offset int
}

type ScanStore interface {
	Create(ctx context.Context, p CreateScanParams) (Scan, error)
	Get(ctx context.Context, id string) (Scan, error)
	List(ctx context.Context, p ListScansParams) ([]Scan, error)
	UpdateStatus(ctx context.Context, id, status string, progress int, errMsg string) error
	UpdateProgress(ctx context.Context, id string, progress int) error
	SaveFindings(ctx context.Context, scanID string, findings []dtos.Finding) error
	ListFindings(ctx context.Context, scanID string) ([]dtos.Finding, error)
	CountFindings(ctx context.Context, scanID string) (int, error)
}

type sqliteStore struct{ db *sql.DB }

func NewSQLiteStore(path string) (ScanStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, err
	}
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(string(schema)); err != nil {
		return nil, err
	}
	return &sqliteStore{db: db}, nil
}

func newID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *sqliteStore) Create(ctx context.Context, p CreateScanParams) (Scan, error) {
	id := newID()
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO scans (id, target_url, openapi_url, scan_type, status, progress, error_message, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 0, '', ?, ?)`,
		id, p.TargetURL, p.OpenAPIURL, p.ScanType, p.Status,
		now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano),
	)
	if err != nil {
		return Scan{}, err
	}
	return s.Get(ctx, id)
}

func (s *sqliteStore) Get(ctx context.Context, id string) (Scan, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, target_url, openapi_url, scan_type, status, progress, error_message, created_at, updated_at
		 FROM scans WHERE id = ?`, id)
	return scanRow(row)
}

func (s *sqliteStore) List(ctx context.Context, p ListScansParams) ([]Scan, error) {
	if p.Limit <= 0 {
		p.Limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, target_url, openapi_url, scan_type, status, progress, error_message, created_at, updated_at
		 FROM scans ORDER BY created_at DESC LIMIT ? OFFSET ?`, p.Limit, p.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Scan
	for rows.Next() {
		sc, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

func (s *sqliteStore) UpdateStatus(ctx context.Context, id, status string, progress int, errMsg string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE scans SET status = ?, progress = ?, error_message = ?, updated_at = ? WHERE id = ?`,
		status, progress, errMsg, time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}

func (s *sqliteStore) UpdateProgress(ctx context.Context, id string, progress int) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE scans SET progress = ?, updated_at = ? WHERE id = ?`,
		progress, time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}

func (s *sqliteStore) SaveFindings(ctx context.Context, scanID string, findings []dtos.Finding) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO findings (id, scan_id, rule_id, name, severity, confidence, url, evidence, cwe, source, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, f := range findings {
		if _, err := stmt.ExecContext(ctx,
			newID(), scanID, f.RuleID, f.Name, f.Severity, f.Confidence,
			f.URL, f.Evidence, f.CWE, f.Source, now,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *sqliteStore) ListFindings(ctx context.Context, scanID string) ([]dtos.Finding, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, scan_id, rule_id, name, severity, confidence, url, evidence, cwe, source
		 FROM findings WHERE scan_id = ? ORDER BY severity, name`, scanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtos.Finding
	for rows.Next() {
		var f dtos.Finding
		if err := rows.Scan(&f.ID, &f.ScanID, &f.RuleID, &f.Name, &f.Severity,
			&f.Confidence, &f.URL, &f.Evidence, &f.CWE, &f.Source); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *sqliteStore) CountFindings(ctx context.Context, scanID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM findings WHERE scan_id = ?`, scanID).Scan(&n)
	return n, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRow(r rowScanner) (Scan, error) {
	var (
		sc         Scan
		createdStr string
		updatedStr string
	)
	if err := r.Scan(&sc.ID, &sc.TargetURL, &sc.OpenAPIURL, &sc.ScanType, &sc.Status,
		&sc.Progress, &sc.ErrorMessage, &createdStr, &updatedStr); err != nil {
		return Scan{}, err
	}
	sc.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdStr)
	sc.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedStr)
	return sc, nil
}
