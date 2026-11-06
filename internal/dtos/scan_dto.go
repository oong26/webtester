package dtos

import "time"

type ScanType string

const (
	ScanBaseline ScanType = "baseline"
	ScanAPI      ScanType = "api"
	ScanFull     ScanType = "full"
)

type ScanAuth struct {
	HeaderName string `json:"header_name" validate:"required"`
	UserA      string `json:"user_a" validate:"required"`
	UserB      string `json:"user_b,omitempty"`
}

type CreateScanRequest struct {
	TargetURL        string    `json:"target_url" validate:"required,url"`
	OpenAPIURL       string    `json:"openapi_url,omitempty" validate:"omitempty,url"`
	ScanType         ScanType  `json:"scan_type" validate:"required,oneof=baseline api full"`
	ConfirmOwnership bool      `json:"confirm_ownership"`
	Auth             *ScanAuth `json:"auth,omitempty"`
	BolaProbeURLs    []string  `json:"bola_probe_urls,omitempty" validate:"omitempty,dive,url"`
}

type CreateScanInput struct {
	TargetURL        string
	OpenAPIURL       string
	ScanType         ScanType
	ConfirmOwnership bool
	Auth             *ScanAuth
	BolaProbeURLs    []string
}

type ScanResult struct {
	ID            string    `json:"id"`
	Status        string    `json:"status"`
	Progress      int       `json:"progress"`
	TargetURL     string    `json:"target_url"`
	ScanType      ScanType  `json:"scan_type"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	FindingsCount int       `json:"findings_count"`
	CreatedAt     time.Time `json:"created_at"`
}

type Finding struct {
	ID         string `json:"id,omitempty"`
	ScanID     string `json:"scan_id"`
	RuleID     string `json:"rule_id"`
	Name       string `json:"name"`
	Severity   string `json:"severity"`
	Confidence string `json:"confidence"`
	URL        string `json:"url"`
	Evidence   string `json:"evidence"`
	CWE        int    `json:"cwe,omitempty"`
	Source     string `json:"source"`
}
