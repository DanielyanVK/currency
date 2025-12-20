package models

import "time"

type LatestRatesResponse struct {
	Date  string            `json:"date"`
	Base  string            `json:"base"`
	Rates map[string]string `json:"rates"`
}

type CurrencyLatestRate struct {
	BaseCCY   string
	QuoteCCY  string
	Rate      string
	AsOfDate  *string
	FetchedAt time.Time
}
