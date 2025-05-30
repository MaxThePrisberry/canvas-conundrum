import { useState, useEffect, useRef, useCallback } from 'react';

export const useHostWebSocket = () => {
  const [isConnected, setIsConnected] = useState(false);
  const [isReconnecting, setIsReconnecting] = useState(false);
  const [lastMessage, setLastMessage] = useState(null);
  const ws = useRef(null);
  const reconnectTimeout = useRef(null);
  const reconnectAttempts = useRef(0);
  const hostCode = useRef(null);
  const messageQueue = useRef([]);
  const isIntentionalClose = useRef(false); // Track intentional disconnections

  const connect = useCallback((code) => {
    if (code) {
      hostCode.current = code;
    }

    if (!hostCode.current) {
      console.error('No host code provided for connection');
      return;
    }

    try {
      // Only close existing connection if we're creating a new one with a different code
      if (ws.current && (code || ws.current.readyState === WebSocket.CONNECTING)) {
        console.log('Closing existing WebSocket connection');
        isIntentionalClose.current = true; // Mark as intentional
        ws.current.close(1000, 'Reconnecting with new parameters');
        ws.current = null;
        // Small delay to ensure clean closure
        setTimeout(() => {
          isIntentionalClose.current = false;
          connectInternal();
        }, 100);
        return;
      }

      connectInternal();
    } catch (error) {
      console.error('Failed to connect host WebSocket:', error);
      setIsConnected(false);
      setIsReconnecting(false);
    }
  }, []);

  const connectInternal = useCallback(() => {
    // Construct host WebSocket URL
    const baseUrl = process.env.REACT_APP_WS_URL || 'ws://localhost:8080';
    const hostUrl = `${baseUrl}/ws/host/${hostCode.current}`;

    console.log('Attempting host WebSocket connection to:', hostUrl);
    ws.current = new WebSocket(hostUrl);

    ws.current.onopen = () => {
      console.log('Host WebSocket connected successfully');
      setIsConnected(true);
      setIsReconnecting(false);
      reconnectAttempts.current = 0;
      isIntentionalClose.current = false;

      // Send any queued messages
      while (messageQueue.current.length > 0) {
        const message = messageQueue.current.shift();
        if (ws.current.readyState === WebSocket.OPEN) {
          ws.current.send(JSON.stringify(message));
        }
      }
    };

    ws.current.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        console.log('Host received message:', message.type, message);
        setLastMessage(message);
      } catch (error) {
        console.error('Error parsing host WebSocket message:', error);
      }
    };

    ws.current.onclose = (event) => {
      console.log('Host WebSocket disconnected:', event.code, event.reason);
      setIsConnected(false);
      ws.current = null;
      
      // Only attempt to reconnect if:
      // 1. Not an intentional close
      // 2. Not a normal closure (1000)
      // 3. Haven't exceeded max attempts
      // 4. We have a host code to reconnect with
      if (!isIntentionalClose.current && 
          event.code !== 1000 && 
          reconnectAttempts.current < 5 && 
          hostCode.current) {
        setIsReconnecting(true);
        reconnectAttempts.current++;
        const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.current), 10000);
        
        console.log(`Host reconnecting in ${delay}ms (attempt ${reconnectAttempts.current})`);
        reconnectTimeout.current = setTimeout(() => {
          connectInternal();
        }, delay);
      } else {
        setIsReconnecting(false);
        if (isIntentionalClose.current) {
          console.log('Host WebSocket closed intentionally');
        } else if (event.code === 1000) {
          console.log('Host WebSocket closed normally');
        } else if (reconnectAttempts.current >= 5) {
          console.log('Host WebSocket max reconnection attempts reached');
        }
      }
    };

    ws.current.onerror = (error) => {
      console.error('Host WebSocket error:', error);
    };
  }, []);

  const sendMessage = useCallback((message) => {
    console.log('Host sending message:', message.type);
    
    if (ws.current && ws.current.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify(message));
    } else {
      console.warn('Host WebSocket not connected, queueing message');
      messageQueue.current.push(message);
      
      // Try to reconnect if not already trying and we have a host code
      if (!isReconnecting && 
          reconnectAttempts.current < 5 && 
          hostCode.current &&
          ws.current?.readyState !== WebSocket.CONNECTING) {
        console.log('Attempting to reconnect for queued message');
        connectInternal();
      }
    }
  }, [isReconnecting, connectInternal]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      console.log('Cleaning up host WebSocket connection');
      if (reconnectTimeout.current) {
        clearTimeout(reconnectTimeout.current);
      }
      if (ws.current) {
        isIntentionalClose.current = true;
        ws.current.close(1000, 'Host component unmounting');
      }
    };
  }, []);

  // Manual reconnect function
  const reconnect = useCallback(() => {
    console.log('Manual host reconnect requested');
    reconnectAttempts.current = 0;
    if (hostCode.current) {
      connectInternal();
    } else {
      console.error('Cannot reconnect: no host code available');
    }
  }, [connectInternal]);

  // Disconnect function for clean disconnection
  const disconnect = useCallback(() => {
    console.log('Host disconnecting intentionally');
    isIntentionalClose.current = true;
    setIsReconnecting(false);
    reconnectAttempts.current = 0;
    hostCode.current = null;
    
    if (reconnectTimeout.current) {
      clearTimeout(reconnectTimeout.current);
    }
    
    if (ws.current) {
      ws.current.close(1000, 'Intentional disconnect');
    }
  }, []);

  return {
    isConnected,
    isReconnecting,
    sendMessage,
    lastMessage,
    connect,
    reconnect,
    disconnect
  };
};
