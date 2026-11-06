package handlers

import (
	"encoding/json"
	"net/http"

	"webtester/internal/auth"
	"webtester/internal/dtos"
	"webtester/internal/services"
	apperr "webtester/pkg/errors"
	"webtester/pkg/response"
)

type AuthHandler struct {
	svc *services.AuthService
}

func NewAuthHandler(svc *services.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	required, err := h.svc.IsSetupRequired(r.Context())
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, http.StatusOK, dtos.SetupStatusResponse{SetupRequired: required})
}

func (h *AuthHandler) Setup(w http.ResponseWriter, r *http.Request) {
	var req dtos.SetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, apperr.Validation("invalid JSON body"))
		return
	}
	if err := validate.Struct(req); err != nil {
		response.Fail(w, apperr.Validation(err.Error()))
		return
	}

	res, err := h.svc.Setup(r.Context(), req)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, res)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dtos.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, apperr.Validation("invalid JSON body"))
		return
	}
	if err := validate.Struct(req); err != nil {
		response.Fail(w, apperr.Validation(err.Error()))
		return
	}

	res, err := h.svc.Login(r.Context(), req)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dtos.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, apperr.Validation("invalid JSON body"))
		return
	}
	if err := validate.Struct(req); err != nil {
		response.Fail(w, apperr.Validation(err.Error()))
		return
	}

	res, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	claims, _ := r.Context().Value(auth.UserContextKey).(*auth.Claims)
	tokenStr, _ := r.Context().Value(auth.TokenContextKey).(string)

	var userID string
	if claims != nil {
		userID = claims.UserID
	}

	if err := h.svc.Logout(r.Context(), tokenStr, userID); err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.UserContextKey).(*auth.Claims)
	if !ok || claims == nil {
		response.Fail(w, apperr.Unauthorized("unauthorized"))
		return
	}

	res, err := h.svc.Me(r.Context(), claims.UserID)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.JSON(w, http.StatusOK, res)
}
