package checks

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ---------- Security headers ----------

type SecurityHeaders struct{ Path string }

func (s SecurityHeaders) Name() string { return "security headers" }

func (s SecurityHeaders) Run(ctx context.Context, t Target) []Result {
	path := s.Path
	if path == "" {
		path = "/"
	}
	resp, err := t.Do(ctx, "GET", path, nil, nil, true)
	if err != nil {
		return []Result{fail(s.Name(), High, "request failed", err.Error())}
	}

	type rule struct {
		header string
		sev    Severity
		hint   string
		ok     func(h http.Header) bool
	}
	rules := []rule{
		{"X-Content-Type-Options", Low, "set to nosniff", func(h http.Header) bool {
			return strings.EqualFold(h.Get("X-Content-Type-Options"), "nosniff")
		}},
		{"Content-Security-Policy", Medium, "define a Content-Security-Policy", func(h http.Header) bool {
			return h.Get("Content-Security-Policy") != ""
		}},
		{"X-Frame-Options", Low, "use DENY/SAMEORIGIN or CSP frame-ancestors", func(h http.Header) bool {
			return h.Get("X-Frame-Options") != "" || strings.Contains(h.Get("Content-Security-Policy"), "frame-ancestors")
		}},
		{"Referrer-Policy", Low, "set a Referrer-Policy", func(h http.Header) bool {
			return h.Get("Referrer-Policy") != ""
		}},
	}
	if strings.HasPrefix(t.BaseURL, "https://") {
		rules = append(rules, rule{"Strict-Transport-Security", Medium, "enable HSTS", func(h http.Header) bool {
			return h.Get("Strict-Transport-Security") != ""
		}})
	}

	var out []Result
	for _, r := range rules {
		name := "header: " + r.header
		if r.ok(resp.Header) {
			out = append(out, pass(name, "present"))
			continue
		}
		val := resp.Header.Get(r.header)
		// A Report-Only CSP is a legitimate rollout step, so call it what it is
		// instead of reporting the header as missing.
		if r.header == "Content-Security-Policy" && val == "" {
			if ro := resp.Header.Get("Content-Security-Policy-Report-Only"); ro != "" {
				out = append(out, fail(name, Low,
					"report-only: violations are reported but the policy is not enforced",
					"Content-Security-Policy-Report-Only: "+ro+" (switch to Content-Security-Policy once it is clean)"))
				continue
			}
		}
		if val == "" {
			out = append(out, fail(name, r.sev, "missing", r.hint))
		} else {
			out = append(out, fail(name, r.sev, "present but weak: "+val, r.hint))
		}
	}
	if v := resp.Header.Get("X-Powered-By"); v != "" {
		out = append(out, fail("header: X-Powered-By", Low, "technology disclosure", v))
	}
	if v := resp.Header.Get("Server"); strings.ContainsAny(v, "0123456789") {
		out = append(out, fail("header: Server", Low, "server version disclosure", v))
	}
	return out
}

// ---------- Exposed paths ----------

type ExposedPaths struct{}

func (ExposedPaths) Name() string { return "exposed paths" }

func (e ExposedPaths) Run(ctx context.Context, t Target) []Result {
	name := e.Name()
	baseline, _ := t.Do(ctx, "GET", fmt.Sprintf("/nonexistent-%d", time.Now().UnixNano()), nil, nil, true)

	paths := []struct {
		p   string
		sev Severity
	}{
		{"/.env", High}, {"/.git/HEAD", High}, {"/config.json", High}, {"/backup.zip", High},
		{"/debug/pprof/", High}, {"/actuator", Medium}, {"/metrics", Medium}, {"/admin", Medium},
		{"/swagger.json", Low}, {"/openapi.json", Low}, {"/server-status", Medium}, {"/.DS_Store", Low},
	}

	var out []Result
	for _, p := range paths {
		if ctx.Err() != nil {
			break
		}
		resp, err := t.Do(ctx, "GET", p.p, nil, nil, true)
		if err != nil || resp.Status != 200 {
			continue
		}
		// SPAs often answer 200 for everything; skip responses identical to the 404 baseline.
		if baseline != nil && baseline.Status == 200 && bytes.Equal(baseline.Body, resp.Body) {
			continue
		}
		out = append(out, fail(name+": "+p.p, p.sev, "reachable (HTTP 200)", truncate(string(resp.Body), 200)))
	}
	if len(out) == 0 {
		out = append(out, pass(name, "no common sensitive paths exposed"))
	}
	return out
}

// ---------- CORS ----------

type CORSCheck struct{ Path string }

func (CORSCheck) Name() string { return "cors" }

func (c CORSCheck) Run(ctx context.Context, t Target) []Result {
	path := c.Path
	if path == "" {
		path = "/"
	}
	const evil = "https://evil.example"
	resp, err := t.Do(ctx, "GET", path, map[string]string{"Origin": evil}, nil, true)
	if err != nil {
		return []Result{fail(c.Name(), High, "request failed", err.Error())}
	}
	acao := resp.Header.Get("Access-Control-Allow-Origin")
	creds := strings.EqualFold(resp.Header.Get("Access-Control-Allow-Credentials"), "true")
	switch {
	case acao == evil && creds:
		return []Result{fail(c.Name(), High, "arbitrary origin reflected with credentials allowed", acao)}
	case acao == evil:
		return []Result{fail(c.Name(), Medium, "arbitrary origin reflected", acao)}
	case acao == "*":
		return []Result{fail(c.Name(), Low, "wildcard CORS policy", "Access-Control-Allow-Origin: *")}
	}
	return []Result{pass(c.Name(), "untrusted origin not allowed")}
}

// ---------- Malformed input ----------

type MalformedInput struct{ Path string }

func (MalformedInput) Name() string { return "malformed input" }

var leakPatterns = []string{
	"panic:", "goroutine ", "runtime error", "stack trace", "sql syntax", "pq:", "sqlite", "mysql", "ora-0", ".go:",
}

func (m MalformedInput) Run(ctx context.Context, t Target) []Result {
	p := m.Path
	if p == "" {
		p = "/"
	}
	sep := "?"
	if strings.Contains(p, "?") {
		sep = "&"
	}
	const xss = `<script>alert(1)</script>`
	probes := []struct {
		name, method, path string
		body               []byte
		ct                 string
		kind               string
	}{
		{"truncated JSON", "POST", p, []byte(`{"a":`), "application/json", ""},
		{"wrong content type", "POST", p, []byte(`<xml/>`), "application/json", ""},
		{"oversized body (256KB)", "POST", p, bytes.Repeat([]byte("A"), 256<<10), "application/json", ""},
		{"SQLi-style query", "GET", p + sep + "q=" + url.QueryEscape("' OR '1'='1' --"), nil, "", ""},
		{"XSS reflection", "GET", p + sep + "q=" + url.QueryEscape(xss), nil, "", "xss"},
		{"path traversal", "GET", p + sep + "file=" + url.QueryEscape("../../../../etc/passwd"), nil, "", "traversal"},
	}

	var out []Result
	for _, pr := range probes {
		if ctx.Err() != nil {
			break
		}
		name := "malformed: " + pr.name
		hdrs := map[string]string{}
		if pr.ct != "" {
			hdrs["Content-Type"] = pr.ct
		}
		resp, err := t.Do(ctx, pr.method, pr.path, hdrs, pr.body, true)
		if err != nil {
			out = append(out, fail(name, Medium, "request failed (server may have dropped the connection)", err.Error()))
			continue
		}
		body := strings.ToLower(string(resp.Body))
		var issue string
		var sev Severity
		for _, lp := range leakPatterns {
			if strings.Contains(body, lp) {
				issue, sev = "response leaks internals (matched "+lp+")", High
				break
			}
		}
		if issue == "" && pr.kind == "xss" && strings.Contains(string(resp.Body), xss) {
			issue, sev = "payload reflected unescaped", Medium
		}
		if issue == "" && pr.kind == "traversal" && strings.Contains(body, "root:x:") {
			issue, sev = "file contents returned", High
		}
		if issue == "" && resp.Status >= 500 {
			issue, sev = fmt.Sprintf("server error %d on bad input", resp.Status), Medium
		}
		if issue != "" {
			out = append(out, fail(name, sev, issue, truncate(string(resp.Body), 300)))
		} else {
			out = append(out, pass(name, fmt.Sprintf("handled gracefully (HTTP %d)", resp.Status)))
		}
	}
	return out
}

// ---------- Auth required ----------

type AuthRequired struct{ Paths []string }

func (AuthRequired) Name() string { return "auth required" }

func (a AuthRequired) Run(ctx context.Context, t Target) []Result {
	var out []Result
	variants := []struct {
		label   string
		headers map[string]string
	}{
		{"no credentials", nil},
		{"invalid token", map[string]string{"Authorization": "Bearer invalid.token.value"}},
	}
	for _, p := range a.Paths {
		for _, v := range variants {
			if ctx.Err() != nil {
				return out
			}
			name := fmt.Sprintf("auth: %s (%s)", p, v.label)
			resp, err := t.Do(ctx, "GET", p, v.headers, nil, false)
			if err != nil {
				out = append(out, fail(name, Medium, "request failed", err.Error()))
				continue
			}
			if resp.Status >= 200 && resp.Status < 300 {
				out = append(out, fail(name, High, fmt.Sprintf("protected endpoint returned %d", resp.Status), truncate(string(resp.Body), 200)))
			} else {
				out = append(out, pass(name, fmt.Sprintf("rejected (HTTP %d)", resp.Status)))
			}
		}
	}
	return out
}

// ---------- Rate limit ----------

type RateLimit struct {
	Path  string
	Burst int
}

func (RateLimit) Name() string { return "rate limit" }

func (r RateLimit) Run(ctx context.Context, t Target) []Result {
	burst := r.Burst
	if burst <= 0 || burst > 500 {
		burst = 60
	}
	path := r.Path
	if path == "" {
		path = "/"
	}
	var limited, ok, errs atomic.Int64
	var wg sync.WaitGroup
	sem := make(chan struct{}, 20)
	for i := 0; i < burst; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			resp, err := t.Do(ctx, "GET", path, nil, nil, true)
			switch {
			case err != nil:
				errs.Add(1)
			case resp.Status == 429:
				limited.Add(1)
			default:
				ok.Add(1)
			}
		}()
	}
	wg.Wait()
	detail := fmt.Sprintf("%d requests: %d ok, %d throttled (429), %d errors", burst, ok.Load(), limited.Load(), errs.Load())
	if limited.Load() > 0 {
		return []Result{pass(r.Name(), "throttling observed. "+detail)}
	}
	return []Result{fail(r.Name(), Low, "no rate limiting observed on this path", detail)}
}
