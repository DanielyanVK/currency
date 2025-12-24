package postgresql

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type APIKeyStorage struct {
	pool *pgxpool.Pool
}

func NewAPIKeyStorage(pool *pgxpool.Pool) *APIKeyStorage {
	return &APIKeyStorage{pool: pool}
}

func (s *APIKeyStorage) GetStatusByHash(ctx context.Context, keyHash string) (exists bool, isActive bool, err error) {
	keyHash = strings.TrimSpace(keyHash)
	if keyHash == "" {
		return false, false, nil
	}

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
