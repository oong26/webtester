package auth

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(addr, password string, db int) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &RedisStore{client: client}, nil
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}

func (s *RedisStore) SaveRefreshToken(ctx context.Context, userID, token string, ttl time.Duration) error {
	key := "refresh:" + userID
	return s.client.Set(ctx, key, token, ttl).Err()
}

func (s *RedisStore) GetRefreshToken(ctx context.Context, userID string) (string, error) {
	key := "refresh:" + userID
	return s.client.Get(ctx, key).Result()
}

func (s *RedisStore) DeleteRefreshToken(ctx context.Context, userID string) error {
	key := "refresh:" + userID
	return s.client.Del(ctx, key).Err()
}

func (s *RedisStore) BlacklistToken(ctx context.Context, token string, ttl time.Duration) error {
	key := "blacklist:" + token
	return s.client.Set(ctx, key, "1", ttl).Err()
}

func (s *RedisStore) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	key := "blacklist:" + token
	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val == "1", nil
}

type SessionData struct {
	UserID    string            `json:"user_id"`
	Email     string            `json:"email"`
	IP        string            `json:"ip"`
	UserAgent string            `json:"user_agent"`
	Metadata  map[string]string `json:"metadata"`
}

func (s *RedisStore) SaveSession(ctx context.Context, sessionID string, data SessionData, ttl time.Duration) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	key := "session:" + sessionID
	return s.client.Set(ctx, key, jsonData, ttl).Err()
}

func (s *RedisStore) GetSession(ctx context.Context, sessionID string) (*SessionData, error) {
	key := "session:" + sessionID
	data, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var session SessionData
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *RedisStore) DeleteSession(ctx context.Context, sessionID string) error {
	key := "session:" + sessionID
	return s.client.Del(ctx, key).Err()
}

func (s *RedisStore) RateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	rkey := "ratelimit:" + key
	current, err := s.client.Incr(ctx, rkey).Result()
	if err != nil {
		return false, err
	}
	if current == 1 {
		s.client.Expire(ctx, rkey, window)
	}
	return current <= int64(limit), nil
}