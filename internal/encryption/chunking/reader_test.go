package chunking

import (
    "bytes"
    "crypto/rand"
    "fmt"
    "io"
    "testing"
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