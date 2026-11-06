package dtos

import "time"

type CreateLoadTestRequest struct {
	TargetURL        string            `json:"target_url" validate:"required,url"`
	Method           string            `json:"method" validate:"required,oneof=GET POST PUT PATCH DELETE HEAD OPTIONS"`
	Headers          map[string]string `json:"headers,omitempty"`
	Body             string            `json:"body,omitempty"`
	DurationSeconds  int               `json:"duration_seconds" validate:"required,min=1,max=600"`
	RequestsPerSec   int               `json:"requests_per_sec" validate:"required,min=1,max=5000"`
	Concurrency      int               `json:"concurrency" validate:"required,min=1,max=500"`
	ConfirmOwnership bool              `json:"confirm_ownership"`
}

type CreateLoadTestInput struct {
	TargetURL        string
	Method           string
	Headers          map[string]string
	Body             string
	DurationSeconds  int
	RequestsPerSec   int
	Concurrency      int
	ConfirmOwnership bool
}

type LoadTestResult struct {
	ID              string           `json:"id"`
	TargetURL       string           `json:"target_url"`
	Method          string           `json:"method"`
	DurationSeconds int              `json:"duration_seconds"`
	RequestsPerSec  int              `json:"requests_per_sec"`
	Concurrency     int              `json:"concurrency"`
	Status          string           `json:"status"`
	ErrorMessage    string           `json:"error_message,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	StartedAt       *time.Time       `json:"started_at,omitempty"`
	FinishedAt      *time.Time       `json:"finished_at,omitempty"`
	TotalRequests   int64            `json:"total_requests"`
	Successful      int64            `json:"successful"`
	Failed          int64            `json:"failed"`
	ErrorRate       float64          `json:"error_rate"`
	RPSAchieved     float64          `json:"rps_achieved"`
	MinMs           float64          `json:"min_ms"`
	MaxMs           float64          `json:"max_ms"`
	MeanMs          float64          `json:"mean_ms"`
	P50Ms           float64          `json:"p50_ms"`
	P90Ms           float64          `json:"p90_ms"`
	P95Ms           float64          `json:"p95_ms"`
	P99Ms           float64          `json:"p99_ms"`
	StatusBreakdown map[string]int64 `json:"status_breakdown"`
	ErrorBreakdown  map[string]int64 `json:"error_breakdown"`
}