'use client';

import { type ReactNode, useEffect, useMemo, useState } from 'react';
import { Save, Check } from 'lucide-react';

import { AppShell } from '@/components/layout/app-shell';
import { Button } from '@/components/ui/button';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';

type WorkspaceSettings = {
  defaultProjectId: string;
  defaultDurationSeconds: number;
  defaultRequestsPerSecond: number;
  chaosModeEnabled: boolean;
  preferDeterministicRuns: boolean;
  region: string;
};

const STORAGE_KEY = 'moktopus.settings.v1';

const defaults: WorkspaceSettings = {
  defaultProjectId: 'demo',
  defaultDurationSeconds: 60,
  defaultRequestsPerSecond: 25,
  chaosModeEnabled: false,
  preferDeterministicRuns: true,
  region: 'local',
};

export default function SettingsPage() {
  const [settings, setSettings] = useState<WorkspaceSettings>(defaults);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return;
    try {
      const parsed = JSON.parse(raw) as Partial<WorkspaceSettings>;
      setSettings({ ...defaults, ...parsed });
    } catch {
      setSettings(defaults);
    }
  }, []);

  const save = () => {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
  };

  return (
    <AppShell>
      <div className="space-y-6 max-w-5xl animate-fade-in">
        <div>
          <h1 className="text-title-lg">Settings</h1>
          <p className="text-body-sm text-ink-secondary mt-1">
            Configure workspace defaults for simulations and infrastructure planning.
          </p>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Simulation Defaults</CardTitle>
            <CardDescription>These values are applied when creating new simulation sessions.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-2">
              <Input
                label="Default Project ID"
                value={settings.defaultProjectId}
                onChange={(e) => setSettings((s) => ({ ...s, defaultProjectId: e.target.value }))}
              />
              <div className="space-y-1.5">
                <label className="text-caption font-medium text-ink-secondary">Region</label>
                <select
                  className="flex h-9 w-full rounded-md border bg-surface px-3 py-2 text-body-sm text-ink transition-colors hover:border-ink-muted focus:border-accent focus:ring-1 focus:ring-accent focus:outline-none"
                  value={settings.region}
                  onChange={(e) => setSettings((s) => ({ ...s, region: e.target.value }))}
                >
                  <option value="local">Local</option>
                  <option value="us-east">US East</option>
                  <option value="eu-west">EU West</option>
                  <option value="ap-south">AP South</option>
                </select>
              </div>
              <Input
                label="Duration (seconds)"
                type="number"
                min={5}
                step={1}
                value={settings.defaultDurationSeconds}
                onKeyDown={(e: React.KeyboardEvent) => { if (e.key === '.' || e.key === ',') e.preventDefault(); }}
                onChange={(e) => setSettings((s) => ({ ...s, defaultDurationSeconds: Math.max(5, Math.round(Number(e.target.value))) }))}
              />
              <Input
                label="Requests/sec"
                type="number"
                min={1}
                step={1}
                value={settings.defaultRequestsPerSecond}
                onKeyDown={(e: React.KeyboardEvent) => { if (e.key === '.' || e.key === ',') e.preventDefault(); }}
                onChange={(e) => setSettings((s) => ({ ...s, defaultRequestsPerSecond: Math.max(1, Math.round(Number(e.target.value))) }))}
              />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Behavior</CardTitle>
            <CardDescription>Control chaos injection and run reproducibility.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              <Toggle
                label="Enable chaos mode by default"
                description="Inject failures into every simulation run automatically."
                checked={settings.chaosModeEnabled}
                onChange={(v) => setSettings((s) => ({ ...s, chaosModeEnabled: v }))}
              />
              <Toggle
                label="Prefer deterministic runs"
                description="Use fixed seeds for reproducible simulation results."
                checked={settings.preferDeterministicRuns}
                onChange={(v) => setSettings((s) => ({ ...s, preferDeterministicRuns: v }))}
              />
            </div>
          </CardContent>
        </Card>

        <div className="flex items-center gap-3">
          <Button onClick={save}>
            {saved ? <><Check size={14} /> Saved</> : <><Save size={14} /> Save settings</>}
          </Button>
          <Badge variant={settings.preferDeterministicRuns ? 'success' : 'info'}>
            {settings.preferDeterministicRuns ? 'Deterministic' : 'Stochastic'} mode
          </Badge>
        </div>
      </div>
    </AppShell>
  );
}

type ToggleProps = { label: string; description?: string; checked: boolean; onChange: (value: boolean) => void };

function Toggle({ label, description, checked, onChange }: ToggleProps) {
  return (
    <label className="flex items-center justify-between rounded-lg border bg-surface-elevated p-4 cursor-pointer hover:border-ink-muted transition-colors">
      <div>
        <p className="text-body-sm font-medium">{label}</p>
        {description && <p className="text-caption text-ink-muted mt-0.5">{description}</p>}
      </div>
      <div className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${checked ? 'bg-accent' : 'bg-surface-overlay border'}`}>
        <span className={`inline-block h-3.5 w-3.5 rounded-full bg-white transition-transform ${checked ? 'translate-x-4' : 'translate-x-0.5'}`} />
      </div>
    </label>
  );
}
