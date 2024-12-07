const os = require('os');
const crypto = require('crypto');
const { machineIdSync } = require('node-machine-id');

class DeviceFingerprint {
    constructor() {
        this.deviceId = null;
        this.hardwareHash = null;
        this.platform = null;
    }

    async initialize() {
        try {
            // Get unique machine ID
            this.deviceId = machineIdSync();

            // Generate hardware hash based on system info
            const hwInfo = {
                cpus: os.cpus().map(cpu => ({
                    model: cpu.model,
                    speed: cpu.speed
                })),
                totalmem: os.totalmem(),
                arch: os.arch(),
                platform: os.platform(),
                release: os.release(),
                hostname: os.hostname()
            };

            // Create a stable hash of hardware info
            const hwString = JSON.stringify(hwInfo, Object.keys(hwInfo).sort());
            this.hardwareHash = crypto
                .createHash('sha256')
                .update(hwString)
                .digest('hex');

            // Set platform info
            this.platform = `${os.platform()}-${os.arch()}`;

            return {
                deviceId: this.deviceId,
                hardwareHash: this.hardwareHash,
                platform: this.platform
            };
        } catch (error) {
            throw new Error(`Failed to initialize device fingerprint: ${error.message}`);
        }
    }

    getDeviceInfo() {
        if (!this.deviceId || !this.hardwareHash || !this.platform) {
            throw new Error('Device fingerprint not initialized');
        }

        return {
            deviceId: this.deviceId,
            hardwareHash: this.hardwareHash,
            platform: this.platform
        };
    }

    // Verify if current device matches stored device info
    verifyDevice(storedInfo) {
        if (!this.deviceId || !this.hardwareHash || !this.platform) {
            throw new Error('Device fingerprint not initialized');
        }

        return (
            storedInfo.deviceId === this.deviceId &&
            storedInfo.hardwareHash === this.hardwareHash &&
            storedInfo.platform === this.platform
        );
    }
}

module.exports = DeviceFingerprint; 