package internal

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type APIKeyRepository interface {
	GetStatusByHash(ctx context.Context, keyHash string) (exists bool, isActive bool, err error)
}

type APIKeyValidator struct {
	repo        APIKeyRepository
	encodingKey string
}

func NewAPIKeyValidator(repo APIKeyRepository, encodingKey string) *APIKeyValidator {
	return &APIKeyValidator{
		repo:        repo,
		encodingKey: strings.TrimSpace(encodingKey),
	}
}

func (v *APIKeyValidator) Validate(ctx context.Context, rawKey string) (exists bool, isActive bool, err error) {
	rawKey = strings.TrimSpace(rawKey)
	if rawKey == "" {
		return false, false, nil
	}

	keyHash := hashKey(rawKey, v.encodingKey)
	return v.repo.GetStatusByHash(ctx, keyHash)
}

func hashKey(rawKey, encodingKey string) string {
	mac := hmac.New(sha256.New, []byte(encodingKey))
	_, _ = mac.Write([]byte(rawKey))
	return hex.EncodeToString(mac.Sum(nil))
}
