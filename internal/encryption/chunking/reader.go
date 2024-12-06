package chunking

import (
    "fmt"
    "io"
    "time"
)

const (
    // Common chunk sizes for video streaming
    DefaultChunkSize = 1024 * 1024    // 1MB default chunk size
    MinChunkSize    = 64 * 1024       // 64KB minimum chunk size
    MaxChunkSize    = 8 * 1024 * 1024 // 8MB maximum chunk size
)

// Stats holds statistics about the chunk reading process
type Stats struct {
    BytesRead       int64         // Total bytes read
    ChunksProcessed int64         // Number of chunks processed
    StartTime       time.Time     // When the reading started
    Duration        time.Duration // Total processing time
    AverageChunkSize float64      // Average size of chunks
    CurrentProgress  float64      // Progress percentage (0-100)
}

type ChunkReader struct {
    reader          io.Reader
    chunkSize       int
    totalSize       int64  // Total size of input (if known)
    stats           Stats
    progressUpdated func(Stats) // Callback for progress updates
}

// NewChunkReader creates a new ChunkReader with the given chunk size
func NewChunkReader(reader io.Reader, chunkSize int) (*ChunkReader, error) {
    if chunkSize < MinChunkSize || chunkSize > MaxChunkSize {
        return nil, fmt.Errorf("invalid chunk size: must be between %d and %d bytes", MinChunkSize, MaxChunkSize)
    }

    // Try to get total size if reader supports it
    var totalSize int64
    if sizer, ok := reader.(interface{ Size() int64 }); ok {
        totalSize = sizer.Size()
    }

    return &ChunkReader{
        reader:    reader,
        chunkSize: chunkSize,
        totalSize: totalSize,
        stats: Stats{
            StartTime: time.Now(),
        },
    }, nil
}

// Read implements io.Reader and updates statistics
func (r *ChunkReader) Read(p []byte) (n int, err error) {
    if len(p) > r.chunkSize {
        p = p[:r.chunkSize]
    }

    n, err = r.reader.Read(p)
    if n > 0 {
        r.updateStats(n)
    }

    return n, err
}

// updateStats updates reading statistics
func (r *ChunkReader) updateStats(bytesRead int) {
    r.stats.BytesRead += int64(bytesRead)
    r.stats.ChunksProcessed++
    r.stats.Duration = time.Since(r.stats.StartTime)
    r.stats.AverageChunkSize = float64(r.stats.BytesRead) / float64(r.stats.ChunksProcessed)

    if r.totalSize > 0 {
        r.stats.CurrentProgress = float64(r.stats.BytesRead) / float64(r.totalSize) * 100
    }

    if r.progressUpdated != nil {
        r.progressUpdated(r.stats)
    }
}

// SetProgressCallback sets a callback function that will be called when progress is updated
func (r *ChunkReader) SetProgressCallback(callback func(Stats)) {
    r.progressUpdated = callback
}

// GetStats returns the current statistics
func (r *ChunkReader) GetStats() Stats {
    return r.stats
}

// SetReader sets a new reader and resets statistics
func (r *ChunkReader) SetReader(reader io.Reader) {
    r.reader = reader
    r.stats = Stats{
        StartTime: time.Now(),
    }
    // Try to get total size if reader supports it
    if sizer, ok := reader.(interface{ Size() int64 }); ok {
        r.totalSize = sizer.Size()
    }
}

// SetChunkSize sets a new chunk size
func (r *ChunkReader) SetChunkSize(size int) error {
    if size < MinChunkSize || size > MaxChunkSize {
        return fmt.Errorf("invalid chunk size: must be between %d and %d bytes", MinChunkSize, MaxChunkSize)
    }
    r.chunkSize = size
    return nil
}