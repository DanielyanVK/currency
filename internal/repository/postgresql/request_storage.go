package postgresql

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RequestLogStorage struct {
	pgpool *pgxpool.Pool
}

func NewRequestLogStorage(pgpool *pgxpool.Pool) *RequestLogStorage {
	return &RequestLogStorage{pgpool: pgpool}
}

func (s *RequestLogStorage) Insert(ctx context.Context, path string, status *int, dateAsOf *string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "unknown"
	}

	_, err := s.pgpool.Exec(ctx, `
insert into request_log (path, status, date_as_of)
values ($1, $2, $3::date);
`, path, status, dateAsOf)
	if err != nil {
		return fmt.Errorf("insert request_log: %w", err)
	}
	return nil
}
