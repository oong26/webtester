package services

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"webtester/internal/dtos"
	"webtester/internal/loadtest"
	"webtester/internal/stores"
	apperr "webtester/pkg/errors"
)

type LoadTestService struct {
	store   stores.LoadTestStore
	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

func NewLoadTestService(store stores.LoadTestStore) *LoadTestService {
	return &LoadTestService{store: store, cancels: make(map[string]context.CancelFunc)}
}

func (s *LoadTestService) Create(ctx context.Context, in dtos.CreateLoadTestInput) (*dtos.LoadTestResult, error) {
	if !in.ConfirmOwnership {
		return nil, apperr.Validation("ownership confirmation is required")
	}
	lt, err := s.store.Create(ctx, stores.CreateLoadTestParams{
		TargetURL:       in.TargetURL,
		Method:          in.Method,
		DurationSeconds: in.DurationSeconds,
		RequestsPerSec:  in.RequestsPerSec,
		Concurrency:     in.Concurrency,
		Status:          "queued",
	})
	if err != nil {
		return nil, apperr.Internal("failed to create load test", err)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancels[lt.ID] = cancel
	s.mu.Unlock()

	go s.run(runCtx, lt.ID, in)
	return toLoadTestResult(lt), nil
}

func (s *LoadTestService) Get(ctx context.Context, id string) (*dtos.LoadTestResult, error) {
	lt, err := s.store.Get(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.NotFound("load test not found")
		}
		return nil, apperr.Internal("failed to load load test", err)
	}
	return toLoadTestResult(lt), nil
}

func (s *LoadTestService) List(ctx context.Context, limit, offset int) ([]*dtos.LoadTestResult, error) {
	lts, err := s.store.List(ctx, limit, offset)
	if err != nil {
		return nil, apperr.Internal("failed to list load tests", err)
	}
	out := make([]*dtos.LoadTestResult, 0, len(lts))
	for _, lt := range lts {
		out = append(out, toLoadTestResult(lt))
	}
	return out, nil
}

func (s *LoadTestService) Cancel(ctx context.Context, id string) error {
	s.mu.Lock()
	cancel, ok := s.cancels[id]
	s.mu.Unlock()
	if !ok {
		return apperr.Validation("load test is not running")
	}
	cancel()
	return nil
}

func (s *LoadTestService) run(runCtx context.Context, id string, in dtos.CreateLoadTestInput) {
	defer func() {
		s.mu.Lock()
		if cancel, ok := s.cancels[id]; ok {
			cancel()
			delete(s.cancels, id)
		}
		s.mu.Unlock()
	}()

	// Store operations use a background context so a user cancel does not
	// abort the final write.
	dbCtx := context.Background()
	started := time.Now().UTC()
	if err := s.store.MarkRunning(dbCtx, id, started); err != nil {
		return
	}

	result, err := loadtest.Run(runCtx, loadtest.Config{
		TargetURL:      in.TargetURL,
		Method:         in.Method,
		Headers:        in.Headers,
		Body:           in.Body,
		DurationSecs:   in.DurationSeconds,
		RequestsPerSec: in.RequestsPerSec,
		Concurrency:    in.Concurrency,
	}, func(done int64, _ time.Duration) {
		_ = s.store.UpdateProgress(dbCtx, id, done)
	})

	finished := time.Now().UTC()
	if runCtx.Err() == context.Canceled {
		_ = s.store.Finish(dbCtx, id, stores.FinishLoadTestParams{
			Status:          "cancelled",
			FinishedAt:      finished,
			TotalRequests:   result.TotalRequests,
			Successful:      result.Successful,
			Failed:          result.Failed,
			ErrorRate:       result.ErrorRate,
			RPSAchieved:     result.RPSAchieved,
			MinMs:           result.MinMs,
			MaxMs:           result.MaxMs,
			MeanMs:          result.MeanMs,
			P50Ms:           result.P50Ms,
			P90Ms:           result.P90Ms,
			P95Ms:           result.P95Ms,
			P99Ms:           result.P99Ms,
			StatusBreakdown: result.StatusBreakdown,
			ErrorBreakdown:  result.ErrorBreakdown,
		})
		return
	}
	if err != nil {
		_ = s.store.Finish(dbCtx, id, stores.FinishLoadTestParams{
			Status:       "failed",
			ErrorMessage: err.Error(),
			FinishedAt:   finished,
		})
		return
	}
	_ = s.store.Finish(dbCtx, id, stores.FinishLoadTestParams{
		Status:          "completed",
		FinishedAt:      finished,
		TotalRequests:   result.TotalRequests,
		Successful:      result.Successful,
		Failed:          result.Failed,
		ErrorRate:       result.ErrorRate,
		RPSAchieved:     result.RPSAchieved,
		MinMs:           result.MinMs,
		MaxMs:           result.MaxMs,
		MeanMs:          result.MeanMs,
		P50Ms:           result.P50Ms,
		P90Ms:           result.P90Ms,
		P95Ms:           result.P95Ms,
		P99Ms:           result.P99Ms,
		StatusBreakdown: result.StatusBreakdown,
		ErrorBreakdown:  result.ErrorBreakdown,
	})
}

func toLoadTestResult(lt stores.LoadTest) *dtos.LoadTestResult {
	return &dtos.LoadTestResult{
		ID:              lt.ID,
		TargetURL:       lt.TargetURL,
		Method:          lt.Method,
		DurationSeconds: lt.DurationSeconds,
		RequestsPerSec:  lt.RequestsPerSec,
		Concurrency:     lt.Concurrency,
		Status:          lt.Status,
		ErrorMessage:    lt.ErrorMessage,
		CreatedAt:       lt.CreatedAt,
		StartedAt:       lt.StartedAt,
		FinishedAt:      lt.FinishedAt,
		TotalRequests:   lt.TotalRequests,
		Successful:      lt.Successful,
		Failed:          lt.Failed,
		ErrorRate:       lt.ErrorRate,
		RPSAchieved:     lt.RPSAchieved,
		MinMs:           lt.MinMs,
		MaxMs:           lt.MaxMs,
		MeanMs:          lt.MeanMs,
		P50Ms:           lt.P50Ms,
		P90Ms:           lt.P90Ms,
		P95Ms:           lt.P95Ms,
		P99Ms:           lt.P99Ms,
		StatusBreakdown: lt.StatusBreakdown,
		ErrorBreakdown:  lt.ErrorBreakdown,
	}
}