package main

import (
    "context"
    "fmt"
    "log"
    //"os"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func main() {
    ctx := context.Background()

    // Use default AWS configuration (from ~/.aws/credentials)
    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        log.Fatalf("Unable to load SDK config: %v", err)
    }

    // Set bucket name
    bucketName := "ves-video-storage"

    // Create S3 client
    client := s3.NewFromConfig(cfg)

    // Check if bucket exists
    _, err = client.HeadBucket(ctx, &s3.HeadBucketInput{
        Bucket: &bucketName,
    })
    if err != nil {
        // Create bucket if it doesn't exist
        fmt.Printf("Creating bucket %s...\n", bucketName)
        input := &s3.CreateBucketInput{
            Bucket: &bucketName,
        }

        // Only add location constraint if not in us-east-1
        if cfg.Region != "us-east-1" {
            input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
                LocationConstraint: types.BucketLocationConstraint(cfg.Region),
            }
        }

        _, err = client.CreateBucket(ctx, input)
        if err != nil {
            log.Fatalf("Unable to create bucket: %v", err)
        }
    } else {
        fmt.Printf("Bucket %s already exists\n", bucketName)
    }

    // Create folder structure
    folders := []string{
        "keys/",
        "videos/",
        "metadata/",
        "raw/",
    }

    for _, folder := range folders {
        _, err = client.PutObject(ctx, &s3.PutObjectInput{
            Bucket: &bucketName,
            Key:    &folder,
        })
        if err != nil {
            log.Printf("Warning: Unable to create folder %s: %v", folder, err)
        } else {
            fmt.Printf("Created folder: %s\n", folder)
        }
    }

    fmt.Println("\nSetup completed successfully!")
    fmt.Println("\nBucket configuration:")
    fmt.Printf("- Name: %s\n", bucketName)
    fmt.Printf("- Region: %s\n", cfg.Region)
    fmt.Println("\nFolder structure:")
    for _, folder := range folders {
        fmt.Printf("- %s\n", folder)
    }
} 