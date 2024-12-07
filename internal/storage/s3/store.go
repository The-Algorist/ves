package s3

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	"ves/internal/storage"
)

type Store struct {
	client *s3.Client
	config storage.Config
}

func New(client *s3.Client, config storage.Config) *Store {
	return &Store{
		client: client,
		config: config,
	}
}

func (s *Store) StoreVideo(ctx context.Context, encryptedData io.Reader, metadata storage.VideoMetadata) error {
	if metadata.ID == "" {
		metadata.ID = uuid.New().String()
	}

	// Store encrypted video data
	videoKey := path.Join(s.config.VideoPrefix, metadata.ID)
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(videoKey),
		Body:   encryptedData,
	})
	if err != nil {
		return fmt.Errorf("failed to store video: %w", err)
	}

	// Store metadata
	metadataKey := path.Join(s.config.MetadataPrefix, metadata.ID+".json")
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(metadataKey),
		Body:   bytes.NewReader(metadataBytes),
	})
	if err != nil {
		return fmt.Errorf("failed to store metadata: %w", err)
	}

	return nil
}

func (s *Store) StoreKey(ctx context.Context, keyID string, key []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.config.BucketName),
		Key:         aws.String(keyID),
		Body:        bytes.NewReader(key),
		ContentType: aws.String("application/octet-stream"),
	})
	if err != nil {
		return fmt.Errorf("failed to store key: %w", err)
	}
	return nil
}

func (s *Store) GenerateKey(ctx context.Context, videoID string) ([]byte, storage.KeyMetadata, error) {
	key := make([]byte, 32) // 256-bit key
	if _, err := rand.Read(key); err != nil {
		return nil, storage.KeyMetadata{}, fmt.Errorf("failed to generate key: %w", err)
	}

	metadata := storage.KeyMetadata{
		ID:            uuid.NewString(),
		VideoID:       videoID,
		CreatedAt:     time.Now(),
		Status:        "Active",
		DeviceBinding: false,
	}

	return key, metadata, nil
}

func (s *Store) GetVideo(ctx context.Context, videoID string) ([]byte, storage.VideoMetadata, error) {
	// First get metadata to get original filename
	metadataKey := path.Join(s.config.MetadataPrefix, videoID+".json")
	metadataResult, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(metadataKey),
	})
	if err != nil {
		return nil, storage.VideoMetadata{}, fmt.Errorf("failed to get metadata: %w", err)
	}
	defer metadataResult.Body.Close()

	// Read and parse metadata
	metadataBytes, err := io.ReadAll(metadataResult.Body)
	if err != nil {
		return nil, storage.VideoMetadata{}, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata storage.VideoMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, storage.VideoMetadata{}, fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Get encrypted video
	videoKey := path.Join(s.config.VideoPrefix, videoID)
	videoResult, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(videoKey),
	})
	if err != nil {
		return nil, metadata, fmt.Errorf("failed to get video: %w", err)
	}
	defer videoResult.Body.Close()

	// Read video data
	videoData, err := io.ReadAll(videoResult.Body)
	if err != nil {
		return nil, metadata, fmt.Errorf("failed to read video data: %w", err)
	}

	return videoData, metadata, nil
}

func (s *Store) GetKey(ctx context.Context, keyID string) ([]byte, error) {
	keyKey := path.Join(s.config.KeyPrefix, keyID)
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(keyKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}
	defer result.Body.Close()

	// Read key data
	keyData, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read key data: %w", err)
	}

	return keyData, nil
}

// GetConfig returns the store configuration
func (s *Store) GetConfig() storage.Config {
	return s.config
}
