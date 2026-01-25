'use client';

import Link from 'next/link';
import { useEffect, useMemo, useRef, useState } from 'react';

import { EventMixChart, TrendChart } from './MetricCharts';
import {
  publishActiveSimulationRunId,
  publishLiveSimulationSnapshot,
  writeSimulationRun,
} from './storage';
import type {
  CostEstimateResponse,
  LiveMetricsSnapshot,
  MetricsDerivedResponse,
  RecommendationResponse,
  RunStateResponse,
  SimulationRandomnessConfig,
  SimulationRunResult,
  StartRunResponse,
  TopologyResponse,
} from './types';

import { apiFetch } from '@/lib/api';

type Props = {
  projectId: string;
};

export default function SimulateWorkspace({ projectId }: Props) {
  const pollerRef = useRef<number | null>(null);
  const [durationSeconds, setDurationSeconds] = useState(60);
  const [requestsPerSecond, setRequestsPerSecond] = useState(25);
  const [maxRetries, setMaxRetries] = useState(1);
  const [networkLatencyMs, setNetworkLatencyMs] = useState(8);
  const [randomnessEnabled, setRandomnessEnabled] = useState(true);
  const [useRandomSeed, setUseRandomSeed] = useState(true);
  const [randomSeed, setRandomSeed] = useState(42);
  const [arrivalJitter, setArrivalJitter] = useState(0.15);
  const [latencyJitter, setLatencyJitter] = useState(0.2);
  const [failureVariance, setFailureVariance] = useState(0.35);

  // Advanced simulation configs
  const [multiRegionEnabled, setMultiRegionEnabled] = useState(false);
  const [multiRegionRegions, setMultiRegionRegions] = useState('us-east-1,eu-west-1');
  const [multiRegionStrategy, setMultiRegionStrategy] = useState('active-active');
  const [serviceMeshEnabled, setServiceMeshEnabled] = useState(false);
  const [serviceMeshSidecar, setServiceMeshSidecar] = useState('envoy');
  const [serviceMeshMtls, setServiceMeshMtls] = useState(true);
  const [autoscalingEnabled, setAutoscalingEnabled] = useState(false);
  const [autoscaleMinReplicas, setAutoscaleMinReplicas] = useState(1);
  const [autoscaleMaxReplicas, setAutoscaleMaxReplicas] = useState(10);
  const [autoscaleCpuTarget, setAutoscaleCpuTarget] = useState(70);

  const [status, setStatus] = useState('Idle');
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<SimulationRunResult | null>(null);
  const [derived, setDerived] = useState<MetricsDerivedResponse | null>(null);
  const [recommendation, setRecommendation] = useState<RecommendationResponse | null>(null);
  const [cost, setCost] = useState<CostEstimateResponse | null>(null);
  const [liveSnapshot, setLiveSnapshot] = useState<LiveMetricsSnapshot | null>(null);

  useEffect(() => {
    return () => {
      if (pollerRef.current) {
        window.clearInterval(pollerRef.current);
      }
    };
  }, []);

  const summaryCards = useMemo(() => {
    if (!result) {
      return [];
    }

    return [
      { label: 'Avg Latency', value: `${result.metrics.avgLatencyMs} ms` },
      { label: 'P95 Latency', value: `${result.metrics.p95LatencyMs} ms` },
      { label: 'RPS', value: `${result.metrics.rps}` },
      { label: 'Error Rate', value: `${(result.metrics.errorRate * 100).toFixed(2)}%` },
      { label: 'Queue Max Depth', value: `${result.metrics.queueDepthMax}` },
      { label: 'CPU Estimate', value: `${result.metrics.cpuEstimate}%` },
      { label: 'Bottleneck Score', value: `${derived?.bottleneckScore ?? '-'} ` },
      { label: 'Estimated Monthly Cost', value: cost ? `$${cost.total}` : '-' },
    ];
  }, [result, derived, cost]);

  const randomnessConfig = useMemo<SimulationRandomnessConfig>(
    () => ({
      enabled: randomnessEnabled,
      seed: useRandomSeed ? 0 : Math.max(0, Math.floor(randomSeed)),
      arrivalJitter,
      latencyJitter,
      failureVariance,
    }),
    [arrivalJitter, failureVariance, latencyJitter, randomSeed, randomnessEnabled, useRandomSeed],
  );

  const runSimulation = async () => {
    setError(null);
    setLiveSnapshot(null);
    publishLiveSimulationSnapshot(projectId, null);
    publishActiveSimulationRunId(projectId, null);
    setStatus('Loading topology from topology-service...');

    try {
      const topologyResponse = await apiFetch(`/api/v1/topology/projects/${encodeURIComponent(projectId)}`);
      if (!topologyResponse.ok) {
        throw new Error('No saved topology found. Save architecture in Builder first.');
      }

      const saved = await parseV1Data<TopologyResponse>(topologyResponse);
      setStatus('Running simulation...');

      const response = await apiFetch(`/api/v1/simulation/runs`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          topology: saved.ctm,
          traffic: {
            durationSeconds,
            requestsPerSecond,
            maxRetries,
            baseNetworkLatencyMs: networkLatencyMs,
            retryPenaltyMs: 20,
            randomness: randomnessConfig,
            ...(multiRegionEnabled && {
              multiRegion: {
                enabled: true,
                regions: multiRegionRegions.split(',').map(r => r.trim()),
                strategy: multiRegionStrategy,
              },
            }),
            ...(serviceMeshEnabled && {
              serviceMesh: {
                enabled: true,
                sidecar: serviceMeshSidecar,
                mtlsEnabled: serviceMeshMtls,
              },
            }),
            ...(autoscalingEnabled && {
              autoscaling: {
                enabled: true,
                minReplicas: autoscaleMinReplicas,
                maxReplicas: autoscaleMaxReplicas,
                targetCpuPercent: autoscaleCpuTarget,
              },
            }),
          },
        }),
      });

      if (!response.ok) {
        throw new Error(`simulation failed (status ${response.status})`);
      }

      const started = await parseV1Data<StartRunResponse>(response);
      publishActiveSimulationRunId(projectId, started.runId);

      const run = await new Promise<SimulationRunResult>((resolve, reject) => {
        pollerRef.current = window.setInterval(async () => {
          try {
            const stateResponse = await apiFetch(
              `/api/v1/simulation/runs/${encodeURIComponent(started.runId)}`,
            );
            if (!stateResponse.ok) {
              throw new Error('unable to read run state');
            }

            const state = await parseV1Data<RunStateResponse>(stateResponse);
            if (state.snapshot) {
              setLiveSnapshot(state.snapshot);
              publishLiveSimulationSnapshot(projectId, state.snapshot);
              setStatus(`Running simulation... ${Math.round(state.snapshot.progress * 100)}%`);
            }

            if (state.status === 'failed') {
              if (pollerRef.current) {
                window.clearInterval(pollerRef.current);
                pollerRef.current = null;
              }
              publishActiveSimulationRunId(projectId, null);
              publishLiveSimulationSnapshot(projectId, null);
              reject(new Error(state.error || 'simulation failed'));
              return;
            }

            if (state.status === 'completed' && state.result) {
              if (pollerRef.current) {
                window.clearInterval(pollerRef.current);
                pollerRef.current = null;
              }
              publishActiveSimulationRunId(projectId, null);
              publishLiveSimulationSnapshot(projectId, null);
              resolve(state.result);
            }
          } catch (pollError) {
            if (pollerRef.current) {
              window.clearInterval(pollerRef.current);
              pollerRef.current = null;
            }
            publishActiveSimulationRunId(projectId, null);
            publishLiveSimulationSnapshot(projectId, null);
            reject(pollError instanceof Error ? pollError : new Error('polling failed'));
          }
        }, 350);
      });

      setStatus('Deriving analytics and cost estimate...');
      const [derivedResponse, recommendationResponse, costResponse] = await Promise.all([
        apiFetch(`/api/v1/metrics/derive`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            projectId,
            metrics: run.metrics,
          }),
        }),
        apiFetch(`/api/v1/recommendation/derive`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            projectId,
            topology: saved.ctm,
            metrics: run.metrics,
          }),
        }),
        apiFetch(`/api/v1/cost/estimate`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            projectId,
            topology: saved.ctm,
          }),
        }),
      ]);

      let nextDerived: MetricsDerivedResponse | null = null;
      if (derivedResponse.ok) {
        nextDerived = await parseV1Data<MetricsDerivedResponse>(derivedResponse);
      }

      let nextRecommendation: RecommendationResponse | null = null;
      if (recommendationResponse.ok) {
        nextRecommendation = await parseV1Data<RecommendationResponse>(recommendationResponse);
      }

      let nextCost: CostEstimateResponse | null = null;
      if (costResponse.ok) {
        nextCost = await parseV1Data<CostEstimateResponse>(costResponse);
      }

      setResult(run);
      setDerived(nextDerived);
      setRecommendation(nextRecommendation);
      setCost(nextCost);
      writeSimulationRun(projectId, {
        runId: started.runId,
        run,
        derived: nextDerived,
        recommendation: nextRecommendation,
        cost: nextCost,
        capturedAt: new Date().toISOString(),
      });
      publishLiveSimulationSnapshot(projectId, null);
      publishActiveSimulationRunId(projectId, null);
      setStatus('Simulation complete. Result stored for Results page.');
    } catch (err) {
      setStatus('Failed');
      setError(err instanceof Error ? err.message : 'Unknown error');
      publishLiveSimulationSnapshot(projectId, null);
      publishActiveSimulationRunId(projectId, null);
    }
  };

  return (
    <section className="grid gap-4 lg:grid-cols-3">
      <div className="rounded-xl border bg-panel p-4 lg:col-span-1">
        <h2 className="text-xl font-semibold">Simulation Controls</h2>
        <p className="mt-1 text-sm text-mute">Configure traffic and run with current CTM topology.</p>

        <div className="mt-4 space-y-3">
          <NumberField label="Duration (seconds)" value={durationSeconds} onChange={setDurationSeconds} min={5} step={5} />
          <NumberField label="Requests per second" value={requestsPerSecond} onChange={setRequestsPerSecond} min={1} step={1} />
          <NumberField label="Max retries" value={maxRetries} onChange={setMaxRetries} min={0} step={1} />
          <NumberField
            label="Network latency (ms)"
            value={networkLatencyMs}
            onChange={setNetworkLatencyMs}
            min={1}
            step={1}
          />
          <CheckboxField label="Enable stochastic mode" checked={randomnessEnabled} onChange={setRandomnessEnabled} />

          {randomnessEnabled ? (
            <>
              <CheckboxField label="Use random seed each run" checked={useRandomSeed} onChange={setUseRandomSeed} />
              {!useRandomSeed ? (
                <NumberField label="Seed" value={randomSeed} onChange={setRandomSeed} min={0} step={1} />
              ) : null}
              <FloatField label="Arrival jitter (0-1)" value={arrivalJitter} onChange={setArrivalJitter} step={0.01} />
              <FloatField label="Latency jitter (0-1)" value={latencyJitter} onChange={setLatencyJitter} step={0.01} />
              <FloatField
                label="Failure variance (0-1)"
                value={failureVariance}
                onChange={setFailureVariance}
                step={0.01}
              />
            </>
          ) : null}
        </div>

        {/* Advanced Simulation Config */}
        <div className="mt-4 space-y-3 border-t pt-4">
          <h3 className="text-sm font-medium">Advanced Config</h3>

          <CheckboxField label="Multi-Region" checked={multiRegionEnabled} onChange={setMultiRegionEnabled} />
          {multiRegionEnabled && (
            <div className="ml-4 space-y-2">
              <div>
                <label className="block text-xs text-mute">Regions (comma-separated)</label>
                <input
                  className="w-full rounded-md border bg-transparent px-3 py-2 text-sm"
                  type="text"
                  value={multiRegionRegions}
                  onChange={(e) => setMultiRegionRegions(e.target.value)}
                />
              </div>
              <div>
                <label className="block text-xs text-mute">Strategy</label>
                <select
                  className="w-full rounded-md border bg-transparent px-3 py-2 text-sm"
                  value={multiRegionStrategy}
                  onChange={(e) => setMultiRegionStrategy(e.target.value)}
                >
                  <option value="active-active">Active-Active</option>
                  <option value="active-passive">Active-Passive</option>
                  <option value="read-replica">Read Replica</option>
                </select>
              </div>
            </div>
          )}

          <CheckboxField label="Service Mesh" checked={serviceMeshEnabled} onChange={setServiceMeshEnabled} />
          {serviceMeshEnabled && (
            <div className="ml-4 space-y-2">
              <div>
                <label className="block text-xs text-mute">Sidecar Proxy</label>
                <select
                  className="w-full rounded-md border bg-transparent px-3 py-2 text-sm"
                  value={serviceMeshSidecar}
                  onChange={(e) => setServiceMeshSidecar(e.target.value)}
                >
                  <option value="envoy">Envoy (Istio)</option>
                  <option value="linkerd">Linkerd</option>
                  <option value="consul-connect">Consul Connect</option>
                </select>
              </div>
              <CheckboxField label="mTLS Enabled" checked={serviceMeshMtls} onChange={setServiceMeshMtls} />
            </div>
          )}

          <CheckboxField label="Autoscaling" checked={autoscalingEnabled} onChange={setAutoscalingEnabled} />
          {autoscalingEnabled && (
            <div className="ml-4 space-y-2">
              <NumberField label="Min Replicas" value={autoscaleMinReplicas} onChange={setAutoscaleMinReplicas} min={1} step={1} />
              <NumberField label="Max Replicas" value={autoscaleMaxReplicas} onChange={setAutoscaleMaxReplicas} min={1} step={1} />
              <NumberField label="Target CPU %" value={autoscaleCpuTarget} onChange={setAutoscaleCpuTarget} min={10} step={5} />
            </div>
          )}
        </div>

        <button className="mt-4 w-full rounded-md bg-accent px-4 py-2 font-medium text-black" onClick={runSimulation}>
          Run Simulation
        </button>

        <p className="mt-3 text-xs text-mute">Status: {status}</p>
        {liveSnapshot ? (
          <div className="mt-3 rounded-md border bg-black/20 p-3 text-xs text-mute">
            <div>Live progress: {(liveSnapshot.progress * 100).toFixed(0)}%</div>
            <div>Live RPS: {liveSnapshot.metrics.rps}</div>
            <div>Live latency: {liveSnapshot.metrics.avgLatencyMs} ms</div>
            <div>Live event: {liveSnapshot.currentEvent}</div>
            <div>Virtual time: {liveSnapshot.currentTimeMs} ms</div>
          </div>
        ) : null}
        {error ? <p className="mt-2 text-xs text-red-300">{error}</p> : null}
        <Link className="mt-3 inline-block text-sm text-accent underline" href={`/project/${projectId}/results`}>
          Open Results Page
        </Link>
      </div>

      <div className="rounded-xl border bg-panel p-4 lg:col-span-2">
        <h2 className="text-xl font-semibold">Run Snapshot</h2>
        {!result ? (
          <p className="mt-3 text-sm text-mute">No simulation run yet.</p>
        ) : (
          <>
            <div className="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
              {summaryCards.map((card) => (
                <article key={card.label} className="rounded-lg border bg-black/30 p-3">
                  <p className="text-xs text-mute">{card.label}</p>
                  <p className="mt-1 text-xl font-semibold">{card.value}</p>
                </article>
              ))}
            </div>

            <div className="mt-4 rounded-lg border bg-black/30 p-3">
              <h3 className="text-sm font-medium">Event Timeline (first 30)</h3>
              <div className="mt-2 max-h-64 overflow-auto text-xs text-mute">
                {result.events.slice(0, 30).map((event, index) => (
                  <div key={`${event.requestId}-${event.type}-${index}`} className="border-b border-line/40 py-1">
                    t={event.timeMs}ms | req #{event.requestId} | {event.type}
                    {event.nodeId ? ` | node: ${event.nodeId}` : ''}
                    {event.detail ? ` | ${event.detail}` : ''}
                  </div>
                ))}
              </div>
            </div>

            {derived?.series?.length ? (
              <div className="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                <TrendChart
                  title="Latency Trend"
                  colorClassName="text-rose-300"
                  series={derived.series}
                  accessor={(point) => point.latencyMs}
                  formatter={(value) => `${value.toFixed(2)} ms`}
                />
                <TrendChart
                  title="RPS Trend"
                  colorClassName="text-emerald-300"
                  series={derived.series}
                  accessor={(point) => point.rps}
                  formatter={(value) => `${value.toFixed(2)}`}
                />
                <TrendChart
                  title="Error Trend"
                  colorClassName="text-amber-300"
                  series={derived.series}
                  accessor={(point) => point.errorRate * 100}
                  formatter={(value) => `${value.toFixed(2)}%`}
                />
              </div>
            ) : null}

            <div className="mt-4">
              <EventMixChart events={result.events} />
            </div>

            {derived?.capacityHint ? (
              <p className="mt-3 text-sm text-mute">Capacity hint: {derived.capacityHint}</p>
            ) : null}

            {recommendation?.recommendations?.length ? (
              <div className="mt-4 rounded-lg border bg-black/30 p-3">
                <h3 className="text-sm font-medium">Top Recommendation</h3>
                <p className="mt-2 text-xs text-mute">{recommendation.recommendations[0].problem}</p>
                <p className="mt-1 text-xs text-mute">{recommendation.recommendations[0].fix}</p>
              </div>
            ) : null}
          </>
        )}
      </div>
    </section>
  );
}

type NumberFieldProps = {
  label: string;
  value: number;
  onChange: (value: number) => void;
  min: number;
  step: number;
};

function NumberField({ label, value, onChange, min, step }: NumberFieldProps) {
  return (
    <div>
      <label className="block text-xs text-mute">{label}</label>
      <input
        className="w-full rounded-md border bg-transparent px-3 py-2 text-sm"
        type="number"
        value={value}
        min={min}
        step={step}
        onKeyDown={(e) => {
          // Block decimal point input when step is integer
          if (step >= 1 && (e.key === '.' || e.key === ',')) {
            e.preventDefault();
          }
        }}
        onChange={(event) => {
          let val = Number(event.target.value);
          // Force integer when step is >= 1
          if (step >= 1) {
            val = Math.round(val);
          }
          onChange(Math.max(min, val));
        }}
      />
    </div>
  );
}

type FloatFieldProps = {
  label: string;
  value: number;
  onChange: (value: number) => void;
  step: number;
};

function FloatField({ label, value, onChange, step }: FloatFieldProps) {
  return (
    <div>
      <label className="block text-xs text-mute">{label}</label>
      <input
        className="w-full rounded-md border bg-transparent px-3 py-2 text-sm"
        type="number"
        value={value}
        min={0}
        max={1}
        step={step}
        onChange={(event) => onChange(Number(event.target.value))}
      />
    </div>
  );
}

type CheckboxFieldProps = {
  label: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
};

function CheckboxField({ label, checked, onChange }: CheckboxFieldProps) {
  return (
    <label className="flex items-center gap-2 text-sm text-mute">
      <input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />
      <span>{label}</span>
    </label>
  );
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
