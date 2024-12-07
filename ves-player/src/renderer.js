let selectedVideo = null;
let selectedKey = null;
let bindingInProgress = false;

// UI Elements
const bindingStatus = document.getElementById('binding-status');
const bindingSpinner = document.getElementById('binding-spinner');
const bindingMessage = document.getElementById('binding-message');
const retryButton = document.getElementById('retry-binding');

function showBindingStatus(message, type) {
    bindingStatus.className = 'status ' + type;
    bindingMessage.textContent = message;
    bindingStatus.style.display = 'block';
}

function hideBindingStatus() {
    bindingStatus.style.display = 'none';
    bindingSpinner.style.display = 'none';
    retryButton.style.display = 'none';
}

function showBindingProgress() {
    bindingStatus.className = 'status warning';
    bindingSpinner.style.display = 'inline-block';
    bindingMessage.textContent = 'Binding device to content...';
    retryButton.style.display = 'none';
    bindingStatus.style.display = 'block';
}

document.getElementById('select-video').addEventListener('click', async () => {
    try {
        const filePath = await window.api.selectVideo();
        if (filePath) {
            selectedVideo = filePath;
            document.getElementById('video-path').textContent = filePath;
            updatePlayButton();
        }
    } catch (error) {
        console.error('Error selecting video file:', error);
        showBindingStatus('Error selecting video file: ' + error.message, 'error');
    }
});

document.getElementById('select-key').addEventListener('click', async () => {
    try {
        const filePath = await window.api.selectKey();
        if (filePath) {
            selectedKey = filePath;
            document.getElementById('key-path').textContent = filePath;
            updatePlayButton();
        }
    } catch (error) {
        console.error('Error selecting key file:', error);
        showBindingStatus('Error selecting key file: ' + error.message, 'error');
    }
});

retryButton.addEventListener('click', async () => {
    hideBindingStatus();
    await playVideo();
});

function updatePlayButton() {
    const playButton = document.getElementById('play-video');
    playButton.disabled = !(selectedVideo && selectedKey) || bindingInProgress;
}

async function playVideo() {
    if (!selectedVideo || !selectedKey) {
        showBindingStatus('Please select both video and key files', 'warning');
        return;
    }

    const playButton = document.getElementById('play-video');
    playButton.disabled = true;
    bindingInProgress = true;
    showBindingProgress();

    try {
        const decryptedPath = await window.api.decryptVideo(selectedVideo, selectedKey);
        console.log('Decrypted file path:', decryptedPath);
        hideBindingStatus();

        // Create a proper file URL
        const videoUrl = `file://${decryptedPath.replace(/\\/g, '/')}`;
        console.log('Video URL:', videoUrl);
        
        // Reset video player
        const videoPlayer = document.getElementById('video-player');
        videoPlayer.pause();
        videoPlayer.removeAttribute('src');
        videoPlayer.load();

        // Set MIME type and source
        const source = document.createElement('source');
        source.src = videoUrl;
        source.type = 'video/mp4';
        
        // Remove any existing sources
        while (videoPlayer.firstChild) {
            videoPlayer.removeChild(videoPlayer.firstChild);
        }
        
        // Add new source
        videoPlayer.appendChild(source);

        // Set up event handlers before loading
        videoPlayer.onloadeddata = () => {
            console.log('Video loaded successfully');
            showBindingStatus('Device successfully bound to content', 'success');
            setTimeout(hideBindingStatus, 3000);
            playButton.textContent = 'Play Video';
            playButton.disabled = false;
            bindingInProgress = false;
            videoPlayer.play().catch(err => {
                console.error('Error playing video:', err);
                showBindingStatus('Error playing video: ' + err.message, 'error');
                playButton.textContent = 'Play Video';
                playButton.disabled = false;
                bindingInProgress = false;
            });
        };

        videoPlayer.onerror = (e) => {
            const error = videoPlayer.error;
            console.error('Video error:', error);
            console.error('Error code:', error.code);
            console.error('Error message:', error.message);
            showBindingStatus(`Error playing video (${error.code}): ${error.message}`, 'error');
            retryButton.style.display = 'inline-block';
            playButton.textContent = 'Play Video';
            playButton.disabled = false;
            bindingInProgress = false;
        };

        // Now load the video
        videoPlayer.load();
    } catch (error) {
        console.error('Error processing video:', error);
        if (error.message.includes('device binding')) {
            showBindingStatus('Device binding failed: ' + error.message, 'error');
            retryButton.style.display = 'inline-block';
        } else {
            showBindingStatus('Failed to process video: ' + error.message, 'error');
        }
        playButton.textContent = 'Play Video';
        playButton.disabled = false;
        bindingInProgress = false;
    }
}

document.getElementById('play-video').addEventListener('click', playVideo); 