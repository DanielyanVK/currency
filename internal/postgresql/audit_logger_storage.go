package postgresql

import (
	"context"
	"fmt"
	"service-currency/internal"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RequestLogStorage struct {
	pgpool *pgxpool.Pool
}

func NewRequestLogStorage(pgpool *pgxpool.Pool) *RequestLogStorage {
	return &RequestLogStorage{pgpool: pgpool}
}

func (s *RequestLogStorage) Insert(ctx context.Context, path string, status *int, dateAsOf *internal.Date) error {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "unknown"
	}

	var asOf *time.Time
	if dateAsOf != nil && !dateAsOf.IsZero() {
		t := time.Date(dateAsOf.Year(), dateAsOf.Month(), dateAsOf.Day(), 0, 0, 0, 0, time.UTC)
		asOf = &t
	}

	_, err := s.pgpool.Exec(ctx, `
insert into request_log (path, status, date_as_of)
values ($1, $2, $3::date);
`, path, status, asOf)
	if err != nil {
		return fmt.Errorf("insert request_log: %w", err)
	}
	return nil
}
