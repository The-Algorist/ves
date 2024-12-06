# Video Encryption Core

Core encryption module for handling large video file encryption using AES-256-GCM.

## Project Structure

video-encryption/
ves/
├── cmd/
│ └── encrypt/ # CLI testing tool
├── internal/
│ ├── core/ # Core domain logic
│ ├── encryption/ # Encryption implementation
│ └── pkg/ # Shared utilities
└── test/ # Integration tests


## Core Components

### Encryption Service
Handles the main encryption logic with chunked processing for large files.

```go
type EncryptionInput struct {
Reader io.Reader
Metadata VideoMetadata
Options EncryptionOptions
}
type EncryptionOutput struct {
EncryptedReader io.Reader
Key []byte
IV []byte
Metadata EncryptionMetadata
}


### Usage Example
go
encryptionService := service.NewEncryptionService(
chunking.NewChunkReader(10 1024 1024), // 10MB chunks
crypto.NewAESEncryptor(),
)
input := domain.EncryptionInput{
Reader: file,
Metadata: domain.VideoMetadata{
FileName: "test.mp4",
Size: fileInfo.Size(),
},
Options: domain.EncryptionOptions{
ChunkSize: 10 1024 1024,
KeySize: 32,
},
}
output, err := encryptionService.Encrypt(context.Background(), input)
```

## Key Features

- Chunk-based processing for large files
- AES-256-GCM encryption
- Memory-efficient streaming
- Progress tracking
- Extensible storage interfaces

## Output Format

### Encrypted Package Structure
plaintext
encrypted-package/
├── video.enc # Encrypted video data
└── metadata.json # Encryption metadata


### Metadata JSON Structure
```json
{
"algorithm": "AES-256-GCM",
"chunkSize": 10485760,
"originalSize": 1234567,
"encryptedSize": 1234789,
"checksum": "sha256-hash",
"createdAt": "2024-03-21T15:04:05Z"
}


## Development

### Prerequisites
- Go 1.21+
- Make (optional, for using Makefile commands)

### Running Tests
Run unit tests

```bash
go test ./...
```
Run integration tests
```bash
go test ./test/...
```


### Building CLI Tool
```bash
go build -o encrypt ./cmd/encrypt
```

### CLI Usage
Encrypt a video file
```bash
./encrypt -input video.mp4 -output encrypted/
```
Show encryption progress
```bash
./encrypt -input video.mp4 -output encrypted/ -verbose
```


## Testing Large Files

For testing with large files:
1. Use chunked processing (default 10MB chunks)
2. Monitor memory usage
3. Check encryption/decryption integrity
4. Verify checksum after process

## Next Steps
- [ ] Implement core encryption
- [ ] Add chunk processing
- [ ] Create basic CLI
- [ ] Add tests
- [ ] Add progress tracking
- [ ] Implement storage interface

## Notes
- Maximum supported file size: 20GB
- Default chunk size: 10MB
- Uses AES-256-GCM for encryption
- Generates random IV for each encryption