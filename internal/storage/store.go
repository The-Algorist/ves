package storage

import (
    "context"
    "time"
)

// DeviceInfo represents hardware-specific information for key binding
type DeviceInfo struct {
    DeviceID      string            // Unique device identifier
    HardwareHash  string            // Hardware-specific hash
    Platform      string            // OS/Platform info
    Fingerprint   map[string]string // Additional device fingerprinting data
}

// VideoMetadata contains information about a stored video
type VideoMetadata struct {
    ID            string
    OriginalName  string
    ContentType   string
    Size          int64
    Duration      time.Duration
    CreatedAt     time.Time
    UpdatedAt     time.Time
    Tags          []string
}

// KeyMetadata contains information about an encryption key
type KeyMetadata struct {
    ID            string
    VideoID       string
    DeviceInfo    DeviceInfo
    CreatedAt     time.Time
    DownloadID    string    // Unique per download session
    Status        string    // Active, Revoked
    DeviceBinding bool      // Whether the key is permanently bound to a device
}

// Store defines the interface for storage operations
type Store interface {
    // Video operations
    StoreVideo(ctx context.Context, videoData []byte, metadata VideoMetadata) error
    GetVideo(ctx context.Context, id string) ([]byte, VideoMetadata, error)
    DeleteVideo(ctx context.Context, id string) error
    
    // Key operations
    GenerateKey(ctx context.Context, videoID string, deviceInfo DeviceInfo) ([]byte, KeyMetadata, error)
    GetKey(ctx context.Context, keyID string, deviceInfo DeviceInfo) ([]byte, error)
    RevokeKey(ctx context.Context, keyID string) error
    ValidateDeviceBinding(ctx context.Context, keyID string, deviceInfo DeviceInfo) error
    
    // Metadata operations
    UpdateMetadata(ctx context.Context, videoID string, metadata VideoMetadata) error
    GetMetadata(ctx context.Context, videoID string) (VideoMetadata, error)
    ListVideos(ctx context.Context, filter map[string]interface{}) ([]VideoMetadata, error)
}

// Config holds configuration for storage services
type Config struct {
    BucketName      string
    Region          string
    KeyPrefix       string
    VideoPrefix     string
    MetadataPrefix  string
    StrictBinding   bool      // Whether to enforce strict device binding
    AllowMultiDevice bool     // Whether to allow multiple devices (default: false)
}

// Options for storage operations
type Options struct {
    Encryption     bool
    Compression    bool
    MaxKeys        int
    MaxDownloads   int
    DeviceBinding  bool      // Whether to bind the key to the device
} 