package auth

import (
	"testing"
	"time"
)

func TestTokenGenerationAndParsing(t *testing.T) {
	cfg := TokenConfig{
		AccessSecret:  "test-access-secret-1234567890",
		RefreshSecret: "test-refresh-secret-1234567890",
		AccessTTL:     5 * time.Minute,
		RefreshTTL:    1 * time.Hour,
	}

	userID := "usr_123"
	email := "test@example.com"

	accessToken, err := GenerateAccessToken(userID, email, cfg)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	claims, err := ParseAccessToken(accessToken, cfg)
	if err != nil {
		t.Fatalf("ParseAccessToken failed: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Errorf("expected Email %s, got %s", email, claims.Email)
	}

	refreshToken, err := GenerateRefreshToken(userID, cfg)
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}

	refreshSub, err := ParseRefreshToken(refreshToken, cfg)
	if err != nil {
		t.Fatalf("ParseRefreshToken failed: %v", err)
	}
	if refreshSub != userID {
		t.Errorf("expected refresh subject %s, got %s", userID, refreshSub)
	}
}

func TestExpiredToken(t *testing.T) {
	cfg := TokenConfig{
		AccessSecret:  "test-secret",
		RefreshSecret: "test-secret-refresh",
		AccessTTL:     -1 * time.Minute,
		RefreshTTL:    -1 * time.Minute,
	}

	token, err := GenerateAccessToken("u1", "u1@example.com", cfg)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	_, err = ParseAccessToken(token, cfg)
	if err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}
