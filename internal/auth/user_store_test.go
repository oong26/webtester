package auth

import (
	"context"
	"os"
	"testing"
)

func TestSQLiteUserStore(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "user_test_*.db")
	if err != nil {
		t.Fatalf("temp file error: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(dbPath)

	store, err := NewUserStore(dbPath)
	if err != nil {
		t.Fatalf("NewUserStore error: %v", err)
	}

	ctx := context.Background()

	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("Count error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}

	user, err := store.Create(ctx, "admin@example.com", "securepassword123", "Admin")
	if err != nil {
		t.Fatalf("Create user error: %v", err)
	}
	if user.Email != "admin@example.com" {
		t.Errorf("expected email admin@example.com, got %s", user.Email)
	}

	count, err = store.Count(ctx)
	if err != nil {
		t.Fatalf("Count error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	if err := VerifyPassword("securepassword123", user.PasswordHash); err != nil {
		t.Errorf("password verification failed: %v", err)
	}
	if err := VerifyPassword("wrongpassword", user.PasswordHash); err == nil {
		t.Errorf("expected error for wrong password, got nil")
	}

	fetched, err := store.GetByEmail(ctx, "admin@example.com")
	if err != nil {
		t.Fatalf("GetByEmail error: %v", err)
	}
	if fetched.ID != user.ID {
		t.Errorf("expected ID %s, got %s", user.ID, fetched.ID)
	}
}
