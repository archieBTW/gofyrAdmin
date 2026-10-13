package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// LocalConfig configures the local disk storage driver.
type LocalConfig struct {
	// BaseDir is the local root directory on disk where files are stored (e.g. "./uploads").
	BaseDir string
	// PublicURLPrefix is the URL prefix to prepend to relative paths (e.g. "/admin/uploads").
	PublicURLPrefix string
}

// Local implements Driver for local disk storage.
type Local struct {
	baseDir         string
	publicURLPrefix string
	mu              sync.RWMutex
}

// NewLocal creates a new local disk storage driver.
func NewLocal(cfg LocalConfig) (*Local, error) {
	if cfg.BaseDir == "" {
		cfg.BaseDir = "./uploads"
	}
	if cfg.PublicURLPrefix == "" {
		cfg.PublicURLPrefix = "/admin/uploads"
	}

	// Ensure base directory exists
	absDir, err := filepath.Abs(cfg.BaseDir)
	if err != nil {
		return nil, fmt.Errorf("storage/local: resolve base directory: %w", err)
	}

	if err := os.MkdirAll(absDir, 0755); err != nil {
		return nil, fmt.Errorf("storage/local: create base directory: %w", err)
	}

	return &Local{
		baseDir:         absDir,
		publicURLPrefix: strings.TrimSuffix(cfg.PublicURLPrefix, "/"),
	}, nil
}

// Name returns "local".
func (l *Local) Name() string {
	return "local"
}

// BaseDir returns the resolved base directory on disk.
func (l *Local) BaseDir() string {
	return l.baseDir
}

// PublicURLPrefix returns the configured URL prefix.
func (l *Local) PublicURLPrefix() string {
	return l.publicURLPrefix
}

// Upload writes the file to disk at key and returns its public URL.
func (l *Local) Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	cleanKey := sanitizeKey(key)
	targetPath := filepath.Join(l.baseDir, cleanKey)

	// Ensure directory exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("storage/local: create subdirectories: %w", err)
	}

	outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return "", fmt.Errorf("storage/local: open output file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, reader); err != nil {
		return "", fmt.Errorf("storage/local: copy data: %w", err)
	}

	return l.GetURL(ctx, cleanKey)
}

// Delete removes the file at key.
func (l *Local) Delete(ctx context.Context, key string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	cleanKey := sanitizeKey(key)
	targetPath := filepath.Join(l.baseDir, cleanKey)

	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage/local: remove file: %w", err)
	}
	return nil
}

// GetURL returns the public URL for the given key.
func (l *Local) GetURL(ctx context.Context, key string) (string, error) {
	cleanKey := sanitizeKey(key)
	cleanKey = strings.TrimPrefix(cleanKey, "/")
	return fmt.Sprintf("%s/%s", l.publicURLPrefix, cleanKey), nil
}

// sanitizeKey cleans the key path to prevent directory traversal attacks.
func sanitizeKey(key string) string {
	// Normalize slashes
	cleaned := strings.ReplaceAll(key, "\\", "/")
	parts := strings.Split(cleaned, "/")
	var safeParts []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || p == "." || p == ".." {
			continue
		}
		// Remove invalid path chars
		p = strings.Map(func(r rune) rune {
			if r < 32 || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
				return '_'
			}
			return r
		}, p)
		safeParts = append(safeParts, p)
	}
	return strings.Join(safeParts, "/")
}
