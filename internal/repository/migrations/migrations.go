package migrations

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Migrations struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Migrations {
	return &Migrations{pool: pool}
}

func (m *Migrations) Setup(ctx context.Context) error {
	if err := m.setupLatestTable(ctx); err != nil {
		return fmt.Errorf("setup currency_rate: %w", err)
	}
	if err := m.setupRequestLogTable(ctx); err != nil {
		return fmt.Errorf("setup request_log: %w", err)
	}
	return nil
}

func (m *Migrations) setupLatestTable(ctx context.Context) error {
	_, err := m.pool.Exec(ctx, `
create table if not exists currency_rate (
  base_ccy   char(3) not null,
  quote_ccy  char(3) not null,
  as_of_date date not null,
  rate       numeric(20, 10) not null,
  fetched_at timestamptz not null default now(),
  primary key (base_ccy, quote_ccy)
);

create index if not exists idx_currency_rate_lookup
  on currency_rate (base_ccy, quote_ccy, as_of_date desc);

create index if not exists idx_currency_rate_fetched_at
  on currency_rate (fetched_at desc);
`)
	if err != nil {
		return fmt.Errorf("ensure table currency_rate: %w", err)
	}
	return nil
}

func (m *Migrations) setupRequestLogTable(ctx context.Context) error {
	_, err := m.pool.Exec(ctx, `
create table if not exists request_log (
  id          bigserial primary key,
  path        text not null,
  status      integer,
  date_as_of  date,
  created_at  timestamptz not null default now()
);

create index if not exists idx_request_log_created_at
  on request_log (created_at desc);

create index if not exists idx_request_log_path_created_at
  on request_log (path, created_at desc);
`)
	if err != nil {
		return fmt.Errorf("ensure table request_log: %w", err)
	}
	return nil
}
