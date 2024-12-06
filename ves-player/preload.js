const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('api', {
    selectVideo: () => ipcRenderer.invoke('select-video'),
    selectKey: () => ipcRenderer.invoke('select-key'),
    decryptVideo: async (videoPath, keyPath) => {
        return await ipcRenderer.invoke('decrypt-video', videoPath, keyPath);
    }
}); 