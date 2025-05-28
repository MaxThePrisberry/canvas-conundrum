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

  const connect = useCallback(() => {
    try {
      // Construct URL with playerId if we're reconnecting
      const url = playerId.current 
        ? `${WS_URL}?playerId=${playerId.current}`
        : WS_URL;

      ws.current = new WebSocket(url);

      ws.current.onopen = () => {
        console.log('WebSocket connected');
        setIsConnected(true);
        setIsReconnecting(false);
        reconnectAttempts.current = 0;
      };

      ws.current.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          
          // Store playerId from available_roles message
          if (message.type === 'available_roles' && message.payload?.playerId) {
            playerId.current = message.payload.playerId;
          }
          
          setLastMessage(message);
        } catch (error) {
          console.error('Error parsing message:', error);
        }
      };

      ws.current.onclose = () => {
        console.log('WebSocket disconnected');
        setIsConnected(false);
        
        // Attempt to reconnect
        if (reconnectAttempts.current < 5) {
          setIsReconnecting(true);
          reconnectAttempts.current++;
          const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.current), 10000);
          
          reconnectTimeout.current = setTimeout(() => {
            console.log(`Reconnecting... (attempt ${reconnectAttempts.current})`);
            connect();
          }, delay);
        }
      };

      ws.current.onerror = (error) => {
        console.error('WebSocket error:', error);
      };
    } catch (error) {
      console.error('Failed to connect:', error);
    }
  }, []);

  const sendMessage = useCallback((message) => {
    if (ws.current && ws.current.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify(message));
    } else {
      console.error('WebSocket is not connected');
    }
  }, []);

  // Initial connection
  useEffect(() => {
    connect();

    return () => {
      if (reconnectTimeout.current) {
        clearTimeout(reconnectTimeout.current);
      }
      if (ws.current) {
        ws.current.close();
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

  return {
    isConnected,
    isReconnecting,
    sendMessage,
    lastMessage
  };
};