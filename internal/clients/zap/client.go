package zap

import "context"

type RunRequest struct {
	TargetURL string
	SpecURL   string
	ScanType  string // baseline | api | full
}

type Alert struct {
	RuleID     string
	Name       string
	Risk       string
	Confidence string
	URL        string
	Evidence   string
	CWE        int
}

type ProgressFunc func(progress int)

type Client interface {
	Run(ctx context.Context, req RunRequest, onProgress ProgressFunc) ([]Alert, error)
}
