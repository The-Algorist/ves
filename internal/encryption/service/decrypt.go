package service

import (
    "bytes"
    "context"
    "fmt"
    "io"
)

func (s *EncryptionService) Decrypt(ctx context.Context, encryptedData io.Reader, key []byte, iv []byte) (io.Reader, error) {
    // Buffer all data since we need to read metadata from the end
    data, err := io.ReadAll(encryptedData)
    if err != nil {
        return nil, fmt.Errorf("failed to read encrypted data: %w", err)
    }

    // fmt.Printf("DEBUG: Total encrypted data size: %d\n", len(data))

    // Read metadata from the end (last 4 bytes contain metadata size)
    if len(data) < 4 {
        return nil, fmt.Errorf("encrypted data too short")
    }

    // Get metadata size from last 4 bytes
    sizeBytes := data[len(data)-4:]
    metadataSize := uint32(sizeBytes[0])<<24 | 
                    uint32(sizeBytes[1])<<16 | 
                    uint32(sizeBytes[2])<<8 | 
                    uint32(sizeBytes[3])

    // fmt.Printf("DEBUG: Metadata size bytes: %v\n", sizeBytes)
    // fmt.Printf("DEBUG: Calculated metadata size: %d\n", metadataSize)

    // Read metadata
    metadataStart := len(data) - 4 - int(metadataSize)
    // fmt.Printf("DEBUG: Metadata start position: %d\n", metadataStart)

    if metadataStart < 0 {
        return nil, fmt.Errorf("invalid metadata size")
    }

    metadata, err := readMetadata(bytes.NewReader(data[metadataStart:]))
    if err != nil {
        return nil, fmt.Errorf("failed to read metadata: %w", err)
    }

    if err := validateMetadata(metadata); err != nil {
        return nil, fmt.Errorf("invalid metadata: %w", err)
    }

    // Create a pipe for streaming decrypted data
    pr, pw := io.Pipe()

    // Start decryption goroutine
    go func() {
        defer pw.Close()

        // Create reader for encrypted chunks (excluding metadata)
        chunkReader := bytes.NewReader(data[:metadataStart])

        for {
            // Check context cancellation
            select {
            case <-ctx.Done():
                pw.CloseWithError(ctx.Err())
                return
            default:
            }

            // Read chunk header
            header := make([]byte, 8)
            _, err := io.ReadFull(chunkReader, header)
            if err == io.EOF {
                return
            }
            if err != nil {
                pw.CloseWithError(fmt.Errorf("failed to read chunk header: %w", err))
                return
            }

            // Parse header
            ivSize := uint32(header[0])<<24 | uint32(header[1])<<16 | uint32(header[2])<<8 | uint32(header[3])
            encryptedSize := uint32(header[4])<<24 | uint32(header[5])<<16 | uint32(header[6])<<8 | uint32(header[7])

            // Read IV
            iv := make([]byte, ivSize)
            if _, err := io.ReadFull(chunkReader, iv); err != nil {
                pw.CloseWithError(fmt.Errorf("failed to read IV: %w", err))
                return
            }

            // Read encrypted chunk
            encryptedChunk := make([]byte, encryptedSize)
            if _, err := io.ReadFull(chunkReader, encryptedChunk); err != nil {
                pw.CloseWithError(fmt.Errorf("failed to read encrypted chunk: %w", err))
                return
            }

            // Decrypt chunk
            decrypted, err := s.encryptor.DecryptChunk(encryptedChunk, key, iv)
            if err != nil {
                pw.CloseWithError(fmt.Errorf("failed to decrypt chunk: %w", err))
                return
            }

            // Write decrypted data
            if _, err := pw.Write(decrypted); err != nil {
                pw.CloseWithError(fmt.Errorf("failed to write decrypted data: %w", err))
                return
            }
        }
    }()

    return pr, nil
}