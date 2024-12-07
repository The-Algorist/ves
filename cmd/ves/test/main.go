package main

import (
    "context"
    "fmt"
    "log"
    // "os"
    // "time"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/aws/aws-sdk-go-v2/service/s3/types"
    "github.com/google/uuid"
    // "github.com/joho/godotenv"

    "ves/internal/device"
    // "ves/internal/storage"
    s3store "ves/internal/storage/s3"  // Aliased import to avoid confusion
)

func main() {
    ctx := context.Background()

    // Initialize device fingerprinting
    fingerprinter := device.New()
    deviceInfo, err := fingerprinter.GetDeviceInfo()
    if err != nil {
        log.Fatalf("Failed to get device info: %v", err)
    }

    fmt.Printf("Device Info:\n")
    fmt.Printf("  ID: %s\n", deviceInfo.DeviceID)
    fmt.Printf("  Hardware Hash: %s\n", deviceInfo.HardwareHash)
    fmt.Printf("  Platform: %s\n", deviceInfo.Platform)

    // Initialize AWS configuration
    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        log.Fatalf("Failed to load AWS config: %v", err)
    }

    // Create S3 client
    s3Client := s3.NewFromConfig(cfg)

    // Ensure bucket exists
    bucketName := "ves-test-bucket"
    region := cfg.Region
    fmt.Printf("\nChecking/creating bucket %s...\n", bucketName)
    
    // Check if bucket exists
    _, err = s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
        Bucket: &bucketName,
    })
    if err != nil {
        // Create bucket if it doesn't exist
        fmt.Printf("Creating bucket %s...\n", bucketName)
        input := &s3.CreateBucketInput{
            Bucket: &bucketName,
        }

        // Only add location constraint if not in us-east-1
        if region != "us-east-1" {
            input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
                LocationConstraint: types.BucketLocationConstraint(region),
            }
        }

        _, err = s3Client.CreateBucket(ctx, input)
        if err != nil {
            log.Fatalf("Failed to create bucket: %v", err)
        }
    }

    // Create storage client
    store, err := s3store.NewClient(ctx, cfg, bucketName)
    if err != nil {
        log.Fatalf("Failed to create storage client: %v", err)
    }

    // Test video ID
    videoID := uuid.NewString()

    // Generate encryption key
    fmt.Printf("\nGenerating encryption key for video %s...\n", videoID)
    _, meta1, err := store.GenerateKey(ctx, videoID)
    if err != nil {
        log.Fatalf("Failed to generate encryption key: %v", err)
    }
    fmt.Printf("Generated encryption key: %s\n", meta1.ID)

    fmt.Println("\nTest completed successfully!")
    fmt.Println("\nSummary:")
    fmt.Printf("- Video ID: %s\n", videoID)
    fmt.Printf("- Key ID: %s\n", meta1.ID)
} 