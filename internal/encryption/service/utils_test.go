package service

import (
    "bytes"
    "testing"
    "time"

    "ves/internal/core/domain"
)

func TestMetadataWriteRead(t *testing.T) {
    original := domain.EncryptionMetadata{
        Algorithm:     "AES-256-GCM",
        ChunkSize:     1024 * 1024,
        OriginalSize:  12345,
        EncryptedSize: 12400,
        Checksum:      "test-checksum",
        CreatedAt:     time.Now().UTC(),
    }

    // Write metadata
    var buf bytes.Buffer
    err := writeMetadata(&buf, original)
    if err != nil {
        t.Fatalf("writeMetadata() error = %v", err)
    }

    // Read metadata back
    read, err := readMetadata(&buf)
    if err != nil {
        t.Fatalf("readMetadata() error = %v", err)
    }

    // Compare
    if original.Algorithm != read.Algorithm {
        t.Errorf("Algorithm mismatch: got %v, want %v", read.Algorithm, original.Algorithm)
    }
    if original.ChunkSize != read.ChunkSize {
        t.Errorf("ChunkSize mismatch: got %v, want %v", read.ChunkSize, original.ChunkSize)
    }
    if original.OriginalSize != read.OriginalSize {
        t.Errorf("OriginalSize mismatch: got %v, want %v", read.OriginalSize, original.OriginalSize)
    }
    if original.EncryptedSize != read.EncryptedSize {
        t.Errorf("EncryptedSize mismatch: got %v, want %v", read.EncryptedSize, original.EncryptedSize)
    }
    if original.Checksum != read.Checksum {
        t.Errorf("Checksum mismatch: got %v, want %v", read.Checksum, original.Checksum)
    }
    if !original.CreatedAt.Equal(read.CreatedAt) {
        t.Errorf("CreatedAt mismatch: got %v, want %v", read.CreatedAt, original.CreatedAt)
    }
}

func TestMakeChunkHeader(t *testing.T) {
    tests := []struct {
        name      string
        iv        []byte
        encrypted []byte
        wantSize  int
    }{
        {
            name:      "Normal case",
            iv:        make([]byte, 12),
            encrypted: make([]byte, 100),
            wantSize:  20, // 8 bytes header + 12 bytes IV
        },
        {
            name:      "Empty encrypted data",
            iv:        make([]byte, 12),
            encrypted: []byte{},
            wantSize:  20,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            header := makeChunkHeader(tt.iv, tt.encrypted)
            if len(header) != tt.wantSize {
                t.Errorf("makeChunkHeader() size = %v, want %v", len(header), tt.wantSize)
            }
        })
    }
}