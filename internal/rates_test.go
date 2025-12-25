package internal_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"service-currency/internal"
	"service-currency/internal/mock"
)

func TestRateConverter_GetPairRate_RUBToUSD(t *testing.T) {
	mockStorage := mock.NewMockStorage(t)

	rate, _ := decimal.NewFromString("0.0105")
	mockStorage.EXPECT().
		GetLatest(testifymock.Anything, internal.RUB, []internal.CurrencyCode{internal.USD}).
		Return([]internal.CurrencyLatestRate{
			{
				BaseCCY:   internal.RUB,
				QuoteCCY:  internal.USD,
				Rate:      rate,
				FetchedAt: time.Now(),
			},
		}, nil).
		Once()

	converter := internal.NewRateConverter(mockStorage)
	result, err := converter.GetPairRate(context.Background(), internal.RUB, internal.USD)

	require.NoError(t, err)
	assert.Equal(t, internal.RUB, result.Base)
	assert.Equal(t, internal.USD, result.Quote)
	assert.Equal(t, "0.0105", result.Rate.StringFixed(4))
}

func TestRateConverter_GetPairRate_USDToRUB(t *testing.T) {
	mockStorage := mock.NewMockStorage(t)

	rate, _ := decimal.NewFromString("0.01")
	mockStorage.EXPECT().
		GetLatest(testifymock.Anything, internal.RUB, []internal.CurrencyCode{internal.USD}).
		Return([]internal.CurrencyLatestRate{
			{
				BaseCCY:   internal.RUB,
				QuoteCCY:  internal.USD,
				Rate:      rate,
				FetchedAt: time.Now(),
			},
		}, nil).
		Once()

	converter := internal.NewRateConverter(mockStorage)
	result, err := converter.GetPairRate(context.Background(), internal.USD, internal.RUB)

	require.NoError(t, err)
	assert.Equal(t, internal.USD, result.Base)
	assert.Equal(t, internal.RUB, result.Quote)
	assert.Equal(t, "100.0000", result.Rate.StringFixed(4))
}

func TestRateConverter_GetPairRate_USDToEUR(t *testing.T) {
	mockStorage := mock.NewMockStorage(t)

	rateUSD, _ := decimal.NewFromString("0.01")
	rateEUR, _ := decimal.NewFromString("0.0095")

	mockStorage.EXPECT().
		GetLatest(testifymock.Anything, internal.RUB, []internal.CurrencyCode{internal.USD}).
		Return([]internal.CurrencyLatestRate{
			{BaseCCY: internal.RUB, QuoteCCY: internal.USD, Rate: rateUSD, FetchedAt: time.Now()},
		}, nil).
		Once()

	mockStorage.EXPECT().
		GetLatest(testifymock.Anything, internal.RUB, []internal.CurrencyCode{internal.EUR}).
		Return([]internal.CurrencyLatestRate{
			{BaseCCY: internal.RUB, QuoteCCY: internal.EUR, Rate: rateEUR, FetchedAt: time.Now()},
		}, nil).
		Once()

	converter := internal.NewRateConverter(mockStorage)
	result, err := converter.GetPairRate(context.Background(), internal.USD, internal.EUR)

	require.NoError(t, err)
	assert.Equal(t, internal.USD, result.Base)
	assert.Equal(t, internal.EUR, result.Quote)
	assert.Equal(t, "0.9500", result.Rate.StringFixed(4))
}

func TestRateConverter_GetPairRate_StorageError(t *testing.T) {
	mockStorage := mock.NewMockStorage(t)

	mockStorage.EXPECT().
		GetLatest(testifymock.Anything, internal.RUB, []internal.CurrencyCode{internal.USD}).
		Return(nil, errors.New("database error")).
		Once()

	converter := internal.NewRateConverter(mockStorage)
	_, err := converter.GetPairRate(context.Background(), internal.RUB, internal.USD)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestRateConverter_GetPairRate_UnsupportedCurrency(t *testing.T) {
	mockStorage := mock.NewMockStorage(t)
	converter := internal.NewRateConverter(mockStorage)

	unsupported := internal.CurrencyCode("XXX")
	_, err := converter.GetPairRate(context.Background(), unsupported, internal.USD)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported currency")
}
