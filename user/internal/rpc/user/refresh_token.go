package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	servicetoken "one_adventure_servicekit/token"
)

var errRefreshTokenNotFound = errors.New("refresh token not found")

// RefreshTokenRecord is intentionally independent from the database entity so
// password hashes and future private fields are never written to Redis.
type RefreshTokenRecord struct {
	User      servicetoken.UserInfo `json:"user"`
	ExpiresAt time.Time             `json:"expires_at"`
}

type refreshTokenStore interface {
	Save(ctx context.Context, token string, record RefreshTokenRecord) error
	Get(ctx context.Context, token string) (RefreshTokenRecord, error)
}

type redisRefreshTokenStore struct {
	keyPrefix string
}

func (s redisRefreshTokenStore) Save(ctx context.Context, token string, record RefreshTokenRecord) error {
	payload, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal refresh token record: %w", err)
	}
	ttl := time.Until(record.ExpiresAt)
	seconds := int64(ttl / time.Second)
	if seconds < 1 {
		return fmt.Errorf("refresh token expiration must be in the future")
	}
	if err = g.Redis().SetEX(ctx, s.keyPrefix+token, string(payload), seconds); err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}
	return nil
}

func (s redisRefreshTokenStore) Get(ctx context.Context, token string) (RefreshTokenRecord, error) {
	value, err := g.Redis().Get(ctx, s.keyPrefix+token)
	if err != nil {
		return RefreshTokenRecord{}, fmt.Errorf("get refresh token: %w", err)
	}
	if value == nil || value.IsNil() || value.IsEmpty() {
		return RefreshTokenRecord{}, errRefreshTokenNotFound
	}
	var record RefreshTokenRecord
	if err = json.Unmarshal(value.Bytes(), &record); err != nil {
		return RefreshTokenRecord{}, fmt.Errorf("unmarshal refresh token record: %w", err)
	}
	return record, nil
}
