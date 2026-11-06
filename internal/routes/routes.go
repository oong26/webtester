package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"webtester/internal/configs"
	"webtester/internal/handlers"
	"webtester/internal/services"
	"webtester/pkg/middleware"
	"webtester/web"
)

func New(_ configs.Config, scanSvc *services.ScanService, loadSvc *services.LoadTestService) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recover)
	r.Use(middleware.AccessLog)

	sh := handlers.NewScanHandler(scanSvc)
	lh := handlers.NewLoadTestHandler(loadSvc)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/scans", sh.List)
		r.Post("/scans", sh.Create)
		r.Get("/scans/{scanId}", sh.Get)
		r.Get("/scans/{scanId}/findings", sh.Findings)

		r.Get("/load-tests", lh.List)
		r.Post("/load-tests", lh.Create)
		r.Get("/load-tests/{id}", lh.Get)
		r.Post("/load-tests/{id}/cancel", lh.Cancel)
	})

	r.Handle("/*", http.FileServer(http.FS(web.FS)))
	return r
}