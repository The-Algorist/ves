package service

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "time"

    "ves/internal/core/domain"
)

func (s *EncryptionService) Encrypt(ctx context.Context, input domain.EncryptionInput) (*domain.EncryptionOutput, error) {
    // Generate encryption key
    key, err := s.encryptor.GenerateKey()
    if err != nil {
        return nil, fmt.Errorf("failed to generate key: %w", err)
    }

    // Create a pipe for streaming encrypted data
    pr, pw := io.Pipe()

    // Create metadata outside the goroutine so we can use it in the return
    metadata := domain.EncryptionMetadata{
        Algorithm:     "AES-256-GCM",
        ChunkSize:     input.Options.ChunkSize,
        OriginalSize:  input.Metadata.Size,
        EncryptedSize: 0,
        Checksum:      "",
        CreatedAt:     time.Now().UTC(),
    }

    // Start encryption goroutine
    go func() {
        defer pw.Close()
        
        hasher := sha256.New()
        var encryptedSize int64

        buffer := make([]byte, input.Options.ChunkSize)
        for {
            // Check context cancellation
            select {
            case <-ctx.Done():
                pw.CloseWithError(ctx.Err())
                return
            default:
            }

            // Read chunk
            n, err := input.Reader.Read(buffer)
            if err != nil && err != io.EOF {
                pw.CloseWithError(fmt.Errorf("failed to read chunk: %w", err))
                return
            }
            if n == 0 {
                break
            }

            // Generate new IV for each chunk
            iv, err := s.encryptor.GenerateIV()
            if err != nil {
                pw.CloseWithError(fmt.Errorf("failed to generate IV: %w", err))
                return
            }

            // Encrypt chunk
            encrypted, err := s.encryptor.EncryptChunk(buffer[:n], key, iv)
            if err != nil {
                pw.CloseWithError(fmt.Errorf("failed to encrypt chunk: %w", err))
                return
            }

            // Write IV size, IV, encrypted size, and encrypted data
            header := makeChunkHeader(iv, encrypted)
            if _, err := pw.Write(header); err != nil {
                pw.CloseWithError(fmt.Errorf("failed to write header: %w", err))
                return
            }
            if _, err := pw.Write(encrypted); err != nil {
                pw.CloseWithError(fmt.Errorf("failed to write encrypted data: %w", err))
                return
            }

            // Update checksum and size
            hasher.Write(encrypted)
            encryptedSize += int64(len(header) + len(encrypted))
        }

        // Update metadata with final values
        metadata.EncryptedSize = encryptedSize
        metadata.Checksum = hex.EncodeToString(hasher.Sum(nil))

        // Write metadata at the end
        if err := writeMetadata(pw, metadata); err != nil {
            pw.CloseWithError(fmt.Errorf("failed to write metadata: %w", err))
            return
        }
    }()

    return &domain.EncryptionOutput{
        EncryptedReader: pr,
        Key:            key,
        Metadata:       metadata,
    }, nil
}