'use client';

import { useState } from 'react';
import type { SimulationEvent, NodeRuntimeStats, SimulationMetrics } from './types';

type NodePerformanceProps = {
  nodeStats: Record<string, NodeRuntimeStats>;
  events: SimulationEvent[];
  metrics: SimulationMetrics;
};

type NodeDetail = {
  id: string;
  touches: number;
  failures: number;
  completed: number;
  queued: number;
  errorRate: number;
  avgLatencyMs: number;
  saturation: number;
};

function deriveNodeDetails(
  nodeStats: Record<string, NodeRuntimeStats>,
  events: SimulationEvent[]
): NodeDetail[] {
  const latencies: Record<string, number[]> = {};
  let lastNodeTime: Record<string, number> = {};

  for (const e of events) {
    if (!e.nodeId) continue;
    if (e.type === 'PROCESS_START') {
      lastNodeTime[e.nodeId] = e.timeMs;
    } else if (e.type === 'PROCESS_COMPLETE' && lastNodeTime[e.nodeId] !== undefined) {
      const dur = e.timeMs - lastNodeTime[e.nodeId];
      if (!latencies[e.nodeId]) latencies[e.nodeId] = [];
      latencies[e.nodeId].push(dur);
      delete lastNodeTime[e.nodeId];
    }
  }

  return Object.entries(nodeStats).map(([id, stats]) => {
    const total = stats.touchCount || 1;
    const nodeLat = latencies[id] || [];
    const avgLatency = nodeLat.length > 0 ? nodeLat.reduce((a, b) => a + b, 0) / nodeLat.length : 0;
    const saturation = Math.min(100, (stats.queuedCount / total) * 100 + (stats.failureCount / total) * 50);

    return {
      id,
      touches: stats.touchCount,
      failures: stats.failureCount,
      completed: stats.completedCount,
      queued: stats.queuedCount,
      errorRate: total > 0 ? (stats.failureCount / total) * 100 : 0,
      avgLatencyMs: Math.round(avgLatency * 100) / 100,
      saturation: Math.round(saturation * 100) / 100,
    };
  });
}

export function NodePerformancePanel({ nodeStats, events, metrics }: NodePerformanceProps) {
  const [sortBy, setSortBy] = useState<'touches' | 'errorRate' | 'saturation' | 'avgLatencyMs'>('saturation');
  const details = deriveNodeDetails(nodeStats, events);
  const sorted = [...details].sort((a, b) => b[sortBy] - a[sortBy]);

  if (!sorted.length) {
    return (
      <article className="rounded-lg border bg-black/30 p-4">
        <h3 className="text-sm font-semibold">Per-Node Performance</h3>
        <p className="mt-2 text-xs text-mute">No node-level data available for this run.</p>
      </article>
    );
  }

  const maxSaturation = Math.max(...sorted.map((n) => n.saturation), 1);

  return (
    <article className="rounded-lg border bg-black/30 p-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold">Per-Node Performance</h3>
        <div className="flex gap-1">
          {(['saturation', 'errorRate', 'avgLatencyMs', 'touches'] as const).map((key) => (
            <button
              key={key}
              onClick={() => setSortBy(key)}
              className={`rounded px-2 py-0.5 text-[10px] transition ${
                sortBy === key ? 'bg-accent text-white' : 'bg-black/30 text-mute hover:bg-black/50'
              }`}
            >
              {key === 'avgLatencyMs' ? 'Latency' : key === 'errorRate' ? 'Errors' : key}
            </button>
          ))}
        </div>
      </div>

      <div className="mt-3 space-y-2 max-h-80 overflow-y-auto">
        {sorted.map((node) => (
          <div key={node.id} className="rounded border border-white/5 bg-black/20 p-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium">{node.id}</span>
              <span className={`text-[10px] px-1.5 py-0.5 rounded ${
                node.saturation > 70 ? 'bg-red-500/20 text-red-400' :
                node.saturation > 40 ? 'bg-yellow-500/20 text-yellow-400' :
                'bg-green-500/20 text-green-400'
              }`}>
                {node.saturation > 70 ? 'HIGH' : node.saturation > 40 ? 'MEDIUM' : 'HEALTHY'}
              </span>
            </div>
            <div className="mt-1.5 grid grid-cols-4 gap-2 text-[10px] text-mute">
              <div>
                <span className="block text-white/60">Requests</span>
                <span className="font-mono">{node.touches}</span>
              </div>
              <div>
                <span className="block text-white/60">Errors</span>
                <span className={`font-mono ${node.errorRate > 5 ? 'text-red-400' : ''}`}>
                  {node.errorRate.toFixed(1)}%
                </span>
              </div>
              <div>
                <span className="block text-white/60">Latency</span>
                <span className="font-mono">{node.avgLatencyMs.toFixed(1)}ms</span>
              </div>
              <div>
                <span className="block text-white/60">Queued</span>
                <span className="font-mono">{node.queued}</span>
              </div>
            </div>
            {/* Saturation bar */}
            <div className="mt-1.5 h-1.5 rounded bg-black/40 overflow-hidden">
              <div
                className={`h-full rounded transition-all ${
                  node.saturation > 70 ? 'bg-red-500' :
                  node.saturation > 40 ? 'bg-yellow-500' :
                  'bg-green-500'
                }`}
                style={{ width: `${(node.saturation / maxSaturation) * 100}%` }}
              />
            </div>
          </div>
        ))}
      </div>

      {/* Summary row */}
      <div className="mt-3 grid grid-cols-3 gap-2 rounded bg-black/40 p-2 text-[10px]">
        <div className="text-center">
          <span className="block text-white/60">Total Requests</span>
          <span className="font-mono text-xs">{metrics.requestsTotal}</span>
        </div>
        <div className="text-center">
          <span className="block text-white/60">Bottleneck</span>
          <span className="font-mono text-xs text-yellow-400">
            {sorted[0]?.id ?? '—'}
          </span>
        </div>
        <div className="text-center">
          <span className="block text-white/60">Max Saturation</span>
          <span className="font-mono text-xs">{sorted[0]?.saturation.toFixed(0)}%</span>
        </div>
      </div>
    </article>
  );
}
