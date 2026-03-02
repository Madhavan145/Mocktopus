'use client';

import type { SimulationEvent, SimulationMetrics } from './types';

type LatencyBreakdownProps = {
  events: SimulationEvent[];
  metrics: SimulationMetrics;
};

type LatencySegment = {
  label: string;
  avgMs: number;
  color: string;
};

function computeLatencyBreakdown(events: SimulationEvent[]): LatencySegment[] {
  let networkTotal = 0, networkCount = 0;
  let processingTotal = 0, processingCount = 0;
  let queueTotal = 0, queueCount = 0;

  const requestStates: Record<number, { lastTime: number; lastType: string }> = {};

  for (const e of events) {
    const state = requestStates[e.requestId];

    switch (e.type) {
      case 'REQUEST_RECEIVED':
        requestStates[e.requestId] = { lastTime: e.timeMs, lastType: e.type };
        break;
      case 'REQUEST_ROUTED':
        if (state) {
          networkTotal += e.timeMs - state.lastTime;
          networkCount++;
        }
        requestStates[e.requestId] = { lastTime: e.timeMs, lastType: e.type };
        break;
      case 'QUEUED':
        requestStates[e.requestId] = { lastTime: e.timeMs, lastType: e.type };
        break;
      case 'PROCESS_START':
        if (state && state.lastType === 'QUEUED') {
          queueTotal += e.timeMs - state.lastTime;
          queueCount++;
        }
        requestStates[e.requestId] = { lastTime: e.timeMs, lastType: e.type };
        break;
      case 'PROCESS_COMPLETE':
        if (state && state.lastType === 'PROCESS_START') {
          processingTotal += e.timeMs - state.lastTime;
          processingCount++;
        }
        requestStates[e.requestId] = { lastTime: e.timeMs, lastType: e.type };
        break;
    }
  }

  return [
    { label: 'Network', avgMs: networkCount > 0 ? networkTotal / networkCount : 0, color: 'bg-blue-500' },
    { label: 'Queue Wait', avgMs: queueCount > 0 ? queueTotal / queueCount : 0, color: 'bg-yellow-500' },
    { label: 'Processing', avgMs: processingCount > 0 ? processingTotal / processingCount : 0, color: 'bg-purple-500' },
  ];
}

export function LatencyBreakdown({ events, metrics }: LatencyBreakdownProps) {
  const segments = computeLatencyBreakdown(events);
  const totalBreakdown = segments.reduce((sum, s) => sum + s.avgMs, 0) || 1;

  return (
    <article className="rounded-lg border bg-black/30 p-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold">Latency Breakdown</h3>
        <span className="text-xs text-mute">avg {metrics.avgLatencyMs.toFixed(1)}ms</span>
      </div>

      {/* Stacked bar */}
      <div className="mt-3 flex h-6 rounded overflow-hidden">
        {segments.map((seg) => {
          const pct = (seg.avgMs / totalBreakdown) * 100;
          if (pct < 1) return null;
          return (
            <div
              key={seg.label}
              className={`${seg.color} flex items-center justify-center text-[9px] font-medium text-white`}
              style={{ width: `${pct}%` }}
              title={`${seg.label}: ${seg.avgMs.toFixed(1)}ms`}
            >
              {pct > 15 ? seg.label : ''}
            </div>
          );
        })}
      </div>

      {/* Legend */}
      <div className="mt-3 grid grid-cols-3 gap-3">
        {segments.map((seg) => (
          <div key={seg.label} className="text-center">
            <div className="flex items-center justify-center gap-1.5">
              <div className={`h-2 w-2 rounded-full ${seg.color}`} />
              <span className="text-[10px] text-mute">{seg.label}</span>
            </div>
            <span className="font-mono text-xs">{seg.avgMs.toFixed(1)}ms</span>
            <span className="block text-[10px] text-mute">
              {((seg.avgMs / totalBreakdown) * 100).toFixed(0)}%
            </span>
          </div>
        ))}
      </div>

      {/* Percentile breakdown */}
      <div className="mt-3 grid grid-cols-5 gap-2 rounded bg-black/40 p-2 text-center">
        <div>
          <span className="block text-[10px] text-white/60">P50</span>
          <span className="font-mono text-xs">{(metrics.p50LatencyMs ?? metrics.avgLatencyMs).toFixed(1)}ms</span>
        </div>
        <div>
          <span className="block text-[10px] text-white/60">P95</span>
          <span className="font-mono text-xs">{(metrics.p95LatencyMs ?? 0).toFixed(1)}ms</span>
        </div>
        <div>
          <span className="block text-[10px] text-white/60">P99</span>
          <span className="font-mono text-xs">{(metrics.p99LatencyMs ?? 0).toFixed(1)}ms</span>
        </div>
        <div>
          <span className="block text-[10px] text-white/60">P99.9</span>
          <span className="font-mono text-xs">{(metrics.p999LatencyMs ?? 0).toFixed(1)}ms</span>
        </div>
        <div>
          <span className="block text-[10px] text-white/60">Std Dev</span>
          <span className="font-mono text-xs">{(metrics.latencyStdDev ?? 0).toFixed(1)}ms</span>
        </div>
      </div>

      {/* Tail latency amplification */}
      {(metrics.p50LatencyMs ?? 0) > 0 && (
        <div className="mt-2 text-[10px] text-white/50 flex justify-between px-1">
          <span>Tail amplification: P99/P50 = {((metrics.p99LatencyMs ?? 0) / (metrics.p50LatencyMs || 1)).toFixed(1)}x</span>
          <span>P99.9/P99 = {((metrics.p99LatencyMs ?? 0) > 0 ? ((metrics.p999LatencyMs ?? 0) / (metrics.p99LatencyMs || 1)).toFixed(1) : '—')}x</span>
        </div>
      )}
    </article>
  );
}
