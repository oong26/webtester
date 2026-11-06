package checks

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"
)

// EndpointTest is a functional assertion against one endpoint.
type EndpointTest struct {
	Title              string            `json:"name"`
	Method             string            `json:"method"`
	Path               string            `json:"path"`
	Headers            map[string]string `json:"headers"`
	Body               string            `json:"body"`
	ExpectStatus       int               `json:"expect_status"`
	ExpectBodyContains string            `json:"expect_body_contains"`
	MaxLatencyMS       int               `json:"max_latency_ms"`
}

func (e EndpointTest) Name() string { return "endpoint: " + e.Title }

func (e EndpointTest) Run(ctx context.Context, t Target) []Result {
	name := e.Name()
	method := strings.ToUpper(strings.TrimSpace(e.Method))
	if method == "" {
		method = "GET"
	}
	want := e.ExpectStatus
	if want == 0 {
		want = 200
	}

	hdrs := map[string]string{}
	for k, v := range e.Headers {
		hdrs[k] = v
	}
	var body []byte
	if e.Body != "" {
		body = []byte(e.Body)
		hasCT := false
		for k := range hdrs {
			if strings.EqualFold(k, "Content-Type") {
				hasCT = true
			}
		}
		if !hasCT {
			hdrs["Content-Type"] = "application/json"
		}
	}

	resp, err := t.Do(ctx, method, e.Path, hdrs, body, true)
	if err != nil {
		return []Result{fail(name, High, "request failed", err.Error())}
	}

	var problems []string
	if resp.Status != want {
		problems = append(problems, fmt.Sprintf("status %d, expected %d", resp.Status, want))
	}
	if e.ExpectBodyContains != "" && !bytes.Contains(resp.Body, []byte(e.ExpectBodyContains)) {
		problems = append(problems, fmt.Sprintf("body does not contain %q", e.ExpectBodyContains))
	}
	if e.MaxLatencyMS > 0 && resp.Latency > time.Duration(e.MaxLatencyMS)*time.Millisecond {
		problems = append(problems, fmt.Sprintf("latency %dms exceeds %dms", resp.Latency.Milliseconds(), e.MaxLatencyMS))
	}

	r := Result{Check: name, Passed: len(problems) == 0, Severity: Info, DurationMS: resp.Latency.Milliseconds()}
	if r.Passed {
		r.Message = fmt.Sprintf("%s %s → %d in %dms", method, e.Path, resp.Status, r.DurationMS)
	} else {
		r.Severity = Medium
		r.Message = strings.Join(problems, "; ")
		r.Detail = truncate(string(resp.Body), 400)
	}
	return []Result{r}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
