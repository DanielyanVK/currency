package currencyFreaks_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"service-currency/internal"
	currencyFreaks "service-currency/internal/currency_freaks"
	"service-currency/internal/currency_freaks/mock"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestClient_LatestRates_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := internal.LatestRatesResponse{
			Date: internal.Date{Time: time.Date(2024, 12, 26, 0, 0, 0, 0, time.UTC)},
			Base: "RUB",
			Rates: map[string]string{
				"USD": "0.0105",
				"EUR": "0.0095",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
	defer server.Close()

	client := currencyFreaks.New("test-api-key", nil)
	client.BaseURL = server.URL

	result, err := client.LatestRates(
		context.Background(),
		internal.RUB,
		[]internal.CurrencyCode{internal.USD, internal.EUR},
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "RUB", result.Base)
	assert.Equal(t, "0.0105", result.Rates["USD"])
	assert.Equal(t, "0.0095", result.Rates["EUR"])
}

func TestClient_LatestRates_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte("internal server error"))
		require.NoError(t, err)
	}))
	defer server.Close()

	client := currencyFreaks.New("test-api-key", nil)
	client.BaseURL = server.URL

	result, err := client.LatestRates(
		context.Background(),
		internal.RUB,
		[]internal.CurrencyCode{internal.USD},
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "500")
	assert.Contains(t, err.Error(), "internal server error")
}

func TestClient_HistoricalRates_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rates/historical", r.URL.Path)
		assert.Equal(t, "2024-12-25", r.URL.Query().Get("date"))
		assert.Equal(t, "test-key", r.URL.Query().Get("apikey"))

		resp := internal.LatestRatesResponse{
			Date: internal.Date{Time: time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC)},
			Base: "RUB",
			Rates: map[string]string{
				"USD": "0.0103",
			},
		}
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
	defer server.Close()

	client := currencyFreaks.New("test-key", nil)
	client.BaseURL = server.URL

	date := internal.Date{Time: time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC)}
	result, err := client.HistoricalRates(
		context.Background(),
		date,
		internal.RUB,
		[]internal.CurrencyCode{internal.USD},
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "RUB", result.Base)
	assert.Equal(t, "0.0103", result.Rates["USD"])
}

func TestClient_HistoricalRates_EmptyDate(t *testing.T) {
	client := currencyFreaks.New("test-api-key", nil)

	result, err := client.HistoricalRates(
		context.Background(),
		internal.Date{},
		internal.RUB,
		[]internal.CurrencyCode{internal.USD},
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "date is empty")
}

func TestClient_FetchAndSaveLatest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := internal.LatestRatesResponse{
			Date: internal.Date{Time: time.Date(2024, 12, 26, 0, 0, 0, 0, time.UTC)},
			Base: "RUB",
			Rates: map[string]string{
				"USD": "0.0105",
				"EUR": "0.0095",
			},
		}
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
	defer server.Close()

	mockStorage := mock.NewMockRatesStorage(t)

	mockStorage.EXPECT().
		UpsertRatesMap(
			testifymock.Anything,
			internal.RUB,
			internal.Date{Time: time.Date(2024, 12, 26, 0, 0, 0, 0, time.UTC)},
			testifymock.MatchedBy(func(rates map[internal.CurrencyCode]decimal.Decimal) bool {
				usd, hasUSD := rates[internal.USD]
				eur, hasEUR := rates[internal.EUR]
				return hasUSD && hasEUR &&
					usd.Equal(decimal.RequireFromString("0.0105")) &&
					eur.Equal(decimal.RequireFromString("0.0095"))
			}),
		).
		Return(nil).
		Once()

	client := currencyFreaks.New("test-api-key", mockStorage)
	client.BaseURL = server.URL

	result, err := client.FetchAndSaveLatest(
		context.Background(),
		mockStorage,
		internal.RUB,
		[]internal.CurrencyCode{internal.USD, internal.EUR},
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "RUB", result.Base)
	assert.Equal(t, 2, len(result.Rates))
	mockStorage.AssertExpectations(t)
}

func TestClient_FetchAndSaveLatest_InvalidCurrency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := internal.LatestRatesResponse{
			Date: internal.Date{Time: time.Date(2024, 12, 26, 0, 0, 0, 0, time.UTC)},
			Base: "RUB",
			Rates: map[string]string{
				"XXX": "0.0105",
			},
		}
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
	defer server.Close()

	mockStorage := mock.NewMockRatesStorage(t)
	client := currencyFreaks.New("test-api-key", mockStorage)
	client.BaseURL = server.URL

	result, err := client.FetchAndSaveLatest(
		context.Background(),
		mockStorage,
		internal.RUB,
		[]internal.CurrencyCode{internal.USD},
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid quote")
}
