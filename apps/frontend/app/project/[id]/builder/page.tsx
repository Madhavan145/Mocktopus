import Link from 'next/link';
import BuilderWorkspace from '@/components/builder/BuilderWorkspace';
import { ArrowLeft } from 'lucide-react';

type PageProps = {
  params: { id: string };
};

export default function BuilderPage({ params }: PageProps) {
  return (
    <div className="flex h-screen flex-col overflow-hidden">
      {/* Compact toolbar header */}
      <header className="flex h-12 shrink-0 items-center justify-between border-b bg-surface px-4">
        <div className="flex items-center gap-3">
          <Link
            href={`/project/${params.id}`}
            className="inline-flex items-center gap-1.5 rounded-md border px-2 py-1 text-xs text-ink-muted hover:text-ink hover:bg-white/5 transition-colors"
          >
            <ArrowLeft size={14} />
            Back
          </Link>
          <span className="text-body-sm font-medium">Topology Builder</span>
          <span className="text-caption text-ink-muted">/ {params.id}</span>
        </div>
        <div className="flex items-center gap-2 text-caption text-ink-muted">
          <kbd className="inline-flex h-5 items-center rounded border bg-surface-elevated px-1.5 text-[10px]">⌘Z</kbd>
          <span>Undo</span>
          <kbd className="inline-flex h-5 items-center rounded border bg-surface-elevated px-1.5 text-[10px]">⌘⇧Z</kbd>
          <span>Redo</span>
          <kbd className="inline-flex h-5 items-center rounded border bg-surface-elevated px-1.5 text-[10px]">Space</kbd>
          <span>Pan</span>
        </div>
      </header>
      <div className="flex-1 min-h-0">
        <BuilderWorkspace projectId={params.id} />
      </div>
    </div>
  );
}
