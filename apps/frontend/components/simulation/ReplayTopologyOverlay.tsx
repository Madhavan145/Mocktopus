'use client';

import { useMemo } from 'react';
import type { ReplaySnapshot, TopologyGraphPayload } from './types';

type Props = {
  topology: TopologyGraphPayload;
  snapshot: ReplaySnapshot | null;
};

/**
 * ReplayTopologyOverlay renders a mini topology graph with nodes colored
 * based on replay snapshot state (health-based gradient).
 * Green = healthy, Yellow = moderate load, Red = high failure rate.
 */
export function ReplayTopologyOverlay({ topology, snapshot }: Props) {
  const nodeStates = useMemo(() => {
    if (!snapshot) return {};
    return snapshot.nodeState;
  }, [snapshot]);

  // Position nodes in a simple grid layout for the mini overlay.
  const positioned = useMemo(() => {
    const cols = Math.ceil(Math.sqrt(topology.nodes.length));
    return topology.nodes.map((node, i) => ({
      ...node,
      x: (i % cols) * 120 + 60,
      y: Math.floor(i / cols) * 80 + 40,
    }));
  }, [topology.nodes]);

  const svgWidth = useMemo(() => {
    const cols = Math.ceil(Math.sqrt(topology.nodes.length));
    return cols * 120 + 60;
  }, [topology.nodes.length]);

  const svgHeight = useMemo(() => {
    const cols = Math.ceil(Math.sqrt(topology.nodes.length));
    const rows = Math.ceil(topology.nodes.length / cols);
    return rows * 80 + 40;
  }, [topology.nodes.length]);

  const nodeMap = useMemo(() => {
    const map: Record<string, { x: number; y: number }> = {};
    for (const n of positioned) {
      map[n.id] = { x: n.x, y: n.y };
    }
    return map;
  }, [positioned]);

  return (
    <div className="rounded-lg border bg-black/40 p-3">
      <h3 className="mb-2 text-sm font-semibold text-white">Topology Overlay</h3>
      <svg
        width={Math.min(svgWidth, 600)}
        height={Math.min(svgHeight, 300)}
        viewBox={`0 0 ${svgWidth} ${svgHeight}`}
        className="w-full"
      >
        {/* Edges */}
        {topology.edges.map((edge) => {
          const src = nodeMap[edge.source];
          const tgt = nodeMap[edge.target];
          if (!src || !tgt) return null;
          return (
            <line
              key={edge.id}
              x1={src.x}
              y1={src.y}
              x2={tgt.x}
              y2={tgt.y}
              stroke="#555"
              strokeWidth={1.5}
              opacity={0.6}
            />
          );
        })}

        {/* Nodes */}
        {positioned.map((node) => {
          const state = nodeStates[node.id];
          const color = getNodeColor(state);
          const radius = 18;

          return (
            <g key={node.id}>
              <circle
                cx={node.x}
                cy={node.y}
                r={radius}
                fill={color}
                stroke={state?.failures ? '#ef4444' : '#444'}
                strokeWidth={state?.failures ? 2.5 : 1}
                opacity={0.9}
              />
              <text
                x={node.x}
                y={node.y + 4}
                textAnchor="middle"
                fontSize={9}
                fill="white"
                fontWeight="bold"
              >
                {node.id.length > 8 ? node.id.slice(0, 7) + '…' : node.id}
              </text>
              {state && (
                <text
                  x={node.x}
                  y={node.y + radius + 12}
                  textAnchor="middle"
                  fontSize={8}
                  fill="#aaa"
                >
                  {state.touches}t / {state.failures}f
                </text>
              )}
            </g>
          );
        })}
      </svg>

      {snapshot && (
        <p className="mt-1 text-xs text-mute">
          t={snapshot.timestamp}ms | event #{snapshot.eventIndex}
        </p>
      )}
    </div>
  );
}

function getNodeColor(state: { touches: number; completed: number; queued: number; failures: number } | undefined): string {
  if (!state || state.touches === 0) return '#374151'; // gray — no traffic
  const failureRate = state.failures / state.touches;
  if (failureRate > 0.5) return '#dc2626'; // red — critical
  if (failureRate > 0.2) return '#f59e0b'; // amber — degraded
  if (failureRate > 0.05) return '#eab308'; // yellow — some errors
  return '#10b981'; // green — healthy
}
