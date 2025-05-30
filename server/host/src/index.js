import React from 'react';
import ReactDOM from 'react-dom/client';
import './index.css';
import HostApp from './HostApp';

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <React.StrictMode>
    <HostApp />
  </React.StrictMode>
);

// Register service worker for PWA capabilities (optional)
if ('serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/service-worker.js').catch(() => {
      // Service worker registration failed - this is optional
    });
  });
}
