package internal

import (
	"context"
	"fmt"
	"strings"
)

type RequestAuditLogger interface {
	LogRequest(ctx context.Context, path string, status *int, dateAsOf *Date) error
}

type AuditLogStorage interface {
	Insert(ctx context.Context, path string, status *int, dateAsOf *Date) error
}

func NewStorageAuditLogger(storage AuditLogStorage) *StorageAuditLogger {
	return &StorageAuditLogger{auditLogStorage: storage}
}

type StorageAuditLogger struct {
	auditLogStorage AuditLogStorage
}

func (l *StorageAuditLogger) LogRequest(ctx context.Context, endpoint string, status *int, dateAsOf *Date) error {
	p := strings.TrimSpace(endpoint)
	p = strings.Trim(p, "/")
	if p == "" {
		p = "unknown"
	}

	err := l.auditLogStorage.Insert(ctx, p, status, dateAsOf)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}
