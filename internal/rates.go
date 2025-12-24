package internal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type LatestRatesResponse struct {
	Date  Date              `json:"date"`
	Base  string            `json:"base"`
	Rates map[string]string `json:"rates"`
}

type CurrencyLatestRate struct {
	BaseCCY   CurrencyCode
	QuoteCCY  CurrencyCode
	Rate      decimal.Decimal
	AsOfDate  *Date
	FetchedAt time.Time
}

type Storage interface {
	GetLatest(ctx context.Context, base CurrencyCode, quotes []CurrencyCode) ([]CurrencyLatestRate, error)
}

type RateConverter struct {
	storage Storage
}

func NewRateConverter(storage Storage) *RateConverter { return &RateConverter{storage: storage} }

type PairRate struct {
	Base  CurrencyCode    `json:"base"`
	Quote CurrencyCode    `json:"quote"`
	Rate  decimal.Decimal `json:"rate"`
	Date  *Date           `json:"date,omitempty"`
}

func (s *RateConverter) GetPairRate(ctx context.Context, base, quote CurrencyCode) (PairRate, error) {
	if !base.IsSupported() {
		return PairRate{}, errors.New("unsupported currency")
	}
	if !quote.IsSupported() {
		return PairRate{}, errors.New("unsupported currency")
	}

	// 1) RUB -> Any
	if base == RUB {
		r, err := s.getLatestRUBTo(ctx, quote)
		if err != nil {
			return PairRate{}, err
		}
		return PairRate{Base: base, Quote: quote, Rate: r.Rate, Date: r.AsOfDate}, nil
	}

	// 2) Any -> RUB
	if quote == RUB {
		r, err := s.getLatestRUBTo(ctx, base) // RUB->base
		if err != nil {
			return PairRate{}, err
		}
		if r.Rate.IsZero() {
			return PairRate{}, fmt.Errorf("rate %s/%s is zero, cannot invert", RUB, base)
		}

		inv := decimal.NewFromInt(1).Div(r.Rate) // base->RUB
		return PairRate{Base: base, Quote: quote, Rate: inv, Date: r.AsOfDate}, nil
	}

	// 3) Any -> Any (через RUB)
	rBase, err := s.getLatestRUBTo(ctx, base)
	if err != nil {
		return PairRate{}, err
	}
	rQuote, err := s.getLatestRUBTo(ctx, quote)
	if err != nil {
		return PairRate{}, err
	}
	if rBase.Rate.IsZero() {
		return PairRate{}, fmt.Errorf("rate %s/%s is zero, cannot divide", RUB, base)
	}

	cross := rQuote.Rate.Div(rBase.Rate) // base -> quote
	return PairRate{Base: base, Quote: quote, Rate: cross, Date: rBase.AsOfDate}, nil
}

func (s *RateConverter) getLatestRUBTo(ctx context.Context, quote CurrencyCode) (CurrencyLatestRate, error) {
	rows, err := s.storage.GetLatest(ctx, RUB, []CurrencyCode{quote})
	if err != nil {
		return CurrencyLatestRate{}, fmt.Errorf("get latest %s/%s: %w", RUB, quote, err)
	}
	if len(rows) == 0 {
		return CurrencyLatestRate{}, errors.New("rate not available")
	}
	return rows[0], nil
}
