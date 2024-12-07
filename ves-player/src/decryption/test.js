const fs = require('fs-extra');
const path = require('path');
const VideoDecryptor = require('./decryptor');
const DeviceFingerprint = require('../device/device');

async function runTests() {
    // Set development mode for detailed logs
    process.env.NODE_ENV = 'development';
    
    console.log('Starting decryptor tests...\n');
    
    const decryptor = new VideoDecryptor();
    await decryptor.initialize();

    try {
        // Test 1: First-time playback
        console.log('Test 1: First-time playback (should create binding)');
        const result1 = await decryptor.decryptFile(
            path.join(__dirname, '../../test/33239530-8665-48d4-9bd9-3024a6557278'),
            path.join(__dirname, '../../test/35729975-f372-41ee-959a-1d5cc10e0faa')
        );
        console.log('✅ First playback successful:', result1);
        console.log('Binding should be created at:', path.join(decryptor.bindingStorePath, '33239530-8665-48d4-9bd9-3024a6557278.json'));
        
        // Test 2: Subsequent playback
        console.log('\nTest 2: Subsequent playback (should verify binding)');
        const result2 = await decryptor.decryptFile(
            path.join(__dirname, '../../test/33239530-8665-48d4-9bd9-3024a6557278'),
            path.join(__dirname, '../../test/35729975-f372-41ee-959a-1d5cc10e0faa')
        );
        console.log('✅ Subsequent playback successful:', result2);

        // Test 3: Different device simulation
        console.log('\nTest 3: Different device simulation (should fail)');
        const bindingPath = path.join(decryptor.bindingStorePath, '33239530-8665-48d4-9bd9-3024a6557278.json');
        const binding = JSON.parse(await fs.readFile(bindingPath, 'utf8'));
        binding.hardwareHash = 'modified-hash';
        await fs.writeFile(bindingPath, JSON.stringify(binding));
        
        try {
            await decryptor.decryptFile(
                path.join(__dirname, '../../test/33239530-8665-48d4-9bd9-3024a6557278'),
                path.join(__dirname, '../../test/35729975-f372-41ee-959a-1d5cc10e0faa')
            );
            console.log('❌ Test failed: Decryption succeeded when it should have failed');
        } catch (error) {
            if (error.message.includes('binding appears to be corrupted')) {
                console.log('✅ Different device correctly rejected:', error.message);
            } else {
                console.log('❌ Unexpected error message:', error.message);
                throw error;
            }
        }

        // Test 4: Corrupted binding file
        console.log('\nTest 4: Corrupted binding file (should fail securely)');
        await fs.writeFile(bindingPath, 'invalid-json-data');
        try {
            await decryptor.decryptFile(
                path.join(__dirname, '../../test/33239530-8665-48d4-9bd9-3024a6557278'),
                path.join(__dirname, '../../test/35729975-f372-41ee-959a-1d5cc10e0faa')
            );
            console.log('❌ Test failed: Decryption succeeded with corrupted binding');
        } catch (error) {
            if (error.message.includes('binding appears to be corrupted')) {
                console.log('✅ Corrupted binding handled correctly:', error.message);
            } else {
                throw error;
            }
        }

        // Test 5: Missing key file
        console.log('\nTest 5: Missing key file (should fail gracefully)');
        // Reset binding to valid state first
        const deviceInfo = await decryptor.deviceFingerprint.getDeviceInfo();
        await fs.writeFile(bindingPath, JSON.stringify(deviceInfo));
        
        try {
            await decryptor.decryptFile(
                path.join(__dirname, '../../test/33239530-8665-48d4-9bd9-3024a6557278'),
                path.join(__dirname, '../../test/non-existent-key')
            );
            console.log('❌ Test failed: Decryption succeeded with missing key');
        } catch (error) {
            if (error.message.includes('Please try again or contact support')) {
                console.log('✅ Missing key handled correctly:', error.message);
            } else {
                throw error;
            }
        }

        // Test 6: Concurrent playback (should use same binding)
        console.log('\nTest 6: Concurrent playback (should use same binding)');
        const [result6a, result6b] = await Promise.all([
            decryptor.decryptFile(
                path.join(__dirname, '../../test/33239530-8665-48d4-9bd9-3024a6557278'),
                path.join(__dirname, '../../test/35729975-f372-41ee-959a-1d5cc10e0faa')
            ),
            decryptor.decryptFile(
                path.join(__dirname, '../../test/33239530-8665-48d4-9bd9-3024a6557278'),
                path.join(__dirname, '../../test/35729975-f372-41ee-959a-1d5cc10e0faa')
            )
        ]);
        console.log('✅ Concurrent playback successful');

        console.log('\nAll tests completed successfully! 🎉');
    } catch (error) {
        console.error('\n❌ Test failed:', error);
        process.exit(1);
    } finally {
        // Cleanup
        await decryptor.cleanup();
        console.log('\nTest cleanup completed');
    }
}

runTests().catch(console.error); 