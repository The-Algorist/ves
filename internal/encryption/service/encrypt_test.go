package service

import (
    "bytes"
    "context"
    "io"
    "testing"
    "time"

    "ves/internal/core/domain"
    "ves/internal/encryption/service/mocks"
)

func TestEncryptionService_Encrypt(t *testing.T) {
    tests := []struct {
        name        string
        input       []byte
        chunkSize   int
        wantErr     bool
        setupMock   func(*mocks.MockEncryptor)
    }{
        {
            name:      "Success - Small file",
            input:     []byte("Hello, World!"),
            chunkSize: 1024,
            wantErr:   false,
            setupMock: nil,
        },
        {
            name:      "Success - Empty file",
            input:     []byte{},
            chunkSize: 1024,
            wantErr:   false,
            setupMock: nil,
        },
        {
            name:      "Success - Multiple chunks",
            input:     bytes.Repeat([]byte("data"), 1000),
            chunkSize: 1024,
            wantErr:   false,
            setupMock: nil,
        },
        {
            name:      "Failure - Key generation fails",
            input:     []byte("test"),
            chunkSize: 1024,
            wantErr:   true,
            setupMock: func(m *mocks.MockEncryptor) {
                m.GenerateKeyFunc = func() ([]byte, error) {
                    return nil, io.ErrUnexpectedEOF
                }
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup mock
            mockEncryptor := mocks.NewMockEncryptor()
            if tt.setupMock != nil {
                tt.setupMock(mockEncryptor)
            }

            // Create service
            svc := NewService(mockEncryptor)

            // Create input
            input := domain.EncryptionInput{
                Reader: bytes.NewReader(tt.input),
                Metadata: domain.VideoMetadata{
                    FileName:    "test.mp4",
                    Size:        int64(len(tt.input)),
                    ContentType: "video/mp4",
                    CreatedAt:   time.Now(),
                },
                Options: domain.EncryptionOptions{
                    ChunkSize: tt.chunkSize,
                    KeySize:   32,
                },
            }

            // Execute
            output, err := svc.Encrypt(context.Background(), input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if !tt.wantErr {
                // Read all encrypted data
                encryptedData, err := io.ReadAll(output.EncryptedReader)
                if err != nil {
                    t.Fatalf("Failed to read encrypted data: %v", err)
                }

                // Verify we got some data
                if len(encryptedData) == 0 && len(tt.input) > 0 {
                    t.Error("Got empty encrypted data for non-empty input")
                }

                // Verify key was generated
                if len(output.Key) == 0 {
                    t.Error("No encryption key generated")
                }
            }
        })
    }
}