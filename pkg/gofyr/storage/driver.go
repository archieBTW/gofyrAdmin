package storage

import (
	"context"
	"io"
)

// Driver defines the pluggable file and photo storage interface for gofyrAdmin.
type Driver interface {
	// Upload stores an object with the specified key, content reader, and MIME type.
	// Returns the public or accessible URL of the uploaded object.
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) (publicURL string, err error)

	// Delete removes the object identified by key.
	Delete(ctx context.Context, key string) error

	// GetURL retrieves the URL for the specified key (e.g. public CDN URL or pre-signed S3 URL).
	GetURL(ctx context.Context, key string) (string, error)

	// Name returns the identifier of this storage driver (e.g. "local", "s3").
	Name() string
}
