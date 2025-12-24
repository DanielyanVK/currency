package postgresql

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"service-currency/internal"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type CurrencyStorage struct {
	pgpool *pgxpool.Pool
}

func NewCurrencyStorage(pgpool *pgxpool.Pool) *CurrencyStorage {
	return &CurrencyStorage{pgpool: pgpool}
}

func (c *CurrencyStorage) UpsertRatesMap(
	ctx context.Context,
	base internal.CurrencyCode,
	asOfDate internal.Date,
	rates map[internal.CurrencyCode]decimal.Decimal,
) error {
	baseStr := strings.ToUpper(strings.TrimSpace(base.String()))
	if baseStr == "" {
		return fmt.Errorf("base currency is empty")
	}
	if asOfDate.IsZero() {
		return fmt.Errorf("as_of_date is empty")
	}

	asOf := time.Date(asOfDate.Year(), asOfDate.Month(), asOfDate.Day(), 0, 0, 0, 0, time.UTC)

	tx, err := c.pgpool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for quote, rate := range rates {
		quoteStr := strings.ToUpper(strings.TrimSpace(quote.String()))

		if quoteStr == "" || quoteStr == baseStr {
			continue
		}

		_, err := tx.Exec(ctx, `
insert into currency_rate (base_ccy, quote_ccy, as_of_date, rate, fetched_at)
values ($1, $2, $3::date, $4::numeric, now())
on conflict (base_ccy, quote_ccy)
do update set
  as_of_date = excluded.as_of_date,
  rate = excluded.rate,
  fetched_at = now();
`, baseStr, quoteStr, asOf, rate.String())
		if err != nil {
			return fmt.Errorf("upsert %s/%s=%q @%s: %w", baseStr, quoteStr, rate.String(), asOf.Format("2006-01-02"), err)
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
	base internal.CurrencyCode,
	quotes []internal.CurrencyCode,
) ([]internal.CurrencyLatestRate, error) {
	baseStr := strings.ToUpper(strings.TrimSpace(base.String()))
	if baseStr == "" {
		return nil, errors.New("base currency is empty")
	}

	if len(quotes) == 0 {
		rows, err := c.pgpool.Query(ctx, `
select distinct on (quote_ccy)
  base_ccy,
  quote_ccy,
  rate::text,
  as_of_date,
  fetched_at
from currency_rate
where base_ccy = $1
order by quote_ccy, as_of_date desc, fetched_at desc;
`, baseStr)
		if err != nil {
			return nil, fmt.Errorf("query latest rates: %w", err)
		}
		defer rows.Close()

		var out []internal.CurrencyLatestRate
		for rows.Next() {
			var r internal.CurrencyLatestRate
			var bRaw, qRaw, rateText string
			var asOf time.Time

			if err := rows.Scan(&bRaw, &qRaw, &rateText, &asOf, &r.FetchedAt); err != nil {
				return nil, fmt.Errorf("scan: %w", err)
			}

			b, err := internal.NewCurrencyCode(strings.TrimSpace(bRaw))
			if err != nil {
				return nil, fmt.Errorf("bad base_ccy from db %q: %w", bRaw, err)
			}
			q, err := internal.NewCurrencyCode(strings.TrimSpace(qRaw))
			if err != nil {
				return nil, fmt.Errorf("bad quote_ccy from db %q: %w", qRaw, err)
			}

			r.BaseCCY = b
			r.QuoteCCY = q

			rateText = strings.TrimSpace(rateText)
			if rateText == "" {
				return nil, fmt.Errorf("empty rate for %s/%s", r.BaseCCY, r.QuoteCCY)
			}
			rate, err := decimal.NewFromString(rateText)
			if err != nil {
				return nil, fmt.Errorf("parse rate %s/%s=%q: %w", r.BaseCCY, r.QuoteCCY, rateText, err)
			}
			r.Rate = rate

			d := internal.Date{Time: time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, time.UTC)}
			r.AsOfDate = &d

			out = append(out, r)
		}
		return out, rows.Err()
	}

	norm := make([]string, 0, len(quotes))
	for _, q := range quotes {
		qs := strings.ToUpper(strings.TrimSpace(q.String()))
		if qs != "" && qs != baseStr {
			norm = append(norm, qs)
		}
	}
	if len(norm) == 0 {
		return []internal.CurrencyLatestRate{}, nil
	}

	rows, err := c.pgpool.Query(ctx, `
select distinct on (quote_ccy)
  base_ccy,
  quote_ccy,
  rate::text,
  as_of_date,
  fetched_at
from currency_rate
where base_ccy = $1 and quote_ccy = any($2)
order by quote_ccy, as_of_date desc, fetched_at desc;
`, baseStr, norm)
	if err != nil {
		return nil, fmt.Errorf("query latest rates: %w", err)
	}
	defer rows.Close()

	var out []internal.CurrencyLatestRate
	for rows.Next() {
		var r internal.CurrencyLatestRate
		var bRaw, qRaw, rateText string
		var asOf time.Time

		if err := rows.Scan(&bRaw, &qRaw, &rateText, &asOf, &r.FetchedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		b, err := internal.NewCurrencyCode(strings.TrimSpace(bRaw))
		if err != nil {
			return nil, fmt.Errorf("bad base_ccy from db %q: %w", bRaw, err)
		}
		q, err := internal.NewCurrencyCode(strings.TrimSpace(qRaw))
		if err != nil {
			return nil, fmt.Errorf("bad quote_ccy from db %q: %w", qRaw, err)
		}

		r.BaseCCY = b
		r.QuoteCCY = q

		rateText = strings.TrimSpace(rateText)
		if rateText == "" {
			return nil, fmt.Errorf("empty rate for %s/%s", r.BaseCCY, r.QuoteCCY)
		}
		rate, err := decimal.NewFromString(rateText)
		if err != nil {
			return nil, fmt.Errorf("parse rate %s/%s=%q: %w", r.BaseCCY, r.QuoteCCY, rateText, err)
		}
		r.Rate = rate

		d := internal.Date{Time: time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, time.UTC)}
		r.AsOfDate = &d

		out = append(out, r)
	}
	return out, rows.Err()
}
