package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/joho/godotenv"
)

const (
	// 10MB part size (larger than minimum to reduce number of parts)
	minPartSize = 10 * 1024 * 1024
	// Reduced concurrency to avoid network issues
	maxConcurrency = 5
	// Maximum retries per part
	maxRetries = 3
)

// uploadPart handles the upload of a single part with retries
func uploadPart(ctx context.Context, client *s3.Client, bucket, key string, uploadID string, partNumber int32, data []byte) (*types.CompletedPart, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := time.Duration(attempt*attempt) * time.Second
			time.Sleep(delay)
			fmt.Printf("Retrying part %d (attempt %d/%d)\n", partNumber, attempt+1, maxRetries)
		}

		response, err := client.UploadPart(ctx, &s3.UploadPartInput{
			Bucket:     aws.String(bucket),
			Key:        aws.String(key),
			PartNumber: aws.Int32(partNumber),
			UploadId:   aws.String(uploadID),
			Body:       bytes.NewReader(data),
		})
		if err == nil {
			return &types.CompletedPart{
				ETag:       response.ETag,
				PartNumber: aws.Int32(partNumber),
			}, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("failed to upload part %d after %d attempts: %w", partNumber, maxRetries, lastErr)
}

// uploadMultipart performs a multipart upload with concurrent part uploads
func uploadMultipart(ctx context.Context, client *s3.Client, bucket, key string, data []byte) error {
	// Create multipart upload
	createResp, err := client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to create multipart upload: %w", err)
	}
	uploadID := *createResp.UploadId

	// Calculate part size and number of parts
	fileSize := len(data)
	partSize := max(minPartSize, fileSize/maxConcurrency)
	numParts := (fileSize + partSize - 1) / partSize

	fmt.Printf("Uploading %d parts of %d bytes each\n", numParts, partSize)

	// Channel for completed parts
	completedParts := make(chan *types.CompletedPart, numParts)
	var wg sync.WaitGroup
	errChan := make(chan error, 1)

	// Semaphore to limit concurrency
	sem := make(chan struct{}, maxConcurrency)

	// Upload parts concurrently
	for i := 0; i < numParts; i++ {
		wg.Add(1)
		go func(partNum int) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			start := partNum * partSize
			end := min((partNum+1)*partSize, fileSize)
			partData := data[start:end]

			fmt.Printf("Starting part %d/%d (%d bytes)\n", partNum+1, numParts, len(partData))
			part, err := uploadPart(ctx, client, bucket, key, uploadID, int32(partNum+1), partData)
			if err != nil {
				select {
				case errChan <- err:
				default:
				}
				return
			}
			fmt.Printf("Completed part %d/%d\n", partNum+1, numParts)

			completedParts <- part
		}(i)
	}

	// Wait for all parts to complete or error
	go func() {
		wg.Wait()
		close(completedParts)
	}()

	// Collect completed parts
	var parts []types.CompletedPart
	for part := range completedParts {
		parts = append(parts, *part)
	}

	// Check for errors
	select {
	case err := <-errChan:
		// Abort upload on error
		_, abortErr := client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
			Bucket:   aws.String(bucket),
			Key:      aws.String(key),
			UploadId: aws.String(uploadID),
		})
		if abortErr != nil {
			log.Printf("Warning: Failed to abort multipart upload: %v", abortErr)
		}
		return err
	default:
	}

	// Complete multipart upload
	_, err = client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: parts,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	return nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	if len(os.Args) < 2 {
		log.Fatal("Please provide the path to a file to upload")
	}
	filePath := os.Args[1]

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Fatalf("Failed to get file info: %v", err)
	}
	fmt.Printf("File size: %.2f MB\n", float64(fileInfo.Size())/(1024*1024))

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	ctx := context.Background()

	// Initialize AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Unable to load SDK config: %v", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)
	bucket := os.Getenv("AWS_BUCKET_NAME")
	key := fmt.Sprintf("test-uploads/%s", filepath.Base(filePath))

	// Upload file
	fmt.Printf("Uploading to s3://%s/%s...\n", bucket, key)
	startTime := time.Now()

	err = uploadMultipart(ctx, client, bucket, key, data)
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}

	duration := time.Since(startTime)
	speed := float64(len(data)) / duration.Seconds() / 1024 / 1024 // MB/s

	fmt.Println("\nUpload completed successfully!")
	fmt.Printf("Time taken: %v\n", duration)
	fmt.Printf("Average speed: %.2f MB/s\n", speed)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
} 