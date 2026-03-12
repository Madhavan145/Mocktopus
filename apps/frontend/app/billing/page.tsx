'use client';

import { useEffect, useMemo, useState } from 'react';
import { TrendingUp, DollarSign, BarChart3, ArrowUpRight, ArrowDownRight } from 'lucide-react';

import { AppShell } from '@/components/layout/app-shell';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { readSimulationHistory } from '@/components/simulation/storage';
import type { SimulationRunRecord } from '@/components/simulation/types';

export default function BillingPage() {
  const [history, setHistory] = useState<SimulationRunRecord[]>([]);

  useEffect(() => {
    setHistory(readSimulationHistory('demo'));
  }, []);

  const rows = useMemo(() => {
    return history
      .filter((r) => r.cost)
      .slice(0, 12)
      .reverse()
      .map((r, i) => ({ label: `Run ${i + 1}`, total: r.cost?.total ?? 0 }));
  }, [history]);

  const average = useMemo(() => rows.length === 0 ? 0 : rows.reduce((s, r) => s + r.total, 0) / rows.length, [rows]);
  const forecast = useMemo(() => rows.length === 0 ? 0 : average * 1.12, [average, rows.length]);
  const maxCost = useMemo(() => Math.max(...rows.map((r) => r.total), 1), [rows]);

  return (
    <AppShell>
      <div className="space-y-6 animate-fade-in">
        <div>
          <h1 className="text-title-lg">Billing & Cost</h1>
          <p className="text-body-sm text-ink-secondary mt-1">
            Infrastructure cost tracking, growth analysis, and optimization insights.
          </p>
        </div>

        {/* Summary Cards */}
        <div className="grid gap-4 md:grid-cols-3">
          <div className="metric-card">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 text-ink-muted">
                <DollarSign size={14} />
                <span className="text-caption">Avg Monthly Cost</span>
              </div>
              <Badge variant="success">
                <ArrowDownRight size={10} className="mr-0.5" /> stable
              </Badge>
            </div>
            <p className="mt-3 text-display text-2xl font-semibold font-mono">${average.toFixed(2)}</p>
          </div>
          <div className="metric-card">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 text-ink-muted">
                <TrendingUp size={14} />
                <span className="text-caption">Projected Next</span>
              </div>
              <Badge variant="warning">
                <ArrowUpRight size={10} className="mr-0.5" /> +12%
              </Badge>
            </div>
            <p className="mt-3 text-display text-2xl font-semibold font-mono">${forecast.toFixed(2)}</p>
          </div>
          <div className="metric-card">
            <div className="flex items-center gap-2 text-ink-muted">
              <BarChart3 size={14} />
              <span className="text-caption">Runs Analyzed</span>
            </div>
            <p className="mt-3 text-display text-2xl font-semibold font-mono">{rows.length}</p>
          </div>
        </div>

        {/* Cost Growth Chart */}
        <Card>
          <CardHeader>
            <CardTitle>Cost Growth Curve</CardTitle>
            <CardDescription>Cost per simulation run over time.</CardDescription>
          </CardHeader>
          <CardContent>
            {rows.length === 0 ? (
              <div className="flex flex-col items-center py-12 text-center">
                <BarChart3 size={28} className="text-ink-muted mb-2" />
                <p className="text-body-sm text-ink-secondary">No cost data yet.</p>
                <p className="text-caption text-ink-muted">Run a simulation with cost estimation enabled.</p>
              </div>
            ) : (
              <div className="space-y-3">
                {rows.map((row) => {
                  const pct = Math.max(4, (row.total / maxCost) * 100);
                  const isHigh = row.total > average * 1.2;
                  return (
                    <div key={row.label} className="group">
                      <div className="flex items-center justify-between mb-1">
                        <span className="text-caption text-ink-muted">{row.label}</span>
                        <span className="text-caption font-mono font-medium">${row.total.toFixed(2)}</span>
                      </div>
                      <div className="h-2 rounded-full bg-surface-overlay overflow-hidden">
                        <div
                          className={`h-full rounded-full transition-all duration-500 ${isHigh ? 'bg-warning' : 'bg-accent'}`}
                          style={{ width: `${pct}%` }}
                        />
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Tradeoff Insight */}
        <Card>
          <CardHeader>
            <CardTitle>Tradeoff Insights</CardTitle>
            <CardDescription>How architectural changes affect cost vs performance.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-3 md:grid-cols-2">
              <div className="rounded-lg border bg-surface-elevated p-4">
                <div className="flex items-center gap-2 mb-2">
                  <Badge variant="success">+ Redis Cache</Badge>
                </div>
                <div className="space-y-1 text-caption text-ink-secondary">
                  <p>Cost <span className="text-warning font-mono">+$200/mo</span></p>
                  <p>Latency <span className="text-success font-mono">-35%</span></p>
                  <p>DB Load <span className="text-success font-mono">-40%</span></p>
                </div>
              </div>
              <div className="rounded-lg border bg-surface-elevated p-4">
                <div className="flex items-center gap-2 mb-2">
                  <Badge variant="info">+ Message Queue</Badge>
                </div>
                <div className="space-y-1 text-caption text-ink-secondary">
                  <p>Cost <span className="text-warning font-mono">+$150/mo</span></p>
                  <p>Throughput <span className="text-success font-mono">+60%</span></p>
                  <p>Failure Rate <span className="text-success font-mono">-50%</span></p>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </AppShell>
  );
}
