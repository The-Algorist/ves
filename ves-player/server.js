const express = require('express');
const path = require('path');
const cors = require('cors');

const app = express();
const port = 3000;

// Enable CORS for Electron
app.use(cors());

// Serve static files from src directory
app.use(express.static(path.join(__dirname, 'src')));

// Serve the main page
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'src', 'index.html'));
});

app.listen(port, () => {
    console.log(`Server running at http://localhost:${port}`);
}); 