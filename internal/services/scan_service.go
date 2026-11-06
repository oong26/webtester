package services

import (
	"context"
	"database/sql"
	"errors"

	"webtester/internal/checks"
	"webtester/internal/clients/zap"
	"webtester/internal/dtos"
	"webtester/internal/stores"
	apperr "webtester/pkg/errors"
)

type ScanService struct {
	store stores.ScanStore
	zap   zap.Client
}

func NewScanService(store stores.ScanStore, z zap.Client) *ScanService {
	return &ScanService{store: store, zap: z}
}

func (s *ScanService) List(ctx context.Context, limit, offset int) ([]*dtos.ScanResult, error) {
	scans, err := s.store.List(ctx, stores.ListScansParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, apperr.Internal("failed to list scans", err)
	}
	out := make([]*dtos.ScanResult, 0, len(scans))
	for _, scan := range scans {
		count, _ := s.store.CountFindings(ctx, scan.ID)
		out = append(out, &dtos.ScanResult{
			ID:            scan.ID,
			Status:        scan.Status,
			Progress:      scan.Progress,
			TargetURL:     scan.TargetURL,
			ScanType:      dtos.ScanType(scan.ScanType),
			ErrorMessage:  scan.ErrorMessage,
			FindingsCount: count,
			CreatedAt:     scan.CreatedAt,
		})
	}
	return out, nil
}

func (s *ScanService) Create(ctx context.Context, in dtos.CreateScanInput) (*dtos.ScanResult, error) {
	if !in.ConfirmOwnership {
		return nil, apperr.Validation("ownership confirmation is required")
	}
	if in.ScanType == dtos.ScanAPI && in.OpenAPIURL == "" {
		return nil, apperr.Validation("openapi_url is required for API scans")
	}

	scan, err := s.store.Create(ctx, stores.CreateScanParams{
		TargetURL:  in.TargetURL,
		OpenAPIURL: in.OpenAPIURL,
		ScanType:   string(in.ScanType),
		Status:     "queued",
	})
	if err != nil {
		return nil, apperr.Internal("failed to create scan", err)
	}

	go s.run(context.Background(), scan.ID, in)

	return &dtos.ScanResult{
		ID:        scan.ID,
		Status:    scan.Status,
		Progress:  scan.Progress,
		TargetURL: scan.TargetURL,
		ScanType:  dtos.ScanType(scan.ScanType),
		CreatedAt: scan.CreatedAt,
	}, nil
}

func (s *ScanService) Get(ctx context.Context, id string) (*dtos.ScanResult, error) {
	scan, err := s.store.Get(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.NotFound("scan not found")
		}
		return nil, apperr.Internal("failed to load scan", err)
	}
	count, _ := s.store.CountFindings(ctx, id)
	return &dtos.ScanResult{
		ID:            scan.ID,
		Status:        scan.Status,
		Progress:      scan.Progress,
		TargetURL:     scan.TargetURL,
		ScanType:      dtos.ScanType(scan.ScanType),
		ErrorMessage:  scan.ErrorMessage,
		FindingsCount: count,
		CreatedAt:     scan.CreatedAt,
	}, nil
}

func (s *ScanService) Findings(ctx context.Context, id string) ([]dtos.Finding, error) {
	if _, err := s.store.Get(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.NotFound("scan not found")
		}
		return nil, apperr.Internal("failed to load scan", err)
	}
	return s.store.ListFindings(ctx, id)
}

func (s *ScanService) run(ctx context.Context, scanID string, in dtos.CreateScanInput) {
	_ = s.store.UpdateStatus(ctx, scanID, "running", 0, "")

	alerts, err := s.zap.Run(ctx, zap.RunRequest{
		TargetURL: in.TargetURL,
		SpecURL:   in.OpenAPIURL,
		ScanType:  string(in.ScanType),
	}, func(p int) {
		_ = s.store.UpdateProgress(ctx, scanID, p)
	})
	if err != nil {
		_ = s.store.UpdateStatus(ctx, scanID, "failed", 0, err.Error())
		return
	}

	findings := normalizeZapAlerts(scanID, alerts)

	for _, f := range checks.PassiveHeaderChecks(ctx, in.TargetURL) {
		f.ScanID = scanID
		findings = append(findings, f)
	}

	if in.Auth != nil && in.Auth.UserB != "" {
		for _, f := range checks.BolaProbe(ctx, in.BolaProbeURLs, checks.BolaAuth{
			HeaderName: in.Auth.HeaderName,
			UserA:      in.Auth.UserA,
			UserB:      in.Auth.UserB,
		}) {
			f.ScanID = scanID
			findings = append(findings, f)
		}
	}

	if err := s.store.SaveFindings(ctx, scanID, findings); err != nil {
		_ = s.store.UpdateStatus(ctx, scanID, "failed", 100, "failed to save findings: "+err.Error())
		return
	}
	_ = s.store.UpdateStatus(ctx, scanID, "completed", 100, "")
}

func normalizeZapAlerts(scanID string, alerts []zap.Alert) []dtos.Finding {
	out := make([]dtos.Finding, 0, len(alerts))
	for _, a := range alerts {
		out = append(out, dtos.Finding{
			ScanID:     scanID,
			RuleID:     a.RuleID,
			Name:       a.Name,
			Severity:   a.Risk,
			Confidence: a.Confidence,
			URL:        a.URL,
			Evidence:   a.Evidence,
			CWE:        a.CWE,
			Source:     "zap",
		})
	}
	return out
}
