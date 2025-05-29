import { useState, useEffect, useRef, useCallback } from 'react';
import { WS_URL } from '../constants';

export const useWebSocket = () => {
  const [isConnected, setIsConnected] = useState(false);
  const [isReconnecting, setIsReconnecting] = useState(false);
  const [lastMessage, setLastMessage] = useState(null);
  const ws = useRef(null);
  const reconnectTimeout = useRef(null);
  const reconnectAttempts = useRef(0);
  const playerId = useRef(null);
  const messageQueue = useRef([]);

  const connect = useCallback(() => {
    try {
      // Clear any existing connection
      if (ws.current) {
        ws.current.close();
      }

      // Construct URL with playerId if we're reconnecting
      const url = playerId.current 
        ? `${WS_URL}?playerId=${playerId.current}`
        : WS_URL;

      console.log('Attempting WebSocket connection to:', url);
      ws.current = new WebSocket(url);

      ws.current.onopen = () => {
        console.log('WebSocket connected successfully');
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
          console.log('Received message:', message.type);
          
          // Store playerId from available_roles message
          if (message.type === 'available_roles' && message.payload?.playerId) {
            playerId.current = message.payload.playerId;
            console.log('Stored playerId:', playerId.current);
          }
          
          setLastMessage(message);
        } catch (error) {
          console.error('Error parsing WebSocket message:', error);
        }
      };

      ws.current.onclose = (event) => {
        console.log('WebSocket disconnected:', event.code, event.reason);
        setIsConnected(false);
        ws.current = null;
        
        // Attempt to reconnect if not a normal closure
        if (event.code !== 1000 && reconnectAttempts.current < 5) {
          setIsReconnecting(true);
          reconnectAttempts.current++;
          const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.current), 10000);
          
          console.log(`Reconnecting in ${delay}ms (attempt ${reconnectAttempts.current})`);
          reconnectTimeout.current = setTimeout(() => {
            connect();
          }, delay);
        }
      };

      ws.current.onerror = (error) => {
        console.error('WebSocket error:', error);
      };
    } catch (error) {
      console.error('Failed to connect WebSocket:', error);
      setIsConnected(false);
      setIsReconnecting(false);
    }
  }, []);

  const sendMessage = useCallback((message) => {
    console.log('Sending message:', message.type);
    
    if (ws.current && ws.current.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify(message));
    } else {
      console.warn('WebSocket not connected, queueing message');
      messageQueue.current.push(message);
      
      // Try to reconnect if not already trying
      if (!isReconnecting && reconnectAttempts.current < 5) {
        connect();
      }
    }
  }, [isReconnecting, connect]);

  // Initial connection
  useEffect(() => {
    connect();

    return () => {
      console.log('Cleaning up WebSocket connection');
      if (reconnectTimeout.current) {
        clearTimeout(reconnectTimeout.current);
      }
      if (ws.current) {
        ws.current.close(1000, 'Component unmounting');
      }
    };
  }, [connect]);

  // Haptic feedback when connection state changes
  useEffect(() => {
    if (window.navigator && window.navigator.vibrate) {
      if (isConnected && !isReconnecting) {
        // Short vibration on successful connection
        window.navigator.vibrate(50);
      } else if (!isConnected && !isReconnecting) {
        // Longer vibration on disconnection
        window.navigator.vibrate([100, 50, 100]);
      }
    }
  }, [isConnected, isReconnecting]);

  // Manual reconnect function
  const reconnect = useCallback(() => {
    console.log('Manual reconnect requested');
    reconnectAttempts.current = 0;
    connect();
  }, [connect]);

  return {
    isConnected,
    isReconnecting,
    sendMessage,
    lastMessage,
    reconnect
  };
};
