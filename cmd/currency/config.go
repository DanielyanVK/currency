package main

import (
	"fmt"
	"os"
	"service-currency/internal"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	APIKey      string

	HTTPPort string

	BaseCCY internal.CurrencyCode
	Symbols []internal.CurrencyCode

	CronSpec string
	Location string

	EncodingKey string
}

func LoadConfig() (Config, error) {
	err := godotenv.Overload()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	rawBaseCCY := "RUB"
	rawBaseSymbols := []string{"EUR", "USD", "JPY"}

	baseCCY, err := internal.NewCurrencyCode(rawBaseCCY)
	if err != nil {
		fmt.Println("invalid base currency: %w", err)
	}

	symbols := make([]internal.CurrencyCode, len(rawBaseSymbols))
	for i, s := range rawBaseSymbols {
		ccy, err := internal.NewCurrencyCode(s)
		if err != nil {
			fmt.Println("invalid symbol %q: %w", s, err)
		}
		symbols[i] = ccy
	}

	cfg := Config{
		HTTPPort: "8080",
		BaseCCY:  baseCCY,
		Symbols:  symbols,
		CronSpec: "0 12 * * *",
		Location: "Europe/Moscow",
	}

	cfg.DatabaseURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is empty")
	}

	cfg.APIKey = strings.TrimSpace(os.Getenv("CURRENCY_API_KEY"))
	if cfg.APIKey == "" {
		return Config{}, fmt.Errorf("CURRENCY_API_KEY is empty")
	}

	cfg.EncodingKey = strings.TrimSpace(os.Getenv("ENCODING_KEY"))
	if cfg.EncodingKey == "" {
		return Config{}, fmt.Errorf("ENCODING_KEY is empty")
	}

	if p := strings.TrimSpace(os.Getenv("PORT")); p != "" {
		cfg.HTTPPort = p
	}

	return cfg, nil
}
