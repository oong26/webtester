package checks

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// LoadTest fires Total requests with Concurrency workers and reports latency percentiles.
type LoadTest struct {
	Path        string
	Method      string
	Total       int
	Concurrency int
}

func (LoadTest) Name() string { return "load test" }

func (l LoadTest) Run(ctx context.Context, t Target) []Result {
	total, conc := l.Total, l.Concurrency
	if total <= 0 {
		total = 200
	}
	if total > 5000 {
		total = 5000
	}
	if conc <= 0 {
		conc = 10
	}
	if conc > 100 {
		conc = 100
	}
	method := strings.ToUpper(l.Method)
	if method == "" {
		method = "GET"
	}

	jobs := make(chan struct{}, total)
	for i := 0; i < total; i++ {
		jobs <- struct{}{}
	}
	close(jobs)

	var mu sync.Mutex
	durs := make([]time.Duration, 0, total)
	var errCount int
	statuses := map[int]int{}

	var wg sync.WaitGroup
	start := time.Now()
	for w := 0; w < conc; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				if ctx.Err() != nil {
					return
				}
				resp, err := t.Do(ctx, method, l.Path, nil, nil, true)
				mu.Lock()
				if err != nil {
					errCount++
				} else {
					durs = append(durs, resp.Latency)
					statuses[resp.Status]++
					if resp.Status >= 500 {
						errCount++
					}
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	done := len(durs)
	if done == 0 {
		return []Result{fail(l.Name(), High, "no successful requests", "")}
	}
	sort.Slice(durs, func(i, j int) bool { return durs[i] < durs[j] })
	p50, p95, p99 := percentile(durs, 50), percentile(durs, 95), percentile(durs, 99)
	rps := float64(done) / elapsed.Seconds()
	errRate := float64(errCount) / float64(total) * 100

	res := Result{
		Check:      l.Name(),
		Passed:     errRate < 1,
		Severity:   Info,
		DurationMS: elapsed.Milliseconds(),
		Message: fmt.Sprintf("%d reqs, %d workers: %.0f req/s, p50=%dms p95=%dms p99=%dms, errors %.1f%%",
			total, conc, rps, p50.Milliseconds(), p95.Milliseconds(), p99.Milliseconds(), errRate),
		Detail: fmt.Sprintf("status codes: %v", statuses),
		Metrics: map[string]float64{
			"rps": rps, "p50_ms": float64(p50.Milliseconds()), "p95_ms": float64(p95.Milliseconds()),
			"p99_ms": float64(p99.Milliseconds()), "error_rate_pct": errRate,
		},
	}
	if !res.Passed {
		res.Severity = Medium
	}
	return []Result{res}
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
