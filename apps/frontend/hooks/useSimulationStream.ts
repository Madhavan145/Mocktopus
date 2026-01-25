"use client";

import { useState, useCallback } from "react";
import { useWebSocket, WSMessage } from "./useWebSocket";

export interface SimulationProgress {
  status: string;
  progress?: number;
  metrics?: Record<string, unknown>;
  error?: string;
}

interface UseSimulationStreamOptions {
  /** Run ID to subscribe to (optional, only for subscribing to existing runs) */
  runId?: string;
  /** Whether to connect */
  enabled?: boolean;
  /** Callback for each progress update */
  onProgress?: (progress: SimulationProgress) => void;
}

/**
 * Hook for streaming simulation progress via WebSocket.
 * Connects to /ws/simulation_updates on the gateway.
 */
export function useSimulationStream({
  runId,
  enabled = true,
  onProgress,
}: UseSimulationStreamOptions) {
  const [progress, setProgress] = useState<SimulationProgress | null>(null);

  const handleMessage = useCallback(
    (msg: WSMessage) => {
      if (msg.type === "simulation_update") {
        const p: SimulationProgress = {
          status: (msg.payload.status as string) || "unknown",
          progress: msg.payload.progress as number | undefined,
          metrics: msg.payload.metrics as Record<string, unknown> | undefined,
          error: msg.payload.error as string | undefined,
        };
        setProgress(p);
        onProgress?.(p);
      }
    },
    [onProgress]
  );

  const queryStr = runId ? `?runId=${encodeURIComponent(runId)}` : "";

  const { status, send, disconnect } = useWebSocket({
    url: `/ws/simulation_updates${queryStr}`,
    enabled,
    onMessage: handleMessage,
  });

  const subscribe = useCallback(
    (targetRunId: string) => {
      send({
        type: "command",
        payload: { action: "subscribe", runId: targetRunId },
      });
    },
    [send]
  );

  return { status, progress, subscribe, send, disconnect };
}
