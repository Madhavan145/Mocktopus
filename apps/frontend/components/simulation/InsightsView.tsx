'use client';

import Link from 'next/link';
import { useEffect, useMemo, useState, useCallback } from 'react';

import { apiFetch } from '@/lib/api';
import { readLastSimulationRun, readSimulationHistory } from './storage';
import type { SimulationRunRecord } from './types';

type Mode = 'compare' | 'cost' | 'recommendations' | 'replay';

const MAX_COMPARE_RUNS = 5;

type ComparisonVerdict = {
  recommended: string;
  score: Record<string, number>;
  reasons: string[];
  tradeoffs: string[];
};

type Props = {
  projectId: string;
  mode: Mode;
};

export default function InsightsView({ projectId, mode }: Props) {
  const [latest, setLatest] = useState<SimulationRunRecord | null>(null);
  const [history, setHistory] = useState<SimulationRunRecord[]>([]);
  const [selectedIndices, setSelectedIndices] = useState<Set<number>>(new Set());
  const [verdict, setVerdict] = useState<ComparisonVerdict | null>(null);
  const [verdictLoading, setVerdictLoading] = useState(false);

  useEffect(() => {
    setLatest(readLastSimulationRun(projectId));
    const hist = readSimulationHistory(projectId);
    setHistory(hist);
    // Auto-select the 2 most recent runs if available
    const initial = new Set<number>();
    if (hist.length > 0) initial.add(0);
    if (hist.length > 1) initial.add(1);
    setSelectedIndices(initial);
  }, [projectId]);

  const selectedRuns = useMemo(() => {
    return Array.from(selectedIndices)
      .sort((a, b) => a - b)
      .map((idx) => history[idx])
      .filter(Boolean);
  }, [history, selectedIndices]);

  const toggleSelection = (idx: number) => {
    setSelectedIndices((prev) => {
      const next = new Set(prev);
      if (next.has(idx)) {
        next.delete(idx);
      } else if (next.size < MAX_COMPARE_RUNS) {
        next.add(idx);
      }
      return next;
    });
  };

  // Fetch backend verdict when runs are selected for comparison.
  const fetchVerdict = useCallback(async () => {
    const runIds = selectedRuns.map((r) => r.runId).filter(Boolean);
    if (runIds.length < 2) {
      setVerdict(null);
      return;
    }
    setVerdictLoading(true);
    try {
      const res = await apiFetch(`/api/v1/simulation/compare`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ runIds }),
      });
      if (res.ok) {
        const envelope = await res.json();
        const data = envelope.data ?? envelope;
        setVerdict(data.verdict ?? null);
      } else {
        setVerdict(null);
      }
    } catch {
      setVerdict(null);
    } finally {
      setVerdictLoading(false);
    }
  }, [selectedRuns]);

  useEffect(() => {
    if (mode === 'compare' && selectedRuns.length >= 2) {
      void fetchVerdict();
    }
  }, [mode, selectedRuns, fetchVerdict]);

  const recommendations = latest?.recommendation?.recommendations ?? [];
  const costs = latest?.cost?.items ?? [];

  const compareMetrics = [
    { key: 'avgLatencyMs', label: 'Avg Latency (ms)', getter: (r: SimulationRunRecord) => r.run.metrics.avgLatencyMs },
    { key: 'p95LatencyMs', label: 'P95 Latency (ms)', getter: (r: SimulationRunRecord) => r.run.metrics.p95LatencyMs },
    { key: 'rps', label: 'RPS', getter: (r: SimulationRunRecord) => r.run.metrics.rps },
    { key: 'errorRate', label: 'Error Rate (%)', getter: (r: SimulationRunRecord) => r.run.metrics.errorRate * 100 },
    { key: 'throughput', label: 'Throughput', getter: (r: SimulationRunRecord) => r.run.metrics.throughput },
    { key: 'queueWaitMs', label: 'Queue Wait (ms)', getter: (r: SimulationRunRecord) => r.run.metrics.queueWaitMs },
    { key: 'cpuEstimate', label: 'CPU Estimate (%)', getter: (r: SimulationRunRecord) => r.run.metrics.cpuEstimate },
    { key: 'cost', label: 'Monthly Cost ($)', getter: (r: SimulationRunRecord) => r.cost?.total ?? 0 },
  ];

  if (!latest) {
    return (
      <section className="rounded-xl border bg-panel p-6">
        <p className="text-sm text-mute">No simulation run found for this project yet.</p>
        <Link className="mt-3 inline-block text-sm text-accent underline" href={`/project/${projectId}/simulate`}>
          Run a simulation first
        </Link>
      </section>
    );
  }

  if (mode === 'replay') {
    return (
      <section className="space-y-4">
        <article className="rounded-xl border bg-panel p-4">
          <h2 className="text-lg font-semibold">Replay Readiness</h2>
          <p className="mt-2 text-sm text-mute">
            Full timeline replay is available in the consolidated results view. This route is a direct deep-link for
            replay-focused workflows.
          </p>
          <p className="mt-2 text-xs text-mute">
            Event taxonomy: {latest.run.metadata?.eventTaxonomyVersion ?? 'legacy'} | Deterministic:{' '}
            {latest.run.metadata?.deterministic ? 'Yes' : 'No'}
          </p>
          <Link className="mt-3 inline-block rounded-md border px-3 py-2 text-sm" href={`/project/${projectId}/results`}>
            Open full replay timeline
          </Link>
        </article>
      </section>
    );
  }

  if (mode === 'compare') {
    return (
      <section className="space-y-4">
        {/* Run selector */}
        <article className="rounded-xl border bg-panel p-4">
          <h2 className="text-lg font-semibold">Select Runs to Compare</h2>
          <p className="mt-1 text-xs text-mute">
            Select up to {MAX_COMPARE_RUNS} simulation runs. {selectedIndices.size}/{MAX_COMPARE_RUNS} selected.
          </p>
          <div className="mt-3 max-h-48 overflow-auto space-y-1">
            {history.map((run, idx) => {
              const isSelected = selectedIndices.has(idx);
              const isDisabled = !isSelected && selectedIndices.size >= MAX_COMPARE_RUNS;
              return (
                <label
                  key={run.capturedAt ?? idx}
                  className={`flex items-center gap-2 rounded-md border px-3 py-2 text-sm cursor-pointer transition-colors ${
                    isSelected ? 'border-accent bg-accent/10' : 'border-line/40 hover:bg-black/20'
                  } ${isDisabled ? 'opacity-50 cursor-not-allowed' : ''}`}
                >
                  <input
                    type="checkbox"
                    checked={isSelected}
                    disabled={isDisabled}
                    onChange={() => toggleSelection(idx)}
                    className="accent-accent"
                  />
                  <span className="font-medium">Run #{history.length - idx}</span>
                  <span className="text-mute text-xs ml-auto">
                    {run.capturedAt ? new Date(run.capturedAt).toLocaleString() : `Index ${idx}`}
                    {' | '}RPS: {run.run.metrics.rps} | Latency: {run.run.metrics.avgLatencyMs}ms
                  </span>
                </label>
              );
            })}
          </div>
        </article>

        {/* Comparison table */}
        <article className="rounded-xl border bg-panel p-4">
          <h2 className="text-lg font-semibold">Architecture Comparison</h2>
          {selectedRuns.length < 2 ? (
            <p className="mt-2 text-sm text-mute">Select at least 2 runs above to compare behavior.</p>
          ) : (
            <div className="mt-3 overflow-auto rounded-md border">
              <table className="min-w-full text-sm">
                <thead className="bg-black/30 text-mute">
                  <tr>
                    <th className="px-3 py-2 text-left sticky left-0 bg-black/30">Metric</th>
                    {selectedRuns.map((run, i) => (
                      <th key={i} className="px-3 py-2 text-left whitespace-nowrap">
                        Run #{history.length - Array.from(selectedIndices).sort((a, b) => a - b)[i]}
                      </th>
                    ))}
                    <th className="px-3 py-2 text-left">Best</th>
                  </tr>
                </thead>
                <tbody>
                  {compareMetrics.map((metric) => {
                    const values = selectedRuns.map((run) => metric.getter(run));
                    // For error rate and latency, lower is better. For RPS/throughput, higher is better.
                    const isLowerBetter = metric.key !== 'rps' && metric.key !== 'throughput';
                    const bestVal = isLowerBetter ? Math.min(...values) : Math.max(...values);
                    const bestIdx = values.indexOf(bestVal);
                    return (
                      <tr key={metric.key} className="border-t border-line/40">
                        <td className="px-3 py-2 font-medium sticky left-0 bg-panel">{metric.label}</td>
                        {values.map((val, i) => (
                          <td
                            key={i}
                            className={`px-3 py-2 ${i === bestIdx ? 'text-emerald-300 font-semibold' : ''}`}
                          >
                            {round2(val)}
                          </td>
                        ))}
                        <td className="px-3 py-2 text-emerald-300 font-semibold">
                          Run #{history.length - Array.from(selectedIndices).sort((a, b) => a - b)[bestIdx]}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </article>

        {/* Delta summary vs latest */}
        {selectedRuns.length >= 2 && (
          <article className="rounded-xl border bg-panel p-4">
            <h2 className="text-lg font-semibold">Delta Summary (vs Latest)</h2>
            <div className="mt-3 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
              {selectedRuns.slice(1).map((run, idx) => {
                const latDelta = round2(run.run.metrics.avgLatencyMs - selectedRuns[0].run.metrics.avgLatencyMs);
                const rpsDelta = round2(run.run.metrics.rps - selectedRuns[0].run.metrics.rps);
                const errDelta = round2((run.run.metrics.errorRate - selectedRuns[0].run.metrics.errorRate) * 100);
                return (
                  <div key={idx} className="rounded-lg border bg-black/30 p-3">
                    <p className="text-xs text-mute font-medium">
                      Run #{history.length - Array.from(selectedIndices).sort((a, b) => a - b)[idx + 1]} vs Latest
                    </p>
                    <p className={`mt-1 text-sm ${latDelta <= 0 ? 'text-emerald-300' : 'text-amber-300'}`}>
                      Latency: {latDelta > 0 ? '+' : ''}{latDelta} ms
                    </p>
                    <p className={`text-sm ${rpsDelta >= 0 ? 'text-emerald-300' : 'text-amber-300'}`}>
                      RPS: {rpsDelta > 0 ? '+' : ''}{rpsDelta}
                    </p>
                    <p className={`text-sm ${errDelta <= 0 ? 'text-emerald-300' : 'text-amber-300'}`}>
                      Error: {errDelta > 0 ? '+' : ''}{errDelta}%
                    </p>
                  </div>
                );
              })}
            </div>
          </article>
        )}

        {/* Backend verdict */}
        {verdict && (
          <article className="rounded-xl border border-accent/30 bg-panel p-4">
            <h2 className="text-lg font-semibold">Architecture Verdict</h2>
            {verdict.recommended ? (
              <>
                <p className="mt-2 text-sm">
                  <span className="text-emerald-300 font-semibold">Recommended:</span>{' '}
                  {verdict.recommended}
                </p>
                {verdict.reasons?.length ? (
                  <p className="mt-1 text-xs text-mute">
                    Reasons: {verdict.reasons.join(', ')}
                  </p>
                ) : null}
                {verdict.tradeoffs?.length ? (
                  <p className="mt-1 text-xs text-amber-300">
                    Tradeoffs: {verdict.tradeoffs.join(', ')}
                  </p>
                ) : null}
              </>
            ) : (
              <p className="mt-2 text-sm text-mute">No clear winner — architectures perform similarly.</p>
            )}
            {verdictLoading && <p className="mt-1 text-xs text-mute animate-pulse">Loading verdict...</p>}
          </article>
        )}
      </section>
    );
  }

  if (mode === 'cost') {
    return (
      <section className="space-y-4">
        <article className="rounded-xl border bg-panel p-4">
          <h2 className="text-lg font-semibold">Cost Analysis</h2>
          <p className="mt-2 text-sm text-mute">
            Estimated monthly spend: {latest.cost ? `$${latest.cost.total} ${latest.cost.currency}` : 'N/A'}
          </p>
          <div className="mt-3 grid gap-3 md:grid-cols-2">
            {costs.map((item) => (
              <div key={item.resourceType} className="rounded-md border bg-black/20 p-3 text-sm">
                <p className="font-medium">{item.resourceType}</p>
                <p className="mt-1 text-mute">
                  {item.count} x ${item.unitMonthly} = ${item.subtotal}
                </p>
              </div>
            ))}
          </div>
          {latest.recommendation?.recommendations?.length ? (
            <p className="mt-3 text-xs text-mute">
              Top tradeoff: {latest.recommendation.recommendations[0].problem} | Cost impact{' '}
              {latest.recommendation.recommendations[0].expectedCostDeltaPct}%
            </p>
          ) : null}
        </article>
      </section>
    );
  }

  return (
    <section className="space-y-4">
      <article className="rounded-xl border bg-panel p-4">
        <h2 className="text-lg font-semibold">Recommendations</h2>
        <p className="mt-2 text-xs text-mute">Confidence: {latest.recommendation?.confidence ?? 0}%</p>
        {recommendations.length === 0 ? (
          <p className="mt-2 text-sm text-mute">No recommendations available for the current run.</p>
        ) : (
          <div className="mt-3 grid gap-3 md:grid-cols-2">
            {recommendations.map((rec) => (
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
        )}
      </article>
    </section>
  );
}

function round2(value: number): number {
  return Math.round(value * 100) / 100;
}
