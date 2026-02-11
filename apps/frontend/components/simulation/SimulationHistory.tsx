'use client';

import { useEffect, useState } from 'react';
import { readSimulationHistory } from './storage';
import type { SimulationRunRecord } from './types';

type SimulationHistoryProps = {
  projectId: string;
  onSelectRun?: (run: SimulationRunRecord) => void;
};

export function SimulationHistory({ projectId, onSelectRun }: SimulationHistoryProps) {
  const [history, setHistory] = useState<SimulationRunRecord[]>([]);
  const [selectedIndex, setSelectedIndex] = useState<number | null>(null);

  useEffect(() => {
    const runs = readSimulationHistory(projectId);
    setHistory(runs);
  }, [projectId]);

  if (!history.length) {
    return (
      <article className="rounded-lg border bg-black/30 p-4">
        <h3 className="text-sm font-semibold">Run History</h3>
        <p className="mt-2 text-xs text-mute">No previous simulation runs found.</p>
      </article>
    );
  }

  return (
    <article className="rounded-lg border bg-black/30 p-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold">Run History</h3>
        <span className="text-[10px] text-mute">{history.length} runs</span>
      </div>

      <div className="mt-3 space-y-1.5 max-h-72 overflow-y-auto">
        {history.map((run, index) => {
          const m = run.run.metrics;
          const isSelected = selectedIndex === index;
          const date = new Date(run.capturedAt);
          const timeStr = date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
          const dateStr = date.toLocaleDateString([], { month: 'short', day: 'numeric' });

          return (
            <button
              key={run.capturedAt}
              onClick={() => {
                setSelectedIndex(index);
                onSelectRun?.(run);
              }}
              className={`w-full rounded border p-2 text-left transition ${
                isSelected
                  ? 'border-accent bg-accent/10'
                  : 'border-white/5 bg-black/20 hover:border-white/10 hover:bg-black/30'
              }`}
            >
              <div className="flex items-center justify-between">
                <span className="text-[10px] text-mute">{dateStr} {timeStr}</span>
                <span className={`text-[10px] px-1.5 py-0.5 rounded ${
                  m.errorRate > 0.1 ? 'bg-red-500/20 text-red-400' :
                  m.errorRate > 0.05 ? 'bg-yellow-500/20 text-yellow-400' :
                  'bg-green-500/20 text-green-400'
                }`}>
                  {(m.errorRate * 100).toFixed(1)}% err
                </span>
              </div>
              <div className="mt-1 grid grid-cols-4 gap-1 text-[10px]">
                <div>
                  <span className="text-white/50">RPS</span>
                  <span className="ml-1 font-mono">{m.rps.toFixed(0)}</span>
                </div>
                <div>
                  <span className="text-white/50">Lat</span>
                  <span className="ml-1 font-mono">{m.avgLatencyMs.toFixed(0)}ms</span>
                </div>
                <div>
                  <span className="text-white/50">P95</span>
                  <span className="ml-1 font-mono">{m.p95LatencyMs.toFixed(0)}ms</span>
                </div>
                <div>
                  <span className="text-white/50">Reqs</span>
                  <span className="ml-1 font-mono">{m.requestsTotal}</span>
                </div>
              </div>
              {/* Mini sparkline */}
              {run.derived?.series && run.derived.series.length > 0 && (
                <svg className="mt-1.5 h-4 w-full opacity-60" viewBox="0 0 100 20" preserveAspectRatio="none">
                  <polyline
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="1.5"
                    className="text-accent"
                    points={run.derived.series
                      .map((p, i) => {
                        const x = (i / Math.max(run.derived!.series.length - 1, 1)) * 100;
                        const maxRps = Math.max(...run.derived!.series.map((s) => s.rps), 1);
                        const y = 20 - (p.rps / maxRps) * 18;
                        return `${x},${y}`;
                      })
                      .join(' ')}
                  />
                </svg>
              )}
            </button>
          );
        })}
      </div>
    </article>
  );
}
