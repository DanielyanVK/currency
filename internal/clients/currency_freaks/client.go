package currencyFreaks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"service-currency/internal/models"
	"strings"
	"time"
)

const maxBodyBytes = 32 << 10

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	storage    RatesStorage
}

func New(apiKey string, storage RatesStorage) *Client {
	return &Client{
		baseURL: "https://api.currencyfreaks.com/v2.0",
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		storage: storage,
	}
}

func (c *Client) doRates(ctx context.Context, endpoint string, q url.Values) (*models.LatestRatesResponse, error) {
	u, err := url.Parse(c.baseURL + endpoint)
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

	var out models.LatestRatesResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &out, nil
}

func (c *Client) LatestRates(ctx context.Context, base string, symbols []string) (*models.LatestRatesResponse, error) {
	q := url.Values{}
	q.Set("apikey", c.apiKey)
	if base != "" {
		q.Set("base", strings.ToUpper(strings.TrimSpace(base)))
	}
	if len(symbols) > 0 {
		q.Set("symbols", strings.Join(symbols, ","))
	}

	return c.doRates(ctx, "/rates/latest", q)
}

func (c *Client) HistoricalRates(ctx context.Context, date string, base string, symbols []string) (*models.LatestRatesResponse, error) {
	q := url.Values{}
	q.Set("apikey", c.apiKey)
	q.Set("date", date)

	if base != "" {
		q.Set("base", base)
	}
	if len(symbols) > 0 {
		q.Set("symbols", strings.Join(symbols, ","))
	}
	return c.doRates(ctx, "/rates/historical", q)
}

func (c *Client) FetchAndSaveLatest(
	ctx context.Context,
	storage RatesStorage,
	base string,
	symbols []string,
) (*models.LatestRatesResponse, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := c.LatestRates(reqCtx, base, symbols)
	if err != nil {
		return nil, fmt.Errorf("latest rates: %w", err)
	}

	if err := storage.UpsertRatesMap(reqCtx, resp.Base, resp.Date, resp.Rates); err != nil {
		return nil, fmt.Errorf("save rates: %w", err)
	}

	return resp, nil
}
