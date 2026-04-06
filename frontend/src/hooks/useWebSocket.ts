import { useEffect, useRef, useCallback } from 'react';
import type { WSMessage } from '../types';

const WS_URL = 'ws://localhost:8080/ws';

type Props = {
  onMessage: (msg: WSMessage) => void;
  onConnect: () => void;
  onDisconnect: () => void;
};

export const useWebSocket = ({ onMessage, onConnect, onDisconnect }: Props) => {
  const wsRef = useRef<WebSocket | null>(null);
  const onMessageRef = useRef(onMessage);

  // onMessageが変わっても最新を参照する
  useEffect(() => {
    onMessageRef.current = onMessage;
  }, [onMessage]);

  const connect = useCallback(() => {
    const ws = new WebSocket(WS_URL);

    ws.onopen = () => {
      console.log('WebSocket connected');
      onConnect();
    };

    ws.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data);
        onMessageRef.current(msg);
      } catch (e) {
        console.error('Failed to parse message', e);
      }
    };

    ws.onclose = () => {
      console.log('WebSocket disconnected, reconnecting...');
      onDisconnect();
      // 3秒後に再接続
      setTimeout(connect, 3000);
    };

    ws.onerror = (e) => {
      console.error('WebSocket error', e);
    };

    wsRef.current = ws;
  }, []);

  useEffect(() => {
    connect();
    return () => {
      wsRef.current?.close();
    };
  }, [connect]);
};
