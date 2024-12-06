// ves/internal/core/ports/encryption.go
package ports

import (
    "context"
    "io"
    "ves/internal/core/domain"
)

type EncryptionService interface {
    Encrypt(ctx context.Context, input domain.EncryptionInput) (*domain.EncryptionOutput, error)
    Decrypt(ctx context.Context, encryptedData io.Reader, key []byte, iv []byte) (io.Reader, error)
}

type Encryptor interface {
    GenerateKey() ([]byte, error)
    GenerateIV() ([]byte, error)
    EncryptChunk(chunk []byte, key []byte, iv []byte) ([]byte, error)
    DecryptChunk(encryptedChunk []byte, key []byte, iv []byte) ([]byte, error)
}