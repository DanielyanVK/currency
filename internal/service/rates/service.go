package rates

import (
	"context"
	"fmt"
	"math"
	"service-currency/internal/models"
	"strconv"
	"strings"
)

const decimals = 2

type Storage interface {
	GetLatest(ctx context.Context, base string, quotes []string) ([]models.CurrencyLatestRate, error)
}

type Service struct {
	st Storage
}

func New(st Storage) *Service { return &Service{st: st} }

var allowed = map[string]struct{}{
	"RUB": {}, "USD": {}, "EUR": {}, "JPY": {},
}

type PairRate struct {
	Base  string  `json:"base"`
	Quote string  `json:"quote"`
	Rate  string  `json:"rate"`
	Date  *string `json:"date,omitempty"`
}

func (s *Service) GetPairRate(ctx context.Context, base, quote string) (*PairRate, error) {
	base = strings.ToUpper(strings.TrimSpace(base))
	quote = strings.ToUpper(strings.TrimSpace(quote))

	if _, ok := allowed[base]; !ok {
		return nil, models.BizError("unsupported_currency", "allowed: RUB,USD,EUR,JPY")
	}
	if _, ok := allowed[quote]; !ok {
		return nil, models.BizError("unsupported_currency", "allowed: RUB,USD,EUR,JPY")
	}
	if base == quote {
		return nil, models.BizError("same_currency", "base and quote must be different")
	}

	// 1) RUB to Any
	if base == "RUB" {
		r, err := s.getLatestRUBTo(ctx, quote)
		if err != nil {
			return nil, err
		}
		return &PairRate{
			Base:  base,
			Quote: quote,
			Rate:  r.Rate,
			Date:  r.AsOfDate,
		}, nil
	}

	// 2) Any to RUB
	if quote == "RUB" {
		r, err := s.getLatestRUBTo(ctx, base)
		if err != nil {
			return nil, err
		}

		f, err := parseFloatRate(r.Rate)
		if err != nil {
			return nil, fmt.Errorf("parse rate RUB/%s=%q: %w", base, r.Rate, err)
		}
		if f == 0 {
			return nil, fmt.Errorf("rate RUB/%s is zero, cannot invert", base)
		}

		inv := 1.0 / f
		return &PairRate{
			Base:  base,
			Quote: quote,
			Rate:  formatRate(inv, decimals),
			Date:  r.AsOfDate,
		}, nil
	}

	// 3) Any to Any
	rBase, err := s.getLatestRUBTo(ctx, base)
	if err != nil {
		return nil, err
	}
	rQuote, err := s.getLatestRUBTo(ctx, quote)
	if err != nil {
		return nil, err
	}

	fBase, err := parseFloatRate(rBase.Rate)
	if err != nil {
		return nil, fmt.Errorf("parse rate RUB/%s=%q: %w", base, rBase.Rate, err)
	}
	fQuote, err := parseFloatRate(rQuote.Rate)
	if err != nil {
		return nil, fmt.Errorf("parse rate RUB/%s=%q: %w", quote, rQuote.Rate, err)
	}
	if fBase == 0 {
		return nil, fmt.Errorf("rate RUB/%s is zero, cannot divide", base)
	}

	cross := fQuote / fBase

	return &PairRate{
		Base:  base,
		Quote: quote,
		Rate:  formatRate(cross, decimals),
		Date:  rBase.AsOfDate,
	}, nil
}

func (s *Service) getLatestRUBTo(ctx context.Context, quote string) (*models.CurrencyLatestRate, error) {
	rows, err := s.st.GetLatest(ctx, "RUB", []string{quote})
	if err != nil {
		return nil, fmt.Errorf("get latest RUB/%s: %w", quote, err)
	}
	if len(rows) == 0 {
		return nil, models.BizError("rate_not_available", fmt.Sprintf("latest rate RUB/%s not found in DB", quote))
	}
	r := rows[0]
	return &r, nil
}

func parseFloatRate(s string) (float64, error) {
	s = strings.TrimSpace(s)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("invalid float %q", s)
	}
	return f, nil
}

func formatRate(v float64, decimals int) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	return strconv.FormatFloat(v, 'f', decimals, 64)
}
