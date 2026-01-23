import SimulateWorkspace from '@/components/simulation/SimulateWorkspace';
import { AppShell } from '@/components/layout/app-shell';

type PageProps = {
  params: { id: string };
};

export default function SimulatePage({ params }: PageProps) {
  return (
    <AppShell projectId={params.id}>
      <div className="space-y-6 animate-fade-in">
        <div>
          <div className="flex items-center gap-2 text-caption text-ink-muted mb-1">
            <span>Project</span><span>/</span><span>{params.id}</span><span>/</span><span className="text-ink-secondary">Simulate</span>
          </div>
          <h1 className="text-title-lg">Simulation</h1>
          <p className="text-body-sm text-ink-secondary mt-1">
            Run deterministic simulation on your topology and inspect latency, throughput, and failure behavior.
          </p>
        </div>
        <SimulateWorkspace projectId={params.id} />
      </div>
    </AppShell>
  );
}
