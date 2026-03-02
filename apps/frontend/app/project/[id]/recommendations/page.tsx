import InsightsView from '@/components/simulation/InsightsView';
import { AppShell } from '@/components/layout/app-shell';

type PageProps = {
  params: { id: string };
};

export default function RecommendationsPage({ params }: PageProps) {
  return (
    <AppShell projectId={params.id}>
      <div className="space-y-6 animate-fade-in">
        <div>
          <div className="flex items-center gap-2 text-caption text-ink-muted mb-1">
            <span>Project</span><span>/</span><span>{params.id}</span><span>/</span><span className="text-ink-secondary">Recommendations</span>
          </div>
          <h1 className="text-title-lg">Recommendations</h1>
          <p className="text-body-sm text-ink-secondary mt-1">
            Actionable optimization guidance with projected performance impact.
          </p>
        </div>
        <InsightsView projectId={params.id} mode="recommendations" />
      </div>
    </AppShell>
  );
}
