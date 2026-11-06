package services

import (
	"context"
	"errors"

	"webtester/internal/auth"
	"webtester/internal/dtos"
	apperr "webtester/pkg/errors"
)

type AuthService struct {
	userStore   auth.UserStore
	redisStore  *auth.RedisStore
	tokenConfig auth.TokenConfig
}

func NewAuthService(userStore auth.UserStore, redisStore *auth.RedisStore, tokenConfig auth.TokenConfig) *AuthService {
	return &AuthService{
		userStore:   userStore,
		redisStore:  redisStore,
		tokenConfig: tokenConfig,
	}
}

func (s *AuthService) IsSetupRequired(ctx context.Context) (bool, error) {
	count, err := s.userStore.Count(ctx)
	if err != nil {
		return false, apperr.Internal("failed to check setup status", err)
	}
	return count == 0, nil
}

func (s *AuthService) Setup(ctx context.Context, req dtos.SetupRequest) (*dtos.TokenResponse, error) {
	count, err := s.userStore.Count(ctx)
	if err != nil {
		return nil, apperr.Internal("failed to check existing users", err)
	}
	if count > 0 {
		return nil, apperr.Conflict("initial setup already completed")
	}

	user, err := s.userStore.Create(ctx, req.Email, req.Password, req.Name)
	if err != nil {
		return nil, apperr.Internal("failed to create admin user", err)
	}

	return s.generateSession(ctx, user)
}

func (s *AuthService) Login(ctx context.Context, req dtos.LoginRequest) (*dtos.TokenResponse, error) {
	user, err := s.userStore.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return nil, apperr.Unauthorized("invalid credentials")
		}
		return nil, apperr.Internal("failed to retrieve user", err)
	}

	if err := auth.VerifyPassword(req.Password, user.PasswordHash); err != nil {
		return nil, apperr.Unauthorized("invalid credentials")
	}

	_ = s.userStore.UpdateLastLogin(ctx, user.ID)

	return s.generateSession(ctx, user)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*dtos.TokenResponse, error) {
	userID, err := auth.ParseRefreshToken(refreshToken, s.tokenConfig)
	if err != nil {
		return nil, apperr.Unauthorized("invalid or expired refresh token")
	}

	storedToken, err := s.redisStore.GetRefreshToken(ctx, userID)
	if err != nil || storedToken != refreshToken {
		return nil, apperr.Unauthorized("refresh token revoked or not recognized")
	}

	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		return nil, apperr.Unauthorized("user not found")
	}

	return s.generateSession(ctx, user)
}

func (s *AuthService) Logout(ctx context.Context, accessToken, userID string) error {
	if userID != "" {
		_ = s.redisStore.DeleteRefreshToken(ctx, userID)
	}
	if accessToken != "" {
		_ = s.redisStore.BlacklistToken(ctx, accessToken, s.tokenConfig.AccessTTL)
	}
	return nil
}

func (s *AuthService) Me(ctx context.Context, userID string) (*dtos.UserResponse, error) {
	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return nil, apperr.NotFound("user not found")
		}
		return nil, apperr.Internal("failed to fetch user", err)
	}
	return toUserResponse(user), nil
}

func (s *AuthService) generateSession(ctx context.Context, user *auth.User) (*dtos.TokenResponse, error) {
	accessToken, err := auth.GenerateAccessToken(user.ID, user.Email, s.tokenConfig)
	if err != nil {
		return nil, apperr.Internal("failed to generate access token", err)
	}

	refreshToken, err := auth.GenerateRefreshToken(user.ID, s.tokenConfig)
	if err != nil {
		return nil, apperr.Internal("failed to generate refresh token", err)
	}

	if err := s.redisStore.SaveRefreshToken(ctx, user.ID, refreshToken, s.tokenConfig.RefreshTTL); err != nil {
		return nil, apperr.Internal("failed to store session in cache", err)
	}

	return &dtos.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.tokenConfig.AccessTTL.Seconds()),
		User:         *toUserResponse(user),
	}, nil
}

func toUserResponse(u *auth.User) *dtos.UserResponse {
	return &dtos.UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
		LastLogin: u.LastLoginAt,
	}
}
