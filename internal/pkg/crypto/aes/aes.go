// ves/internal/pkg/crypto/aes/aes.go
package aes

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "fmt"
)

const (
    GCMNonceSize = 12 // GCM standard nonce size
)

type AESEncryptor struct {
    keySize int
}

func NewAESEncryptor(keySize int) *AESEncryptor {
    return &AESEncryptor{
        keySize: keySize,
    }
}

func (e *AESEncryptor) GenerateKey() ([]byte, error) {
    key := make([]byte, e.keySize)
    if _, err := rand.Read(key); err != nil {
        return nil, fmt.Errorf("failed to generate key: %w", err)
    }
    return key, nil
}

func (e *AESEncryptor) GenerateIV() ([]byte, error) {
    iv := make([]byte, 12) // GCM standard nonce size
    if _, err := rand.Read(iv); err != nil {
        return nil, fmt.Errorf("failed to generate IV: %w", err)
    }
    return iv, nil
}

func (e *AESEncryptor) EncryptChunk(chunk []byte, key []byte, iv []byte) ([]byte, error) {
    // Validate inputs
    if len(key) != e.keySize {
        return nil, fmt.Errorf("invalid key size: expected %d, got %d", e.keySize, len(key))
    }
    
    if len(iv) != GCMNonceSize {
        return nil, fmt.Errorf("invalid IV size: expected %d, got %d", GCMNonceSize, len(iv))
    }

    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, fmt.Errorf("failed to create cipher: %w", err)
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, fmt.Errorf("failed to create GCM: %w", err)
    }

    return gcm.Seal(nil, iv, chunk, nil), nil
}

func (e *AESEncryptor) DecryptChunk(encryptedChunk []byte, key []byte, iv []byte) ([]byte, error) {
    // Validate inputs
    if len(key) != e.keySize {
        return nil, fmt.Errorf("invalid key size: expected %d, got %d", e.keySize, len(key))
    }
    
    if len(iv) != GCMNonceSize {
        return nil, fmt.Errorf("invalid IV size: expected %d, got %d", GCMNonceSize, len(iv))
    }

    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, fmt.Errorf("failed to create cipher: %w", err)
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, fmt.Errorf("failed to create GCM: %w", err)
    }

    return gcm.Open(nil, iv, encryptedChunk, nil)
}