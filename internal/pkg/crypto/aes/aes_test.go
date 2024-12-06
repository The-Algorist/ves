// ves/internal/pkg/crypto/aes/aes_test.go
package aes

import (
    "bytes"
    "crypto/rand"
    "fmt"
    // "io"
    "strings"
    "testing"
    // "crypto/cipher"
    // "crypto/aes"
)

// Mock rand.Reader for testing error cases
type mockErrorReader struct{}

func (m mockErrorReader) Read(p []byte) (n int, err error) {
    return 0, fmt.Errorf("mock random reader error")
}

func TestAESEncryption(t *testing.T) {
    tests := []struct {
        name        string
        inputData   []byte
        keySize     int
        shouldError bool
    }{
        {
            name:        "Basic encryption/decryption",
            inputData:   []byte("Hello, this is a test message!"),
            keySize:     32,
            shouldError: false,
        },
        {
            name:        "Empty data",
            inputData:   []byte(""),
            keySize:     32,
            shouldError: false,
        },
        {
            name:        "Large data",
            inputData:   bytes.Repeat([]byte("Large data test "), 1000),
            keySize:     32,
            shouldError: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            encryptor := NewAESEncryptor(tt.keySize)

    // Generate key and IV
    key, err := encryptor.GenerateKey()
    if err != nil {
        t.Fatalf("Failed to generate key: %v", err)
    }

    iv, err := encryptor.GenerateIV()
    if err != nil {
        t.Fatalf("Failed to generate IV: %v", err)
    }

            // Encrypt
            encrypted, err := encryptor.EncryptChunk(tt.inputData, key, iv)
            if (err != nil) != tt.shouldError {
                t.Fatalf("Encryption error = %v, shouldError = %v", err, tt.shouldError)
            }

            if tt.shouldError {
                return
            }

            // Verify encrypted data is different from input
            if bytes.Equal(encrypted, tt.inputData) {
                t.Error("Encrypted data is identical to input data")
            }

    // Decrypt
    decrypted, err := encryptor.DecryptChunk(encrypted, key, iv)
    if err != nil {
        t.Fatalf("Decryption failed: %v", err)
    }

            // Compare
            if !bytes.Equal(tt.inputData, decrypted) {
                t.Errorf("Decrypted data doesn't match original data")
            }
        })
    }
}

// Test invalid cases
func TestAESEncryption_Invalid(t *testing.T) {
    encryptor := NewAESEncryptor(32)
    data := []byte("Test data")

    // Generate valid key and IV
    key, _ := encryptor.GenerateKey()
    iv, _ := encryptor.GenerateIV()

    t.Run("Invalid key size", func(t *testing.T) {
        invalidKey := make([]byte, 31) // Wrong key size
        _, err := encryptor.EncryptChunk(data, invalidKey, iv)
        if err == nil {
            t.Error("Expected error with invalid key size, got none")
        }
    })

    t.Run("Invalid IV size", func(t *testing.T) {
        invalidIV := make([]byte, 11) // Wrong IV size
        _, err := encryptor.EncryptChunk(data, key, invalidIV)
        if err == nil {
            t.Error("Expected error with invalid IV size, got none")
        }
        if err != nil && err.Error() != fmt.Sprintf("invalid IV size: expected %d, got %d", GCMNonceSize, 11) {
            t.Errorf("Unexpected error message: %v", err)
        }
    })

    t.Run("Nil key", func(t *testing.T) {
        _, err := encryptor.EncryptChunk(data, nil, iv)
        if err == nil {
            t.Error("Expected error with nil key, got none")
        }
    })

    t.Run("Nil IV", func(t *testing.T) {
        _, err := encryptor.EncryptChunk(data, key, nil)
        if err == nil {
            t.Error("Expected error with nil IV, got none")
        }
    })
}

func TestAESEncryptor_GenerateKey(t *testing.T) {
    tests := []struct {
        name    string
        keySize int
        wantErr bool
    }{
        {
            name:    "Valid key size (32 bytes)",
            keySize: 32,
            wantErr: false,
        },
        {
            name:    "Valid key size (16 bytes)",
            keySize: 16,
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            e := NewAESEncryptor(tt.keySize)
            key, err := e.GenerateKey()
            if (err != nil) != tt.wantErr {
                t.Errorf("GenerateKey() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !tt.wantErr && len(key) != tt.keySize {
                t.Errorf("GenerateKey() got key length = %v, want %v", len(key), tt.keySize)
            }
        })
    }
}

func TestAESEncryptor_GenerateIV(t *testing.T) {
    e := NewAESEncryptor(32)
    
    t.Run("Generate IV", func(t *testing.T) {
        iv, err := e.GenerateIV()
        if err != nil {
            t.Errorf("GenerateIV() unexpected error = %v", err)
            return
        }
        if len(iv) != GCMNonceSize {
            t.Errorf("GenerateIV() got IV length = %v, want %v", len(iv), GCMNonceSize)
        }
    })

    t.Run("Multiple IVs should be different", func(t *testing.T) {
        iv1, _ := e.GenerateIV()
        iv2, _ := e.GenerateIV()
        if bytes.Equal(iv1, iv2) {
            t.Error("Generated IVs should be different")
        }
    })
}

func TestAESEncryptor_DecryptChunk(t *testing.T) {
    e := NewAESEncryptor(32)
    key, _ := e.GenerateKey()
    iv, _ := e.GenerateIV()

    tests := []struct {
        name    string
        data    []byte
        wantErr bool
    }{
        {
            name:    "Decrypt valid data",
            data:    []byte("Test encryption and decryption"),
            wantErr: false,
        },
        {
            name:    "Decrypt empty data",
            data:    []byte{},
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // First encrypt
            encrypted, err := e.EncryptChunk(tt.data, key, iv)
            if err != nil {
                t.Fatalf("Failed to encrypt: %v", err)
            }

            // Then decrypt
            decrypted, err := e.DecryptChunk(encrypted, key, iv)
            if (err != nil) != tt.wantErr {
                t.Errorf("DecryptChunk() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if !tt.wantErr && !bytes.Equal(decrypted, tt.data) {
                t.Errorf("DecryptChunk() = %v, want %v", decrypted, tt.data)
            }
        })
    }

    t.Run("Decrypt with wrong key", func(t *testing.T) {
        data := []byte("Test data")
        encrypted, _ := e.EncryptChunk(data, key, iv)
        wrongKey, _ := e.GenerateKey()
        _, err := e.DecryptChunk(encrypted, wrongKey, iv)
        if err == nil {
            t.Error("Expected error when decrypting with wrong key")
        }
    })

    t.Run("Decrypt corrupted data", func(t *testing.T) {
        data := []byte("Test data")
        encrypted, _ := e.EncryptChunk(data, key, iv)
        corrupted := append(encrypted, byte(0))
        _, err := e.DecryptChunk(corrupted, key, iv)
        if err == nil {
            t.Error("Expected error when decrypting corrupted data")
        }
    })
}

func TestAESEncryptor_GenerateKeyError(t *testing.T) {
    // Replace rand.Reader temporarily
    oldReader := rand.Reader
    rand.Reader = mockErrorReader{}
    defer func() { rand.Reader = oldReader }()

    e := NewAESEncryptor(32)
    _, err := e.GenerateKey()
    if err == nil {
        t.Error("Expected error from GenerateKey, got nil")
    }
    if !strings.Contains(err.Error(), "failed to generate key") {
        t.Errorf("Expected 'failed to generate key' error, got: %v", err)
    }
}

func TestAESEncryptor_GenerateIVError(t *testing.T) {
    // Replace rand.Reader temporarily
    oldReader := rand.Reader
    rand.Reader = mockErrorReader{}
    defer func() { rand.Reader = oldReader }()

    e := NewAESEncryptor(32)
    _, err := e.GenerateIV()
    if err == nil {
        t.Error("Expected error from GenerateIV, got nil")
    }
    if !strings.Contains(err.Error(), "failed to generate IV") {
        t.Errorf("Expected 'failed to generate IV' error, got: %v", err)
    }
}

func TestAESEncryptor_CipherErrors(t *testing.T) {
    e := NewAESEncryptor(32)
    invalidKey := make([]byte, 31) // Invalid key size for AES
    validIV, _ := e.GenerateIV()
    data := []byte("test data")

    // Test NewCipher error in EncryptChunk
    _, err := e.EncryptChunk(data, invalidKey, validIV)
    if err == nil || !strings.Contains(err.Error(), "invalid key size") {
        t.Errorf("Expected invalid key size error, got: %v", err)
    }

    // Test NewCipher error in DecryptChunk
    _, err = e.DecryptChunk(data, invalidKey, validIV)
    if err == nil || !strings.Contains(err.Error(), "invalid key size") {
        t.Errorf("Expected invalid key size error, got: %v", err)
    }

    // Test GCM creation error (this is harder to trigger as it rarely fails)
    // but we're including it for completeness
    validKey, _ := e.GenerateKey()
    invalidData := []byte("invalid data")
    _, err = e.DecryptChunk(invalidData, validKey, validIV)
    if err == nil {
        t.Error("Expected error when decrypting invalid data")
    }
}

func TestAESEncryptor_CompleteFlow(t *testing.T) {
    e := NewAESEncryptor(32)
    key, err := e.GenerateKey()
    if err != nil {
        t.Fatalf("Failed to generate key: %v", err)
    }

    iv, err := e.GenerateIV()
    if err != nil {
        t.Fatalf("Failed to generate IV: %v", err)
    }

    originalData := []byte("test data for complete flow")
    
    // Encrypt
    encrypted, err := e.EncryptChunk(originalData, key, iv)
    if err != nil {
        t.Fatalf("Failed to encrypt: %v", err)
    }

    // Decrypt
    decrypted, err := e.DecryptChunk(encrypted, key, iv)
    if err != nil {
        t.Fatalf("Failed to decrypt: %v", err)
    }

    if !bytes.Equal(originalData, decrypted) {
        t.Error("Decrypted data doesn't match original")
    }
}

func TestAESEncryptor_CipherCreationErrors(t *testing.T) {
    e := NewAESEncryptor(32)
    validIV, _ := e.GenerateIV()
    data := []byte("test data")

    // Create an invalid key that will pass size check but fail cipher creation
    invalidKey := make([]byte, 32)
    for i := range invalidKey {
        invalidKey[i] = byte(i + 1) // Non-zero pattern that should cause AES key schedule to fail
    }

    t.Run("Invalid key in EncryptChunk", func(t *testing.T) {
        _, err := e.EncryptChunk(data, invalidKey[:31], validIV) // Use wrong size to trigger error
        if err == nil || !strings.Contains(err.Error(), "invalid key size") {
            t.Errorf("Expected invalid key size error, got: %v", err)
        }
    })

    t.Run("Invalid key in DecryptChunk", func(t *testing.T) {
        _, err := e.DecryptChunk(data, invalidKey[:31], validIV) // Use wrong size to trigger error
        if err == nil || !strings.Contains(err.Error(), "invalid key size") {
            t.Errorf("Expected invalid key size error, got: %v", err)
        }
    })
}

func TestAESEncryptor_GCMErrors(t *testing.T) {
    e := NewAESEncryptor(32)
    validKey, _ := e.GenerateKey()
    data := []byte("test data")

    // Test with invalid IV sizes
    invalidIVSizes := []int{0, 11, 13, 16}
    for _, size := range invalidIVSizes {
        t.Run(fmt.Sprintf("Invalid IV size %d", size), func(t *testing.T) {
            invalidIV := make([]byte, size)
            
            // Test encryption
            _, err := e.EncryptChunk(data, validKey, invalidIV)
            if err == nil || !strings.Contains(err.Error(), "invalid IV size") {
                t.Errorf("Expected invalid IV size error, got: %v", err)
            }
            
            // Test decryption
            _, err = e.DecryptChunk(data, validKey, invalidIV)
            if err == nil || !strings.Contains(err.Error(), "invalid IV size") {
                t.Errorf("Expected invalid IV size error, got: %v", err)
            }
        })
    }

    // Test decryption with corrupted ciphertext
    t.Run("Corrupted ciphertext", func(t *testing.T) {
        validIV, _ := e.GenerateIV()
        originalData := []byte("test data")
        encrypted, _ := e.EncryptChunk(originalData, validKey, validIV)
        
        // Corrupt the encrypted data
        corrupted := make([]byte, len(encrypted))
        copy(corrupted, encrypted)
        for i := range corrupted {
            corrupted[i] ^= 0xff // Flip all bits
        }
        
        _, err := e.DecryptChunk(corrupted, validKey, validIV)
        if err == nil {
            t.Error("Expected error when decrypting corrupted data")
        }
    })

    // Test with malformed ciphertext
    t.Run("Malformed ciphertext", func(t *testing.T) {
        validIV, _ := e.GenerateIV()
        malformed := []byte("too short to be valid ciphertext")
        _, err := e.DecryptChunk(malformed, validKey, validIV)
        if err == nil {
            t.Error("Expected error when decrypting malformed data")
        }
    })
}

func TestAESEncryptor_DecryptionAuthenticationError(t *testing.T) {
    e := NewAESEncryptor(32)
    validKey, _ := e.GenerateKey()
    validIV, _ := e.GenerateIV()
    data := []byte("test data")

    // Create valid ciphertext
    encrypted, err := e.EncryptChunk(data, validKey, validIV)
    if err != nil {
        t.Fatalf("Failed to encrypt: %v", err)
    }

    // Modify the authentication tag
    encrypted[len(encrypted)-1] ^= 0xff

    // Attempt to decrypt
    _, err = e.DecryptChunk(encrypted, validKey, validIV)
    if err == nil {
        t.Error("Expected authentication error, got nil")
    }
}

func TestAESEncryptor_EdgeCases(t *testing.T) {
    e := NewAESEncryptor(32)
    validKey, _ := e.GenerateKey()
    validIV, _ := e.GenerateIV()

    // Test with empty data
    t.Run("Empty data encryption", func(t *testing.T) {
        _, err := e.EncryptChunk([]byte{}, validKey, validIV)
        if err != nil {
            t.Errorf("Expected no error for empty data, got: %v", err)
        }
    })

    // Test with nil data
    t.Run("Nil data encryption", func(t *testing.T) {
        _, err := e.EncryptChunk(nil, validKey, validIV)
        if err != nil {
            t.Errorf("Expected no error for nil data, got: %v", err)
        }
    })

    // Test with maximum size data
    t.Run("Large data encryption", func(t *testing.T) {
        largeData := make([]byte, 1024*1024) // 1MB
        _, err := e.EncryptChunk(largeData, validKey, validIV)
        if err != nil {
            t.Errorf("Expected no error for large data, got: %v", err)
        }
    })
}