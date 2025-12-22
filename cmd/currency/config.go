package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	APIKey      string

	HTTPPort string

	BaseCCY string
	Symbols []string

	CronSpec string
	Location string

	EncodingKey string
}

func LoadConfig() (Config, error) {
	if err := godotenv.Overload(); err != nil {
		log.Println(errors.New("Error loading .env file"))
	}

	cfg := Config{
		HTTPPort: "8080",
		BaseCCY:  "RUB",
		Symbols:  []string{"EUR", "USD", "JPY"},
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
