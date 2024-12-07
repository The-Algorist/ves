package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "path/filepath"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/joho/godotenv"

    "ves/internal/storage/s3"
)

func main() {
    if err := godotenv.Load(); err != nil {
        log.Fatal("Error loading .env file")
    }

    if len(os.Args) < 3 {
        log.Fatal("Usage: download <video-id> <output-dir>")
    }
    videoID := os.Args[1]
    outputDir := os.Args[2]

    // Ensure output directory exists
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        log.Fatalf("Failed to create output directory: %v", err)
    }

    ctx := context.Background()

    // Initialize AWS configuration
    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        log.Fatalf("Unable to load SDK config: %v", err)
    }

    // Create storage client
    store, err := s3.NewClient(ctx, cfg, os.Getenv("AWS_BUCKET_NAME"))
    if err != nil {
        log.Fatalf("Failed to create storage client: %v", err)
    }

    // Get video and metadata
    fmt.Printf("Downloading video %s...\n", videoID)
    videoData, metadata, err := store.GetVideo(ctx, videoID)
    if err != nil {
        log.Fatalf("Failed to get video: %v", err)
    }

    // Save video with original filename + .enc
    videoPath := filepath.Join(outputDir, metadata.OriginalName+".enc")
    if err := os.WriteFile(videoPath, videoData, 0644); err != nil {
        log.Fatalf("Failed to save video: %v", err)
    }
    fmt.Printf("Saved encrypted video to: %s\n", videoPath)

    // Get key metadata
    fmt.Println("\nGetting key metadata...")
    _, keyMeta, err := store.GenerateKey(ctx, videoID)
    if err != nil {
        log.Fatalf("Failed to get key metadata: %v", err)
    }

    // Get and save key
    fmt.Printf("Downloading key %s...\n", keyMeta.ID)
    keyData, err := store.GetKey(ctx, keyMeta.ID)
    if err != nil {
        log.Fatalf("Failed to get key: %v", err)
    }

    // Save key with same pattern as CLI
    keyPath := videoPath + ".key"
    if err := os.WriteFile(keyPath, keyData, 0600); err != nil {
        log.Fatalf("Failed to save key: %v", err)
    }
    fmt.Printf("Saved key to: %s\n", keyPath)

    fmt.Println("\nDownload completed successfully!")
    fmt.Printf("Video saved as: %s\n", videoPath)
    fmt.Printf("Key saved as: %s\n", keyPath)
} 