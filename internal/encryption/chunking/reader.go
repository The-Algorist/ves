package chunking

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "runtime"
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
    
    // Memory thresholds for adaptive sizing
    MemoryThresholdHigh = 85  // High memory usage threshold (percentage)
    MemoryThresholdLow  = 60  // Low memory usage threshold (percentage)
    ChunkSizeDecreaseRate = 0.75  // Decrease chunk size by 25% when memory is high
    ChunkSizeIncreaseRate = 1.25  // Increase chunk size by 25% when memory is low
    MemoryCheckInterval   = 5 * time.Second  // How often to check memory usage
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

// SystemStats holds system resource information
type SystemStats struct {
    MemoryUsage     float64   // Current memory usage percentage
    LastCheck       time.Time // Last time memory was checked
    AvgProcessingTime time.Duration // Average time to process a chunk
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

// ChunkState represents the processing state of a chunk
type ChunkState int

const (
    ChunkStatePending ChunkState = iota
    ChunkStateProcessing
    ChunkStateCompleted
    ChunkStateFailed
)

// Checkpoint represents a saved state of the chunk processing
type Checkpoint struct {
    Position         int64
    ProcessedChunks  []ChunkInfo
    LastChunkState   ChunkState
    Stats           Stats
    SystemStats     SystemStats
    Timestamp       time.Time
}

// ChunkReader struct update
type ChunkReader struct {
    reader          io.Reader
    chunkSize       int
    totalSize       int64
    stats           Stats
    progressUpdated func(Stats)
    currentChunk    ChunkInfo
    validateChunks  bool
    retryEnabled    bool
    workerPool      *WorkerPool
    concurrency     bool
    systemStats     SystemStats
    adaptiveSizing  bool
    lastSizeChange  time.Time
    checkpointFile  string
    resumeEnabled   bool
    processedChunks map[int64]ChunkState
    lastCheckpoint  *Checkpoint
    memCheckInterval time.Duration  // Add configurable interval
}

// NewChunkReader update
func NewChunkReader(reader io.Reader, chunkSize int) (*ChunkReader, error) {
    if chunkSize < MinChunkSize || chunkSize > MaxChunkSize {
        return nil, fmt.Errorf("invalid chunk size: must be between %d and %d bytes", MinChunkSize, MaxChunkSize)
    }

    var totalSize int64
    if sizer, ok := reader.(interface{ Size() int64 }); ok {
        totalSize = sizer.Size()
    }

    return &ChunkReader{
        reader:         reader,
        chunkSize:     chunkSize,
        totalSize:     totalSize,
        validateChunks: true,
        retryEnabled:   true,
        adaptiveSizing: false,
        resumeEnabled:  false,
        processedChunks: make(map[int64]ChunkState),
        memCheckInterval: MemoryCheckInterval, // Use constant as default
        stats: Stats{
            StartTime: time.Now(),
        },
        systemStats: SystemStats{
            LastCheck: time.Now(),
        },
    }, nil
}

// EnableResume enables or disables resume capabilities
func (r *ChunkReader) EnableResume(enable bool, checkpointFile string) {
    r.resumeEnabled = enable
    r.checkpointFile = checkpointFile
    if enable && r.lastCheckpoint == nil {
        r.lastCheckpoint = &Checkpoint{
            Timestamp: time.Now(),
        }
    }
}

// SaveCheckpoint saves the current processing state
func (r *ChunkReader) SaveCheckpoint() error {
    if !r.resumeEnabled {
        return nil
    }

    checkpoint := &Checkpoint{
        Position:        r.stats.BytesRead,
        ProcessedChunks: r.stats.ValidChunks,
        LastChunkState:  r.processedChunks[r.currentChunk.Number],
        Stats:          r.stats,
        SystemStats:    r.systemStats,
        Timestamp:      time.Now(),
    }

    // Save checkpoint to file
    data, err := json.Marshal(checkpoint)
    if err != nil {
        return fmt.Errorf("failed to marshal checkpoint: %w", err)
    }

    if err := os.WriteFile(r.checkpointFile, data, 0644); err != nil {
        return fmt.Errorf("failed to write checkpoint file: %w", err)
    }

    r.lastCheckpoint = checkpoint
    return nil
}

// LoadCheckpoint loads the last saved processing state
func (r *ChunkReader) LoadCheckpoint() error {
    if !r.resumeEnabled {
        return nil
    }

    data, err := os.ReadFile(r.checkpointFile)
    if err != nil {
        if os.IsNotExist(err) {
            return nil // No checkpoint file exists
        }
        return fmt.Errorf("failed to read checkpoint file: %w", err)
    }

    checkpoint := &Checkpoint{}
    if err := json.Unmarshal(data, checkpoint); err != nil {
        return fmt.Errorf("failed to unmarshal checkpoint: %w", err)
    }

    // Verify checkpoint position is valid
    if seeker, ok := r.reader.(io.Seeker); ok {
        // Try to seek to verify position is valid
        _, err := seeker.Seek(0, io.SeekEnd)
        if err != nil {
            return fmt.Errorf("failed to seek to end: %w", err)
        }
        
        size, err := seeker.Seek(0, io.SeekCurrent)
        if err != nil {
            return fmt.Errorf("failed to get file size: %w", err)
        }
        
        if checkpoint.Position > size {
            return fmt.Errorf("invalid checkpoint position %d: exceeds file size %d", 
                checkpoint.Position, size)
        }

        // Seek back to checkpoint position
        if _, err := seeker.Seek(checkpoint.Position, io.SeekStart); err != nil {
            return fmt.Errorf("failed to seek to checkpoint position: %w", err)
        }
    }

    // Restore state from checkpoint
    r.stats = checkpoint.Stats
    r.systemStats = checkpoint.SystemStats
    r.lastCheckpoint = checkpoint

    // Restore processed chunks state
    for _, chunk := range checkpoint.ProcessedChunks {
        r.processedChunks[chunk.Number] = ChunkStateCompleted
    }

    return nil
}

// calculateChecksum generates a SHA-256 checksum for a chunk
func calculateChecksum(data []byte) string {
    hash := sha256.Sum256(data)
    return hex.EncodeToString(hash[:])
}

// Read method update
func (r *ChunkReader) Read(p []byte) (n int, err error) {
    // Check and adjust chunk size if adaptive sizing is enabled
    if r.adaptiveSizing {
        if err := r.adjustChunkSize(); err != nil {
            fmt.Printf("Warning: Failed to adjust chunk size: %v\n", err)
        }
    }

    if len(p) > r.chunkSize {
        p = p[:r.chunkSize]
    }

    // Initialize chunk info
    r.currentChunk = ChunkInfo{
        Number: r.stats.ChunksProcessed + 1,
        Offset: r.stats.BytesRead,
    }

    // Check if chunk was already processed
    if state, exists := r.processedChunks[r.currentChunk.Number]; exists && state == ChunkStateCompleted {
        // Skip already processed chunk
        if seeker, ok := r.reader.(io.Seeker); ok {
            _, err := seeker.Seek(int64(r.chunkSize), io.SeekCurrent)
            if err != nil {
                return 0, fmt.Errorf("failed to seek past processed chunk: %w", err)
            }
            return r.chunkSize, nil
        }
    }

    // Update chunk state
    r.processedChunks[r.currentChunk.Number] = ChunkStateProcessing

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

        // Update chunk state after successful processing
        r.processedChunks[r.currentChunk.Number] = ChunkStateCompleted
        
        // Save checkpoint periodically
        if r.resumeEnabled && r.stats.ChunksProcessed%10 == 0 {
            if err := r.SaveCheckpoint(); err != nil {
                fmt.Printf("Warning: Failed to save checkpoint: %v\n", err)
            }
        }
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

    // Update system stats
    if r.adaptiveSizing {
        r.systemStats.AvgProcessingTime = r.stats.Duration / time.Duration(r.stats.ChunksProcessed)
    }

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

// EnableAdaptiveSizing enables or disables adaptive chunk sizing
func (r *ChunkReader) EnableAdaptiveSizing(enable bool) {
    r.adaptiveSizing = enable
    r.lastSizeChange = time.Now()
}

// MemoryReader interface for readers that can report memory usage
type MemoryReader interface {
    io.Reader
    GetMemoryUsage() float64
}

// getSystemMemoryUsage returns the current system memory usage percentage
func (r *ChunkReader) getSystemMemoryUsage() (float64, error) {
    // If we have a reader that reports memory usage, use that
    if mr, ok := r.reader.(MemoryReader); ok {
        return mr.GetMemoryUsage(), nil
    }
    
    // Otherwise use real memory stats
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    memoryUsage := float64(m.Alloc) / float64(m.Sys) * 100
    return memoryUsage, nil
}

// adjustChunkSize dynamically adjusts the chunk size based on system memory usage
func (r *ChunkReader) adjustChunkSize() error {
    // Only check periodically
    if time.Since(r.systemStats.LastCheck) < r.memCheckInterval {
        return nil
    }

    memUsage, err := r.getSystemMemoryUsage()
    if err != nil {
        return fmt.Errorf("failed to get memory usage: %w", err)
    }

    r.systemStats.MemoryUsage = memUsage
    r.systemStats.LastCheck = time.Now()

    // Only adjust if enough time has passed since last change
    if time.Since(r.lastSizeChange) < r.memCheckInterval*2 {
        return nil
    }

    currentSize := r.chunkSize
    newSize := currentSize

    switch {
    case memUsage >= MemoryThresholdHigh:
        // Decrease chunk size
        newSize = int(float64(currentSize) * ChunkSizeDecreaseRate)
    case memUsage <= MemoryThresholdLow:
        // Increase chunk size
        newSize = int(float64(currentSize) * ChunkSizeIncreaseRate)
    }

    // Ensure new size is within bounds
    if newSize < MinChunkSize {
        newSize = MinChunkSize
    } else if newSize > MaxChunkSize {
        newSize = MaxChunkSize
    }

    // Only update if size actually changed
    if newSize != currentSize {
        r.chunkSize = newSize
        r.lastSizeChange = time.Now()
    }

    return nil
}

// GetProcessingState returns the current processing state
func (r *ChunkReader) GetProcessingState() map[int64]ChunkState {
    return r.processedChunks
}

// GetLastCheckpoint returns the last saved checkpoint
func (r *ChunkReader) GetLastCheckpoint() *Checkpoint {
    return r.lastCheckpoint
}

// SetMemoryCheckInterval allows configuring the memory check interval
func (r *ChunkReader) SetMemoryCheckInterval(interval time.Duration) {
    r.memCheckInterval = interval
}