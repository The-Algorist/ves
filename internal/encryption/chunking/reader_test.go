package chunking

import (
    "bytes"
    "crypto/rand"
    "fmt"
    "io"
    "os"
    "testing"
    "time"
)

func generateVideoData(size int) []byte {
    data := make([]byte, size)
    rand.Read(data) // Generate random bytes to simulate video data
    return data
}

func TestChunkReader(t *testing.T) {
    tests := []struct {
        name      string
        input     []byte
        chunkSize int
        wantErr   bool
    }{
        {
            name:      "Default chunk size",
            input:     bytes.Repeat([]byte{1}, DefaultChunkSize),
            chunkSize: DefaultChunkSize,
            wantErr:   false,
        },
        {
            name:      "Minimum chunk size",
            input:     bytes.Repeat([]byte{1}, MinChunkSize),
            chunkSize: MinChunkSize,
            wantErr:   false,
        },
        {
            name:      "Maximum chunk size",
            input:     bytes.Repeat([]byte{1}, MaxChunkSize),
            chunkSize: MaxChunkSize,
            wantErr:   false,
        },
        {
            name:      "Below minimum chunk size",
            input:     bytes.Repeat([]byte{1}, 1024),
            chunkSize: MinChunkSize - 1,
            wantErr:   true,
        },
        {
            name:      "Above maximum chunk size",
            input:     bytes.Repeat([]byte{1}, 1024),
            chunkSize: MaxChunkSize + 1,
            wantErr:   true,
        },
        {
            name:      "1MB chunk of video data",
            input:     generateVideoData(1024 * 1024), // 1MB
            chunkSize: 1024 * 1024,                    // 1MB chunks
            wantErr:   false,
        },
        {
            name:      "4MB video in 1MB chunks",
            input:     generateVideoData(4 * 1024 * 1024), // 4MB
            chunkSize: 1024 * 1024,                        // 1MB chunks
            wantErr:   false,
        },
        {
            name:      "Empty video file",
            input:     []byte{},
            chunkSize: 1024 * 1024,
            wantErr:   false,
        },
        {
            name:      "Invalid chunk size",
            input:     generateVideoData(1024), // 1KB
            chunkSize: 0,
            wantErr:   true,
        },
        {
            name:      "Small chunk size for video",
            input:     generateVideoData(10 * 1024 * 1024), // 10MB
            chunkSize: 64 * 1024,                           // 64KB chunks
            wantErr:   false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            reader, err := NewChunkReader(bytes.NewReader(tt.input), tt.chunkSize)
            if (err != nil) != tt.wantErr {
                t.Errorf("NewChunkReader() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if tt.wantErr {
                return
            }

            // Read all chunks
            totalRead := 0
            for {
                chunk := make([]byte, tt.chunkSize)
                n, err := reader.Read(chunk)
                if err == io.EOF {
                    break
                }
                if err != nil {
                    if !tt.wantErr {
                        t.Errorf("ChunkReader.Read() error = %v, wantErr %v", err, tt.wantErr)
                    }
                    return
                }
                totalRead += n

                // Verify chunk size
                if n > tt.chunkSize {
                    t.Errorf("Chunk size exceeded: got %d, want <= %d", n, tt.chunkSize)
                }
            }

            // Verify total bytes read
            if totalRead != len(tt.input) {
                t.Errorf("Total bytes read %d, want %d", totalRead, len(tt.input))
            }
        })
    }
}

func TestChunkReader_ReadAtEnd(t *testing.T) {
    // Test reading beyond EOF with video-sized chunks
    input := generateVideoData(5 * 1024 * 1024) // 5MB
    chunkSize := 2 * 1024 * 1024                // 2MB chunks
    reader, err := NewChunkReader(bytes.NewReader(input), chunkSize)
    if err != nil {
        t.Fatalf("Failed to create chunk reader: %v", err)
    }

    // First reads should return full chunks
    totalRead := 0
    chunk := make([]byte, chunkSize)
    for i := 0; i < 2; i++ {
        n, err := reader.Read(chunk)
        if err != nil && err != io.EOF {
            t.Errorf("Read %d failed: %v", i+1, err)
        }
        totalRead += n
        if n != chunkSize {
            t.Errorf("Expected to read %d bytes, got %d", chunkSize, n)
        }
    }

    // Last read should return remaining 1MB
    n, err := reader.Read(chunk)
    if err != nil && err != io.EOF {
        t.Errorf("Final read failed: %v", err)
    }
    totalRead += n
    if n != 1024*1024 {
        t.Errorf("Expected to read 1MB, got %d", n)
    }

    // Next read should return EOF
    n, err = reader.Read(chunk)
    if err != io.EOF {
        t.Errorf("Expected EOF, got %v", err)
    }
    if n != 0 {
        t.Errorf("Expected to read 0 bytes at EOF, got %d", n)
    }
}

func TestChunkReader_SetOperations(t *testing.T) {
    originalData := bytes.Repeat([]byte{1}, 4*1024*1024) // 4MB
    newData := bytes.Repeat([]byte{2}, 2*1024*1024)      // 2MB

    reader, err := NewChunkReader(bytes.NewReader(originalData), 1024*1024) // 1MB chunks
    if err != nil {
        t.Fatalf("Failed to create chunk reader: %v", err)
    }

    // Read first chunk
    chunk := make([]byte, 1024*1024)
    n, err := reader.Read(chunk)
    if err != nil || n != 1024*1024 {
        t.Errorf("First read failed: got %d bytes, error: %v", n, err)
    }

    // Change reader
    reader.SetReader(bytes.NewReader(newData))

    // Change chunk size
    if err := reader.SetChunkSize(512 * 1024); err != nil { // 512KB chunks
        t.Errorf("Failed to set chunk size: %v", err)
    }

    // Read from new data
    chunk = make([]byte, 512*1024)
    n, err = reader.Read(chunk)
    if err != nil || n != 512*1024 {
        t.Errorf("Read after changes failed: got %d bytes, error: %v", n, err)
    }
}

func TestChunkReader_Statistics(t *testing.T) {
    // Create a 1MB test file
    input := bytes.Repeat([]byte{1}, 1024*1024)
    chunkSize := 64 * 1024 // 64KB chunks

    reader, err := NewChunkReader(bytes.NewReader(input), chunkSize)
    if err != nil {
        t.Fatalf("Failed to create chunk reader: %v", err)
    }

    // Add progress callback
    var lastStats Stats
    progressCalled := false
    reader.SetProgressCallback(func(stats Stats) {
        progressCalled = true
        lastStats = stats
    })

    // Read all data
    buf := make([]byte, chunkSize)
    totalRead := 0
    for {
        n, err := reader.Read(buf)
        if err == io.EOF {
            break
        }
        if err != nil {
            t.Fatalf("Read failed: %v", err)
        }
        totalRead += n
    }

    // Get final stats
    stats := reader.GetStats()

    // Verify statistics
    if stats.BytesRead != int64(len(input)) {
        t.Errorf("BytesRead = %d, want %d", stats.BytesRead, len(input))
    }

    expectedChunks := (len(input) + chunkSize - 1) / chunkSize
    if stats.ChunksProcessed != int64(expectedChunks) {
        t.Errorf("ChunksProcessed = %d, want %d", stats.ChunksProcessed, expectedChunks)
    }

    if stats.Duration <= 0 {
        t.Error("Duration should be positive")
    }

    if stats.AverageChunkSize <= 0 {
        t.Error("AverageChunkSize should be positive")
    }

    if !progressCalled {
        t.Error("Progress callback was not called")
    }

    if lastStats.CurrentProgress != 100 {
        t.Errorf("Final progress = %.2f, want 100", lastStats.CurrentProgress)
    }
}

func TestChunkReader_Validation(t *testing.T) {
    // Create test data
    input := bytes.Repeat([]byte{1}, 256*1024) // 256KB
    reader, err := NewChunkReader(bytes.NewReader(input), 64*1024) // 64KB chunks
    if err != nil {
        t.Fatalf("Failed to create chunk reader: %v", err)
    }

    // Read all chunks
    buf := make([]byte, 64*1024)
    var checksums []string
    for {
        n, err := reader.Read(buf)
        if err == io.EOF {
            break
        }
        if err != nil {
            t.Fatalf("Read failed: %v", err)
        }

        // Verify chunk info
        chunkInfo := reader.GetChunkInfo()
        if chunkInfo.Size != n {
            t.Errorf("Chunk size mismatch: got %d, want %d", chunkInfo.Size, n)
        }
        if chunkInfo.Checksum == "" {
            t.Error("Checksum should not be empty when validation is enabled")
        }
        checksums = append(checksums, chunkInfo.Checksum)
    }

    stats := reader.GetStats()
    if len(stats.ValidChunks) != len(checksums) {
        t.Errorf("ValidChunks count mismatch: got %d, want %d", 
            len(stats.ValidChunks), len(checksums))
    }
}

type errorReader struct {
    data      []byte
    position  int
    failCount int
    reads     int
}

func (r *errorReader) Read(p []byte) (n int, err error) {
    r.reads++
    
    // Fail on the second Read call and keep failing
    if r.reads > 1 {
        r.failCount++
        return 0, fmt.Errorf("simulated error (attempt %d)", r.failCount)
    }
    
    if r.position >= len(r.data) {
        return 0, io.EOF
    }
    
    n = copy(p, r.data[r.position:])
    r.position += n
    return n, nil
}

func TestChunkReader_ErrorRecovery(t *testing.T) {
    // Create test data for exactly 3 chunks
    chunkSize := 1024 * 1024 // 1MB chunks
    input := bytes.Repeat([]byte{1}, chunkSize*3) // 3MB total
    
    er := &errorReader{
        data: input,
    }
    
    reader, err := NewChunkReader(er, chunkSize)
    if err != nil {
        t.Fatalf("Failed to create chunk reader: %v", err)
    }

    // Read all chunks
    buf := make([]byte, chunkSize)
    chunks := 0
    
    // First read should succeed
    n, err := reader.Read(buf)
    if err != nil {
        t.Fatalf("First read failed: %v", err)
    }
    if n != chunkSize {
        t.Fatalf("Expected to read %d bytes, got %d", chunkSize, n)
    }
    chunks++

    // Second read should fail after retries
    _, err = reader.Read(buf)
    if err == nil {
        t.Fatal("Expected error on second read")
    }
    
    chunkErr, ok := err.(*ChunkError)
    if !ok {
        t.Fatalf("Expected ChunkError, got %T: %v", err, err)
    }
    
    // Verify error details
    if chunkErr.RetryCount != MaxRetries {
        t.Errorf("Expected %d retries in error, got %d", MaxRetries, chunkErr.RetryCount)
    }
    if chunkErr.ChunkNumber != 2 {
        t.Errorf("Expected error on chunk 2, got chunk %d", chunkErr.ChunkNumber)
    }

    stats := reader.GetStats()
    if stats.Retries != MaxRetries {
        t.Errorf("Expected %d retries, got %d", MaxRetries, stats.Retries)
    }
    if len(stats.FailedChunks) != 1 {
        t.Errorf("Expected 1 failed chunk, got %d", len(stats.FailedChunks))
    }
    if chunks != 1 {
        t.Errorf("Expected to read 1 chunk successfully before error, got %d", chunks)
    }
}

func TestChunkReader_ParallelProcessing(t *testing.T) {
    // Create test data - 10MB total, using 1MB chunks
    dataSize := 10 * 1024 * 1024
    chunkSize := 1024 * 1024
    input := bytes.Repeat([]byte{1}, dataSize)
    
    reader, err := NewChunkReader(bytes.NewReader(input), chunkSize)
    if err != nil {
        t.Fatalf("Failed to create chunk reader: %v", err)
    }

    // Test enabling concurrency
    err = reader.EnableConcurrency(true, DefaultWorkers)
    if err != nil {
        t.Fatalf("Failed to enable concurrency: %v", err)
    }

    // Read all chunks
    buf := make([]byte, chunkSize)
    expectedChunks := dataSize / chunkSize
    processedChunks := 0

    for {
        n, err := reader.Read(buf)
        if err == io.EOF {
            break
        }
        if err != nil {
            t.Fatalf("Read failed: %v", err)
        }
        if n != chunkSize && processedChunks < expectedChunks-1 {
            t.Errorf("Expected chunk size %d, got %d", chunkSize, n)
        }
        processedChunks++
    }

    // Verify results
    stats := reader.GetStats()
    
    if stats.ChunksProcessed != int64(expectedChunks) {
        t.Errorf("Expected %d chunks processed, got %d", 
            expectedChunks, stats.ChunksProcessed)
    }

    if len(stats.WorkerStats) != DefaultWorkers {
        t.Errorf("Expected %d worker stats, got %d", 
            DefaultWorkers, len(stats.WorkerStats))
    }

    // Test worker utilization
    totalProcessed := int64(0)
    for _, ws := range stats.WorkerStats {
        if ws.ChunksProcessed <= 0 {
            t.Errorf("Worker %d processed no chunks", ws.WorkerID)
        }
        totalProcessed += ws.ChunksProcessed
    }

    if totalProcessed != int64(expectedChunks) {
        t.Errorf("Total chunks processed by workers (%d) doesn't match expected (%d)",
            totalProcessed, expectedChunks)
    }
}

func TestChunkReader_ConcurrencyControl(t *testing.T) {
    reader, err := NewChunkReader(bytes.NewReader([]byte{1}), DefaultChunkSize)
    if err != nil {
        t.Fatalf("Failed to create chunk reader: %v", err)
    }

    // Test invalid worker counts
    tests := []struct {
        name       string
        numWorkers int
        wantErr    bool
    }{
        {"Zero workers", 0, true},
        {"Negative workers", -1, true},
        {"Too many workers", MaxWorkers + 1, true},
        {"Minimum workers", MinWorkers, false},
        {"Maximum workers", MaxWorkers, false},
        {"Default workers", DefaultWorkers, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := reader.EnableConcurrency(true, tt.numWorkers)
            if (err != nil) != tt.wantErr {
                t.Errorf("EnableConcurrency() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }

    // Test disabling concurrency
    err = reader.EnableConcurrency(false, DefaultWorkers)
    if err != nil {
        t.Errorf("Failed to disable concurrency: %v", err)
    }
    if reader.concurrency {
        t.Error("Concurrency still enabled after disabling")
    }
}

// mockMemoryReader simulates high memory usage to trigger adaptations
type mockMemoryReader struct {
    *bytes.Reader
    memoryUsage float64
    readCount   int
}

func (m *mockMemoryReader) Read(p []byte) (n int, err error) {
    m.readCount++
    // Simulate memory usage pattern: increase -> spike -> decrease
    switch {
    case m.readCount < 5:
        m.memoryUsage += 10.0 // Gradual increase
    case m.readCount == 5:
        m.memoryUsage = MemoryThresholdHigh + 5 // Spike
    case m.readCount < 10:
        m.memoryUsage = MemoryThresholdLow - 5 // Drop to low
    default:
        m.memoryUsage += 5.0 // Start increasing again
    }
    return m.Reader.Read(p)
}

func (m *mockMemoryReader) GetMemoryUsage() float64 {
    return m.memoryUsage
}

func TestChunkReader_AdaptiveSizing(t *testing.T) {
    // Create a large test file to trigger memory adaptations
    dataSize := 20 * 1024 * 1024 // 20MB
    input := generateVideoData(dataSize)
    initialChunkSize := 1024 * 1024 // 1MB

    // Use mock reader that simulates increasing memory usage
    mockReader := &mockMemoryReader{
        Reader: bytes.NewReader(input),
        memoryUsage: 50.0, // Start at 50% memory usage
    }

    reader, err := NewChunkReader(mockReader, initialChunkSize)
    if err != nil {
        t.Fatalf("Failed to create chunk reader: %v", err)
    }

    // Enable adaptive sizing and set shorter check interval for testing
    reader.EnableAdaptiveSizing(true)
    reader.SetMemoryCheckInterval(50 * time.Millisecond)

    // Read chunks and monitor size adaptations
    buf := make([]byte, MaxChunkSize) // Use max size buffer to accommodate any adaptation
    chunkSizes := make([]int, 0)
    lastSize := initialChunkSize
    totalBytesRead := 0

    // Force initial memory stats collection
    reader.systemStats.LastCheck = time.Now().Add(-2 * reader.memCheckInterval)

    for {
        n, err := reader.Read(buf)
        if err == io.EOF {
            break
        }
        if err != nil {
            t.Fatalf("Read failed: %v", err)
        }
        totalBytesRead += n

        currentSize := reader.chunkSize
        if currentSize != lastSize {
            // Store the size change
            chunkSizes = append(chunkSizes, currentSize)
            lastSize = currentSize

            // Verify size change is within adaptation rates
            sizeRatio := float64(currentSize) / float64(lastSize)
            if sizeRatio > ChunkSizeIncreaseRate || sizeRatio < ChunkSizeDecreaseRate {
                t.Errorf("Chunk size change ratio %.2f outside allowed range [%.2f, %.2f]",
                    sizeRatio, ChunkSizeDecreaseRate, ChunkSizeIncreaseRate)
            }
        }

        // Verify chunk size bounds
        if currentSize < MinChunkSize || currentSize > MaxChunkSize {
            t.Errorf("Chunk size out of bounds: got %d, want between %d and %d",
                currentSize, MinChunkSize, MaxChunkSize)
        }

        // Add small delay to allow memory stats to update
        time.Sleep(5 * time.Millisecond)
    }

    // Verify total bytes read
    if totalBytesRead != dataSize {
        t.Errorf("Total bytes read = %d, want %d", totalBytesRead, dataSize)
    }

    // Verify that memory stats were collected
    if reader.systemStats.MemoryUsage == 0 {
        t.Error("Memory usage stats not collected")
    }

    if reader.systemStats.AvgProcessingTime == 0 {
        t.Error("Processing time stats not collected")
    }

    // Verify that chunk sizes were actually adapted
    if len(chunkSizes) == 0 {
        t.Error("No chunk size adaptations occurred during test")
    } else {
        t.Logf("Memory usage pattern: %v", mockReader.memoryUsage)
        t.Logf("Chunk size adaptations: initial=%d, changes=%v", initialChunkSize, chunkSizes)
    }
}

func TestChunkReader_StateManagement(t *testing.T) {
    // Create test data
    dataSize := 5 * 1024 * 1024 // 5MB
    chunkSize := 1024 * 1024    // 1MB chunks
    input := generateVideoData(dataSize)
    checkpointFile := "test_checkpoint.json"

    // Clean up checkpoint file after test
    defer os.Remove(checkpointFile)

    reader, err := NewChunkReader(bytes.NewReader(input), chunkSize)
    if err != nil {
        t.Fatalf("Failed to create chunk reader: %v", err)
    }

    // Enable resume capability
    reader.EnableResume(true, checkpointFile)

    // Read half the chunks
    buf := make([]byte, chunkSize)
    chunksToRead := dataSize / (2 * chunkSize)
    for i := 0; i < chunksToRead; i++ {
        n, err := reader.Read(buf)
        if err != nil {
            t.Fatalf("Read failed: %v", err)
        }
        if n != chunkSize {
            t.Errorf("Expected chunk size %d, got %d", chunkSize, n)
        }
    }

    // Verify checkpoint was saved
    if err := reader.SaveCheckpoint(); err != nil {
        t.Fatalf("Failed to save checkpoint: %v", err)
    }

    // Create new reader and load checkpoint
    newReader, err := NewChunkReader(bytes.NewReader(input), chunkSize)
    if err != nil {
        t.Fatalf("Failed to create new reader: %v", err)
    }

    newReader.EnableResume(true, checkpointFile)
    if err := newReader.LoadCheckpoint(); err != nil {
        t.Fatalf("Failed to load checkpoint: %v", err)
    }

    // Verify restored state
    if newReader.stats.ChunksProcessed != int64(chunksToRead) {
        t.Errorf("Restored chunks processed = %d, want %d",
            newReader.stats.ChunksProcessed, chunksToRead)
    }

    // Verify chunk states
    state := newReader.GetProcessingState()
    for i := int64(1); i <= int64(chunksToRead); i++ {
        if state[i] != ChunkStateCompleted {
            t.Errorf("Chunk %d state = %v, want %v", i, state[i], ChunkStateCompleted)
        }
    }

    // Continue reading remaining chunks
    remainingChunks := (dataSize/chunkSize) - chunksToRead
    for i := 0; i < remainingChunks; i++ {
        n, err := newReader.Read(buf)
        if err != nil && err != io.EOF {
            t.Fatalf("Read failed: %v", err)
        }
        if err != io.EOF && n != chunkSize {
            t.Errorf("Expected chunk size %d, got %d", chunkSize, n)
        }
    }

    // Verify final state
    finalStats := newReader.GetStats()
    expectedTotalChunks := dataSize / chunkSize
    if finalStats.ChunksProcessed != int64(expectedTotalChunks) {
        t.Errorf("Final chunks processed = %d, want %d",
            finalStats.ChunksProcessed, expectedTotalChunks)
    }
}

func TestChunkReader_CheckpointRecovery(t *testing.T) {
    // Test recovery from invalid/corrupted checkpoint file
    reader, err := NewChunkReader(bytes.NewReader([]byte{1}), DefaultChunkSize)
    if err != nil {
        t.Fatalf("Failed to create reader: %v", err)
    }

    // Test with non-existent file
    reader.EnableResume(true, "nonexistent.json")
    if err := reader.LoadCheckpoint(); err != nil {
        t.Errorf("Loading non-existent checkpoint should not error: %v", err)
    }

    // Test with invalid JSON
    invalidFile := "invalid_checkpoint.json"
    if err := os.WriteFile(invalidFile, []byte("invalid json"), 0644); err != nil {
        t.Fatalf("Failed to create invalid checkpoint file: %v", err)
    }
    defer os.Remove(invalidFile)

    reader.EnableResume(true, invalidFile)
    err = reader.LoadCheckpoint()
    if err == nil {
        t.Error("Expected error loading invalid checkpoint file")
    }

    // Test with invalid seek position
    checkpointFile := "test_checkpoint.json"
    defer os.Remove(checkpointFile)

    // Create a reader with some data
    dataSize := 1024 * 1024 // 1MB
    input := generateVideoData(dataSize)
    reader, err = NewChunkReader(bytes.NewReader(input), DefaultChunkSize)
    if err != nil {
        t.Fatalf("Failed to create reader: %v", err)
    }

    reader.EnableResume(true, checkpointFile)
    reader.stats.BytesRead = int64(dataSize * 2) // Set position beyond file size
    
    // Save checkpoint with invalid position
    err = reader.SaveCheckpoint()
    if err != nil {
        t.Fatalf("Failed to save checkpoint: %v", err)
    }

    // Try to load checkpoint with invalid position
    newReader, err := NewChunkReader(bytes.NewReader(input), DefaultChunkSize)
    if err != nil {
        t.Fatalf("Failed to create new reader: %v", err)
    }
    newReader.EnableResume(true, checkpointFile)
    
    err = newReader.LoadCheckpoint()
    if err == nil {
        t.Error("Expected error loading checkpoint with invalid seek position")
    } else {
        t.Logf("Got expected error: %v", err)
    }
}