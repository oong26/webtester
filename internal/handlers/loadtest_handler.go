package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"webtester/internal/dtos"
	"webtester/internal/services"
	apperr "webtester/pkg/errors"
	"webtester/pkg/response"
)

type LoadTestHandler struct {
	svc *services.LoadTestService
}

func NewLoadTestHandler(svc *services.LoadTestService) *LoadTestHandler {
	return &LoadTestHandler{svc: svc}
}

func (h *LoadTestHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dtos.CreateLoadTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, apperr.Validation("invalid JSON body"))
		return
	}
	if err := validate.Struct(req); err != nil {
		response.Fail(w, apperr.Validation(err.Error()))
		return
	}
	if req.Headers != nil {
		for k := range req.Headers {
			if !isValidHeaderName(k) {
				response.Fail(w, apperr.Validation(fmt.Sprintf("invalid header field name: %q", k)))
				return
			}
		}
	}
	res, err := h.svc.Create(r.Context(), toLoadTestInput(req))
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, http.StatusAccepted, res)
}

func (h *LoadTestHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := h.svc.Get(r.Context(), id)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *LoadTestHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	res, err := h.svc.List(r.Context(), limit, offset)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *LoadTestHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Cancel(r.Context(), id); err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "cancelling"})
}

func toLoadTestInput(req dtos.CreateLoadTestRequest) dtos.CreateLoadTestInput {
	return dtos.CreateLoadTestInput{
		TargetURL:        req.TargetURL,
		Method:           req.Method,
		Headers:          req.Headers,
		Body:             req.Body,
		DurationSeconds:  req.DurationSeconds,
		RequestsPerSec:   req.RequestsPerSec,
		Concurrency:      req.Concurrency,
		ConfirmOwnership: req.ConfirmOwnership,
	}
}

// isValidHeaderName reports whether name is a valid HTTP token (RFC 7230 §3.2.6).
func isValidHeaderName(name string) bool {
	if name == "" {
		return false
	}
	return strings.IndexFunc(name, func(r rune) bool {
		switch {
		case r >= 'a' && r <= 'z':
			return false
		case r >= 'A' && r <= 'Z':
			return false
		case r >= '0' && r <= '9':
			return false
		case strings.ContainsRune("!#$%&'*+-.^_`|~", r):
			return false
		default:
			return true
		}
	}) == -1
}