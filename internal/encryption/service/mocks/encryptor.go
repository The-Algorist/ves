package mocks

import (
    // "crypto/rand"
    "bytes"
)

type MockEncryptor struct {
    GenerateKeyFunc    func() ([]byte, error)
    GenerateIVFunc     func() ([]byte, error)
    EncryptChunkFunc   func(chunk []byte, key []byte, iv []byte) ([]byte, error)
    DecryptChunkFunc   func(encryptedChunk []byte, key []byte, iv []byte) ([]byte, error)
}

func NewMockEncryptor() *MockEncryptor {
    return &MockEncryptor{
        GenerateKeyFunc: func() ([]byte, error) {
            return bytes.Repeat([]byte{1}, 32), nil
        },
        GenerateIVFunc: func() ([]byte, error) {
            return bytes.Repeat([]byte{2}, 12), nil
        },
        EncryptChunkFunc: func(chunk []byte, key []byte, iv []byte) ([]byte, error) {
            return chunk, nil
        },
        DecryptChunkFunc: func(encryptedChunk []byte, key []byte, iv []byte) ([]byte, error) {
            return encryptedChunk, nil
        },
    }
}

func (m *MockEncryptor) GenerateKey() ([]byte, error) {
    return m.GenerateKeyFunc()
}

func (m *MockEncryptor) GenerateIV() ([]byte, error) {
    return m.GenerateIVFunc()
}

func (m *MockEncryptor) EncryptChunk(chunk []byte, key []byte, iv []byte) ([]byte, error) {
    return m.EncryptChunkFunc(chunk, key, iv)
}

func (m *MockEncryptor) DecryptChunk(encryptedChunk []byte, key []byte, iv []byte) ([]byte, error) {
    return m.DecryptChunkFunc(encryptedChunk, key, iv)
}