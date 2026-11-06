package checks

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"webtester/internal/dtos"
)

type BolaAuth struct {
	HeaderName string
	UserA      string
	UserB      string
}

func BolaProbe(ctx context.Context, probeURLs []string, auth BolaAuth) []dtos.Finding {
	if len(probeURLs) == 0 || auth.UserA == "" || auth.UserB == "" {
		return nil
	}
	client := &http.Client{Timeout: 15 * time.Second}
	var findings []dtos.Finding
	for _, u := range probeURLs {
		bodyA, statusA := fetch(client, ctx, u, auth.HeaderName, auth.UserA)
		if statusA != http.StatusOK {
			continue
		}
		bodyB, statusB := fetch(client, ctx, u, auth.HeaderName, auth.UserB)
		if statusB != http.StatusOK {
			continue
		}
		if similar(bodyA, bodyB) {
			findings = append(findings, dtos.Finding{
				RuleID:     "webtester/bola",
				Name:       "Possible broken object-level authorization",
				Severity:   "High",
				Confidence: "Low",
				URL:        u,
				Evidence:   "User B receives a response similar to User A",
				Source:     "webtester",
			})
		}
	}
	return findings
}

func fetch(c *http.Client, ctx context.Context, url, header, token string) ([]byte, int) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0
	}
	req.Header.Set(header, token)
	resp, err := c.Do(req)
	if err != nil {
		return nil, 0
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return b, resp.StatusCode
}

func similar(a, b []byte) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	ratio := float64(len(b)) / float64(len(a))
	if ratio < 0.9 || ratio > 1.1 {
		return false
	}
	n := len(a)
	if n > 256 {
		n = 256
	}
	return strings.EqualFold(string(a[:n]), string(b[:n]))
}
