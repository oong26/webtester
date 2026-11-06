package stores

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"time"

	_ "modernc.org/sqlite"

	"webtester/internal/dtos"
)

//go:embed loadtest_schema.sql
var loadTestSchemaFS embed.FS

type LoadTest struct {
	ID              string
	TargetURL       string
	Method          string
	DurationSeconds int
	RequestsPerSec  int
	Concurrency     int
	Status          string
	ErrorMessage    string
	CreatedAt       time.Time
	StartedAt       *time.Time
	FinishedAt      *time.Time
	TotalRequests   int64
	Successful      int64
	Failed          int64
	ErrorRate       float64
	RPSAchieved     float64
	MinMs           float64
	MaxMs           float64
	MeanMs          float64
	P50Ms           float64
	P90Ms           float64
	P95Ms           float64
	P99Ms           float64
	StatusBreakdown map[string]int64
	ErrorBreakdown  map[string]int64
}

type CreateLoadTestParams struct {
	TargetURL       string
	Method          string
	DurationSeconds int
	RequestsPerSec  int
	Concurrency     int
	Status          string
}

type FinishLoadTestParams struct {
	Status          string
	ErrorMessage    string
	FinishedAt      time.Time
	TotalRequests   int64
	Successful      int64
	Failed          int64
	ErrorRate       float64
	RPSAchieved     float64
	MinMs           float64
	MaxMs           float64
	MeanMs          float64
	P50Ms           float64
	P90Ms           float64
	P95Ms           float64
	P99Ms           float64
	StatusBreakdown map[string]int64
	ErrorBreakdown  map[string]int64
}

type LoadTestStore interface {
	Create(ctx context.Context, p CreateLoadTestParams) (LoadTest, error)
	Get(ctx context.Context, id string) (LoadTest, error)
	List(ctx context.Context, limit, offset int) ([]LoadTest, error)
	MarkRunning(ctx context.Context, id string, startedAt time.Time) error
	UpdateProgress(ctx context.Context, id string, totalRequests int64) error
	Finish(ctx context.Context, id string, p FinishLoadTestParams) error
}

type sqliteLoadTestStore struct{ db *sql.DB }

func NewLoadTestStore(path string) (LoadTestStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, err
	}
	schema, err := loadTestSchemaFS.ReadFile("loadtest_schema.sql")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(string(schema)); err != nil {
		return nil, err
	}
	return &sqliteLoadTestStore{db: db}, nil
}

func (s *sqliteLoadTestStore) Create(ctx context.Context, p CreateLoadTestParams) (LoadTest, error) {
	id := newID()
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO load_tests (id, target_url, method, duration_seconds, requests_per_sec, concurrency, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, p.TargetURL, p.Method, p.DurationSeconds, p.RequestsPerSec,
		p.Concurrency, p.Status, now.Format(time.RFC3339Nano))
	if err != nil {
		return LoadTest{}, err
	}
	return s.Get(ctx, id)
}

func (s *sqliteLoadTestStore) Get(ctx context.Context, id string) (LoadTest, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, target_url, method, duration_seconds, requests_per_sec, concurrency,
		        status, error_message, created_at, started_at, finished_at,
		        total_requests, successful, failed, error_rate, rps_achieved,
		        min_ms, max_ms, mean_ms, p50_ms, p90_ms, p95_ms, p99_ms, status_breakdown, error_breakdown
		 FROM load_tests WHERE id = ?`, id)
	return loadTestRow(row)
}

func (s *sqliteLoadTestStore) List(ctx context.Context, limit, offset int) ([]LoadTest, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, target_url, method, duration_seconds, requests_per_sec, concurrency,
		        status, error_message, created_at, started_at, finished_at,
		        total_requests, successful, failed, error_rate, rps_achieved,
		        min_ms, max_ms, mean_ms, p50_ms, p90_ms, p95_ms, p99_ms, status_breakdown, error_breakdown
		 FROM load_tests ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LoadTest
	for rows.Next() {
		lt, err := loadTestRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, lt)
	}
	return out, rows.Err()
}

func (s *sqliteLoadTestStore) MarkRunning(ctx context.Context, id string, startedAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE load_tests SET status = 'running', started_at = ? WHERE id = ?`,
		startedAt.Format(time.RFC3339Nano), id)
	return err
}

func (s *sqliteLoadTestStore) UpdateProgress(ctx context.Context, id string, total int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE load_tests SET total_requests = ? WHERE id = ?`, total, id)
	return err
}

func (s *sqliteLoadTestStore) Finish(ctx context.Context, id string, p FinishLoadTestParams) error {
	breakdown, _ := json.Marshal(p.StatusBreakdown)
	errBreakdown, _ := json.Marshal(p.ErrorBreakdown)
	_, err := s.db.ExecContext(ctx,
		`UPDATE load_tests SET
		    status = ?, error_message = ?, finished_at = ?,
		    total_requests = ?, successful = ?, failed = ?,
		    error_rate = ?, rps_achieved = ?,
		    min_ms = ?, max_ms = ?, mean_ms = ?,
		    p50_ms = ?, p90_ms = ?, p95_ms = ?, p99_ms = ?,
		    status_breakdown = ?, error_breakdown = ?
		 WHERE id = ?`,
		p.Status, p.ErrorMessage, p.FinishedAt.Format(time.RFC3339Nano),
		p.TotalRequests, p.Successful, p.Failed,
		p.ErrorRate, p.RPSAchieved,
		p.MinMs, p.MaxMs, p.MeanMs,
		p.P50Ms, p.P90Ms, p.P95Ms, p.P99Ms,
		string(breakdown), string(errBreakdown), id)
	return err
}

func loadTestRow(r rowScanner) (LoadTest, error) {
	var (
		lt         LoadTest
		createdStr string
		startedStr string
		finishStr  string
		breakdown  string
		errBreakdown  string
	)
	if err := r.Scan(
		&lt.ID, &lt.TargetURL, &lt.Method, &lt.DurationSeconds, &lt.RequestsPerSec,
		&lt.Concurrency, &lt.Status, &lt.ErrorMessage,
		&createdStr, &startedStr, &finishStr,
		&lt.TotalRequests, &lt.Successful, &lt.Failed,
		&lt.ErrorRate, &lt.RPSAchieved,
		&lt.MinMs, &lt.MaxMs, &lt.MeanMs,
		&lt.P50Ms, &lt.P90Ms, &lt.P95Ms, &lt.P99Ms,
		&breakdown, &errBreakdown,
	); err != nil {
		return LoadTest{}, err
	}
	lt.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdStr)
	if startedStr != "" {
		t, _ := time.Parse(time.RFC3339Nano, startedStr)
		lt.StartedAt = &t
	}
	if finishStr != "" {
		t, _ := time.Parse(time.RFC3339Nano, finishStr)
		lt.FinishedAt = &t
	}
	lt.StatusBreakdown = map[string]int64{}
	_ = json.Unmarshal([]byte(breakdown), &lt.StatusBreakdown)
	lt.ErrorBreakdown = map[string]int64{}
	_ = json.Unmarshal([]byte(errBreakdown), &lt.ErrorBreakdown)
	return lt, nil
}

// Silence unused-import when dtos isn't referenced directly.
var _ = dtos.ScanType("")