package chunking

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "sync"
    "time"
)

const (
    // Common chunk sizes for video streaming
    DefaultChunkSize = 1024 * 1024    // 1MB default chunk size
    MinChunkSize    = 64 * 1024       // 64KB minimum chunk size
    MaxChunkSize    = 8 * 1024 * 1024 // 8MB maximum chunk size
    MaxRetries      = 3               // Maximum number of retry attempts
    DefaultWorkers = 4               // Default number of worker goroutines
    MaxWorkers     = 16             // Maximum allowed workers
    MinWorkers     = 1              // Minimum allowed workers
    QueueSize      = 100            // Size of the chunk processing queue
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
    WorkerStats     []WorkerStats  // Statistics for each worker
}

// ChunkProcessor represents a chunk processing worker
type ChunkProcessor struct {
    id        int
    queue     chan *Chunk
    results   chan *ProcessedChunk
    wg        *sync.WaitGroup
    processor func(*Chunk) (*ProcessedChunk, error)
    stats     *WorkerStats  // Add stats field
}

// Chunk represents a data chunk to be processed
type Chunk struct {
    Number   int64
    Data     []byte
    Offset   int64
    Size     int
}

// ProcessedChunk represents a processed chunk with results
type ProcessedChunk struct {
    Chunk
    Error    error
    Duration time.Duration
    Checksum string
}

// WorkerPool manages concurrent chunk processing
type WorkerPool struct {
    workers    []*ChunkProcessor
    queue      chan *Chunk
    results    chan *ProcessedChunk
    wg         sync.WaitGroup
    numWorkers int
}

// WorkerStats holds statistics for a single worker
type WorkerStats struct {
    WorkerID         int
    ChunksProcessed  int64
    TotalDuration    time.Duration
    AverageLatency   time.Duration
}

func NewWorkerPool(numWorkers int, processor func(*Chunk) (*ProcessedChunk, error)) (*WorkerPool, error) {
    if numWorkers < MinWorkers || numWorkers > MaxWorkers {
        return nil, fmt.Errorf("invalid number of workers: must be between %d and %d", MinWorkers, MaxWorkers)
    }

    pool := &WorkerPool{
        queue:      make(chan *Chunk, QueueSize),
        results:    make(chan *ProcessedChunk, QueueSize),
        numWorkers: numWorkers,
    }

    // Initialize worker stats slice
    workerStats := make([]WorkerStats, numWorkers)
    
    // Start workers
    for i := 0; i < numWorkers; i++ {
        workerStats[i] = WorkerStats{WorkerID: i + 1}
        worker := &ChunkProcessor{
            id:        i + 1,
            queue:     pool.queue,
            results:   pool.results,
            wg:        &pool.wg,
            processor: processor,
            stats:     &workerStats[i],
        }
        pool.workers = append(pool.workers, worker)
        go worker.start()
    }

    return pool, nil
}

func (w *ChunkProcessor) start() {
    w.wg.Add(1)
    defer w.wg.Done()

    for chunk := range w.queue {
        start := time.Now()
        processed, err := w.processor(chunk)
        if processed == nil {
            processed = &ProcessedChunk{Chunk: *chunk}
        }
        duration := time.Since(start)
        processed.Duration = duration
        processed.Error = err
        
        // Update worker statistics
        w.stats.ChunksProcessed++
        w.stats.TotalDuration += duration
        w.stats.AverageLatency = w.stats.TotalDuration / time.Duration(w.stats.ChunksProcessed)
        
        w.results <- processed
    }
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
    workerPool  *WorkerPool
    concurrency bool
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

    // Read the chunk
    n, err = r.reader.Read(p)
    if err != nil && err != io.EOF {
        // Handle retries
        var lastErr error
        retryCount := 0
        
        for retryCount < MaxRetries {
            n, err = r.reader.Read(p)
            if err == nil || err == io.EOF {
                break
            }
            lastErr = err
            r.stats.Retries++
            retryCount++
            fmt.Printf("Retry %d: %v\n", retryCount, err)
        }

        if lastErr != nil {
            chunkErr := &ChunkError{
                ChunkNumber: r.currentChunk.Number,
                Offset:     r.currentChunk.Offset,
                Err:        lastErr,
                RetryCount: retryCount,
            }
            r.stats.FailedChunks = append(r.stats.FailedChunks, r.currentChunk)
            return 0, chunkErr
        }
    }

    if n > 0 {
        // Process the chunk
        if r.concurrency && r.workerPool != nil {
            chunk := &Chunk{
                Number: r.currentChunk.Number,
                Data:   make([]byte, n),
                Offset: r.currentChunk.Offset,
                Size:   n,
            }
            copy(chunk.Data, p[:n])

            // Send chunk to worker pool
            r.workerPool.queue <- chunk

            // Wait for result
            processed := <-r.workerPool.results
            if processed.Error != nil {
                return 0, processed.Error
            }

            // Update chunk info with processed result
            r.currentChunk.Size = processed.Size
            r.currentChunk.Checksum = processed.Checksum
            
            // Update worker stats in the main stats
            if len(r.stats.WorkerStats) != len(r.workerPool.workers) {
                r.stats.WorkerStats = make([]WorkerStats, len(r.workerPool.workers))
            }
            for i, worker := range r.workerPool.workers {
                r.stats.WorkerStats[i] = *worker.stats
            }
        } else {
            // Process synchronously
            r.currentChunk.Size = n
            if r.validateChunks {
                r.currentChunk.Checksum = calculateChecksum(p[:n])
            }
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

func (r *ChunkReader) EnableConcurrency(enable bool, numWorkers int) error {
    if enable && r.workerPool == nil {
        pool, err := NewWorkerPool(numWorkers, r.processChunk)
        if err != nil {
            return err
        }
        r.workerPool = pool
    }
    r.concurrency = enable
    return nil
}

func (r *ChunkReader) processChunk(chunk *Chunk) (*ProcessedChunk, error) {
    // Process chunk (validation, encryption, etc.)
    processed := &ProcessedChunk{
        Chunk: *chunk,
    }
    
    if r.validateChunks {
        processed.Checksum = calculateChecksum(chunk.Data)
    }
    
    return processed, nil
}