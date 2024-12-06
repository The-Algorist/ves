let selectedVideo = null;
let selectedKey = null;

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
        alert('Error selecting video file: ' + error.message);
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
        alert('Error selecting key file: ' + error.message);
    }
});

function updatePlayButton() {
    const playButton = document.getElementById('play-video');
    playButton.disabled = !(selectedVideo && selectedKey);
}

document.getElementById('play-video').addEventListener('click', async () => {
    if (!selectedVideo || !selectedKey) {
        alert('Please select both video and key files');
        return;
    }

    try {
        const decryptedPath = await window.api.decryptVideo(selectedVideo, selectedKey);
        console.log('Decrypted file path:', decryptedPath);

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

        // Show loading state
        const playButton = document.getElementById('play-video');
        playButton.disabled = true;
        playButton.textContent = 'Decrypting...';

        // Set up event handlers before loading
        videoPlayer.onloadeddata = () => {
            console.log('Video loaded successfully');
            playButton.textContent = 'Play Video';
            playButton.disabled = false;
            videoPlayer.play().catch(err => {
                console.error('Error playing video:', err);
                alert('Error playing video: ' + err.message);
                playButton.textContent = 'Play Video';
                playButton.disabled = false;
            });
        };

        videoPlayer.onerror = (e) => {
            const error = videoPlayer.error;
            console.error('Video error:', error);
            console.error('Error code:', error.code);
            console.error('Error message:', error.message);
            alert(`Error playing video (${error.code}): ${error.message}`);
            playButton.textContent = 'Play Video';
            playButton.disabled = false;
        };

        // Now load the video
        videoPlayer.load();
    } catch (error) {
        console.error('Error processing video:', error);
        alert('Failed to process video: ' + error.message);
        playButton.textContent = 'Play Video';
        playButton.disabled = false;
    }
}); 