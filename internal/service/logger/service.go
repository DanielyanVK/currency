package logger

import (
	"context"
	"fmt"
	"strings"
)

type DBRequestLogger struct {
	storage LoggerStorage
}

func New(storage LoggerStorage) *DBRequestLogger {
	return &DBRequestLogger{storage: storage}
}

func (l *DBRequestLogger) LogRequest(ctx context.Context, endpoint string, status *int, dateAsOf *string) error {
	p := strings.TrimSpace(endpoint)
	p = strings.Trim(p, "/")
	if p == "" {
		p = "unknown"
	}

	err := l.storage.Insert(ctx, p, status, dateAsOf)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}
