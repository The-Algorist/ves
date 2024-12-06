package service

import (
    "bytes"
    "context"
    "fmt"
    "io"
    "testing"
    "time"

    "ves/internal/core/domain"
    "ves/internal/encryption/service/mocks"
)

func TestEncryptionService_Decrypt(t *testing.T) {
    tests := []struct {
        name          string
        setupEncrypt  func([]byte) (io.Reader, []byte, error)
        input         []byte
        wantErr       bool
    }{
        {
            name: "Success - Small file",
            setupEncrypt: func(data []byte) (io.Reader, []byte, error) {
                mockEncryptor := mocks.NewMockEncryptor()
                svc := NewService(mockEncryptor)
                
                input := domain.EncryptionInput{
                    Reader: bytes.NewReader(data),
                    Metadata: domain.VideoMetadata{
                        Size:      int64(len(data)),
                        CreatedAt: time.Now(),
                    },
                    Options: domain.EncryptionOptions{
                        ChunkSize: 1024,
                        KeySize:   32,
                    },
                }

                output, err := svc.Encrypt(context.Background(), input)
                if err != nil {
                    return nil, nil, err
                }

                // Read all encrypted data into buffer
                encryptedBuf := new(bytes.Buffer)
                if _, err = io.Copy(encryptedBuf, output.EncryptedReader); err != nil {
                    return nil, nil, fmt.Errorf("failed to read encrypted data: %w", err)
                }

                return bytes.NewReader(encryptedBuf.Bytes()), output.Key, nil
            },
            input:   []byte("Hello, World!"),
            wantErr: false,
        },
        {
            name: "Failure - Invalid metadata",
            setupEncrypt: func(data []byte) (io.Reader, []byte, error) {
                // Return invalid data to trigger metadata error
                return bytes.NewReader([]byte("invalid data")), make([]byte, 32), nil
            },
            input:   []byte("test"),
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // First encrypt the data
            encryptedReader, key, err := tt.setupEncrypt(tt.input)
            if err != nil {
                t.Fatalf("Setup failed: %v", err)
            }

            // Create service for decryption with the same mock encryptor
            mockEncryptor := mocks.NewMockEncryptor()
            svc := NewService(mockEncryptor)

            // Decrypt (IV is read from the encrypted data)
            decryptedReader, err := svc.Decrypt(context.Background(), encryptedReader, key, nil)
            if (err != nil) != tt.wantErr {
                t.Errorf("Decrypt() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if !tt.wantErr && decryptedReader != nil {
                // Read decrypted data
                decrypted, err := io.ReadAll(decryptedReader)
                if err != nil {
                    t.Fatalf("Failed to read decrypted data: %v", err)
                }

                // Compare with original
                if !bytes.Equal(decrypted, tt.input) {
                    t.Errorf("Decrypted data doesn't match original\nwant: %v\ngot:  %v", tt.input, decrypted)
                }
            }
        })
    }
}