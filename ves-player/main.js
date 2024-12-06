const { app, BrowserWindow, ipcMain, protocol } = require('electron');
const path = require('path');
const VideoDecryptor = require('./src/decryption/decryptor');

// Start the server
require('./server');

let decryptor = null;
let mainWindow = null;

function createWindow() {
    console.log('Creating window...');
    mainWindow = new BrowserWindow({
        width: 800,
        height: 600,
        webPreferences: {
            nodeIntegration: false,
            contextIsolation: true,
            preload: path.join(__dirname, 'preload.js'),
            webSecurity: false  // Allow loading local files
        },
        autoHideMenuBar: true,
        backgroundColor: '#ffffff'
    });

    // Load from local server
    const serverUrl = 'http://localhost:3000';
    console.log('Loading from server:', serverUrl);
    
    mainWindow.loadURL(serverUrl).catch(err => {
        console.error('Failed to load from server:', err);
    });
    
    if (process.env.NODE_ENV === 'development') {
        mainWindow.webContents.openDevTools();
    }

    mainWindow.webContents.on('crashed', () => {
        console.log('Window crashed! Attempting to reload...');
        mainWindow.reload();
    });

    mainWindow.on('closed', () => {
        mainWindow = null;
    });

    // Register protocol for local files
    protocol.registerFileProtocol('local-file', (request, callback) => {
        const filePath = request.url.replace('local-file://', '');
        callback({ path: filePath });
    });
}

// Handle startup
app.whenReady().then(async () => {
    try {
        console.log('Initializing decryptor...');
        decryptor = new VideoDecryptor();
        await decryptor.initialize();
        
        // Wait a bit for the server to start
        setTimeout(createWindow, 1000);
    } catch (error) {
        console.error('Failed to initialize:', error);
    }
});

app.on('window-all-closed', async () => {
    if (decryptor) {
        await decryptor.cleanup();
    }
    app.quit();
});

app.on('activate', () => {
    if (mainWindow === null) {
        createWindow();
    }
});

// Handle video file selection
ipcMain.handle('select-video', async () => {
    console.log('Handling video selection...');
    const { dialog } = require('electron');
    try {
        const result = await dialog.showOpenDialog(mainWindow, {
            properties: ['openFile'],
            filters: [
                { name: 'Encrypted Videos', extensions: ['enc'] }
            ]
        });
        console.log('Selected video file:', result.filePaths[0]);
        return result.filePaths[0];
    } catch (error) {
        console.error('File selection error:', error);
        return null;
    }
});

// Handle key file selection
ipcMain.handle('select-key', async () => {
    console.log('Handling key selection...');
    const { dialog } = require('electron');
    try {
        const result = await dialog.showOpenDialog(mainWindow, {
            properties: ['openFile'],
            filters: [
                { name: 'Key Files', extensions: ['key'] }
            ]
        });
        console.log('Selected key file:', result.filePaths[0]);
        return result.filePaths[0];
    } catch (error) {
        console.error('File selection error:', error);
        return null;
    }
});

// Handle video decryption
ipcMain.handle('decrypt-video', async (event, videoPath, keyPath) => {
    console.log('Handling video decryption:', videoPath);
    console.log('Using key file:', keyPath);
    if (!decryptor) {
        throw new Error('Decryptor not initialized');
    }
    return await decryptor.decryptFile(videoPath, keyPath);
}); 