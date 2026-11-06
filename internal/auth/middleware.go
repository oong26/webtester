package auth

import (
	"context"
	"net/http"
	"strings"

	apperr "webtester/pkg/errors"
	"webtester/pkg/response"
)

type contextKey string

const (
	UserContextKey  contextKey = "user"
	TokenContextKey contextKey = "token"
)

type Middleware struct {
	store       *RedisStore
	tokenConfig TokenConfig
}

func NewMiddleware(store *RedisStore, tokenConfig TokenConfig) *Middleware {
	return &Middleware{
		store:       store,
		tokenConfig: tokenConfig,
	}
}

func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			response.Fail(w, apperr.Unauthorized("missing authorization token"))
			return
		}
		blacklisted, err := m.store.IsTokenBlacklisted(r.Context(), token)
		if err != nil {
			response.Fail(w, apperr.Internal("failed to check token", err))
			return
		}
		if blacklisted {
			response.Fail(w, apperr.Unauthorized("token is blacklisted"))
			return
		}
		claims, err := ParseAccessToken(token, m.tokenConfig)
		if err != nil {
			response.Fail(w, apperr.Unauthorized(err.Error()))
			return
		}
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		ctx = context.WithValue(ctx, TokenContextKey, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) GetClaims(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*Claims)
	return claims, ok
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	switch strings.ToLower(parts[0]) {
	case "bearer":
		return strings.TrimSpace(parts[1])
	default:
		return ""
	}
}