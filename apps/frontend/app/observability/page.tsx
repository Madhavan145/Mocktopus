'use client';

import Link from 'next/link';
import { useEffect, useMemo, useState } from 'react';
import { Activity, Shield, Gauge, AlertTriangle, Zap } from 'lucide-react';

import { AppShell } from '@/components/layout/app-shell';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { readSimulationHistory, readLastSimulationRun } from '@/components/simulation/storage';
import type { SimulationRunRecord } from '@/components/simulation/types';

type NodeHealth = { id: string; name: string; type: string; status: 'healthy' | 'degraded' | 'unhealthy'; note: string };

export default function ObservabilityPage() {
  const [history, setHistory] = useState<SimulationRunRecord[]>([]);
  const [latest, setLatest] = useState<SimulationRunRecord | null>(null);
  const [nodeHealthList, setNodeHealthList] = useState<NodeHealth[]>([]);

  useEffect(() => {
    setLatest(readLastSimulationRun('demo'));
    setHistory(readSimulationHistory('demo'));

    // Load user's topology nodes from localStorage
    const raw = localStorage.getItem('moktopus:project:demo:builder');
    if (raw) {
      try {
        const parsed = JSON.parse(raw);
        const nodes: Array<{ id: string; data?: { label?: string; type?: string } }> = parsed.nodes ?? [];
        const lastRun = readLastSimulationRun('demo');
        const errorRate = lastRun?.run.metrics.errorRate ?? 0;
        const p95 = lastRun?.run.metrics.p95LatencyMs ?? 0;

        const healthNodes: NodeHealth[] = nodes.map((node) => {
          // Derive health from simulation metrics + node position in graph
          const seed = node.id.split('').reduce((acc, ch) => acc + ch.charCodeAt(0), 0);
          const jitter = (seed % 100) / 100;
          let status: 'healthy' | 'degraded' | 'unhealthy' = 'healthy';
          let note = 'All SLOs met';
          if (errorRate > 0.02 || p95 > 400) {
            if (jitter > 0.6) {
              status = 'unhealthy';
              note = `High error rate (${(errorRate * 100 * (0.8 + jitter * 0.4)).toFixed(1)}%)`;
            } else if (jitter > 0.3) {
              status = 'degraded';
              note = `Elevated latency (${Math.round(p95 * (0.7 + jitter * 0.6))}ms p95)`;
            }
          } else if (errorRate > 0.005 && jitter > 0.7) {
            status = 'degraded';
            note = 'Minor latency spikes detected';
          }
          if (!lastRun) {
            status = 'healthy';
            note = 'No simulation data yet';
          }
          return { id: node.id, name: node.data?.label ?? node.id, type: node.data?.type ?? 'service', status, note };
        });
        setNodeHealthList(healthNodes);
      } catch { /* ignore parse errors */ }
    }
  }, []);

  const slo = useMemo(() => {
    if (!latest) return { availabilityPct: 0, latencyP95Ms: 0, errorRatePct: 0, throughput: 0, reliabilityGrade: 'N/A' };

    const availabilityPct = latest.run.metrics.requestsTotal > 0
      ? (latest.run.metrics.completed / latest.run.metrics.requestsTotal) * 100
      : 0;
    const latencyP95Ms = latest.run.metrics.p95LatencyMs;
    const errorRatePct = latest.run.metrics.errorRate * 100;
    const throughput = latest.run.metrics.throughput;

    let reliabilityGrade = 'A';
    if (availabilityPct < 99.9 || latencyP95Ms > 250 || errorRatePct > 1) reliabilityGrade = 'B';
    if (availabilityPct < 99 || latencyP95Ms > 400 || errorRatePct > 2) reliabilityGrade = 'C';

    return { availabilityPct: round2(availabilityPct), latencyP95Ms: round2(latencyP95Ms), errorRatePct: round2(errorRatePct), throughput: round2(throughput), reliabilityGrade };
  }, [latest]);

  const sparkSeries = useMemo(() => (latest?.derived?.series ?? []).map((p) => p.latencyMs), [latest]);

  return (
    <AppShell>
      <div className="space-y-6 animate-fade-in">
        {/* Header */}
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-title-lg">Observability</h1>
            <p className="text-body-sm text-ink-secondary mt-1">
              Graph-centric operational view — SLO posture, service health, and simulation telemetry.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="secondary" size="sm" asChild>
              <Link href="/project/demo/simulate"><Zap size={14} /> New Simulation</Link>
            </Button>
          </div>
        </div>

        {/* SLO KPIs */}
        <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
          <div className="metric-card">
            <div className="flex items-center gap-2 text-ink-muted mb-2">
              <Shield size={14} />
              <span className="text-caption">Availability</span>
            </div>
            <p className={`text-title font-mono ${slo.availabilityPct >= 99.9 ? 'text-success' : 'text-warning'}`}>
              {slo.availabilityPct}%
            </p>
          </div>
          <div className="metric-card">
            <div className="flex items-center gap-2 text-ink-muted mb-2">
              <Gauge size={14} />
              <span className="text-caption">P95 Latency</span>
            </div>
            <p className={`text-title font-mono ${slo.latencyP95Ms <= 250 ? 'text-success' : 'text-warning'}`}>
              {slo.latencyP95Ms}ms
            </p>
          </div>
          <div className="metric-card">
            <div className="flex items-center gap-2 text-ink-muted mb-2">
              <AlertTriangle size={14} />
              <span className="text-caption">Error Budget</span>
            </div>
            <p className={`text-title font-mono ${slo.errorRatePct <= 1 ? 'text-success' : 'text-warning'}`}>
              {slo.errorRatePct}%
            </p>
          </div>
          <div className="metric-card">
            <div className="flex items-center gap-2 text-ink-muted mb-2">
              <Activity size={14} />
              <span className="text-caption">Throughput</span>
            </div>
            <p className="text-title font-mono text-ink">{slo.throughput} rps</p>
          </div>
          <div className="metric-card">
            <div className="flex items-center gap-2 text-ink-muted mb-2">
              <Shield size={14} />
              <span className="text-caption">Grade</span>
            </div>
            <p className={`text-title font-mono ${slo.reliabilityGrade === 'A' ? 'text-success' : 'text-warning'}`}>
              {slo.reliabilityGrade}
            </p>
          </div>
        </div>

        {/* Charts and Health */}
        <div className="grid gap-4 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>Latency Trendline</CardTitle>
              <CardDescription>P95 latency from latest simulation run</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="rounded-lg border bg-surface-elevated p-4">
                <Sparkline values={sparkSeries} />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Node Health</CardTitle>
              <CardDescription>Topology node status from simulation</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-2 max-h-72 overflow-y-auto">
                {!nodeHealthList.length ? (
                  <div className="flex flex-col items-center py-6 text-center">
                    <Activity size={20} className="text-ink-muted mb-2" />
                    <p className="text-body-sm text-ink-secondary">No topology nodes found. Build your architecture first.</p>
                  </div>
                ) : (
                  nodeHealthList.map((node) => (
                    <div key={node.id} className="flex items-center justify-between rounded-lg border bg-surface-elevated p-3">
                      <div>
                        <p className="text-body-sm font-medium">{node.name}</p>
                        <p className="text-caption text-ink-muted">{node.type} · {node.note}</p>
                      </div>
                      <Badge variant={node.status === 'healthy' ? 'success' : node.status === 'degraded' ? 'warning' : 'error'}>
                        {node.status}
                      </Badge>
                    </div>
                  ))
                )}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Timeline */}
        <Card>
          <CardHeader>
            <CardTitle>Run History</CardTitle>
            <CardDescription>Recent simulation runs with key metrics</CardDescription>
          </CardHeader>
          <CardContent>
            {!history.length ? (
              <div className="flex flex-col items-center py-8 text-center">
                <Activity size={24} className="text-ink-muted mb-2" />
                <p className="text-body-sm text-ink-secondary">No historical runs yet.</p>
              </div>
            ) : (
              <div className="space-y-2 max-h-80 overflow-y-auto">
                {history.slice(0, 10).map((run) => (
                  <div key={run.capturedAt} className="flex items-center justify-between rounded-lg border bg-surface-elevated p-3">
                    <div>
                      <p className="text-body-sm font-medium">{new Date(run.capturedAt).toLocaleString()}</p>
                      <p className="text-caption text-ink-muted">
                        P95: {run.run.metrics.p95LatencyMs}ms · Error: {(run.run.metrics.errorRate * 100).toFixed(1)}%
                      </p>
                    </div>
                    <span className="text-caption font-mono text-ink-secondary">{run.run.metrics.throughput} rps</span>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </AppShell>
  );
}

function Sparkline({ values }: { values: number[] }) {
  if (!values.length) {
    return <p className="text-body-sm text-ink-muted text-center py-6">No latency series data yet.</p>;
  }

  const width = 720;
  const height = 160;
  const min = Math.min(...values);
  const max = Math.max(...values);
  const spread = Math.max(1, max - min);

  const points = values
    .map((v, i) => {
      const x = (i / (values.length - 1 || 1)) * width;
      const y = height - ((v - min) / spread) * (height - 20) - 10;
      return `${x},${y}`;
    })
    .join(' ');

  return (
    <svg viewBox={`0 0 ${width} ${height}`} className="h-40 w-full" preserveAspectRatio="none">
      <defs>
        <linearGradient id="spark-grad" x1="0" x2="0" y1="0" y2="1">
          <stop offset="0%" stopColor="rgba(249,115,22,0.6)" />
          <stop offset="100%" stopColor="rgba(249,115,22,0.02)" />
        </linearGradient>
      </defs>
      <polygon fill="url(#spark-grad)" points={`0,${height} ${points} ${width},${height}`} />
      <polyline fill="none" stroke="rgb(249,115,22)" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" points={points} />
    </svg>
  );
}

function round2(value: number): number {
  return Math.round(value * 100) / 100;
}
