package zap

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type dockerClient struct {
	image   string
	timeout time.Duration
}

func NewDockerClient(image string, timeout time.Duration) Client {
	if image == "" {
		image = "ghcr.io/zaproxy/zaproxy:stable"
	}
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	return &dockerClient{image: image, timeout: timeout}
}

func (c *dockerClient) Run(ctx context.Context, req RunRequest, onProgress ProgressFunc) ([]Alert, error) {
	workdir, err := os.MkdirTemp("", "zap-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(workdir)
	_ = os.Chmod(workdir, 0o777)

	script, target, extra := scriptFor(req)
	args := []string{
		"run", "--rm",
		"--network=host",
		"-v", workdir + ":/zap/wrk:rw",
		c.image,
		script, "-t", target, "-J", "report.json", "-I",
	}
	args = append(args, extra...)

	runCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	if onProgress != nil {
		onProgress(5)
	}
	out, err := exec.CommandContext(runCtx, "docker", args...).CombinedOutput()
	if onProgress != nil {
		onProgress(90)
	}
	if err != nil {
		return nil, fmt.Errorf("zap scan failed: %w: %s", err, truncate(string(out), 2000))
	}

	data, err := os.ReadFile(filepath.Join(workdir, "report.json"))
	if err != nil {
		return nil, fmt.Errorf("zap report not found: %w", err)
	}
	alerts, err := parseReport(data)
	if err != nil {
		return nil, err
	}
	if onProgress != nil {
		onProgress(100)
	}
	return alerts, nil
}

func scriptFor(req RunRequest) (script, target string, extra []string) {
	switch req.ScanType {
	case "api":
		return "zap-api-scan.py", req.SpecURL, []string{"-f", "openapi"}
	case "full":
		return "zap-full-scan.py", req.TargetURL, nil
	default:
		return "zap-baseline.py", req.TargetURL, nil
	}
}

type zapReport struct {
	Site []struct {
		Alerts []struct {
			PluginID   string `json:"pluginid"`
			Alert      string `json:"alert"`
			RiskCode   string `json:"riskcode"`
			Confidence string `json:"confidence"`
			Instances  []struct {
				URI      string `json:"uri"`
				Evidence string `json:"evidence"`
			} `json:"instances"`
		} `json:"alerts"`
	} `json:"site"`
}

func parseReport(data []byte) ([]Alert, error) {
	var r zapReport
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse zap report: %w", err)
	}
	var out []Alert
	for _, site := range r.Site {
		for _, a := range site.Alerts {
			risk := riskFromCode(a.RiskCode)
			conf := confidenceFromCode(a.Confidence)
			if len(a.Instances) == 0 {
				out = append(out, Alert{RuleID: a.PluginID, Name: a.Alert, Risk: risk, Confidence: conf})
				continue
			}
			for _, inst := range a.Instances {
				out = append(out, Alert{
					RuleID:     a.PluginID,
					Name:       a.Alert,
					Risk:       risk,
					Confidence: conf,
					URL:        inst.URI,
					Evidence:   inst.Evidence,
				})
			}
		}
	}
	return out, nil
}

func riskFromCode(c string) string {
	switch c {
	case "3":
		return "High"
	case "2":
		return "Medium"
	case "1":
		return "Low"
	default:
		return "Info"
	}
}

func confidenceFromCode(c string) string {
	switch c {
	case "3":
		return "High"
	case "2":
		return "Medium"
	case "1":
		return "Low"
	default:
		return "Low"
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
