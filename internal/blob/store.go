package blob

import (
	"context"
	"time"
)

type PresignedRequest struct {
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers,omitempty"`
	ExpiresAt time.Time         `json:"expiresAt"`
}

type ObjectMetadata struct {
	Size        int64
	ContentType string
}

type Store interface {
	UploadOrigin() string
	PresignPut(context.Context, string, string, int64, time.Duration) (PresignedRequest, error)
	PresignGet(context.Context, string, string, time.Duration) (PresignedRequest, error)
	Head(context.Context, string) (ObjectMetadata, error)
	Delete(context.Context, string) error
}
