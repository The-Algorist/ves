// ves/internal/core/domain/types.go
package domain

import (
    "io"
    "time"
)

type VideoMetadata struct {
    FileName    string
    Size        int64
    ContentType string
    CreatedAt   time.Time
}

type EncryptionOptions struct {
    ChunkSize      int
    KeySize        int
    UseCompression bool
}

type EncryptionMetadata struct {
    Algorithm     string
    ChunkSize     int
    OriginalSize  int64
    EncryptedSize int64
    Checksum      string
    CreatedAt     time.Time
}

type EncryptionInput struct {
    Reader   io.Reader
    Metadata VideoMetadata
    Options  EncryptionOptions
}

type EncryptionOutput struct {
    EncryptedReader io.Reader
    Key            []byte
    IV             []byte
    Metadata       EncryptionMetadata
}