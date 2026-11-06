package loadtest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type Config struct {
	TargetURL      string
	Method         string
	Headers        map[string]string
	Body           string
	DurationSecs   int
	RequestsPerSec int
	Concurrency    int
}

type Result struct {
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

type ProgressFunc func(done int64, elapsed time.Duration)

// Run performs a closed-loop load test with a rate cap. The producer submits
// jobs at RequestsPerSec; Concurrency workers consume them. If the target is
// slow, backpressure lowers the achieved rate — the reported RPSAchieved is
// the honest number.
//
// Redirects are followed (default Go behavior). Response bodies are discarded
// so memory stays flat regardless of body size. Latencies are kept as int64
// nanoseconds in a slice — for a 600s test at 5000 RPS that is 24MB, which is
// acceptable for a tool that already runs ZAP.
func Run(ctx context.Context, cfg Config, onProgress ProgressFunc) (Result, error) {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}
	if cfg.RequestsPerSec <= 0 {
		cfg.RequestsPerSec = 1
	}
	if cfg.DurationSecs <= 0 {
		cfg.DurationSecs = 1
	}

	runCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.DurationSecs)*time.Second)
	defer cancel()

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        cfg.Concurrency * 2,
			MaxIdleConnsPerHost: cfg.Concurrency * 2,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	bodyReader := func() io.Reader {
		if cfg.Body == "" {
			return nil
		}
		return strings.NewReader(cfg.Body)
	}

	type outcome struct {
		status  int
		latency int64 // nanoseconds
		err     error
	}

	jobs := make(chan struct{}, cfg.Concurrency)
	results := make(chan outcome, cfg.Concurrency*4)

	var wg sync.WaitGroup
	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				if runCtx.Err() != nil {
					return
				}
				start := time.Now()
				req, err := http.NewRequestWithContext(runCtx, cfg.Method, cfg.TargetURL, bodyReader())
				if err != nil {
					results <- outcome{err: err, latency: time.Since(start).Nanoseconds()}
					continue
				}
				for k, v := range cfg.Headers {
					req.Header.Set(k, v)
				}
				resp, err := client.Do(req)
				lat := time.Since(start).Nanoseconds()
				if err != nil {
					results <- outcome{err: err, latency: lat}
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				results <- outcome{status: resp.StatusCode, latency: lat}
			}
		}()
	}

	start := time.Now()

	go func() {
		defer close(jobs)
		interval := time.Second / time.Duration(cfg.RequestsPerSec)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				select {
				case jobs <- struct{}{}:
				case <-runCtx.Done():
					return
				}
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	latencies := make([]int64, 0, cfg.DurationSecs*cfg.RequestsPerSec)
	var total, successful, failed int64
	statusCounts := make(map[int]int64)
	lastProgress := time.Now()
	errCounts := make(map[string]int64)

	for r := range results {
		total++
		if r.err != nil {
			failed++
			errCounts[r.err.Error()]++
		} else {
			statusCounts[r.status]++
			if r.status >= 200 && r.status < 400 {
				successful++
			} else {
				failed++
			}
			latencies = append(latencies, r.latency)
		}
		if onProgress != nil && time.Since(lastProgress) > 500*time.Millisecond {
			onProgress(total, time.Since(start))
			lastProgress = time.Now()
		}
	}

	elapsed := time.Since(start)
	res := Result{
		TotalRequests:   total,
		Successful:      successful,
		Failed:          failed,
		StatusBreakdown: make(map[string]int64, len(statusCounts)),
		ErrorBreakdown:  errCounts,
	}
	if total > 0 {
		res.ErrorRate = float64(failed) / float64(total) * 100
	}
	if elapsed > 0 {
		res.RPSAchieved = float64(total) / elapsed.Seconds()
	}
	for code, n := range statusCounts {
		res.StatusBreakdown[fmt.Sprintf("%d", code)] = n
	}
	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		toMs := func(ns int64) float64 { return float64(ns) / 1e6 }
		res.MinMs = toMs(latencies[0])
		res.MaxMs = toMs(latencies[len(latencies)-1])
		var sum int64
		for _, l := range latencies {
			sum += l
		}
		res.MeanMs = toMs(sum / int64(len(latencies)))
		pct := func(p float64) float64 {
			idx := int(float64(len(latencies)-1) * p)
			return toMs(latencies[idx])
		}
		res.P50Ms = pct(0.50)
		res.P90Ms = pct(0.90)
		res.P95Ms = pct(0.95)
		res.P99Ms = pct(0.99)
	}
	if onProgress != nil {
		onProgress(total, elapsed)
	}
	return res, nil
}