package postgresql

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type APIKeyStorage struct {
	pool        *pgxpool.Pool
	encodingKey string
}

func New(pool *pgxpool.Pool, encodingKey string) *APIKeyStorage {
	return &APIKeyStorage{
		pool:        pool,
		encodingKey: strings.TrimSpace(encodingKey),
	}
}

func (s *APIKeyStorage) Validate(ctx context.Context, rawKey string) (exists bool, isActive bool, err error) {
	rawKey = strings.TrimSpace(rawKey)
	if rawKey == "" {
		return false, false, nil
	}

	keyHash := hashKey(rawKey, s.encodingKey)
	fmt.Println("DEBUG api key hash:", keyHash, "encodingKey:", s.encodingKey, "rawKey:", rawKey)

	err = s.pool.QueryRow(ctx, `
select is_active
from api_keys
where key_hash = $1;
`, keyHash).Scan(&isActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, nil
		}
		return false, false, fmt.Errorf("select api_keys: %w", err)
	}

	return true, isActive, nil
}

func hashKey(rawKey, encodingKey string) string {
	mac := hmac.New(sha256.New, []byte(encodingKey))
	_, _ = mac.Write([]byte(rawKey))
	return hex.EncodeToString(mac.Sum(nil))
}
