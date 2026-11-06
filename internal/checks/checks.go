package checks

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

type Severity string

const (
	Info   Severity = "info"
	Low    Severity = "low"
	Medium Severity = "medium"
	High   Severity = "high"
)

// Target describes the application under test.
type Target struct {
	BaseURL string
	Headers map[string]string // default headers (e.g. Authorization)
	Client  *http.Client
}

// Result is a single finding or assertion outcome.
type Result struct {
	Check      string             `json:"check"`
	Passed     bool               `json:"passed"`
	Severity   Severity           `json:"severity"`
	Message    string             `json:"message"`
	Detail     string             `json:"detail,omitempty"`
	DurationMS int64              `json:"duration_ms"`
	Metrics    map[string]float64 `json:"metrics,omitempty"`
}

// Check is the unit of work. Add a new test by implementing this interface.
type Check interface {
	Name() string
	Run(ctx context.Context, t Target) []Result
}

// Response is a trimmed-down HTTP response.
type Response struct {
	Status  int
	Header  http.Header
	Body    []byte
	Latency time.Duration
}

// NewClient returns a client that does not follow redirects, so checks see raw responses.
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// Do sends a request. If useDefaults is true the target's default headers are applied first.
func (t Target) Do(ctx context.Context, method, path string, headers map[string]string, body []byte, useDefaults bool) (*Response, error) {
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(t.BaseURL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if useDefaults {
		for k, v := range t.Headers {
			req.Header.Set(k, v)
		}
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	start := time.Now()
	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return &Response{Status: resp.StatusCode, Header: resp.Header, Body: b, Latency: time.Since(start)}, nil
}

func pass(name, msg string) Result {
	return Result{Check: name, Passed: true, Severity: Info, Message: msg}
}

func fail(name string, sev Severity, msg, detail string) Result {
	return Result{Check: name, Passed: false, Severity: sev, Message: msg, Detail: detail}
}
