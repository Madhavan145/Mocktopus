"use client";

import { useCallback, useRef } from "react";
import type { Edge, Node } from "reactflow";
import type { BuilderNodeData } from "./types";

export type HistoryEntry = {
  nodes: Array<Node<BuilderNodeData>>;
  edges: Array<Edge>;
};

const MAX_HISTORY = 50;

/**
 * useUndoRedo provides undo/redo capability for the builder canvas.
 * Maintains a history stack of (nodes, edges) snapshots.
 */
export function useUndoRedo(
  setNodes: (nodes: Array<Node<BuilderNodeData>>) => void,
  setEdges: (edges: Array<Edge>) => void
) {
  const historyRef = useRef<HistoryEntry[]>([]);
  const pointerRef = useRef(-1);
  const isProgrammaticRef = useRef(false);

  /**
   * Call this after every user-initiated change to push a new history entry.
   * Should be called with the new state (after change).
   */
  const pushHistory = useCallback(
    (nodes: Array<Node<BuilderNodeData>>, edges: Array<Edge>) => {
      if (isProgrammaticRef.current) return;

      const history = historyRef.current;
      const pointer = pointerRef.current;

      // Trim any redo entries beyond current pointer.
      historyRef.current = history.slice(0, pointer + 1);

      // Push new entry.
      historyRef.current.push({ nodes: structuredClone(nodes), edges: structuredClone(edges) });

      // Cap history size.
      if (historyRef.current.length > MAX_HISTORY) {
        historyRef.current = historyRef.current.slice(historyRef.current.length - MAX_HISTORY);
      }

      pointerRef.current = historyRef.current.length - 1;
    },
    []
  );

  const undo = useCallback(() => {
    if (pointerRef.current <= 0) return;
    pointerRef.current--;
    const entry = historyRef.current[pointerRef.current];
    if (entry) {
      isProgrammaticRef.current = true;
      setNodes(structuredClone(entry.nodes));
      setEdges(structuredClone(entry.edges));
      isProgrammaticRef.current = false;
    }
  }, [setNodes, setEdges]);

  const redo = useCallback(() => {
    if (pointerRef.current >= historyRef.current.length - 1) return;
    pointerRef.current++;
    const entry = historyRef.current[pointerRef.current];
    if (entry) {
      isProgrammaticRef.current = true;
      setNodes(structuredClone(entry.nodes));
      setEdges(structuredClone(entry.edges));
      isProgrammaticRef.current = false;
    }
  }, [setNodes, setEdges]);

  const canUndo = useCallback(() => pointerRef.current > 0, []);
  const canRedo = useCallback(
    () => pointerRef.current < historyRef.current.length - 1,
    []
  );

  return { pushHistory, undo, redo, canUndo, canRedo, isProgrammatic: isProgrammaticRef };
}
