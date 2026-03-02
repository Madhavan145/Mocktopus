'use client';

import type { SimulationMetrics, MetricsDerivedResponse } from './types';

type SystemHealthProps = {
  metrics: SimulationMetrics;
  derived: MetricsDerivedResponse | null;
};

function GaugeRing({ value, max, label, color, unit }: { value: number; max: number; label: string; color: string; unit?: string }) {
  const pct = Math.min((value / max) * 100, 100);
  const radius = 32;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference - (pct / 100) * circumference;

  return (
    <div className="flex flex-col items-center">
      <svg width="76" height="76" className="transform -rotate-90">
        <circle cx="38" cy="38" r={radius} fill="none" stroke="currentColor" strokeWidth="6" className="text-white/5" />
        <circle
          cx="38" cy="38" r={radius} fill="none" stroke="currentColor" strokeWidth="6"
          strokeDasharray={circumference} strokeDashoffset={offset}
          strokeLinecap="round" className={color}
        />
      </svg>
      <span className="mt-1 text-xs font-mono">{value.toFixed(1)}{unit ?? ''}</span>
      <span className="text-[10px] text-white/50">{label}</span>
    </div>
  );
}

function GradeBadge({ grade, label }: { grade: string; label: string }) {
  const colors: Record<string, string> = {
    A: 'bg-green-500/20 text-green-400',
    B: 'bg-blue-500/20 text-blue-400',
    C: 'bg-yellow-500/20 text-yellow-400',
    D: 'bg-orange-500/20 text-orange-400',
    F: 'bg-red-500/20 text-red-400',
  };
  const cls = colors[grade] ?? 'bg-white/10 text-white/60';
  return (
    <div className="flex flex-col items-center gap-0.5">
      <span className={`text-lg font-bold px-2 py-0.5 rounded ${cls}`}>{grade}</span>
      <span className="text-[10px] text-white/50">{label}</span>
    </div>
  );
}

function HistogramBar({ count, maxCount, label }: { count: number; maxCount: number; label: string }) {
  const height = maxCount > 0 ? Math.max(2, (count / maxCount) * 48) : 2;
  return (
    <div className="flex flex-col items-center gap-0.5">
      <div className="w-3 bg-white/5 rounded-sm relative" style={{ height: 48 }}>
        <div
          className="absolute bottom-0 w-full bg-blue-500/60 rounded-sm transition-all"
          style={{ height }}
        />
      </div>
      <span className="text-[8px] text-white/40">{label}</span>
    </div>
  );
}

function MetricTile({ label, value, unit, alert }: { label: string; value: string | number; unit?: string; alert?: boolean }) {
  return (
    <div className="rounded bg-black/40 border border-white/5 p-2">
      <span className="text-[10px] text-white/50 block">{label}</span>
      <span className={`block font-mono text-sm ${alert ? 'text-red-400' : ''}`}>
        {value}{unit ? <span className="text-[10px] text-white/40 ml-0.5">{unit}</span> : null}
      </span>
    </div>
  );
}

export function SystemHealthDashboard({ metrics, derived }: SystemHealthProps) {
  const bottleneck = derived?.bottleneckScore ?? 0;
  const healthStatus = bottleneck > 80 ? 'CRITICAL' : bottleneck > 60 ? 'DEGRADED' : bottleneck > 30 ? 'MODERATE' : 'HEALTHY';
  const healthColor = bottleneck > 80 ? 'text-red-400' : bottleneck > 60 ? 'text-yellow-400' : bottleneck > 30 ? 'text-blue-400' : 'text-green-400';
  const healthBg = bottleneck > 80 ? 'bg-red-500/10' : bottleneck > 60 ? 'bg-yellow-500/10' : bottleneck > 30 ? 'bg-blue-500/10' : 'bg-green-500/10';

  const histLabels = ['5', '10', '25', '50', '100', '250', '500', '1K', '2.5K', '5K', '10K', '∞'];
  const histogram = metrics.latencyHistogram ?? Array(12).fill(0);
  const maxBucket = Math.max(...histogram, 1);

  const avail = metrics.availability ?? 0;
  const apdex = metrics.apdexScore ?? 0;
  const sloBurn = metrics.sloBurnRate ?? 0;
  const p50 = metrics.p50LatencyMs ?? metrics.avgLatencyMs ?? 0;
  const p95 = metrics.p95LatencyMs ?? 0;
  const p99 = metrics.p99LatencyMs ?? 0;
  const p999 = metrics.p999LatencyMs ?? 0;
  const goodput = metrics.goodput ?? 0;
  const jitter = metrics.latencyJitter ?? 0;
  const mttf = metrics.mttfMs ?? 0;
  const mttr = metrics.mttrMs ?? 0;
  const mtbf = metrics.mtbfMs ?? 0;
  const concurrency = metrics.concurrencyPeak ?? 0;
  const costPer = metrics.costPerRequest ?? 0;

  return (
    <article className="rounded-lg border border-white/10 bg-black/30 p-4 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold">System Health Overview</h3>
        <div className="flex items-center gap-2">
          <span className={`text-[10px] px-2 py-0.5 rounded font-medium ${healthBg} ${healthColor}`}>
            {healthStatus}
          </span>
          {derived?.reliabilityTier && (
            <span className="text-[10px] px-2 py-0.5 rounded bg-white/5 text-white/60">
              {derived.reliabilityTier}
            </span>
          )}
        </div>
      </div>

      {/* Grades Row */}
      {derived && (
        <div className="flex justify-around py-2 border-y border-white/5">
          <GradeBadge grade={derived.scalabilityGrade ?? 'B'} label="Scalability" />
          <GradeBadge grade={derived.latencyGrade ?? 'B'} label="Latency" />
          <div className="flex flex-col items-center gap-0.5">
            <span className="text-lg font-bold px-2 py-0.5 rounded bg-white/10 text-white/80">
              {apdex.toFixed(2)}
            </span>
            <span className="text-[10px] text-white/50">Apdex</span>
          </div>
          <div className="flex flex-col items-center gap-0.5">
            <span className={`text-lg font-bold px-2 py-0.5 rounded ${
              sloBurn > 1 ? 'bg-red-500/20 text-red-400' : 'bg-green-500/20 text-green-400'
            }`}>
              {sloBurn.toFixed(1)}x
            </span>
            <span className="text-[10px] text-white/50">SLO Burn</span>
          </div>
        </div>
      )}

      {/* Gauges */}
      <div className="flex justify-around">
        <GaugeRing value={metrics.rps} max={metrics.rps * 1.5 || 100} label="RPS" color="text-blue-500" />
        <GaugeRing value={metrics.cpuEstimate ?? 0} max={100} label="CPU %" color={(metrics.cpuEstimate ?? 0) > 80 ? 'text-red-500' : (metrics.cpuEstimate ?? 0) > 50 ? 'text-yellow-500' : 'text-green-500'} unit="%" />
        <GaugeRing value={avail * 100} max={100} label="Avail." color={avail >= 0.999 ? 'text-green-500' : avail >= 0.99 ? 'text-yellow-500' : 'text-red-500'} unit="%" />
        <GaugeRing value={bottleneck} max={100} label="Risk" color={bottleneck > 75 ? 'text-red-500' : bottleneck > 45 ? 'text-yellow-500' : 'text-green-500'} />
      </div>

      {/* Latency Distribution Histogram */}
      <div>
        <h4 className="text-[10px] text-white/50 mb-1 uppercase tracking-wider">Latency Distribution (ms)</h4>
        <div className="flex justify-between items-end gap-0.5">
          {histogram.map((count: number, i: number) => (
            <HistogramBar key={i} count={count} maxCount={maxBucket} label={histLabels[i]} />
          ))}
        </div>
      </div>

      {/* Core Metrics Grid */}
      <div className="grid grid-cols-4 gap-2 text-xs">
        <MetricTile label="P50 Latency" value={p50.toFixed(1)} unit="ms" />
        <MetricTile label="P95 Latency" value={p95.toFixed(1)} unit="ms" />
        <MetricTile label="P99 Latency" value={p99.toFixed(1)} unit="ms" />
        <MetricTile label="P99.9 Latency" value={p999.toFixed(1)} unit="ms" alert={p999 > 1000} />
        <MetricTile label="Goodput" value={goodput.toFixed(1)} unit="req/s" />
        <MetricTile label="Throughput" value={(metrics.throughput ?? 0).toFixed(1)} unit="req/s" />
        <MetricTile label="Error Rate" value={((metrics.errorRate ?? 0) * 100).toFixed(3)} unit="%" alert={(metrics.errorRate ?? 0) > 0.01} />
        <MetricTile label="Concurrency" value={concurrency} />
      </div>

      {/* Reliability Metrics */}
      <div className="grid grid-cols-4 gap-2 text-xs">
        <MetricTile label="MTTF" value={formatDuration(mttf)} />
        <MetricTile label="MTTR" value={formatDuration(mttr)} />
        <MetricTile label="MTBF" value={formatDuration(mtbf)} />
        <MetricTile label="Jitter (CV)" value={jitter.toFixed(3)} alert={jitter > 1} />
      </div>

      {/* Resource & Cost */}
      <div className="grid grid-cols-3 gap-2 text-xs">
        <MetricTile label="Queue Depth" value={metrics.queueDepthMax ?? 0} />
        <MetricTile label="Cost/Request" value={formatCost(costPer)} />
        <MetricTile label="Cost Efficiency" value={derived?.costEfficiency ?? 'N/A'} />
      </div>

      {/* Capacity Hint */}
      {derived?.capacityHint && (
        <div className="rounded bg-white/5 border border-white/10 p-2 text-xs text-white/70">
          <span className="text-[10px] text-white/40 uppercase tracking-wider">Recommendation</span>
          <p className="mt-0.5">{derived.capacityHint}</p>
        </div>
      )}

      {/* ─── Industry-Standard Resilience Metrics ─── */}

      {/* Resilience Score */}
      {metrics.resilienceScore != null && (
        <div className="flex items-center justify-between rounded bg-black/40 border border-white/5 p-3">
          <div>
            <span className="text-[10px] text-white/40 uppercase tracking-wider">Resilience Score</span>
            <p className="text-2xl font-bold font-mono mt-0.5">{metrics.resilienceScore}<span className="text-sm text-white/40">/100</span></p>
          </div>
          <div className="w-24 h-2 bg-white/5 rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full transition-all ${
                metrics.resilienceScore >= 80 ? 'bg-green-500' : metrics.resilienceScore >= 60 ? 'bg-yellow-500' : metrics.resilienceScore >= 40 ? 'bg-orange-500' : 'bg-red-500'
              }`}
              style={{ width: `${metrics.resilienceScore}%` }}
            />
          </div>
        </div>
      )}

      {/* Error Budget (Google SRE) */}
      {metrics.errorBudget && (
        <div className="rounded bg-black/40 border border-white/5 p-3 space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-[10px] text-white/40 uppercase tracking-wider">Error Budget (SRE)</span>
            <span className={`text-[10px] px-2 py-0.5 rounded font-medium ${
              metrics.errorBudget.alertSeverity === 'page' ? 'bg-red-500/20 text-red-400' :
              metrics.errorBudget.alertSeverity === 'critical' ? 'bg-orange-500/20 text-orange-400' :
              metrics.errorBudget.alertSeverity === 'warning' ? 'bg-yellow-500/20 text-yellow-400' :
              'bg-green-500/20 text-green-400'
            }`}>
              {metrics.errorBudget.alertSeverity === 'none' ? 'HEALTHY' : metrics.errorBudget.alertSeverity.toUpperCase()}
            </span>
          </div>
          <div className="w-full h-3 bg-white/5 rounded-full overflow-hidden relative">
            <div
              className={`h-full rounded-full ${metrics.errorBudget.budgetConsumed > 80 ? 'bg-red-500' : metrics.errorBudget.budgetConsumed > 50 ? 'bg-yellow-500' : 'bg-green-500'}`}
              style={{ width: `${Math.min(metrics.errorBudget.budgetConsumed, 100)}%` }}
            />
          </div>
          <div className="grid grid-cols-3 gap-2 text-xs">
            <MetricTile label="Budget Used" value={`${metrics.errorBudget.budgetConsumed.toFixed(1)}%`} alert={metrics.errorBudget.budgetConsumed > 80} />
            <MetricTile label="Burn Rate (1h)" value={`${metrics.errorBudget.burnRate1h.toFixed(1)}x`} alert={metrics.errorBudget.burnRate1h > 6} />
            <MetricTile label="Time to Exhaust" value={metrics.errorBudget.timeToExhaustion > 9999 ? '∞' : `${metrics.errorBudget.timeToExhaustion.toFixed(0)}m`} />
          </div>
        </div>
      )}

      {/* Blast Radius */}
      {metrics.blastRadius && (
        <div className="rounded bg-black/40 border border-white/5 p-3 space-y-2">
          <span className="text-[10px] text-white/40 uppercase tracking-wider">Blast Radius Analysis</span>
          <div className="grid grid-cols-3 gap-2 text-xs">
            <MetricTile label="Impact %" value={`${metrics.blastRadius.impactPercentage.toFixed(0)}%`} alert={metrics.blastRadius.impactPercentage > 60} />
            <MetricTile label="User Impact" value={`${metrics.blastRadius.userImpactEstimate.toFixed(1)}%`} alert={metrics.blastRadius.userImpactEstimate > 50} />
            <MetricTile label="Vulnerable Node" value={metrics.blastRadius.mostVulnerableNode?.split('-')[0] ?? '—'} />
          </div>
          {metrics.blastRadius.impactedNodes > 0 && (
            <p className="text-[10px] text-white/50">
              {metrics.blastRadius.impactedNodes}/{metrics.blastRadius.totalNodes} nodes at risk • Highest vulnerability: {(metrics.blastRadius.mostVulnerableScore * 100).toFixed(0)}%
            </p>
          )}
        </div>
      )}

      {/* Cascade Chains */}
      {metrics.cascadeChain && metrics.cascadeChain.length > 0 && (
        <div className="rounded bg-black/40 border border-white/5 p-3 space-y-2">
          <span className="text-[10px] text-white/40 uppercase tracking-wider">Cascading Failures Detected</span>
          <div className="space-y-1 max-h-24 overflow-y-auto">
            {metrics.cascadeChain.slice(0, 5).map((c, i) => (
              <div key={i} className="flex items-center gap-1 text-[10px] font-mono">
                <span className="text-red-400">{c.sourceNode.split('-')[0]}</span>
                <span className="text-white/30">→</span>
                <span className="text-orange-400">{c.affectedNode.split('-')[0]}</span>
                <span className="text-white/20 ml-auto">{c.propagationMs}ms • {c.mechanism}</span>
                {c.depth > 1 && <span className="text-yellow-500/60">depth:{c.depth}</span>}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Critical Path */}
      {metrics.criticalPath && metrics.criticalPath.length > 0 && (
        <div className="rounded bg-black/40 border border-white/5 p-3 space-y-2">
          <span className="text-[10px] text-white/40 uppercase tracking-wider">Critical Path Analysis</span>
          <div className="space-y-1">
            {metrics.criticalPath.map((node, i) => (
              <div key={i} className="flex items-center gap-2 text-[10px]">
                <span className="font-mono text-white/70 w-20 truncate">{node.nodeId.split('-')[0]}</span>
                <div className="flex-1 h-2 bg-white/5 rounded-full overflow-hidden">
                  <div
                    className={`h-full rounded-full ${node.bottleneck ? 'bg-red-500' : 'bg-blue-500/60'}`}
                    style={{ width: `${node.contribPercentage}%` }}
                  />
                </div>
                <span className="text-white/40 w-10 text-right">{node.contribPercentage.toFixed(0)}%</span>
                {node.bottleneck && <span className="text-red-400 text-[8px]">BOTTLENECK</span>}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Steady-State Deviation */}
      {(metrics.steadyStateDeviation ?? 0) > 5 && (
        <div className="rounded bg-yellow-500/5 border border-yellow-500/20 p-2 text-xs text-yellow-300/80">
          <span className="text-[10px] text-yellow-500/60 uppercase tracking-wider">Steady-State Alert</span>
          <p className="mt-0.5">System deviated {metrics.steadyStateDeviation?.toFixed(1)}% from baseline under sustained load</p>
        </div>
      )}
    </article>
  );
}

function formatDuration(ms: number): string {
  if (!ms || ms <= 0) return '—';
  if (ms < 1000) return `${ms.toFixed(0)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

function formatCost(cost: number): string {
  if (!cost || cost <= 0) return '—';
  if (cost < 0.000001) return `$${(cost * 1e9).toFixed(1)}n`;
  if (cost < 0.001) return `$${(cost * 1e6).toFixed(1)}µ`;
  return `$${cost.toFixed(6)}`;
}
