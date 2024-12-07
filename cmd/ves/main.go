package main

import (
    "context"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
    "time"
    "bytes"

    "ves/internal/core/domain"
    "ves/internal/encryption/service"
    "ves/internal/pkg/crypto/aes"
)

// Update paths to use Windows vesplayer directory
const (
    // WSL path that maps to D:\Personal\vesplayer\ves-player\ves-videos
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
    fmt.Println("VES - Video Encryption Service")
    
    // Show operation menu
    fmt.Println("\nSelect operation:")
    fmt.Println("1. Encrypt video")
    fmt.Println("2. Decrypt video")
    
    var choice int
    fmt.Print("\nChoice: ")
    fmt.Scanln(&choice)

    switch choice {
    case 1:
        handleEncryption(rawVideoDir, encryptedVideoDir)
    case 2:
        handleDecryption(encryptedVideoDir, decryptedVideoDir)
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

    // Create encryption service
    encryptor := aes.NewAESEncryptor(32)  // 32 bytes for AES-256
    encryptionService := service.NewService(encryptor)

    fmt.Printf("\nEncrypting %s...\n", filename)
    startTime := time.Now()

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
            ChunkSize: 1024 * 1024, // 1MB chunks
            KeySize:   32,          // AES-256
        },
    }

    // Encrypt the file
    encOutput, err := encryptionService.Encrypt(context.Background(), encInput)
    if err != nil {
        fmt.Printf("Encryption error: %v\n", err)
        os.Exit(1)
    }

    // Copy encrypted data to output file
    written, err := io.Copy(output, encOutput.EncryptedReader)
    if err != nil {
        fmt.Printf("Write error: %v\n", err)
        os.Exit(1)
    }

    duration := time.Since(startTime)
    fmt.Printf("\nEncryption completed in %v\n", duration)

    fmt.Printf("\nFinal Statistics:\n")
    fmt.Printf("Total Bytes: %d\n", written)
    fmt.Printf("Processing Rate: %.2f MB/s\n", 
        float64(written)/(1024*1024*duration.Seconds()))

    // Save the key
    keyPath := outputPath + ".key"
    if err := os.WriteFile(keyPath, encOutput.Key, 0600); err != nil {
        fmt.Printf("Error saving encryption key: %v\n", err)
        os.Exit(1)
    }
    fmt.Printf("\nEncryption key saved to: %s\n", keyPath)
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
        if strings.HasSuffix(f.Name(), ".enc") {
            encFiles = append(encFiles, f)
        }
    }

    if len(encFiles) == 0 {
        fmt.Printf("No encrypted files found in %s\n", encDir)
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

    // Read key file
    keyPath := inputPath + ".key"
    key, err := os.ReadFile(keyPath)
    if err != nil {
        fmt.Printf("Error reading encryption key: %v\n", err)
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

    // Read all encrypted data
    encryptedData, err := io.ReadAll(input)
    if err != nil {
        fmt.Printf("Error reading encrypted file: %v\n", err)
        os.Exit(1)
    }

    // Decrypt the file
    decryptedReader, err := encryptionService.Decrypt(context.Background(), bytes.NewReader(encryptedData), key, nil)
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