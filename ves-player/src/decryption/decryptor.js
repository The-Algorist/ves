const fs = require('fs-extra');
const path = require('path');
const os = require('os');
const crypto = require('crypto');

class VideoDecryptor {
    constructor() {
        this.tempDir = path.join(os.tmpdir(), 'ves-player');
    }

    async initialize() {
        await fs.ensureDir(this.tempDir);
    }

    async cleanup() {
        try {
            await fs.remove(this.tempDir);
        } catch (error) {
            console.error('Error cleaning up temp files:', error);
        }
    }

    readUInt32BE(buffer, offset) {
        return (buffer[offset] << 24) |
               (buffer[offset + 1] << 16) |
               (buffer[offset + 2] << 8) |
               buffer[offset + 3];
    }

    async decryptFile(encryptedFilePath, keyPath) {
        try {
            console.log('Starting decryption...');
            
            // Create a unique temp file path
            const tempFilePath = path.join(this.tempDir, `dec_${Date.now()}.mp4`);
            
            // Read the encrypted file as a buffer
            const encryptedData = await fs.readFile(encryptedFilePath);
            console.log('Read encrypted data:', encryptedData.length, 'bytes');
            
            // Read the key file
            const key = await fs.readFile(keyPath);
            console.log('Read key:', key.length, 'bytes');

            // Read metadata size from last 4 bytes
            const metadataSize = this.readUInt32BE(encryptedData, encryptedData.length - 4);
            console.log('Metadata size:', metadataSize);

            // Calculate where the encrypted data ends and metadata begins
            const metadataStart = encryptedData.length - 4 - metadataSize;
            console.log('Metadata starts at:', metadataStart);

            // Create write stream for decrypted data
            const writeStream = fs.createWriteStream(tempFilePath);

            // Process encrypted data in chunks
            let position = 0;
            while (position < metadataStart) {
                // Read chunk header (8 bytes: 4 for IV size, 4 for encrypted size)
                const ivSize = this.readUInt32BE(encryptedData, position);
                const encryptedSize = this.readUInt32BE(encryptedData, position + 4);
                position += 8;

                if (ivSize !== 12) {
                    throw new Error(`Invalid IV size: ${ivSize}, expected 12`);
                }

                console.log(`Processing chunk - IV size: ${ivSize}, Encrypted size: ${encryptedSize}`);

                // Read IV (nonce) - it's appended to the header
                const iv = encryptedData.slice(position, position + ivSize);
                position += ivSize;

                // Read encrypted chunk (includes auth tag)
                const encryptedChunk = encryptedData.slice(position, position + encryptedSize);
                position += encryptedSize;

                // The last 16 bytes of the encrypted chunk is the auth tag
                const authTag = encryptedChunk.slice(-16);
                const ciphertext = encryptedChunk.slice(0, -16);

                try {
                    // Create decipher
                    const decipher = crypto.createDecipheriv('aes-256-gcm', key, iv);
                    decipher.setAuthTag(authTag);

                    // Decrypt
                    const decrypted = Buffer.concat([
                        decipher.update(ciphertext),
                        decipher.final()
                    ]);

                    // Write decrypted chunk
                    writeStream.write(decrypted);
                    console.log(`Processed chunk of ${decrypted.length} bytes`);
                } catch (error) {
                    console.error('Chunk decryption error:', error);
                    throw new Error('Failed to decrypt chunk: ' + error.message);
                }
            }

            // Close the write stream
            await new Promise((resolve, reject) => {
                writeStream.end((err) => {
                    if (err) reject(err);
                    else resolve();
                });
            });

            console.log('Decryption complete');
            console.log('Wrote decrypted file:', tempFilePath);
            
            return tempFilePath;
        } catch (error) {
            console.error('Decryption error:', error);
            throw new Error('Failed to decrypt video file: ' + error.message);
        }
    }
}

module.exports = VideoDecryptor;