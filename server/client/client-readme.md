# Canvas Conundrum - Mobile Client

A React-based mobile web application for the Canvas Conundrum collaborative puzzle game.

## Features

- **Stunning Animations**: Complex phase transitions, token animations, and visual effects using Framer Motion
- **Mobile-First Design**: Optimized for portrait mobile devices with touch interactions and haptic feedback
- **QR Code Scanning**: Built-in QR scanner for resource station verification
- **Real-time WebSocket**: Live game updates and synchronization with the server
- **Progressive Web App**: Installable on mobile devices with offline capabilities
- **Beautiful UI**: Clean, minimalist design with a turquoise/teal color scheme

## Prerequisites

- Node.js (v14 or higher)
- npm or yarn
- Canvas Conundrum server running on `ws://localhost:8080/ws`

## Installation

1. Navigate to the client directory:
```bash
cd client
```

2. Install dependencies:
```bash
npm install
```

## Running the Application

### Development Mode
```bash
npm start
```
The app will open at `http://localhost:3000`

### Production Build
```bash
npm run build
```
Creates an optimized build in the `build` folder

### Environment Variables

Create a `.env` file in the client directory:
```env
REACT_APP_WS_URL=ws://your-server-url:8080/ws
```

## Project Structure

```
client/
├── public/
│   ├── index.html          # HTML template
│   ├── manifest.json       # PWA manifest
│   └── images/            # Static images
│       └── puzzles/       # Puzzle images (to be added)
├── src/
│   ├── components/        # React components
│   │   ├── SetupPhase.js         # Role/specialty selection
│   │   ├── ResourceGatheringPhase.js  # QR scanning & trivia
│   │   ├── PuzzleAssemblyPhase.js     # Puzzle solving
│   │   ├── PostGamePhase.js           # Analytics display
│   │   ├── TokenHeader.js             # Token progress display
│   │   ├── TriviaQuestion.js          # Trivia component
│   │   ├── PhaseTransition.js         # Phase animations
│   │   ├── ConnectionOverlay.js       # Connection status
│   │   └── SwapRequestList.js         # Swap request UI
│   ├── hooks/
│   │   └── useWebSocket.js    # WebSocket connection hook
│   ├── constants.js           # Game constants
│   ├── App.js                # Main app component
│   ├── App.css               # Main styles
│   ├── index.js              # Entry point
│   └── index.css             # Global styles
└── package.json
```

## Game Phases

### 1. Setup Phase
- Players select their role (Art Enthusiast, Detective, Tourist, or Janitor)
- Choose 1-2 trivia specialties for bonus points
- Beautiful waiting animation while other players join

### 2. Resource Gathering
- Navigate to physical QR code stations
- Scan codes to verify location
- Answer trivia questions to earn tokens
- Real-time token progress tracking

### 3. Puzzle Assembly
- Solve individual 16-piece puzzle segments
- Collaborate on master puzzle grid
- Swap puzzle pieces with gesture controls
- Handle incoming swap requests

### 4. Post-Game Analytics
- View personal and team performance
- Interactive charts and statistics
- Leaderboard rankings
- Victory celebrations with confetti

## Key Features Implementation

### QR Code Scanning
Uses `html5-qrcode` library for reliable mobile scanning with:
- Auto-focus and zoom controls
- Torch/flashlight support
- Visual scanning frame

### Animations
Powered by Framer Motion:
- Smooth phase transitions
- Token increase animations
- Puzzle piece movements
- Celebration effects

### Haptic Feedback
Strategic vibration patterns for:
- Button presses
- Correct/incorrect answers
- Phase transitions
- Victory celebrations

### WebSocket Communication
- Automatic reconnection with exponential backoff
- State synchronization on reconnect
- Real-time message handling
- Connection status overlay

## Adding Game Assets

### Role Images
Place role images in `public/images/roles/`:
- `art_enthusiast.png`
- `detective.png`
- `tourist.png`
- `janitor.png`

### Token Images
Place token images in `public/images/tokens/`:
- `anchor.png`
- `chronos.png`
- `guide.png`
- `clarity.png`

### Puzzle Images
Puzzle images should follow the URL pattern:
```
/images/puzzles/{imageId}/{segmentId}.png
```

## Customization

### Colors
Edit color constants in `src/constants.js`:
```javascript
export const Colors = {
  primary: '#2DD4BF',    // Turquoise
  secondary: '#14B8A6',  // Darker turquoise
  // ... etc
};
```

### Animation Durations
Adjust timing in `src/constants.js`:
```javascript
export const AnimationDuration = {
  SHORT: 0.3,
  MEDIUM: 0.6,
  LONG: 1.0,
  // ... etc
};
```

## Performance Optimization

- Lazy loading of heavy components
- Optimized image loading
- Minimal re-renders with proper React patterns
- Hardware acceleration for animations
- Touch event optimization

## Browser Support

- iOS Safari 13+
- Chrome for Android 80+
- Samsung Internet 12+
- Firefox for Android 68+

## Troubleshooting

### QR Scanner Issues
- Ensure camera permissions are granted
- Check for adequate lighting
- Clean camera lens
- Try enabling torch/flashlight

### WebSocket Connection
- Verify server is running
- Check CORS settings
- Ensure correct WebSocket URL
- Look for firewall issues

### Performance Issues
- Clear browser cache
- Close other apps
- Ensure stable internet connection
- Check for memory constraints

## Contributing

1. Follow the existing code style
2. Test on real mobile devices
3. Ensure animations are smooth
4. Maintain accessibility standards
5. Document any new features

## License

See main project LICENSE file.