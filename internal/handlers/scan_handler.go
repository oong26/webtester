package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"webtester/internal/dtos"
	"webtester/internal/services"
	apperr "webtester/pkg/errors"
	"webtester/pkg/response"
)

var validate = validator.New()

type ScanHandler struct {
	svc *services.ScanService
}

func NewScanHandler(svc *services.ScanService) *ScanHandler {
	return &ScanHandler{svc: svc}
}

func (h *ScanHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	res, err := h.svc.List(r.Context(), limit, offset)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *ScanHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dtos.CreateScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, apperr.Validation("invalid JSON body"))
		return
	}
	if err := validate.Struct(req); err != nil {
		response.Fail(w, apperr.Validation(err.Error()))
		return
	}
	res, err := h.svc.Create(r.Context(), toInput(req))
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, http.StatusAccepted, res)
}

func (h *ScanHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "scanId")
	res, err := h.svc.Get(r.Context(), id)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *ScanHandler) Findings(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "scanId")
	res, err := h.svc.Findings(r.Context(), id)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, http.StatusOK, res)
}

func toInput(req dtos.CreateScanRequest) dtos.CreateScanInput {
	return dtos.CreateScanInput{
		TargetURL:        req.TargetURL,
		OpenAPIURL:       req.OpenAPIURL,
		ScanType:         req.ScanType,
		ConfirmOwnership: req.ConfirmOwnership,
		Auth:             req.Auth,
		BolaProbeURLs:    req.BolaProbeURLs,
	}
}
