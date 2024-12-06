package chunking

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "time"
)

const (
    // Common chunk sizes for video streaming
    DefaultChunkSize = 1024 * 1024    // 1MB default chunk size
    MinChunkSize    = 64 * 1024       // 64KB minimum chunk size
    MaxChunkSize    = 8 * 1024 * 1024 // 8MB maximum chunk size
    MaxRetries      = 3               // Maximum number of retry attempts
)

// ChunkError represents an error that occurred during chunk processing
type ChunkError struct {
    ChunkNumber int64
    Offset      int64
    Err         error
    RetryCount  int
}

func (e *ChunkError) Error() string {
    return fmt.Sprintf("chunk %d at offset %d failed: %v (retries: %d)", 
        e.ChunkNumber, e.Offset, e.Err, e.RetryCount)
}

// ChunkInfo contains validation information for a chunk
type ChunkInfo struct {
    Number   int64
    Size     int
    Checksum string
    Offset   int64
}

// Stats holds statistics about the chunk reading process
type Stats struct {
    BytesRead        int64
    ChunksProcessed  int64
    StartTime        time.Time
    Duration         time.Duration
    AverageChunkSize float64
    CurrentProgress  float64
    Retries         int           // Number of retries performed
    FailedChunks    []ChunkInfo  // Information about failed chunks
    ValidChunks     []ChunkInfo  // Information about successful chunks
}

type ChunkReader struct {
    reader          io.Reader
    chunkSize       int
    totalSize       int64
    stats           Stats
    progressUpdated func(Stats)
    currentChunk    ChunkInfo
    validateChunks  bool         // Enable/disable chunk validation
    retryEnabled    bool         // Enable/disable retry mechanism
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
        reader:         reader,
        chunkSize:     chunkSize,
        totalSize:     totalSize,
        validateChunks: true,  // Enable validation by default
        retryEnabled:   true,  // Enable retries by default
        stats: Stats{
            StartTime: time.Now(),
        },
    }, nil
}

// calculateChecksum generates a SHA-256 checksum for a chunk
func calculateChecksum(data []byte) string {
    hash := sha256.Sum256(data)
    return hex.EncodeToString(hash[:])
}

func (r *ChunkReader) Read(p []byte) (n int, err error) {
    if len(p) > r.chunkSize {
        p = p[:r.chunkSize]
    }

    // Initialize chunk info
    r.currentChunk = ChunkInfo{
        Number: r.stats.ChunksProcessed + 1,
        Offset: r.stats.BytesRead,
    }

    // Try reading with retries
    var lastErr error
    retryCount := 0
    
    for retryCount < MaxRetries {
        n, err = r.reader.Read(p)
        
        if err == nil || err == io.EOF {
            // Successful read or EOF
            break
        }
        
        // Record retry attempt and continue
        lastErr = err
        r.stats.Retries++
        retryCount++
        fmt.Printf("Retry %d: %v\n", retryCount, err)
        
        // Don't break, keep retrying until MaxRetries
        continue
    }

    // Try one final time after all retries
    if lastErr != nil {
        n, err = r.reader.Read(p)
        if err != nil {
            chunkErr := &ChunkError{
                ChunkNumber: r.currentChunk.Number,
                Offset:     r.currentChunk.Offset,
                Err:        err,
                RetryCount: retryCount,
            }
            r.stats.FailedChunks = append(r.stats.FailedChunks, r.currentChunk)
            return 0, chunkErr
        }
    }

    if n > 0 {
        // Update chunk info
        r.currentChunk.Size = n
        if r.validateChunks {
            r.currentChunk.Checksum = calculateChecksum(p[:n])
        }
        r.stats.ValidChunks = append(r.stats.ValidChunks, r.currentChunk)
        r.updateStats(n)
    }

    return n, err
}

// EnableValidation turns chunk validation on/off
func (r *ChunkReader) EnableValidation(enable bool) {
    r.validateChunks = enable
}

// EnableRetries turns retry mechanism on/off
func (r *ChunkReader) EnableRetries(enable bool) {
    r.retryEnabled = enable
}

// GetChunkInfo returns information about the current chunk
func (r *ChunkReader) GetChunkInfo() ChunkInfo {
    return r.currentChunk
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