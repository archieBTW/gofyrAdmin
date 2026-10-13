package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Config configures the S3 storage driver.
type S3Config struct {
	Bucket          string
	Region          string
	Endpoint        string // Custom endpoint for MinIO, Cloudflare R2, DigitalOcean Spaces
	AccessKey       string
	SecretKey       string
	UsePathStyle    bool   // Force path-style addressing (s3.example.com/bucket instead of bucket.s3.example.com)
	PublicURLBase   string // Optional public CDN or bucket base URL
	PresignLifetime time.Duration
}

// S3 implements Driver for Amazon S3 and S3-compatible object storage.
type S3 struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
	publicURLBase string
	presignLife   time.Duration
}

// NewS3 creates a new S3 storage driver using aws-sdk-go-v2.
func NewS3(ctx context.Context, cfg S3Config) (*S3, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("storage/s3: bucket name is required")
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	if cfg.PresignLifetime == 0 {
		cfg.PresignLifetime = time.Hour
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if cfg.Endpoint != "" {
			return aws.Endpoint{
				URL:               cfg.Endpoint,
				HostnameImmutable: true,
				SigningRegion:     cfg.Region,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	opts := []func(*awsConfig.LoadOptions) error{
		awsConfig.WithRegion(cfg.Region),
		awsConfig.WithEndpointResolverWithOptions(customResolver),
	}

	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		opts = append(opts, awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		)))
	}

	awsCfg, err := awsConfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("storage/s3: load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.UsePathStyle
	})

	presignClient := s3.NewPresignClient(client)

	return &S3{
		client:        client,
		presignClient: presignClient,
		bucket:        cfg.Bucket,
		publicURLBase: strings.TrimSuffix(cfg.PublicURLBase, "/"),
		presignLife:   cfg.PresignLifetime,
	}, nil
}

// Name returns "s3".
func (s *S3) Name() string {
	return "s3"
}

// Upload uploads the reader stream to S3.
func (s *S3) Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	cleanKey := sanitizeKey(key)

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(cleanKey),
		Body:   reader,
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("storage/s3: put object: %w", err)
	}

	return s.GetURL(ctx, cleanKey)
}

// Delete deletes the object with the given key.
func (s *S3) Delete(ctx context.Context, key string) error {
	cleanKey := sanitizeKey(key)
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(cleanKey),
	})
	if err != nil {
		return fmt.Errorf("storage/s3: delete object: %w", err)
	}
	return nil
}

// GetURL returns the public CDN URL if configured, or a pre-signed URL for private buckets.
func (s *S3) GetURL(ctx context.Context, key string) (string, error) {
	cleanKey := sanitizeKey(key)

	if s.publicURLBase != "" {
		return fmt.Sprintf("%s/%s", s.publicURLBase, cleanKey), nil
	}

	// Generate pre-signed URL
	req, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(cleanKey),
	}, s3.WithPresignExpires(s.presignLife))
	if err != nil {
		return "", fmt.Errorf("storage/s3: presign get object: %w", err)
	}

	return req.URL, nil
}
