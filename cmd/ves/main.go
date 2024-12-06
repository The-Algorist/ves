package main

import (
    "context"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
    "time"

    "ves/internal/core/domain"
    "ves/internal/encryption/chunking"
    "ves/internal/encryption/service"
    "ves/internal/pkg/crypto/aes"
)

func main() {
    fmt.Println("VES - Video Encryption Service Test")
    
    // Setup directories
    rawDir := "videos/raw"
    encDir := "videos/encrypted"
    decDir := "videos/decrypted"

    // Create directories if they don't exist
    for _, dir := range []string{rawDir, encDir, decDir} {
        if err := os.MkdirAll(dir, 0755); err != nil {
            fmt.Printf("Error creating directory %s: %v\n", dir, err)
            os.Exit(1)
        }
    }

    // Show operation menu
    fmt.Println("\nSelect operation:")
    fmt.Println("1. Encrypt video")
    fmt.Println("2. Decrypt video")
    
    var choice int
    fmt.Print("\nChoice: ")
    fmt.Scanln(&choice)

    switch choice {
    case 1:
        handleEncryption(rawDir, encDir)
    case 2:
        handleDecryption(encDir, decDir)
    default:
        fmt.Println("Invalid choice")
        os.Exit(1)
    }
}

func handleEncryption(rawDir, encDir string) {
    // List available raw videos
    files, err := os.ReadDir(rawDir)
    if err != nil {
        fmt.Printf("Error reading raw directory: %v\n", err)
        os.Exit(1)
    }

    if len(files) == 0 {
        fmt.Printf("No files found in %s. Please add a video file.\n", rawDir)
        os.Exit(1)
    }

    // Show available files
    fmt.Println("\nAvailable video files:")
    for i, f := range files {
        fmt.Printf("%d. %s\n", i+1, f.Name())
    }

    // Get user choice
    var choice int
    fmt.Print("\nSelect file number to process: ")
    fmt.Scanln(&choice)
    
    if choice < 1 || choice > len(files) {
        fmt.Println("Invalid choice")
        os.Exit(1)
    }

    filename := files[choice-1].Name()
    inputPath := filepath.Join(rawDir, filename)
    outputPath := filepath.Join(encDir, filename+".enc")

    // Open input file
    input, err := os.Open(inputPath)
    if err != nil {
        fmt.Printf("Error opening input file: %v\n", err)
        os.Exit(1)
    }
    defer input.Close()

    // Get file info for size
    fileInfo, err := input.Stat()
    if err != nil {
        fmt.Printf("Error getting file info: %v\n", err)
        os.Exit(1)
    }

    // Create output file
    output, err := os.Create(outputPath)
    if err != nil {
        fmt.Printf("Error creating output file: %v\n", err)
        os.Exit(1)
    }
    defer output.Close()

    // Create chunk reader
    reader, err := chunking.NewChunkReader(input, chunking.DefaultChunkSize)
    if err != nil {
        fmt.Printf("Failed to create chunk reader: %v\n", err)
        os.Exit(1)
    }

    // Enable features
    reader.EnableValidation(true)
    if err := reader.EnableConcurrency(true, chunking.DefaultWorkers); err != nil {
        fmt.Printf("Failed to enable concurrency: %v\n", err)
        os.Exit(1)
    }

    // Create encryption service
    encryptor := aes.NewAESEncryptor(32)  // 32 bytes for AES-256
    encryptionService := service.NewService(encryptor)

    fmt.Printf("\nEncrypting %s...\n", filename)

    var key []byte
    var iv []byte

    // Record start time
    start := time.Now()

    // Create encryption input for the entire file
    encInput := domain.EncryptionInput{
        Reader: input,
        Metadata: domain.VideoMetadata{
            FileName:    filename,
            Size:        fileInfo.Size(),
            ContentType: "video/mp4",
            CreatedAt:   time.Now(),
        },
        Options: domain.EncryptionOptions{
            ChunkSize: chunking.DefaultChunkSize,
            KeySize:   32,
        },
    }

    // Encrypt the entire file
    encOutput, err := encryptionService.Encrypt(context.Background(), encInput)
    if err != nil {
        fmt.Printf("Encryption error: %v\n", err)
        os.Exit(1)
    }

    // Store the key and IV
    key = encOutput.Key
    iv = encOutput.IV

    // Copy encrypted data to output file
    written, err := io.Copy(output, encOutput.EncryptedReader)
    if err != nil {
        fmt.Printf("Write error: %v\n", err)
        os.Exit(1)
    }

    duration := time.Since(start)
    fmt.Printf("\nEncryption completed in %v\n", duration)
    
    // Show final statistics
    fmt.Printf("\nFinal Statistics:\n")
    fmt.Printf("Total Bytes: %d\n", written)
    fmt.Printf("Processing Rate: %.2f MB/s\n", 
        float64(written)/(1024*1024*duration.Seconds()))

    // Save the key and IV for decryption
    keyPath := outputPath + ".key"
    ivPath := outputPath + ".iv"
    
    if err := os.WriteFile(keyPath, key, 0600); err != nil {
        fmt.Printf("Error saving encryption key: %v\n", err)
        os.Exit(1)
    }
    if err := os.WriteFile(ivPath, iv, 0600); err != nil {
        fmt.Printf("Error saving IV: %v\n", err)
        os.Exit(1)
    }
    fmt.Printf("\nEncryption key saved to: %s\n", keyPath)
    fmt.Printf("IV saved to: %s\n", ivPath)
}

func handleDecryption(encDir, decDir string) {
    // List available encrypted videos
    files, err := os.ReadDir(encDir)
    if err != nil {
        fmt.Printf("Error reading encrypted directory: %v\n", err)
        os.Exit(1)
    }

    // Filter for .enc files
    var encFiles []os.DirEntry
    for _, f := range files {
        if filepath.Ext(f.Name()) == ".enc" {
            encFiles = append(encFiles, f)
        }
    }

    if len(encFiles) == 0 {
        fmt.Printf("No encrypted files found in %s.\n", encDir)
        os.Exit(1)
    }

    // Show available files
    fmt.Println("\nAvailable encrypted files:")
    for i, f := range encFiles {
        fmt.Printf("%d. %s\n", i+1, f.Name())
    }

    // Get user choice
    var choice int
    fmt.Print("\nSelect file number to decrypt: ")
    fmt.Scanln(&choice)
    
    if choice < 1 || choice > len(encFiles) {
        fmt.Println("Invalid choice")
        os.Exit(1)
    }

    filename := encFiles[choice-1].Name()
    inputPath := filepath.Join(encDir, filename)
    outputPath := filepath.Join(decDir, strings.TrimSuffix(filename, ".enc"))
    keyPath := inputPath + ".key"

    // Read the encryption key and IV
    key, err := os.ReadFile(keyPath)
    if err != nil {
        fmt.Printf("Error reading encryption key: %v\n", err)
        os.Exit(1)
    }

    ivPath := inputPath + ".iv"
    iv, err := os.ReadFile(ivPath)
    if err != nil {
        fmt.Printf("Error reading IV: %v\n", err)
        os.Exit(1)
    }

    // Open input file
    input, err := os.Open(inputPath)
    if err != nil {
        fmt.Printf("Error opening encrypted file: %v\n", err)
        os.Exit(1)
    }
    defer input.Close()

    // Create output file
    output, err := os.Create(outputPath)
    if err != nil {
        fmt.Printf("Error creating output file: %v\n", err)
        os.Exit(1)
    }
    defer output.Close()

    // Create encryption service
    encryptor := aes.NewAESEncryptor(32)
    encryptionService := service.NewService(encryptor)

    fmt.Printf("\nDecrypting %s...\n", filename)
    start := time.Now()

    // Decrypt the file
    decryptedReader, err := encryptionService.Decrypt(context.Background(), input, key, iv)
    if err != nil {
        fmt.Printf("Decryption error: %v\n", err)
        os.Exit(1)
    }

    // Copy decrypted data to output file
    written, err := io.Copy(output, decryptedReader)
    if err != nil {
        fmt.Printf("Write error: %v\n", err)
        os.Exit(1)
    }

    duration := time.Since(start)
    fmt.Printf("\nDecryption completed in %v\n", duration)
    fmt.Printf("Bytes written: %d\n", written)
    fmt.Printf("Processing Rate: %.2f MB/s\n", 
        float64(written)/(1024*1024*duration.Seconds()))
}