#!/bin/bash
# Make this script executable with: chmod +x start.sh

# Canvas Conundrum Client - Development Start Script

echo "🎨 Canvas Conundrum - Starting Development Server"
echo "================================================"

# Check if node_modules exists
if [ ! -d "node_modules" ]; then
    echo "📦 Installing dependencies..."
    npm install
    if [ $? -ne 0 ]; then
        echo "❌ Failed to install dependencies"
        exit 1
    fi
else
    echo "✅ Dependencies already installed"
fi

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo "⚠️  No .env file found. Creating from .env.example..."
    if [ -f ".env.example" ]; then
        cp .env.example .env
        echo "✅ Created .env file. Please update it with your configuration."
    else
        echo "❌ No .env.example file found. Creating basic .env..."
        echo "REACT_APP_WS_URL=ws://localhost:8080/ws" > .env
    fi
fi

# Check if the server is running
SERVER_URL="localhost:8080"
echo "🔍 Checking if server is running at $SERVER_URL..."
if nc -z localhost 8080 2>/dev/null; then
    echo "✅ Server is running"
else
    echo "⚠️  Warning: Server doesn't appear to be running at $SERVER_URL"
    echo "   Make sure to start the server before playing the game"
fi

# Display environment info
echo ""
echo "📋 Environment Information:"
echo "   Node Version: $(node --version)"
echo "   NPM Version: $(npm --version)"
echo "   React Scripts: $(npm list react-scripts | grep react-scripts | head -1)"
echo ""

# Check for image assets
echo "🖼️  Checking for image assets..."
IMAGES_DIR="public/images"
if [ ! -d "$IMAGES_DIR/roles" ] || [ ! -d "$IMAGES_DIR/tokens" ]; then
    echo "⚠️  Missing image directories. Creating them..."
    mkdir -p "$IMAGES_DIR/roles"
    mkdir -p "$IMAGES_DIR/tokens"
    mkdir -p "$IMAGES_DIR/puzzles"
    echo "   Created: $IMAGES_DIR/roles"
    echo "   Created: $IMAGES_DIR/tokens"
    echo "   Created: $IMAGES_DIR/puzzles"
    echo "   ⚠️  Don't forget to add the actual images!"
fi

# Start the development server
echo ""
echo "🚀 Starting React development server..."
echo "   The app will open at http://localhost:3000"
echo "   Press Ctrl+C to stop the server"
echo ""

# Start the app
npm start