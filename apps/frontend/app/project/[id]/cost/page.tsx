'use client';

import { useState, useEffect, useCallback } from 'react';
import InsightsView from '@/components/simulation/InsightsView';
import { AppShell } from '@/components/layout/app-shell';
import { apiFetch } from '@/lib/api';
import { DollarSign, TrendingDown, BarChart3 } from 'lucide-react';

type PageProps = {
  params: { id: string };
};

interface CostBreakdown {
  compute: number;
  database: number;
  cache: number;
  queue: number;
  network: number;
  storage: number;
  other: number;
}

interface ProviderComparison {
  provider: string;
  total: number;
  savingsPercent: number;
}

interface CostResponse {
  monthlyTotal: number;
  annualEstimate: number;
  breakdown: CostBreakdown;
  comparison: ProviderComparison[];
  lineItems: any[];
}

export default function CostPage({ params }: PageProps) {
  const [costData, setCostData] = useState<CostResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [provider, setProvider] = useState('aws');
  const [topologyVersion, setTopologyVersion] = useState(0);

  // Listen for localStorage changes (e.g. when builder saves topology)
  useEffect(() => {
    const handleStorage = (e: StorageEvent) => {
      if (e.key === `moktopus:project:${params.id}:builder`) {
        setTopologyVersion((v) => v + 1);
      }
    };
    window.addEventListener('storage', handleStorage);
    return () => window.removeEventListener('storage', handleStorage);
  }, [params.id]);

  // Also poll for changes periodically (same-tab localStorage writes don't fire StorageEvent)
  useEffect(() => {
    const key = `moktopus:project:${params.id}:builder`;
    let lastRaw = localStorage.getItem(key);
    const interval = setInterval(() => {
      const current = localStorage.getItem(key);
      if (current !== lastRaw) {
        lastRaw = current;
        setTopologyVersion((v) => v + 1);
      }
    }, 2000);
    return () => clearInterval(interval);
  }, [params.id]);

  const fetchCostEstimate = useCallback(async () => {
    setLoading(true);
    try {
      // First try reading topology from localStorage (most up-to-date)
      const localRaw = localStorage.getItem(`moktopus:project:${params.id}:builder`);
      let ctm: any = null;

      if (localRaw) {
        try {
          const parsed = JSON.parse(localRaw);
          // Convert ReactFlow nodes to CTM format for cost service
          const nodes = (parsed.nodes ?? []).map((n: any) => ({
            id: n.id,
            type: n.data?.type ?? 'service',
            config: { ...(n.data?.config ?? {}), label: n.data?.label ?? n.id },
          }));
          const edges = (parsed.edges ?? []).map((e: any) => ({
            source: e.source,
            target: e.target,
          }));
          ctm = { nodes, edges };
        } catch { /* fall through to API */ }
      }

      if (!ctm) {
        // Fallback to topology service API
        const topologyRes = await apiFetch(`/api/v1/topology/projects/${encodeURIComponent(params.id)}`);
        if (!topologyRes.ok) return;
        const topologyData = await topologyRes.json();
        ctm = topologyData.data?.ctm || topologyData.ctm;
      }

      if (!ctm) return;

      const res = await apiFetch(`/api/v1/cost/estimate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          projectId: params.id,
          topology: ctm,
          provider,
          region: 'us-east-1',
        }),
      });
      if (res.ok) {
        const json = await res.json();
        if (json.success) setCostData(json.data);
      }
    } catch (e) {
      console.error('Failed to fetch cost estimate:', e);
    } finally {
      setLoading(false);
    }
  }, [params.id, provider]);

  useEffect(() => {
    fetchCostEstimate();
  }, [fetchCostEstimate, topologyVersion]);

  const breakdownCategories = costData?.breakdown
    ? Object.entries(costData.breakdown)
        .filter(([, v]) => v > 0)
        .sort(([, a], [, b]) => b - a)
    : [];

  const maxBreakdownValue = breakdownCategories.length > 0
    ? Math.max(...breakdownCategories.map(([, v]) => v))
    : 1;

  return (
    <AppShell projectId={params.id}>
      <div className="space-y-6 animate-fade-in">
        <div>
          <div className="flex items-center gap-2 text-caption text-ink-muted mb-1">
            <span>Project</span><span>/</span><span>{params.id}</span><span>/</span><span className="text-ink-secondary">Cost</span>
          </div>
          <h1 className="text-title-lg">Cost Analysis</h1>
          <p className="text-body-sm text-ink-secondary mt-1">
            Infrastructure cost breakdown with multi-cloud comparison and instance-level pricing.
          </p>
        </div>

        {/* Provider Selector */}
        <div className="flex items-center gap-2">
          {['aws', 'gcp', 'azure'].map((p) => (
            <button
              key={p}
              onClick={() => setProvider(p)}
              className={`rounded-lg px-4 py-2 text-sm border transition-colors ${
                provider === p
                  ? 'bg-accent text-white border-accent'
                  : 'text-ink-muted border-border hover:border-accent/50'
              }`}
            >
              {p.toUpperCase()}
            </button>
          ))}
        </div>

        {/* Summary Cards */}
        {costData && (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="rounded-xl border bg-surface p-5">
              <div className="flex items-center gap-2 text-ink-muted mb-2">
                <DollarSign size={16} />
                <span className="text-xs font-medium">Monthly Estimate</span>
              </div>
              <p className="text-3xl font-bold text-accent">${costData.monthlyTotal?.toFixed(2)}</p>
              <p className="text-xs text-ink-muted mt-1">{provider.toUpperCase()} • us-east-1</p>
            </div>
            <div className="rounded-xl border bg-surface p-5">
              <div className="flex items-center gap-2 text-ink-muted mb-2">
                <BarChart3 size={16} />
                <span className="text-xs font-medium">Annual Projection</span>
              </div>
              <p className="text-3xl font-bold text-ink">${costData.annualEstimate?.toFixed(0)}</p>
              <p className="text-xs text-ink-muted mt-1">Based on current topology</p>
            </div>
            <div className="rounded-xl border bg-surface p-5">
              <div className="flex items-center gap-2 text-ink-muted mb-2">
                <TrendingDown size={16} />
                <span className="text-xs font-medium">Potential Savings</span>
              </div>
              <p className="text-3xl font-bold text-green-400">
                {costData.comparison?.length > 1
                  ? `${Math.max(...costData.comparison.filter(c => c.provider !== provider).map(c => c.savingsPercent))}%`
                  : '—'}
              </p>
              <p className="text-xs text-ink-muted mt-1">Best alternative provider</p>
            </div>
          </div>
        )}

        {/* Multi-Cloud Comparison */}
        {costData?.comparison && costData.comparison.length > 0 && (
          <div className="rounded-xl border bg-surface p-5">
            <h3 className="text-sm font-medium text-ink mb-4">Multi-Cloud Cost Comparison</h3>
            <div className="space-y-3">
              {costData.comparison.map((comp) => (
                <div key={comp.provider} className="flex items-center gap-4">
                  <span className="w-16 text-xs font-medium text-ink">{comp.provider.toUpperCase()}</span>
                  <div className="flex-1 h-8 rounded-lg bg-surface-elevated overflow-hidden relative">
                    <div
                      className={`h-full rounded-lg transition-all ${
                        comp.provider === provider ? 'bg-accent' : 'bg-accent/40'
                      }`}
                      style={{
                        width: `${(comp.total / Math.max(...costData.comparison.map(c => c.total))) * 100}%`,
                      }}
                    />
                    <span className="absolute inset-0 flex items-center px-3 text-xs font-medium text-ink">
                      ${comp.total.toFixed(2)}/mo
                      {comp.savingsPercent > 0 && (
                        <span className="ml-2 text-green-400">({comp.savingsPercent}% savings)</span>
                      )}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Cost Breakdown */}
        {breakdownCategories.length > 0 && (
          <div className="rounded-xl border bg-surface p-5">
            <h3 className="text-sm font-medium text-ink mb-4">Cost Breakdown by Category</h3>
            <div className="space-y-3">
              {breakdownCategories.map(([category, value]) => (
                <div key={category} className="flex items-center gap-4">
                  <span className="w-20 text-xs text-ink-muted capitalize">{category}</span>
                  <div className="flex-1 h-6 rounded bg-surface-elevated overflow-hidden">
                    <div
                      className="h-full rounded bg-accent/60"
                      style={{ width: `${(value / maxBreakdownValue) * 100}%` }}
                    />
                  </div>
                  <span className="w-20 text-right text-xs font-medium text-ink">${value.toFixed(2)}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Line Items */}
        {costData?.lineItems && costData.lineItems.length > 0 && (
          <div className="rounded-xl border bg-surface overflow-hidden">
            <div className="px-5 py-3 border-b bg-surface-elevated/50">
              <h3 className="text-sm font-medium text-ink">Instance-Level Breakdown</h3>
            </div>
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b">
                  <th className="px-5 py-2 text-left text-xs font-medium text-ink-muted">Node</th>
                  <th className="px-5 py-2 text-left text-xs font-medium text-ink-muted">Instance</th>
                  <th className="px-5 py-2 text-left text-xs font-medium text-ink-muted">Tier</th>
                  <th className="px-5 py-2 text-right text-xs font-medium text-ink-muted">$/month</th>
                </tr>
              </thead>
              <tbody>
                {costData.lineItems.map((item: any, i: number) => (
                  <tr key={i} className="border-b border-border/50">
                    <td className="px-5 py-2 text-ink">{item.label || item.nodeId}</td>
                    <td className="px-5 py-2 font-mono text-xs text-ink-muted">{item.instanceType || '—'}</td>
                    <td className="px-5 py-2 text-xs text-ink-muted">{item.tier || '—'}</td>
                    <td className="px-5 py-2 text-right font-medium text-accent">${item.monthlyCost?.toFixed(2)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Legacy insights view fallback */}
        {!costData && !loading && (
          <InsightsView projectId={params.id} mode="cost" />
        )}

        {loading && (
          <div className="flex items-center justify-center h-32">
            <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
          </div>
        )}
      </div>
    </AppShell>
  );
}
