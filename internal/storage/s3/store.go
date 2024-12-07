package s3

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "math/rand"
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

func (s *Store) StoreVideo(ctx context.Context, videoData []byte, metadata storage.VideoMetadata) error {
    if metadata.ID == "" {
        metadata.ID = uuid.New().String()
    }
    
    // Store video data
    videoKey := path.Join(s.config.VideoPrefix, metadata.ID)
    _, err := s.client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String(s.config.BucketName),
        Key:    aws.String(videoKey),
        Body:   bytes.NewReader(videoData),
        Metadata: map[string]string{
            "ContentType": metadata.ContentType,
            "VideoID":    metadata.ID,
        },
    })
    if err != nil {
        return fmt.Errorf("failed to store video: %w", err)
    }

    // Store metadata
    metadataKey := path.Join(s.config.MetadataPrefix, metadata.ID + ".json")
    metadataBytes, err := json.Marshal(metadata)
    if err != nil {
        return fmt.Errorf("failed to marshal metadata: %w", err)
    }

    _, err = s.client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String(s.config.BucketName),
        Key:    aws.String(metadataKey),
        Body:   bytes.NewReader(metadataBytes),
        ContentType: aws.String("application/json"),
    })
    if err != nil {
        return fmt.Errorf("failed to store metadata: %w", err)
    }

    return nil
}

func (s *Store) GenerateKey(ctx context.Context, videoID string) ([]byte, storage.KeyMetadata, error) {
    // Generate a unique encryption key
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

    // Store key without device binding
    keyKey := path.Join(s.config.KeyPrefix, metadata.ID)
    _, err := s.client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String(s.config.BucketName),
        Key:    aws.String(keyKey),
        Body:   bytes.NewReader(key),
        Metadata: map[string]string{
            "VideoID": videoID,
            "KeyID":   metadata.ID,
            "Status":  metadata.Status,
        },
    })
    if err != nil {
        return nil, storage.KeyMetadata{}, fmt.Errorf("failed to store key: %w", err)
    }

    return key, metadata, nil
}

func (s *Store) GetKey(ctx context.Context, keyID string, deviceInfo storage.DeviceInfo) ([]byte, error) {
    // Get key and verify device binding
    keyKey := path.Join(s.config.KeyPrefix, keyID)
    result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(s.config.BucketName),
        Key:    aws.String(keyKey),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get key: %w", err)
    }
    defer result.Body.Close()

    // Verify device binding
    if err := s.validateDeviceBinding(result.Metadata, deviceInfo); err != nil {
        return nil, err
    }

    // Read key
    key := make([]byte, 32)
    if _, err := result.Body.Read(key); err != nil {
        return nil, fmt.Errorf("failed to read key: %w", err)
    }

    return key, nil
}

func (s *Store) ValidateDeviceBinding(ctx context.Context, keyID string, deviceInfo storage.DeviceInfo) error {
    keyKey := path.Join(s.config.KeyPrefix, keyID)
    result, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
        Bucket: aws.String(s.config.BucketName),
        Key:    aws.String(keyKey),
    })
    if err != nil {
        return fmt.Errorf("failed to get key metadata: %w", err)
    }

    return s.validateDeviceBinding(result.Metadata, deviceInfo)
}

func (s *Store) validateDeviceBinding(metadata map[string]string, deviceInfo storage.DeviceInfo) error {
    if metadata["DeviceID"] != deviceInfo.DeviceID {
        return fmt.Errorf("key is not bound to this device ID")
    }
    if metadata["HardwareHash"] != deviceInfo.HardwareHash {
        return fmt.Errorf("key is not bound to this hardware configuration")
    }
    if metadata["Platform"] != deviceInfo.Platform {
        return fmt.Errorf("key is not bound to this platform")
    }
    if metadata["Status"] != "Active" {
        return fmt.Errorf("key is not active")
    }
    return nil
}

// Additional methods implementation... 
