'use client';

import Link from 'next/link';
import { useEffect, useMemo, useState, useCallback } from 'react';

import { EventMixChart, TrendChart } from './MetricCharts';
import { LatencyBreakdown } from './LatencyBreakdown';
import { NodePerformancePanel } from './NodePerformancePanel';
import { ReplayTopologyOverlay } from './ReplayTopologyOverlay';
import { SimulationHistory } from './SimulationHistory';
import { SystemHealthDashboard } from './SystemHealthDashboard';
import {
  readActiveSimulationRunId,
  readLastSimulationRun,
  readLiveSimulationSnapshot,
  readSimulationHistory,
  subscribeActiveSimulationRunId,
  subscribeLiveSimulationSnapshot,
} from './storage';
import type { LiveMetricsSnapshot, RunStateResponse, SimulationRunRecord } from './types';
import type { ReplayReconstructResponse, TopologyGraphPayload } from './types';

import { apiFetch } from '@/lib/api';

type Props = {
  projectId: string;
};

export default function ResultsView({ projectId }: Props) {
  const [record, setRecord] = useState<SimulationRunRecord | null>(null);
  const [history, setHistory] = useState<SimulationRunRecord[]>([]);
  const [liveSnapshot, setLiveSnapshot] = useState<LiveMetricsSnapshot | null>(null);
  const [activeRunId, setActiveRunId] = useState<string | null>(null);
  const [replay, setReplay] = useState<ReplayReconstructResponse | null>(null);
  const [snapshotIndex, setSnapshotIndex] = useState(0);
  const [topology, setTopology] = useState<TopologyGraphPayload | null>(null);

  // Load topology from localStorage for replay overlay.
  useEffect(() => {
    try {
      const raw = localStorage.getItem(`moktopus:project:${projectId}:builder`);
      if (raw) {
        const parsed = JSON.parse(raw);
        // Builder stores { nodes, edges } or { graph: { nodes, edges } }
        const graph = parsed.graph ?? parsed;
        if (graph?.nodes) {
          setTopology(graph as TopologyGraphPayload);
        }
      }
    } catch {
      // Ignore parse errors
    }
  }, [projectId]);

  useEffect(() => {
    setRecord(readLastSimulationRun(projectId));
    setHistory(readSimulationHistory(projectId));

    setLiveSnapshot(readLiveSimulationSnapshot(projectId));
    setActiveRunId(readActiveSimulationRunId(projectId));
    const unsubscribe = subscribeLiveSimulationSnapshot(projectId, (snapshot) => {
      setLiveSnapshot(snapshot);
      if (!snapshot) {
        setRecord(readLastSimulationRun(projectId));
        setHistory(readSimulationHistory(projectId));
      }
    });

    const unsubscribeRunId = subscribeActiveSimulationRunId(projectId, (runId) => {
      setActiveRunId(runId);
      if (!runId) {
        setRecord(readLastSimulationRun(projectId));
        setHistory(readSimulationHistory(projectId));
      }
    });

    return () => {
      unsubscribe();
      unsubscribeRunId();
    };
  }, [projectId]);

  useEffect(() => {
    if (!activeRunId) {
      return;
    }

    const intervalId = window.setInterval(async () => {
      const response = await apiFetch(`/api/v1/simulation/runs/${encodeURIComponent(activeRunId)}`);
      if (!response.ok) {
        return;
      }

      const state = await parseV1Data<RunStateResponse>(response);
      setLiveSnapshot(state.snapshot ?? null);
      if (state.status !== 'running') {
        setRecord(readLastSimulationRun(projectId));
        setHistory(readSimulationHistory(projectId));
      }
    }, 500);

    return () => window.clearInterval(intervalId);
  }, [activeRunId, projectId]);

  useEffect(() => {
    if (!record) {
      setReplay(null);
      setSnapshotIndex(0);
      return;
    }

    const loadReplay = async () => {
      const runId = record.runId ?? `run-${record.capturedAt}`;

      // Try server-side replay reconstruction first
      try {
        const response = await apiFetch(`/api/v1/replay/reconstruct`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            runId,
            snapshotStride: 25,
            events: record.run.events,
          }),
        });

        if (response.ok) {
          const payload = await parseV1Data<ReplayReconstructResponse>(response);
          setReplay(payload);
          setSnapshotIndex(Math.max(0, payload.snapshots.length - 1));
          return;
        }
      } catch {
        // Server unavailable, fall through to client-side reconstruction
      }

      // Client-side fallback: reconstruct snapshots from events locally
      const events = record.run.events;
      if (!events.length) return;

      const stride = 25;
      const snapshots: ReplayReconstructResponse['snapshots'] = [];
      const nodeState: Record<string, { touches: number; completed: number; queued: number; failures: number }> = {};

      for (let i = 0; i < events.length; i++) {
        const ev = events[i];
        if (ev.nodeId) {
          if (!nodeState[ev.nodeId]) {
            nodeState[ev.nodeId] = { touches: 0, completed: 0, queued: 0, failures: 0 };
          }
          nodeState[ev.nodeId].touches++;
          if (ev.type === 'PROCESS_COMPLETE' || ev.type === 'REQUEST_COMPLETE') {
            nodeState[ev.nodeId].completed++;
          } else if (ev.type === 'QUEUED') {
            nodeState[ev.nodeId].queued++;
          } else if (ev.type === 'FAILED' || ev.type === 'REQUEST_FAILED') {
            nodeState[ev.nodeId].failures++;
          }
        }

        if ((i + 1) % stride === 0 || i === events.length - 1) {
          snapshots.push({
            timestamp: ev.timeMs,
            eventIndex: i,
            nodeState: JSON.parse(JSON.stringify(nodeState)),
          });
        }
      }

      setReplay({ runId, snapshots, snapshotStride: stride, eventsProcessed: events.length });
      setSnapshotIndex(Math.max(0, snapshots.length - 1));
    };

    void loadReplay();
  }, [record]);

  // Keyboard shortcuts for replay: Left/Right arrow keys
  const handleReplayKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (!replay?.snapshots?.length) return;
      if (e.key === 'ArrowLeft') {
        setSnapshotIndex((current) => Math.max(0, current - 1));
      } else if (e.key === 'ArrowRight') {
        setSnapshotIndex((current) => Math.min(replay.snapshots.length - 1, current + 1));
      }
    },
    [replay]
  );

  useEffect(() => {
    window.addEventListener('keydown', handleReplayKeyDown);
    return () => window.removeEventListener('keydown', handleReplayKeyDown);
  }, [handleReplayKeyDown]);

  const cards = useMemo(() => {
    if (!record) {
      return [];
    }

    const run = record.run;

    return [
      { label: 'Requests Total', value: `${run.metrics.requestsTotal}` },
      { label: 'Completed', value: `${run.metrics.completed}` },
      { label: 'Failed', value: `${run.metrics.failed}` },
      { label: 'RPS', value: `${run.metrics.rps}` },
      { label: 'Goodput', value: `${run.metrics.goodput ?? 0} req/s` },
      { label: 'Avg Latency', value: `${run.metrics.avgLatencyMs} ms` },
      { label: 'P50 Latency', value: `${run.metrics.p50LatencyMs ?? 0} ms` },
      { label: 'P95 Latency', value: `${run.metrics.p95LatencyMs} ms` },
      { label: 'P99 Latency', value: `${run.metrics.p99LatencyMs ?? 0} ms` },
      { label: 'P99.9 Latency', value: `${run.metrics.p999LatencyMs ?? 0} ms` },
      { label: 'Apdex Score', value: `${(run.metrics.apdexScore ?? 0).toFixed(2)}` },
      { label: 'Error Rate', value: `${((run.metrics.errorRate ?? 0) * 100).toFixed(3)}%` },
      { label: 'Availability', value: `${((run.metrics.availability ?? 0) * 100).toFixed(3)}%` },
      { label: 'SLO Burn Rate', value: `${(run.metrics.sloBurnRate ?? 0).toFixed(2)}x` },
      { label: 'Queue Max Depth', value: `${run.metrics.queueDepthMax}` },
      { label: 'Concurrency Peak', value: `${run.metrics.concurrencyPeak ?? 0}` },
      { label: 'Throughput', value: `${run.metrics.throughput}` },
      { label: 'CPU Estimate', value: `${run.metrics.cpuEstimate}%` },
      { label: 'Bottleneck Score', value: `${record.derived?.bottleneckScore ?? '-'}` },
      {
        label: 'Monthly Cost',
        value: record.cost ? `$${record.cost.total} ${record.cost.currency}` : '-',
      },
    ];
  }, [record]);

  if (!record) {
    return (
      <section className="rounded-xl border bg-panel p-6">
        <p className="text-sm text-mute">No run found yet. Execute simulation first.</p>
        <Link className="mt-3 inline-block text-sm text-accent underline" href={`/project/${projectId}/simulate`}>
          Go to simulate page
        </Link>
      </section>
    );
  }

  const run = record.run;
  const previous = history.length > 1 ? history[1] : null;
  const compareCards = previous
    ? [
        {
          label: 'Avg Latency Delta',
          value: formatDelta(run.metrics.avgLatencyMs - previous.run.metrics.avgLatencyMs, ' ms'),
        },
        {
          label: 'P95 Latency Delta',
          value: formatDelta(run.metrics.p95LatencyMs - previous.run.metrics.p95LatencyMs, ' ms'),
        },
        {
          label: 'RPS Delta',
          value: formatDelta(run.metrics.rps - previous.run.metrics.rps, ''),
        },
        {
          label: 'Error Rate Delta',
          value: formatDelta((run.metrics.errorRate - previous.run.metrics.errorRate) * 100, '%'),
        },
      ]
    : [];

  const activeSnapshot = replay?.snapshots[snapshotIndex] ?? null;

  return (
    <section className="space-y-4">
      {liveSnapshot ? (
        <article className="rounded-xl border bg-panel p-4">
          <h2 className="text-lg font-semibold">Active Run Progress</h2>
          <div className="mt-3 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <div className="rounded-lg border bg-black/30 p-3">
              <p className="text-xs text-mute">Progress</p>
              <p className="mt-1 text-xl font-semibold">{Math.round(liveSnapshot.progress * 100)}%</p>
            </div>
            <div className="rounded-lg border bg-black/30 p-3">
              <p className="text-xs text-mute">Live RPS</p>
              <p className="mt-1 text-xl font-semibold">{liveSnapshot.metrics.rps}</p>
            </div>
            <div className="rounded-lg border bg-black/30 p-3">
              <p className="text-xs text-mute">Live Latency</p>
              <p className="mt-1 text-xl font-semibold">{liveSnapshot.metrics.avgLatencyMs} ms</p>
            </div>
            <div className="rounded-lg border bg-black/30 p-3">
              <p className="text-xs text-mute">Current Event</p>
              <p className="mt-1 text-sm font-semibold">{liveSnapshot.currentEvent}</p>
            </div>
          </div>
        </article>
      ) : null}

      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        {cards.map((card) => (
          <article key={card.label} className="rounded-lg border bg-panel p-4">
            <p className="text-xs text-mute">{card.label}</p>
            <p className="mt-1 text-xl font-semibold">{card.value}</p>
          </article>
        ))}
      </div>

      {record.derived ? (
        <article className="rounded-xl border bg-panel p-4">
          <h2 className="text-lg font-semibold">Analytics</h2>
          <p className="mt-2 text-sm text-mute">{record.derived.capacityHint}</p>
          {record.derived.series.length ? (
            <div className="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
              <TrendChart
                title="Latency Trend"
                colorClassName="text-rose-300"
                series={record.derived.series}
                accessor={(point) => point.latencyMs}
                formatter={(value) => `${value.toFixed(2)} ms`}
              />
              <TrendChart
                title="RPS Trend"
                colorClassName="text-emerald-300"
                series={record.derived.series}
                accessor={(point) => point.rps}
                formatter={(value) => `${value.toFixed(2)}`}
              />
              <TrendChart
                title="Error Trend"
                colorClassName="text-amber-300"
                series={record.derived.series}
                accessor={(point) => point.errorRate * 100}
                formatter={(value) => `${value.toFixed(2)}%`}
              />
            </div>
          ) : null}
        </article>
      ) : null}

      {/* Enhanced visualization panels */}
      <div className="grid gap-4 lg:grid-cols-2">
        <SystemHealthDashboard metrics={run.metrics} derived={record.derived} />
        <LatencyBreakdown events={run.events} metrics={run.metrics} />
      </div>

      {run.metadata && liveSnapshot?.nodeStats ? (
        <NodePerformancePanel
          nodeStats={liveSnapshot.nodeStats}
          events={run.events}
          metrics={run.metrics}
        />
      ) : replay?.snapshots?.length ? (
        <NodePerformancePanel
          nodeStats={
            Object.fromEntries(
              Object.entries(replay.snapshots[replay.snapshots.length - 1]?.nodeState ?? {}).map(
                ([id, s]) => [id, { touchCount: s.touches, failureCount: s.failures, completedCount: s.completed, queuedCount: s.queued }]
              )
            )
          }
          events={run.events}
          metrics={run.metrics}
        />
      ) : null}

      {record.recommendation?.recommendations?.length ? (
        <article className="rounded-xl border bg-panel p-4">
          <h2 className="text-lg font-semibold">Recommendations</h2>
          <p className="mt-2 text-xs text-mute">Confidence: {record.recommendation.confidence}%</p>
          <div className="mt-3 grid gap-3 md:grid-cols-2">
            {record.recommendation.recommendations.map((rec) => (
              <div key={rec.id} className="rounded-lg border bg-black/30 p-3">
                <p className="text-sm font-semibold">{rec.problem}</p>
                <p className="mt-1 text-xs text-mute">{rec.reason}</p>
                <p className="mt-2 text-xs">Fix: {rec.fix}</p>
                <p className="mt-1 text-xs text-mute">
                  Latency {rec.expectedLatencyDeltaPct}% | Cost {rec.expectedCostDeltaPct}% | Risk {rec.riskScore}
                </p>
              </div>
            ))}
          </div>
        </article>
      ) : null}

      {record.cost ? (
        <article className="rounded-xl border bg-panel p-4">
          <h2 className="text-lg font-semibold">Cost Breakdown</h2>
          <div className="mt-2 space-y-1 text-sm text-mute">
            {record.cost.items.map((item) => (
              <div key={item.resourceType}>
                {item.resourceType}: {item.count} x ${item.unitMonthly} = ${item.subtotal}
              </div>
            ))}
          </div>
        </article>
      ) : null}

      {previous ? (
        <article className="rounded-xl border bg-panel p-4">
          <h2 className="text-lg font-semibold">Compare Behavior</h2>
          <p className="mt-2 text-sm text-mute">Latest run compared to previous run.</p>
          <div className="mt-3 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            {compareCards.map((card) => (
              <div key={card.label} className="rounded-lg border bg-black/30 p-3">
                <p className="text-xs text-mute">{card.label}</p>
                <p className="mt-1 text-lg font-semibold">{card.value}</p>
              </div>
            ))}
          </div>
        </article>
      ) : null}

      <article className="rounded-xl border bg-panel p-4">
        <h2 className="text-lg font-semibold">Replay Timeline (first 120 events)</h2>
        {replay?.snapshots?.length ? (
          <div className="mt-3 rounded-lg border bg-black/30 p-3">
            <div className="flex flex-wrap items-center gap-2">
              <button
                className="rounded border px-2 py-1 text-xs"
                onClick={() => setSnapshotIndex((current) => Math.max(0, current - 1))}
              >
                Step Back
              </button>
              <button
                className="rounded border px-2 py-1 text-xs"
                onClick={() => setSnapshotIndex((current) => Math.min((replay?.snapshots.length ?? 1) - 1, current + 1))}
              >
                Step Forward
              </button>
              <input
                className="w-64"
                type="range"
                min={0}
                max={Math.max(0, replay.snapshots.length - 1)}
                value={snapshotIndex}
                onChange={(event) => setSnapshotIndex(Number(event.target.value))}
              />
              <p className="text-xs text-mute">
                Snapshot {snapshotIndex + 1}/{replay.snapshots.length}
              </p>
            </div>

            {activeSnapshot ? (
              <div className="mt-2 text-xs text-mute">
                t={activeSnapshot.timestamp}ms | event index {activeSnapshot.eventIndex}
              </div>
            ) : null}
          </div>
        ) : null}
        <div className="mt-3 max-h-[520px] overflow-auto text-xs text-mute">
          {run.events.slice(0, 120).map((event, index) => (
            <div key={`${event.requestId}-${event.type}-${index}`} className="border-b border-line/40 py-1">
              t={event.timeMs}ms | req #{event.requestId} | {event.type}
              {event.nodeId ? ` | node: ${event.nodeId}` : ''}
              {event.detail ? ` | ${event.detail}` : ''}
            </div>
          ))}
        </div>
      </article>

      <EventMixChart events={run.events} />

      {activeSnapshot ? (
        <article className="rounded-xl border bg-panel p-4">
          <h2 className="text-lg font-semibold">Snapshot Node State</h2>
          <div className="mt-2 grid gap-2 md:grid-cols-2 xl:grid-cols-3 text-xs text-mute">
            {Object.entries(activeSnapshot.nodeState)
              .slice(0, 12)
              .map(([nodeId, nodeState]) => (
                <div key={nodeId} className="rounded-md border bg-black/30 p-2">
                  <p className="font-semibold text-white">{nodeId}</p>
                  <p>Touches: {nodeState.touches}</p>
                  <p>Completed: {nodeState.completed}</p>
                  <p>Queued: {nodeState.queued}</p>
                  <p>Failures: {nodeState.failures}</p>
                </div>
              ))}
          </div>
        </article>
      ) : null}

      {topology && activeSnapshot ? (
        <article className="rounded-xl border bg-panel p-4">
          <h2 className="text-lg font-semibold">Replay Topology View</h2>
          <p className="mt-1 mb-3 text-xs text-mute">
            Node colors reflect health at current snapshot. Green = healthy, Yellow = degraded, Red = critical.
          </p>
          <ReplayTopologyOverlay topology={topology} snapshot={activeSnapshot} />
        </article>
      ) : null}

      <SimulationHistory
        projectId={projectId}
        onSelectRun={(selectedRun) => {
          setRecord(selectedRun);
        }}
      />
    </section>
  );
}

function formatDelta(value: number, unit: string): string {
  const rounded = Math.round(value * 100) / 100;
  if (rounded > 0) {
    return `+${rounded}${unit}`;
  }
  return `${rounded}${unit}`;
}

type V1Envelope<T> = {
  success: boolean;
  data: T;
  error: { code: string; message: string } | null;
};

async function parseV1Data<T>(response: Response): Promise<T> {
  const text = await response.text();
  if (!text || !text.trim()) {
    throw new Error(`Empty response body (status ${response.status})`);
  }
  let payload: Partial<V1Envelope<T>>;
  try {
    payload = JSON.parse(text) as Partial<V1Envelope<T>>;
  } catch {
    throw new Error(`Invalid JSON response (status ${response.status})`);
  }
  if (typeof payload !== 'object' || payload === null || typeof payload.success !== 'boolean') {
    throw new Error('Invalid v1 envelope response');
  }
  if (!payload.success) {
    throw new Error(payload.error?.message ?? `Request failed with status ${response.status}`);
  }
  if (!("data" in payload)) {
    throw new Error('Missing v1 envelope data field');
  }
  return payload.data as T;
}
