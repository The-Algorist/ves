package service

import (
    "encoding/json"
    "fmt"
    "io"
    "ves/internal/core/domain"
)

func makeChunkHeader(iv, encrypted []byte) []byte {
    header := make([]byte, 8)
    // Write IV size
    header[0] = byte(len(iv) >> 24)
    header[1] = byte(len(iv) >> 16)
    header[2] = byte(len(iv) >> 8)
    header[3] = byte(len(iv))
    // Write encrypted size
    header[4] = byte(len(encrypted) >> 24)
    header[5] = byte(len(encrypted) >> 16)
    header[6] = byte(len(encrypted) >> 8)
    header[7] = byte(len(encrypted))
    return append(header, iv...)
}

func writeMetadata(w io.Writer, metadata domain.EncryptionMetadata) error {
    // Convert metadata to JSON
    metadataJSON, err := json.Marshal(metadata)
    if err != nil {
        return fmt.Errorf("failed to marshal metadata: %w", err)
    }

    // Create metadata block with size prefix
    size := uint32(len(metadataJSON))
    header := make([]byte, 4)
    header[0] = byte(size >> 24)
    header[1] = byte(size >> 16)
    header[2] = byte(size >> 8)
    header[3] = byte(size)

    // fmt.Printf("DEBUG: Writing metadata size: %d (bytes: %v)\n", size, header)
    // fmt.Printf("DEBUG: Writing metadata JSON: %s\n", string(metadataJSON))

    // Write metadata JSON first
    if _, err := w.Write(metadataJSON); err != nil {
        return fmt.Errorf("failed to write metadata: %w", err)
    }

    // Write size header at the end
    if _, err := w.Write(header); err != nil {
        return fmt.Errorf("failed to write metadata size: %w", err)
    }

    return nil
}

func readMetadata(r io.Reader) (domain.EncryptionMetadata, error) {
    var metadata domain.EncryptionMetadata

    // Read all data from reader
    data, err := io.ReadAll(r)
    if err != nil {
        return metadata, fmt.Errorf("failed to read metadata data: %w", err)
    }

    if len(data) < 4 {
        return metadata, fmt.Errorf("metadata data too short")
    }

    // Get size from the last 4 bytes
    size := uint32(data[len(data)-4])<<24 | 
            uint32(data[len(data)-3])<<16 | 
            uint32(data[len(data)-2])<<8 | 
            uint32(data[len(data)-1])

    // fmt.Printf("DEBUG: Reading metadata size: %d\n", size)

    // Validate size
    if int(size) > len(data)-4 {
        return metadata, fmt.Errorf("invalid metadata size")
    }

    // Get metadata JSON (excluding size bytes)
    metadataJSON := data[:len(data)-4]
    // fmt.Printf("DEBUG: Reading metadata JSON: %s\n", string(metadataJSON))

    // Parse metadata
    if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
        return metadata, fmt.Errorf("failed to unmarshal metadata: %w", err)
    }

    return metadata, nil
}

func validateMetadata(metadata domain.EncryptionMetadata) error {
    if metadata.Algorithm == "" {
        return fmt.Errorf("missing algorithm in metadata")
    }
    if metadata.ChunkSize <= 0 {
        return fmt.Errorf("invalid chunk size in metadata: %d", metadata.ChunkSize)
    }
    if metadata.OriginalSize < 0 {
        return fmt.Errorf("invalid original size in metadata: %d", metadata.OriginalSize)
    }
    if metadata.EncryptedSize < 0 {
        return fmt.Errorf("invalid encrypted size in metadata: %d", metadata.EncryptedSize)
    }
    if metadata.Checksum == "" {
        return fmt.Errorf("missing checksum in metadata")
    }
    if metadata.CreatedAt.IsZero() {
        return fmt.Errorf("missing creation time in metadata")
    }
    return nil
}