package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	_ "modernc.org/sqlite"
)

//go:embed user_schema.sql
var userSchemaFS embed.FS

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrUserExists      = errors.New("user already exists")
	ErrInvalidPassword = errors.New("invalid password")
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastLoginAt  *time.Time
}

type UserStore interface {
	Create(ctx context.Context, email, password, name string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	UpdatePassword(ctx context.Context, id, newPasswordHash string) error
	UpdateLastLogin(ctx context.Context, id string) error
	Count(ctx context.Context) (int, error)
}

type sqliteUserStore struct {
	db *sql.DB
}

func NewUserStore(dbPath string) (UserStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, err
	}
	schema, err := userSchemaFS.ReadFile("user_schema.sql")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(string(schema)); err != nil {
		return nil, err
	}
	return &sqliteUserStore{db: db}, nil
}

func (s *sqliteUserStore) Create(ctx context.Context, email, password, name string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	id := newID()
	now := time.Now().UTC()
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, name, role, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, email, string(hash), name, "admin", now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *sqliteUserStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, name, role, created_at, updated_at, last_login_at
		 FROM users WHERE email = ?`, email)
	return scanUser(row)
}

func (s *sqliteUserStore) GetByID(ctx context.Context, id string) (*User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, name, role, created_at, updated_at, last_login_at
		 FROM users WHERE id = ?`, id)
	return scanUser(row)
}

func (s *sqliteUserStore) UpdatePassword(ctx context.Context, id, newPasswordHash string) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`,
		newPasswordHash, now.Format(time.RFC3339Nano), id)
	return err
}

func (s *sqliteUserStore) UpdateLastLogin(ctx context.Context, id string) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET last_login_at = ?, updated_at = ? WHERE id = ?`,
		now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), id)
	return err
}

func (s *sqliteUserStore) Count(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

func scanUser(row interface {
	Scan(dest ...interface{}) error
}) (*User, error) {
	var (
		u           User
		createdStr  string
		updatedStr  string
		lastLoginStr sql.NullString
	)
	if err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role,
		&createdStr, &updatedStr, &lastLoginStr,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdStr)
	u.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedStr)
	if lastLoginStr.Valid {
		t, _ := time.Parse(time.RFC3339Nano, lastLoginStr.String)
		u.LastLoginAt = &t
	}
	return &u, nil
}

func newID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func VerifyPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}