package postgresql

import (
	"context"
	"fmt"
	"service-currency/internal/models"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const baseCCY = "RUB"

type CurrencyStorage struct {
	pgpool *pgxpool.Pool
}

func NewCurrencyStorage(pgpool *pgxpool.Pool) *CurrencyStorage {
	return &CurrencyStorage{pgpool: pgpool}
}

func (c *CurrencyStorage) UpsertRatesMap(
	ctx context.Context,
	base string,
	asOfDate string,
	rates map[string]string,
) error {
	base = strings.ToUpper(strings.TrimSpace(base))
	if base == "" {
		base = baseCCY
	}

	asOf := strings.TrimSpace(asOfDate)
	if asOf == "" {
		return fmt.Errorf("as_of_date is empty")
	}
	if len(asOf) >= 10 {
		asOf = asOf[:10]
	}
	if len(asOf) != 10 {
		return fmt.Errorf("as_of_date must be YYYY-MM-DD, got %q", asOf)
	}

	tx, err := c.pgpool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for quote, rate := range rates {
		quote = strings.ToUpper(strings.TrimSpace(quote))
		rate = strings.TrimSpace(rate)

		if quote == "" || quote == base {
			continue
		}
		if rate == "" {
			return fmt.Errorf("empty rate for %s/%s", base, quote)
		}

		_, err := tx.Exec(ctx, `
insert into currency_rate (base_ccy, quote_ccy, as_of_date, rate, fetched_at)
values ($1, $2, $3::date, $4::numeric, now())
on conflict (base_ccy, quote_ccy, as_of_date)
do update set
  rate = excluded.rate,
  fetched_at = now();
`, base, quote, asOf, rate)
		if err != nil {
			return fmt.Errorf("upsert %s/%s=%q @%s: %w", base, quote, rate, asOf, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// GetLatest возвращает последний (по as_of_date) курс для каждой quote.
// Если quotes пустой — вернёт все quotes, которые есть в БД для base.
func (c *CurrencyStorage) GetLatest(
	ctx context.Context,
	base string,
	quotes []string,
) ([]models.CurrencyLatestRate, error) {
	base = strings.ToUpper(strings.TrimSpace(base))
	if base == "" {
		base = baseCCY
	}

	if len(quotes) == 0 {
		rows, err := c.pgpool.Query(ctx, `
select distinct on (quote_ccy)
  base_ccy,
  quote_ccy,
  rate::text,
  to_char(as_of_date, 'YYYY-MM-DD') as as_of_date,
  fetched_at
from currency_rate
where base_ccy = $1
order by quote_ccy, as_of_date desc, fetched_at desc;
`, base)
		if err != nil {
			return nil, fmt.Errorf("query latest rates: %w", err)
		}
		defer rows.Close()

		var out []models.CurrencyLatestRate
		for rows.Next() {
			var r models.CurrencyLatestRate
			var b, q, rate, asOf string
			if err := rows.Scan(&b, &q, &rate, &asOf, &r.FetchedAt); err != nil {
				return nil, fmt.Errorf("scan: %w", err)
			}
			r.BaseCCY = strings.TrimSpace(b)
			r.QuoteCCY = strings.TrimSpace(q)
			r.Rate = rate
			asOfCopy := asOf
			r.AsOfDate = &asOfCopy
			out = append(out, r)
		}
		return out, rows.Err()
	}

	norm := make([]string, 0, len(quotes))
	for _, q := range quotes {
		q = strings.ToUpper(strings.TrimSpace(q))
		if q != "" && q != base {
			norm = append(norm, q)
		}
	}

	rows, err := c.pgpool.Query(ctx, `
select distinct on (quote_ccy)
  base_ccy,
  quote_ccy,
  rate::text,
  to_char(as_of_date, 'YYYY-MM-DD') as as_of_date,
  fetched_at
from currency_rate
where base_ccy = $1 and quote_ccy = any($2)
order by quote_ccy, as_of_date desc, fetched_at desc;
`, base, norm)
	if err != nil {
		return nil, fmt.Errorf("query latest rates: %w", err)
	}
	defer rows.Close()

	var out []models.CurrencyLatestRate
	for rows.Next() {
		var r models.CurrencyLatestRate
		var b, q, rate, asOf string
		if err := rows.Scan(&b, &q, &rate, &asOf, &r.FetchedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		r.BaseCCY = strings.TrimSpace(b)
		r.QuoteCCY = strings.TrimSpace(q)
		r.Rate = rate
		asOfCopy := asOf
		r.AsOfDate = &asOfCopy
		out = append(out, r)
	}
	return out, rows.Err()
}
