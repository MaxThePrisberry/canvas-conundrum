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

  const connect = useCallback((code) => {
    if (code) {
      hostCode.current = code;
    }

    if (!hostCode.current) {
      console.error('No host code provided for connection');
      return;
    }

    try {
      // Clear any existing connection
      if (ws.current) {
        ws.current.close();
      }

      // Construct host WebSocket URL
      const baseUrl = process.env.REACT_APP_WS_URL || 'ws://localhost:8080';
      const hostUrl = `${baseUrl}/host/${hostCode.current}`;

      console.log('Attempting host WebSocket connection to:', hostUrl);
      ws.current = new WebSocket(hostUrl);

      ws.current.onopen = () => {
        console.log('Host WebSocket connected successfully');
        setIsConnected(true);
        setIsReconnecting(false);
        reconnectAttempts.current = 0;

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
          console.log('Host received message:', message.type);
          setLastMessage(message);
        } catch (error) {
          console.error('Error parsing host WebSocket message:', error);
        }
      };

      ws.current.onclose = (event) => {
        console.log('Host WebSocket disconnected:', event.code, event.reason);
        setIsConnected(false);
        ws.current = null;
        
        // Attempt to reconnect if not a normal closure
        if (event.code !== 1000 && reconnectAttempts.current < 5) {
          setIsReconnecting(true);
          reconnectAttempts.current++;
          const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.current), 10000);
          
          console.log(`Host reconnecting in ${delay}ms (attempt ${reconnectAttempts.current})`);
          reconnectTimeout.current = setTimeout(() => {
            connect();
          }, delay);
        }
      };

      ws.current.onerror = (error) => {
        console.error('Host WebSocket error:', error);
      };
    } catch (error) {
      console.error('Failed to connect host WebSocket:', error);
      setIsConnected(false);
      setIsReconnecting(false);
    }
  }, []);

  const sendMessage = useCallback((message) => {
    console.log('Host sending message:', message.type);
    
    if (ws.current && ws.current.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify(message));
    } else {
      console.warn('Host WebSocket not connected, queueing message');
      messageQueue.current.push(message);
      
      // Try to reconnect if not already trying
      if (!isReconnecting && reconnectAttempts.current < 5) {
        connect();
      }
    }
  }, [isReconnecting, connect]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      console.log('Cleaning up host WebSocket connection');
      if (reconnectTimeout.current) {
        clearTimeout(reconnectTimeout.current);
      }
      if (ws.current) {
        ws.current.close(1000, 'Host component unmounting');
      }
    };
  }, []);

  // Manual reconnect function
  const reconnect = useCallback(() => {
    console.log('Manual host reconnect requested');
    reconnectAttempts.current = 0;
    connect();
  }, [connect]);

  return {
    isConnected,
    isReconnecting,
    sendMessage,
    lastMessage,
    connect,
    reconnect
  };
};
