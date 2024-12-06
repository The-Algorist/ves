package service

import (
    "context"
    "io"
    "ves/internal/core/domain"
    "ves/internal/core/ports"
)

type Service interface {
    Encrypt(ctx context.Context, input domain.EncryptionInput) (*domain.EncryptionOutput, error)
    Decrypt(ctx context.Context, encryptedData io.Reader, key []byte, iv []byte) (io.Reader, error)
}

type EncryptionService struct {
    encryptor ports.Encryptor
}

func NewService(encryptor ports.Encryptor) Service {
    return &EncryptionService{
        encryptor: encryptor,
    }
}