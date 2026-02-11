import ResultsView from '@/components/simulation/ResultsView';
import { AppShell } from '@/components/layout/app-shell';

type PageProps = {
  params: { id: string };
};

export default function ResultsPage({ params }: PageProps) {
  return (
    <AppShell projectId={params.id}>
      <div className="space-y-6 animate-fade-in">
        <div>
          <div className="flex items-center gap-2 text-caption text-ink-muted mb-1">
            <span>Project</span><span>/</span><span>{params.id}</span><span>/</span><span className="text-ink-secondary">Results</span>
          </div>
          <h1 className="text-title-lg">Simulation Results</h1>
          <p className="text-body-sm text-ink-secondary mt-1">
            View simulation metrics, timeline events, and performance analytics.
          </p>
        </div>
        <ResultsView projectId={params.id} />
      </div>
    </AppShell>
  );
}
