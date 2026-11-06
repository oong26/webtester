package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/url"
	"strings"
	"time"

	"webtester/internal/checks"
)

type SecurityConfig struct {
	Headers        bool     `json:"headers"`
	ExposedPaths   bool     `json:"exposed_paths"`
	CORS           bool     `json:"cors"`
	MalformedInput bool     `json:"malformed_input"`
	ProbePath      string   `json:"probe_path"`
	AuthPaths      []string `json:"auth_paths"`
	RateLimitPath  string   `json:"rate_limit_path"`
}

type LoadConfig struct {
	Path        string `json:"path"`
	Method      string `json:"method"`
	Total       int    `json:"total"`
	Concurrency int    `json:"concurrency"`
}

type Request struct {
	BaseURL          string                `json:"base_url"`
	Headers          map[string]string     `json:"headers"`
	ConfirmOwnership bool                  `json:"confirm_ownership"`
	Endpoints        []checks.EndpointTest `json:"endpoints"`
	Security         SecurityConfig        `json:"security"`
	Load             *LoadConfig           `json:"load"`
}

type Summary struct {
	Total  int `json:"total"`
	Passed int `json:"passed"`
	Failed int `json:"failed"`
	High   int `json:"high"`
	Medium int `json:"medium"`
	Low    int `json:"low"`
}

type Report struct {
	ID         string          `json:"id"`
	Target     string          `json:"target"`
	StartedAt  time.Time       `json:"started_at"`
	DurationMS int64           `json:"duration_ms"`
	Results    []checks.Result `json:"results"`
	Summary    Summary         `json:"summary"`
}

func Run(ctx context.Context, req Request) (*Report, error) {
	if !req.ConfirmOwnership {
		return nil, errors.New("confirm that you own this target or have written permission to test it")
	}
	u, err := url.Parse(strings.TrimSpace(req.BaseURL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, errors.New("base_url must be a valid http:// or https:// URL")
	}
	if len(req.Endpoints) > 200 {
		return nil, errors.New("too many endpoint tests (max 200)")
	}

	t := checks.Target{
		BaseURL: strings.TrimRight(u.String(), "/"),
		Headers: req.Headers,
		Client:  checks.NewClient(15 * time.Second),
	}

	var list []checks.Check
	for _, e := range req.Endpoints {
		list = append(list, e)
	}
	s := req.Security
	if s.Headers {
		list = append(list, checks.SecurityHeaders{Path: s.ProbePath})
	}
	if s.CORS {
		list = append(list, checks.CORSCheck{Path: s.ProbePath})
	}
	if s.ExposedPaths {
		list = append(list, checks.ExposedPaths{})
	}
	if s.MalformedInput {
		list = append(list, checks.MalformedInput{Path: s.ProbePath})
	}
	if len(s.AuthPaths) > 0 {
		list = append(list, checks.AuthRequired{Paths: s.AuthPaths})
	}
	if s.RateLimitPath != "" {
		list = append(list, checks.RateLimit{Path: s.RateLimitPath})
	}
	if l := req.Load; l != nil && l.Path != "" {
		list = append(list, checks.LoadTest{Path: l.Path, Method: l.Method, Total: l.Total, Concurrency: l.Concurrency})
	}
	if len(list) == 0 {
		return nil, errors.New("nothing to run: add an endpoint test or enable a security check")
	}

	id := make([]byte, 4)
	_, _ = rand.Read(id)
	rep := &Report{ID: hex.EncodeToString(id), Target: t.BaseURL, StartedAt: time.Now(), Results: []checks.Result{}}

	for _, c := range list {
		if ctx.Err() != nil {
			break
		}
		start := time.Now()
		rs := c.Run(ctx, t)
		ms := time.Since(start).Milliseconds()
		for i := range rs {
			if rs[i].DurationMS == 0 {
				rs[i].DurationMS = ms
			}
		}
		rep.Results = append(rep.Results, rs...)
	}
	rep.DurationMS = time.Since(rep.StartedAt).Milliseconds()

	for _, r := range rep.Results {
		rep.Summary.Total++
		if r.Passed {
			rep.Summary.Passed++
			continue
		}
		rep.Summary.Failed++
		switch r.Severity {
		case checks.High:
			rep.Summary.High++
		case checks.Medium:
			rep.Summary.Medium++
		case checks.Low:
			rep.Summary.Low++
		}
	}
	return rep, nil
}
