package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/google/uuid"
	"github.com/joho/godotenv"

	"ves/internal/core/domain"
	"ves/internal/encryption/service"
	"ves/internal/pkg/crypto/aes"
	"ves/internal/storage"
	s3store "ves/internal/storage/s3"
)

const (
    baseDir           = "/mnt/d/Personal/vesplayer/ves-player/ves-videos"
    rawVideoDir       = baseDir + "/raw"
    encryptedVideoDir = baseDir + "/encrypted"
    decryptedVideoDir = baseDir + "/decrypted"
    keysDir           = baseDir + "/keys"
)

func init() {
    // Ensure directories exist
    dirs := []string{rawVideoDir, encryptedVideoDir, decryptedVideoDir, keysDir}
    for _, dir := range dirs {
        if err := os.MkdirAll(dir, 0755); err != nil {
            fmt.Printf("Failed to create directory %s: %v\n", dir, err)
        }
    }
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	if len(os.Args) < 2 {
		log.Fatal("Please provide the path to a video file")
	}
	videoPath := os.Args[1]

	// Get file info and verify size
	fileInfo, err := os.Stat(videoPath)
	if err != nil {
		log.Fatalf("Failed to get file info: %v", err)
	}
	fmt.Printf("File size: %.2f MB\n", float64(fileInfo.Size())/(1024*1024))

	ctx := context.Background()

	// Initialize AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Unable to load SDK config: %v", err)
	}

	fmt.Printf("Using AWS Region: %s\n", cfg.Region)

	// Print caller identity for debugging
	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		log.Printf("Warning: Unable to get caller identity: %v", err)
	} else {
		fmt.Printf("AWS Account: %s\n", *identity.Account)
		fmt.Printf("AWS User ARN: %s\n", *identity.Arn)
	}

	// Create storage client
	store, err := s3store.NewClient(ctx, cfg, os.Getenv("AWS_BUCKET_NAME"))
	if err != nil {
		log.Fatalf("Failed to create storage client: %v", err)
	}

	// Open video file
	videoFile, err := os.Open(videoPath)
	if err != nil {
		log.Fatalf("Failed to open video file: %v", err)
	}
	defer videoFile.Close()

	// Create encryption service
	encryptor := aes.NewAESEncryptor(32)
	encryptionService := service.NewService(encryptor)

	// Create video metadata
	videoID := uuid.NewString()
	metadata := storage.VideoMetadata{
		ID:           videoID,
		OriginalName: filepath.Base(videoPath),
		ContentType:  "video/mp4",
		Size:         fileInfo.Size(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Create encryption input
	encInput := domain.EncryptionInput{
		Reader: videoFile,
		Metadata: domain.VideoMetadata{
			FileName:    filepath.Base(videoPath),
			Size:        fileInfo.Size(),
			ContentType: "video/mp4",
			CreatedAt:   time.Now(),
		},
		Options: domain.EncryptionOptions{
			ChunkSize: 1024 * 1024, // 1MB chunks
			KeySize:   32,
		},
	}

	// Start timer
	startTime := time.Now()

	// Encrypt the video
	fmt.Println("Encrypting video...")
	encOutput, err := encryptionService.Encrypt(ctx, encInput)
	if err != nil {
		log.Fatalf("Failed to encrypt video: %v", err)
	}

	// Read all encrypted data
	encryptedData, err := io.ReadAll(encOutput.EncryptedReader)
	if err != nil {
		log.Fatalf("Failed to read encrypted data: %v", err)
	}

	// Save locally first
	fmt.Println("Saving files locally...")
	filename := filepath.Base(videoPath)
	outputPath := filepath.Join(encryptedVideoDir, filename+".enc")

	// Save encrypted video
	if err := os.WriteFile(outputPath, encryptedData, 0644); err != nil {
		log.Fatalf("Failed to save encrypted file: %v", err)
	}

	// Save key
	keyPath := outputPath + ".key"
	if err := os.WriteFile(keyPath, encOutput.Key, 0600); err != nil {
		log.Fatalf("Failed to save key file: %v", err)
	}

	fmt.Printf("Saved encrypted video to: %s\n", outputPath)
	fmt.Printf("Saved key to: %s\n", keyPath)

	// Upload to S3
	fmt.Println("\nUploading to S3...")
	s3VideoKey := fmt.Sprintf("videos/%s.enc", metadata.OriginalName)
	s3KeyPath := fmt.Sprintf("keys/%s.enc.key", metadata.OriginalName)

	// Update metadata with consistent paths
	metadata.ID = s3VideoKey // Use the full path as ID

	if err := store.StoreVideo(ctx, bytes.NewReader(encryptedData), metadata); err != nil {
		log.Fatalf("Failed to store video: %v", err)
	}

	// Store encryption key in S3
	fmt.Println("Storing encryption key in S3...")
	if err := store.StoreKey(ctx, s3KeyPath, encOutput.Key); err != nil {
		log.Fatalf("Failed to store encryption key: %v", err)
	}

	// Print summary
	duration := time.Since(startTime)
	speed := float64(fileInfo.Size()) / duration.Seconds() / 1024 / 1024 // MB/s

	fmt.Println("\nUpload completed successfully!")
	fmt.Printf("Time taken: %v\n", duration)
	fmt.Printf("Average speed: %.2f MB/s\n", speed)
	fmt.Printf("S3 Video path: %s\n", s3VideoKey)
	fmt.Printf("S3 Key path: %s\n", s3KeyPath)
}
