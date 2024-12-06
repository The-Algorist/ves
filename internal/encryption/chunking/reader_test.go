package chunking

import (
    "bytes"
    "crypto/rand"
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