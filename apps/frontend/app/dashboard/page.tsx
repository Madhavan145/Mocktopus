'use client';

import Link from 'next/link';
import { useEffect, useMemo, useRef, useState } from 'react';
import {
  Activity,
  Clock,
  Gauge,
  AlertTriangle,
  DollarSign,
  Plus,
  Layers,
  Play,
  ArrowUpRight,
  Zap,
  FolderOpen,
  Pencil,
  Trash2,
} from 'lucide-react';

import { AppShell } from '@/components/layout/app-shell';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
import { EventMixChart, TrendChart } from '@/components/simulation/MetricCharts';
import {
  readActiveSimulationRunId,
  readLastSimulationRun,
  readLiveSimulationSnapshot,
  subscribeActiveSimulationRunId,
  subscribeLiveSimulationSnapshot,
} from '@/components/simulation/storage';
import type { LiveMetricsSnapshot, RunStateResponse, SimulationRunRecord } from '@/components/simulation/types';
import { apiFetch, parseV1 } from '@/lib/api';

const GATEWAY_URL = process.env.NEXT_PUBLIC_GATEWAY_URL ?? 'http://localhost:8080';

export default function DashboardPage() {
  const [record, setRecord] = useState<SimulationRunRecord | null>(null);
  const [liveSnapshot, setLiveSnapshot] = useState<LiveMetricsSnapshot | null>(null);
  const [activeRunId, setActiveRunId] = useState<string | null>(null);
  const [projects, setProjects] = useState<Array<{ id: string; name: string; description: string }>>([]);
  const [projectName, setProjectName] = useState('');
  const [projectDescription, setProjectDescription] = useState('');
  const [projectError, setProjectError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [editingProject, setEditingProject] = useState<{ id: string; name: string; description: string } | null>(null);
  const [editName, setEditName] = useState('');
  const [editDescription, setEditDescription] = useState('');
  const [deletingProject, setDeletingProject] = useState<{ id: string; name: string } | null>(null);
  const createFormRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (showCreate && createFormRef.current) {
      createFormRef.current.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
  }, [showCreate]);

  useEffect(() => {
    setRecord(readLastSimulationRun('demo'));
    setLiveSnapshot(readLiveSimulationSnapshot('demo'));
    setActiveRunId(readActiveSimulationRunId('demo'));

    const unsubscribe = subscribeLiveSimulationSnapshot('demo', (snapshot) => {
      setLiveSnapshot(snapshot);
      if (!snapshot) setRecord(readLastSimulationRun('demo'));
    });

    const unsubscribeRunId = subscribeActiveSimulationRunId('demo', (runId) => {
      setActiveRunId(runId);
      if (!runId) setRecord(readLastSimulationRun('demo'));
    });

    return () => { unsubscribe(); unsubscribeRunId(); };
  }, []);

  useEffect(() => {
    const loadProjects = async () => {
      try {
        const response = await apiFetch(`/api/v1/projects?limit=50`);
        if (!response.ok) throw new Error('unable to load projects');
        const payload = await parseV1<{ projects: Array<{ id: string; name: string; description: string }> }>(response);
        setProjects(payload.projects);
      } catch (err) {
        setProjectError(err instanceof Error ? err.message : 'unable to load projects');
      }
    };
    void loadProjects();
  }, []);

  useEffect(() => {
    if (!activeRunId) return;
    const intervalId = window.setInterval(async () => {
      const response = await apiFetch(`/api/v1/simulation/runs/${encodeURIComponent(activeRunId)}`);
      if (!response.ok) return;
      const state = await parseV1<RunStateResponse>(response);
      setLiveSnapshot(state.snapshot ?? null);
      if (state.status !== 'running') setRecord(readLastSimulationRun('demo'));
    }, 500);
    return () => window.clearInterval(intervalId);
  }, [activeRunId]);

  const metrics = useMemo(() => {
    if (!record) return null;
    return {
      avgLatency: record.run.metrics.avgLatencyMs,
      p95Latency: record.run.metrics.p95LatencyMs,
      rps: record.run.metrics.rps,
      errorRate: record.run.metrics.errorRate,
      bottleneck: record.derived?.bottleneckScore ?? 0,
      cost: record.cost?.total ?? 0,
    };
  }, [record]);

  const activeProjectId = projects[0]?.id ?? 'demo';

  const createProject = async () => {
    if (projectName.trim() === '') { setProjectError('Project name is required'); return; }
    setProjectError(null);
    try {
      const response = await apiFetch(`/api/v1/projects`, {
        method: 'POST',
        body: JSON.stringify({ name: projectName.trim(), description: projectDescription.trim() }),
      });
      if (!response.ok) {
        const json = await response.json().catch(() => null);
        throw new Error(json?.error?.message || 'unable to create project');
      }
      const created = await parseV1<{ id: string; name: string; description: string }>(response);
      setProjects((prev) => [created, ...prev]);
      setProjectName('');
      setProjectDescription('');
      setShowCreate(false);
    } catch (err) {
      setProjectError(err instanceof Error ? err.message : 'unable to create project');
    }
  };

  const startEdit = (project: { id: string; name: string; description: string }) => {
    setEditingProject(project);
    setEditName(project.name);
    setEditDescription(project.description || '');
    setProjectError(null);
  };

  const saveEdit = async () => {
    if (!editingProject) return;
    if (editName.trim() === '') { setProjectError('Project name is required'); return; }
    setProjectError(null);
    try {
      const response = await apiFetch(`/api/v1/projects/${editingProject.id}`, {
        method: 'PUT',
        body: JSON.stringify({ name: editName.trim(), description: editDescription.trim() }),
      });
      if (!response.ok) throw new Error('unable to update project');
      const updated = await parseV1<{ id: string; name: string; description: string }>(response);
      setProjects((prev) => prev.map((p) => (p.id === updated.id ? updated : p)));
      setEditingProject(null);
    } catch (err) {
      setProjectError(err instanceof Error ? err.message : 'unable to update project');
    }
  };

  const confirmDelete = async () => {
    if (!deletingProject) return;
    try {
      const response = await apiFetch(`/api/v1/projects/${deletingProject.id}`, { method: 'DELETE' });
      if (!response.ok) throw new Error('unable to delete project');
      setProjects((prev) => prev.filter((p) => p.id !== deletingProject.id));
      setDeletingProject(null);
    } catch (err) {
      setProjectError(err instanceof Error ? err.message : 'unable to delete project');
      setDeletingProject(null);
    }
  };

  return (
    <AppShell projectId={activeProjectId}>
      <div className="space-y-8 max-w-7xl animate-fade-in">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-title-lg">Dashboard</h1>
            <p className="text-body-sm text-ink-secondary mt-1">
              Infrastructure simulation analytics and project overview.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="secondary" size="sm" asChild>
              <Link href={`/project/${activeProjectId}/simulate`}>
                <Play size={14} /> Run Simulation
              </Link>
            </Button>
            <Button size="sm" onClick={() => setShowCreate(true)}>
              <Plus size={14} /> New Project
            </Button>
          </div>
        </div>

        {/* Live Run Banner */}
        {liveSnapshot && (
          <div className="rounded-lg border border-accent/30 bg-accent-muted p-4 animate-slide-up">
            <div className="flex items-center gap-3 mb-3">
              <div className="h-2 w-2 rounded-full bg-accent animate-pulse-subtle" />
              <span className="text-body-sm font-medium text-accent">Simulation Running</span>
              <Badge variant="default">{Math.round(liveSnapshot.progress * 100)}%</Badge>
            </div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div>
                <p className="text-caption text-ink-muted">RPS</p>
                <p className="text-title font-mono">{liveSnapshot.metrics.rps}</p>
              </div>
              <div>
                <p className="text-caption text-ink-muted">Latency</p>
                <p className="text-title font-mono">{liveSnapshot.metrics.avgLatencyMs}ms</p>
              </div>
              <div>
                <p className="text-caption text-ink-muted">Progress</p>
                <div className="mt-1.5 h-2 rounded-full bg-surface-overlay overflow-hidden">
                  <div
                    className="h-full rounded-full bg-accent transition-all duration-300"
                    style={{ width: `${liveSnapshot.progress * 100}%` }}
                  />
                </div>
              </div>
              <div>
                <p className="text-caption text-ink-muted">Current Event</p>
                <p className="text-body-sm font-mono truncate">{liveSnapshot.currentEvent}</p>
              </div>
            </div>
          </div>
        )}

        {/* Metrics Grid */}
        {metrics && (
          <div className="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-6 gap-3">
            <div className="metric-card">
              <div className="flex items-center gap-2 text-ink-muted mb-2">
                <Clock size={14} />
                <span className="text-caption">Avg Latency</span>
              </div>
              <p className="text-title font-mono">{metrics.avgLatency}<span className="text-caption text-ink-muted ml-1">ms</span></p>
            </div>
            <div className="metric-card">
              <div className="flex items-center gap-2 text-ink-muted mb-2">
                <Gauge size={14} />
                <span className="text-caption">P95 Latency</span>
              </div>
              <p className="text-title font-mono">{metrics.p95Latency}<span className="text-caption text-ink-muted ml-1">ms</span></p>
            </div>
            <div className="metric-card">
              <div className="flex items-center gap-2 text-ink-muted mb-2">
                <Activity size={14} />
                <span className="text-caption">RPS</span>
              </div>
              <p className="text-title font-mono">{metrics.rps}</p>
            </div>
            <div className="metric-card">
              <div className="flex items-center gap-2 text-ink-muted mb-2">
                <AlertTriangle size={14} />
                <span className="text-caption">Error Rate</span>
              </div>
              <p className="text-title font-mono">{(metrics.errorRate * 100).toFixed(1)}<span className="text-caption text-ink-muted ml-1">%</span></p>
            </div>
            <div className="metric-card">
              <div className="flex items-center gap-2 text-ink-muted mb-2">
                <Zap size={14} />
                <span className="text-caption">Bottleneck</span>
              </div>
              <p className="text-title font-mono">{metrics.bottleneck || '-'}</p>
            </div>
            <div className="metric-card">
              <div className="flex items-center gap-2 text-ink-muted mb-2">
                <DollarSign size={14} />
                <span className="text-caption">Est. Cost</span>
              </div>
              <p className="text-title font-mono">${metrics.cost || '-'}<span className="text-caption text-ink-muted ml-1">/mo</span></p>
            </div>
          </div>
        )}

        {/* Charts */}
        {record?.derived?.series?.length ? (
          <div className="grid gap-4 md:grid-cols-3">
            <Card>
              <CardContent className="mt-0 pt-4">
                <TrendChart
                  title="Latency"
                  colorClassName="text-rose-400"
                  series={record.derived.series}
                  accessor={(p) => p.latencyMs}
                  formatter={(v) => `${v.toFixed(1)} ms`}
                />
              </CardContent>
            </Card>
            <Card>
              <CardContent className="mt-0 pt-4">
                <TrendChart
                  title="Throughput"
                  colorClassName="text-emerald-400"
                  series={record.derived.series}
                  accessor={(p) => p.rps}
                  formatter={(v) => `${v.toFixed(0)} rps`}
                />
              </CardContent>
            </Card>
            <Card>
              <CardContent className="mt-0 pt-4">
                <TrendChart
                  title="Error Rate"
                  colorClassName="text-amber-400"
                  series={record.derived.series}
                  accessor={(p) => p.errorRate * 100}
                  formatter={(v) => `${v.toFixed(1)}%`}
                />
              </CardContent>
            </Card>
          </div>
        ) : null}

        {record ? <EventMixChart events={record.run.events} /> : null}

        {/* Projects */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>Projects</CardTitle>
                <CardDescription>Your architecture simulation projects.</CardDescription>
              </div>
              <Button variant="secondary" size="sm" onClick={() => setShowCreate(!showCreate)}>
                <Plus size={14} /> New
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            {showCreate && (
              <div ref={createFormRef} className="mb-4 rounded-lg border bg-surface-elevated p-4 space-y-3 animate-slide-down">
                <div className="grid gap-3 md:grid-cols-2">
                  <Input placeholder="Project name" value={projectName} onChange={(e) => setProjectName(e.target.value)} />
                  <Input placeholder="Description (optional)" value={projectDescription} onChange={(e) => setProjectDescription(e.target.value)} />
                </div>
                <div className="flex items-center gap-2">
                  <Button size="sm" onClick={createProject}>Create</Button>
                  <Button variant="ghost" size="sm" onClick={() => setShowCreate(false)}>Cancel</Button>
                </div>
                {projectError && <p className="text-caption text-error">{projectError}</p>}
              </div>
            )}

            {projects.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-12 text-center">
                <FolderOpen size={32} className="text-ink-muted mb-3" />
                <p className="text-body-sm text-ink-secondary">No projects yet.</p>
                <p className="text-caption text-ink-muted">Create your first project to get started.</p>
              </div>
            ) : (
              <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                {projects.map((project) => (
                  <div
                    key={project.id}
                    className="group relative rounded-lg border bg-surface-elevated p-4 hover:border-ink-muted transition-all duration-200"
                  >
                    <Link href={`/project/${project.id}`} className="block">
                      <div className="flex items-start justify-between">
                        <div className="flex items-center gap-2">
                          <Layers size={16} className="text-accent" />
                          <span className="text-body-sm font-medium">{project.name}</span>
                        </div>
                        <ArrowUpRight size={14} className="text-ink-muted opacity-0 group-hover:opacity-100 transition-opacity" />
                      </div>
                      <p className="mt-2 text-caption text-ink-muted line-clamp-2">
                        {project.description || 'No description'}
                      </p>
                    </Link>
                    <div className="mt-3 flex items-center justify-between">
                      <div className="flex gap-2">
                        <Badge variant="secondary">Builder</Badge>
                        <Badge variant="secondary">Simulate</Badge>
                      </div>
                      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                        <button
                          onClick={(e) => { e.preventDefault(); startEdit(project); }}
                          className="p-1 rounded hover:bg-surface-overlay text-ink-muted hover:text-ink transition-colors"
                          title="Edit project"
                        >
                          <Pencil size={13} />
                        </button>
                        <button
                          onClick={(e) => { e.preventDefault(); setDeletingProject(project); }}
                          className="p-1 rounded hover:bg-red-500/10 text-ink-muted hover:text-red-400 transition-colors"
                          title="Delete project"
                        >
                          <Trash2 size={13} />
                        </button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Edit Project Modal */}
        {editingProject && (
          <div className="fixed inset-0 z-50 flex items-center justify-center">
            <div className="absolute inset-0 bg-black/50 animate-fade-in" onClick={() => setEditingProject(null)} />
            <div className="relative z-10 w-full max-w-md rounded-lg border bg-surface-elevated p-6 shadow-xl animate-slide-up space-y-4">
              <h2 className="text-body font-semibold text-ink">Edit Project</h2>
              <Input placeholder="Project name" value={editName} onChange={(e) => setEditName(e.target.value)} />
              <Input placeholder="Description (optional)" value={editDescription} onChange={(e) => setEditDescription(e.target.value)} />
              {projectError && <p className="text-caption text-error">{projectError}</p>}
              <div className="flex items-center justify-end gap-2">
                <Button variant="ghost" size="sm" onClick={() => setEditingProject(null)}>Cancel</Button>
                <Button size="sm" onClick={saveEdit}>Save</Button>
              </div>
            </div>
          </div>
        )}

        {/* Delete Project Confirmation */}
        <ConfirmDialog
          open={!!deletingProject}
          onConfirm={confirmDelete}
          onCancel={() => setDeletingProject(null)}
          title="Delete project?"
          description={`Are you sure you want to delete "${deletingProject?.name}"? This action cannot be undone.`}
          confirmLabel="Delete"
          cancelLabel="Cancel"
          variant="destructive"
        />

        {!record && !liveSnapshot && (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <div className="rounded-full bg-surface-elevated p-4 mb-4">
              <Activity size={24} className="text-ink-muted" />
            </div>
            <p className="text-body text-ink-secondary">No simulation data yet</p>
            <p className="text-caption text-ink-muted mt-1">Run your first simulation to see analytics here.</p>
            <Button variant="secondary" size="sm" className="mt-4" asChild>
              <Link href={`/project/${activeProjectId}/simulate`}>
                <Play size={14} /> Start Simulation
              </Link>
            </Button>
          </div>
        )}
      </div>
    </AppShell>
  );
}


