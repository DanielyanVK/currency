package logger

import "context"

type RequestLogger interface {
	LogRequest(ctx context.Context, path string, status *int, dateAsOf *string) error
}

type LoggerStorage interface {
	Insert(ctx context.Context, path string, status *int, dateAsOf *string) error
}
