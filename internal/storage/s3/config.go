package s3

import (
    "context"
    "fmt"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/s3"

    "ves/internal/storage"
)

// DefaultConfig provides default configuration values
var DefaultConfig = storage.Config{
    BucketName:       "ves-video-storage",
    Region:           "us-east-1",
    KeyPrefix:        "keys/",
    VideoPrefix:      "videos/",
    MetadataPrefix:   "metadata/",
    StrictBinding:    true,
    AllowMultiDevice: false,
}

// NewClient creates a new S3 client with the given configuration
func NewClient(ctx context.Context, cfg aws.Config, bucket string) (*Store, error) {
    client := s3.NewFromConfig(cfg)

    // Verify bucket exists and is accessible
    _, err := client.HeadBucket(ctx, &s3.HeadBucketInput{
        Bucket: aws.String(bucket),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to access bucket %s: %w", bucket, err)
    }

    config := DefaultConfig
    config.BucketName = bucket
    config.Region = cfg.Region

    return New(client, config), nil
}

// WithStrictBinding sets whether to enforce strict device binding
func WithStrictBinding(strict bool) func(*storage.Config) {
    return func(c *storage.Config) {
        c.StrictBinding = strict
    }
}

// WithMultiDevice sets whether to allow multiple devices
func WithMultiDevice(allow bool) func(*storage.Config) {
    return func(c *storage.Config) {
        c.AllowMultiDevice = allow
    }
}

// WithPrefixes sets custom prefixes for different types of objects
func WithPrefixes(key, video, metadata string) func(*storage.Config) {
    return func(c *storage.Config) {
        if key != "" {
            c.KeyPrefix = key
        }
        if video != "" {
            c.VideoPrefix = video
        }
        if metadata != "" {
            c.MetadataPrefix = metadata
        }
    }
} 