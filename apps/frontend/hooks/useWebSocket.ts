"use client";

import { useEffect, useRef, useCallback, useState } from "react";

/**
 * WebSocket message format per SRS Section 20:
 * { type: string, runId?: string, payload: Record<string, unknown> }
 */
export interface WSMessage {
  type: string;
  runId?: string;
  payload: Record<string, unknown>;
}

export type WSStatus = "connecting" | "open" | "closed" | "error";

interface UseWebSocketOptions {
  /** WebSocket namespace path, e.g. "/ws/simulation_updates" */
  url: string;
  /** Auto-reconnect on close (default: true) */
  reconnect?: boolean;
  /** Reconnect interval in ms (default: 3000) */
  reconnectInterval?: number;
  /** Max reconnect attempts (default: 10) */
  maxRetries?: number;
  /** Callback when a message is received */
  onMessage?: (message: WSMessage) => void;
  /** Callback when connection opens */
  onOpen?: () => void;
  /** Callback when connection closes */
  onClose?: () => void;
  /** Whether the hook should connect (default: true) */
  enabled?: boolean;
}

/**
 * React hook for WebSocket connections with auto-reconnect.
 * Connects to the gateway WebSocket namespaces:
 * - /ws/simulation_updates
 * - /ws/metrics_updates
 * - /ws/replay_updates
 * - /ws/notifications
 */
export function useWebSocket({
  url,
  reconnect = true,
  reconnectInterval = 3000,
  maxRetries = 10,
  onMessage,
  onOpen,
  onClose,
  enabled = true,
}: UseWebSocketOptions) {
  const [status, setStatus] = useState<WSStatus>("closed");
  const [lastMessage, setLastMessage] = useState<WSMessage | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const retriesRef = useRef(0);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const getFullUrl = useCallback(() => {
    if (typeof window === "undefined") return "";
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const host = process.env.NEXT_PUBLIC_WS_HOST || window.location.host;
    return `${protocol}//${host}${url}`;
  }, [url]);

  const connect = useCallback(() => {
    if (!enabled) return;
    const fullUrl = getFullUrl();
    if (!fullUrl) return;

    setStatus("connecting");

    const ws = new WebSocket(fullUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      setStatus("open");
      retriesRef.current = 0;
      onOpen?.();
    };

    ws.onmessage = (event) => {
      try {
        const data: WSMessage = JSON.parse(event.data);
        setLastMessage(data);
        onMessage?.(data);
      } catch {
        // Non-JSON message, ignore.
      }
    };

    ws.onerror = () => {
      setStatus("error");
    };

    ws.onclose = () => {
      setStatus("closed");
      wsRef.current = null;
      onClose?.();

      if (reconnect && retriesRef.current < maxRetries) {
        retriesRef.current++;
        // Exponential backoff: 1s, 2s, 4s, 8s... capped at 30s
        const backoff = Math.min(reconnectInterval * Math.pow(2, retriesRef.current - 1), 30000);
        reconnectTimerRef.current = setTimeout(connect, backoff);
      }
    };
  }, [enabled, getFullUrl, reconnect, reconnectInterval, maxRetries, onMessage, onOpen, onClose]);

  const disconnect = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setStatus("closed");
  }, []);

  const send = useCallback((message: WSMessage) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message));
    }
  }, []);

  useEffect(() => {
    connect();
    return () => {
      disconnect();
    };
  }, [connect, disconnect]);

  return { status, lastMessage, send, disconnect, reconnect: connect };
}
