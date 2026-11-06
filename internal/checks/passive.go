package checks

import (
	"context"
	"net/http"
	"time"

	"webtester/internal/dtos"
)

var requiredHeaders = map[string]string{
	"Strict-Transport-Security": "HSTS header is missing",
	"X-Content-Type-Options":    "X-Content-Type-Options is missing",
	"X-Frame-Options":           "X-Frame-Options is missing",
	"Content-Security-Policy":   "Content-Security-Policy is missing",
	"Referrer-Policy":           "Referrer-Policy is missing",
}

func PassiveHeaderChecks(ctx context.Context, target string) []dtos.Finding {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var findings []dtos.Finding
	for h, msg := range requiredHeaders {
		if resp.Header.Get(h) == "" {
			findings = append(findings, dtos.Finding{
				RuleID:     "webtester/missing-header/" + h,
				Name:       msg,
				Severity:   "Low",
				Confidence: "High",
				URL:        target,
				Evidence:   "response headers for " + target + " lack " + h,
				Source:     "webtester",
			})
		}
	}
	if srv := resp.Header.Get("Server"); srv != "" {
		findings = append(findings, dtos.Finding{
			RuleID:     "webtester/server-disclosure",
			Name:       "Server header discloses software",
			Severity:   "Info",
			Confidence: "High",
			URL:        target,
			Evidence:   "Server: " + srv,
			Source:     "webtester",
		})
	}
	return findings
}
