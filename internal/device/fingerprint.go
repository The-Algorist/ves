package device

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "os"
    "runtime"
    "strings"

    "github.com/denisbrodbeck/machineid"
    "github.com/jaypipes/ghw"
    
    "ves/internal/storage"
)

// Fingerprinter generates device-specific information
type Fingerprinter struct {
    // Add any configuration if needed
}

// New creates a new Fingerprinter
func New() *Fingerprinter {
    return &Fingerprinter{}
}

// GetDeviceInfo collects hardware-specific information
func (f *Fingerprinter) GetDeviceInfo() (storage.DeviceInfo, error) {
    // Get machine ID (unique per device)
    machineID, err := machineid.ID()
    if err != nil {
        return storage.DeviceInfo{}, fmt.Errorf("failed to get machine ID: %w", err)
    }

    // Get hardware information
    cpu, err := ghw.CPU()
    if err != nil {
        return storage.DeviceInfo{}, fmt.Errorf("failed to get CPU info: %w", err)
    }

    memory, err := ghw.Memory()
    if err != nil {
        return storage.DeviceInfo{}, fmt.Errorf("failed to get memory info: %w", err)
    }

    // Collect hardware fingerprints
    fingerprints := map[string]string{
        "cpu_model":     cpu.Processors[0].Model,
        "cpu_vendor":    cpu.Processors[0].Vendor,
        "total_memory":  fmt.Sprintf("%d", memory.TotalPhysicalBytes),
        "os":           runtime.GOOS,
        "arch":         runtime.GOARCH,
        "hostname":     getHostname(),
    }

    // Generate hardware hash
    hashInput := []string{
        machineID,
        fmt.Sprintf("%d", cpu.Processors[0].ID),
        fmt.Sprintf("%d", memory.TotalPhysicalBytes),
        runtime.GOOS,
        runtime.GOARCH,
    }
    hardwareHash := generateHash(strings.Join(hashInput, "|"))

    return storage.DeviceInfo{
        DeviceID:     machineID,
        HardwareHash: hardwareHash,
        Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
        Fingerprint:  fingerprints,
    }, nil
}

// ValidateDevice checks if the current device matches the stored device info
func (f *Fingerprinter) ValidateDevice(storedInfo storage.DeviceInfo) (bool, error) {
    currentInfo, err := f.GetDeviceInfo()
    if err != nil {
        return false, fmt.Errorf("failed to get current device info: %w", err)
    }

    // Check primary identifiers
    if currentInfo.DeviceID != storedInfo.DeviceID ||
       currentInfo.HardwareHash != storedInfo.HardwareHash {
        return false, nil
    }

    // Check if essential hardware components match
    if currentInfo.Fingerprint["cpu_model"] != storedInfo.Fingerprint["cpu_model"] ||
       currentInfo.Fingerprint["total_memory"] != storedInfo.Fingerprint["total_memory"] {
        return false, nil
    }

    return true, nil
}

func generateHash(input string) string {
    hash := sha256.New()
    hash.Write([]byte(input))
    return hex.EncodeToString(hash.Sum(nil))
}

func getHostname() string {
    hostname, err := os.Hostname()
    if err != nil {
        return "unknown"
    }
    return hostname
} 