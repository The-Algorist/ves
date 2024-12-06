package chunking

import (
    "fmt"
    "io"
)

const (
    // Common chunk sizes for video streaming
    DefaultChunkSize = 1024 * 1024    // 1MB default chunk size
    MinChunkSize    = 64 * 1024       // 64KB minimum chunk size
    MaxChunkSize    = 8 * 1024 * 1024 // 8MB maximum chunk size
)

type ChunkReader struct {
    reader    io.Reader
    chunkSize int
}

func NewChunkReader(reader io.Reader, chunkSize int) (*ChunkReader, error) {
    if chunkSize < MinChunkSize || chunkSize > MaxChunkSize {
        return nil, fmt.Errorf("invalid chunk size: must be between %d and %d bytes", MinChunkSize, MaxChunkSize)
    }
    
    return &ChunkReader{
        reader:    reader,
        chunkSize: chunkSize,
    }, nil
}

func (r *ChunkReader) Read(p []byte) (n int, err error) {
    if len(p) > r.chunkSize {
        p = p[:r.chunkSize]
    }
    return r.reader.Read(p)
}

func (r *ChunkReader) SetReader(reader io.Reader) {
    r.reader = reader
}

func (r *ChunkReader) SetChunkSize(size int) error {
    if size < MinChunkSize || size > MaxChunkSize {
        return fmt.Errorf("invalid chunk size: must be between %d and %d bytes", MinChunkSize, MaxChunkSize)
    }
    r.chunkSize = size
    return nil
}