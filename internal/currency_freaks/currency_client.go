package currencyFreaks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"service-currency/internal"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const maxBodyBytes = 32 << 10

type RatesClient interface {
	LatestRates(ctx context.Context, base internal.CurrencyCode, symbols []internal.CurrencyCode) (*internal.LatestRatesResponse, error)
	HistoricalRates(ctx context.Context, date internal.Date, base internal.CurrencyCode, symbols []internal.CurrencyCode) (*internal.LatestRatesResponse, error)
	FetchAndSaveLatest(ctx context.Context, storage RatesStorage, base internal.CurrencyCode, symbols []internal.CurrencyCode) (*internal.LatestRatesResponse, error)
}

type RatesStorage interface {
	UpsertRatesMap(ctx context.Context, base internal.CurrencyCode, asOfDate internal.Date, rates map[internal.CurrencyCode]decimal.Decimal) error
}

type Client struct {
	BaseURL    string
	apiKey     string
	httpClient *http.Client
	storage    RatesStorage
}

func New(apiKey string, storage RatesStorage) *Client {
	return &Client{
		BaseURL: "https://api.currencyfreaks.com/v2.0",
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		storage: storage,
	}
}

func (c *Client) doRates(ctx context.Context, endpoint string, q url.Values) (*internal.LatestRatesResponse, error) {
	u, err := url.Parse(c.BaseURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("currencyfreaks http %d: %s", resp.StatusCode, string(body))
	}

	var out internal.LatestRatesResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &out, nil
}

func (c *Client) LatestRates(ctx context.Context, base internal.CurrencyCode, symbols []internal.CurrencyCode) (*internal.LatestRatesResponse, error) {
	q := url.Values{}
	q.Set("apikey", c.apiKey)
	if base != "" {
		q.Set("base", strings.ToUpper(strings.TrimSpace(string(base))))
	}
	if len(symbols) > 0 {
		symbolStrs := make([]string, len(symbols))
		for i, s := range symbols {
			symbolStrs[i] = string(s)
		}
		q.Set("symbols", strings.Join(symbolStrs, ","))
	}

	return c.doRates(ctx, "/rates/latest", q)
}

func (c *Client) HistoricalRates(ctx context.Context, date internal.Date, base internal.CurrencyCode, symbols []internal.CurrencyCode) (*internal.LatestRatesResponse, error) {
	if date.IsZero() {
		return nil, fmt.Errorf("date is empty")
	}

	q := url.Values{}
	q.Set("apikey", c.apiKey)
	q.Set("date", date.Time.Format("2006-01-02"))

	if base != "" {
		q.Set("base", strings.ToUpper(strings.TrimSpace(string(base))))
	}
	if len(symbols) > 0 {
		symbolStrs := make([]string, len(symbols))
		for i, s := range symbols {
			symbolStrs[i] = string(s)
		}
		q.Set("symbols", strings.Join(symbolStrs, ","))
	}
	return c.doRates(ctx, "/rates/historical", q)
}

func (c *Client) FetchAndSaveLatest(
	ctx context.Context,
	storage RatesStorage,
	base internal.CurrencyCode,
	symbols []internal.CurrencyCode,
) (*internal.LatestRatesResponse, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := c.LatestRates(reqCtx, base, symbols)
	if err != nil {
		return nil, fmt.Errorf("latest rates: %w", err)
	}

	baseCCY, err := internal.NewCurrencyCode(resp.Base)
	if err != nil {
		return nil, fmt.Errorf("invalid base %q: %w", resp.Base, err)
	}

	typedRates := make(map[internal.CurrencyCode]decimal.Decimal, len(resp.Rates))
	for quoteStr, rateStr := range resp.Rates {
		quote, err := internal.NewCurrencyCode(quoteStr)
		if err != nil {
			return nil, fmt.Errorf("invalid quote %q: %w", quoteStr, err)
		}

		rateStr = strings.TrimSpace(rateStr)
		rate, err := decimal.NewFromString(rateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid rate %s/%s=%q: %w", baseCCY, quote, rateStr, err)
		}

		typedRates[quote] = rate
	}

	if err := storage.UpsertRatesMap(reqCtx, baseCCY, resp.Date, typedRates); err != nil {
		return nil, fmt.Errorf("save rates: %w", err)
	}

	return resp, nil
}
