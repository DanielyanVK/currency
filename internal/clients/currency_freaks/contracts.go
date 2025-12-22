package currencyFreaks

import (
	"context"
	"service-currency/internal/models"
)

type RatesClient interface {
	LatestRates(ctx context.Context, base string, symbols []string) (*models.LatestRatesResponse, error)
	HistoricalRates(ctx context.Context, date string, base string, symbols []string) (*models.LatestRatesResponse, error)
	FetchAndSaveLatest(ctx context.Context, storage RatesStorage, base string, symbols []string) (*models.LatestRatesResponse, error)
}

type RatesStorage interface {
	UpsertRatesMap(ctx context.Context, base string, asOfDate string, rates map[string]string) error
}
