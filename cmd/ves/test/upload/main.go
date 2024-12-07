package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "time"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/sts"
    "github.com/google/uuid"
    "github.com/joho/godotenv"

    "ves/internal/storage"
    "ves/internal/storage/s3"
)

func main() {
    if err := godotenv.Load(); err != nil {
        log.Fatal("Error loading .env file")
    }

    if len(os.Args) < 2 {
        log.Fatal("Please provide the path to a video file")
    }
    videoPath := os.Args[1]

    // Read video file
    videoData, err := os.ReadFile(videoPath)
    if err != nil {
        log.Fatalf("Failed to read video file: %v", err)
    }

    ctx := context.Background()

    // Initialize AWS configuration
    cfg, err := config.LoadDefaultConfig(ctx,
        config.WithRegion("us-east-1"),
        config.WithSharedConfigProfile("default"),
    )
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
    store, err := s3.NewClient(ctx, cfg, os.Getenv("AWS_BUCKET_NAME"))
    if err != nil {
        // Try to get more details about the error
        fmt.Printf("Error details: %v\n", err)
        fmt.Printf("AWS Region: %s\n", cfg.Region)
        fmt.Printf("Bucket: %s\n", os.Getenv("AWS_BUCKET_NAME"))
        log.Fatalf("Failed to create storage client: %v", err)
    }

    // Create video metadata
    videoID := uuid.NewString()
    metadata := storage.VideoMetadata{
        ID:           videoID,
        OriginalName: filepath.Base(videoPath),
        ContentType:  "video/mp4",
        Size:        int64(len(videoData)),
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
        Tags:        []string{"test"},
    }

    // Store video
    fmt.Println("Uploading video...")
    if err := store.StoreVideo(ctx, videoData, metadata); err != nil {
        log.Fatalf("Failed to store video: %v", err)
    }
    fmt.Printf("Video uploaded successfully with ID: %s\n", videoID)

    // Generate encryption key
    fmt.Println("\nGenerating encryption key...")
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