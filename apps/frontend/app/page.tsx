import Link from 'next/link';
import { Zap, ArrowRight, Layers, Play, Activity } from 'lucide-react';

export default function HomePage() {
  return (
    <div className="flex min-h-screen flex-col">
      {/* Hero */}
      <main className="flex flex-1 flex-col items-center justify-center px-6 text-center">
        <div className="inline-flex items-center gap-2 rounded-full border bg-surface px-4 py-1.5 mb-8">
          <Zap size={14} className="text-accent" />
          <span className="text-caption text-ink-secondary">Infrastructure Simulation Platform</span>
        </div>

        <h1 className="text-display max-w-3xl">
          Simulate before<br />
          you <span className="text-accent">ship</span>.
        </h1>

        <p className="mt-6 max-w-lg text-body text-ink-secondary">
          Model distributed systems, inject chaos, and discover bottlenecks
          before they reach production. Built for platform engineering teams.
        </p>

        <div className="mt-10 flex items-center gap-3">
          <Link
            href="/playground"
            className="inline-flex items-center gap-2 rounded-md bg-accent px-5 py-2.5 text-body-sm font-medium text-white hover:bg-accent-hover transition-colors"
          >
            Get Started <ArrowRight size={16} />
          </Link>
          <Link
            href="/login"
            className="inline-flex items-center gap-2 rounded-md border px-5 py-2.5 text-body-sm font-medium text-ink-secondary hover:text-ink hover:border-ink-muted transition-colors"
          >
            Sign In
          </Link>
        </div>

        {/* Feature pills */}
        <div className="mt-16 grid grid-cols-1 md:grid-cols-3 gap-4 max-w-2xl w-full">
          <div className="flex items-center gap-3 rounded-lg border bg-surface p-4">
            <Layers size={18} className="text-accent shrink-0" />
            <div className="text-left">
              <p className="text-body-sm font-medium">Topology Builder</p>
              <p className="text-caption text-ink-muted">Figma-like drag & drop</p>
            </div>
          </div>
          <div className="flex items-center gap-3 rounded-lg border bg-surface p-4">
            <Play size={18} className="text-success shrink-0" />
            <div className="text-left">
              <p className="text-body-sm font-medium">Simulation Engine</p>
              <p className="text-caption text-ink-muted">Deterministic runs</p>
            </div>
          </div>
          <div className="flex items-center gap-3 rounded-lg border bg-surface p-4">
            <Activity size={18} className="text-info shrink-0" />
            <div className="text-left">
              <p className="text-body-sm font-medium">Observability</p>
              <p className="text-caption text-ink-muted">Graph-centric metrics</p>
            </div>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t py-4 px-6 text-center">
        <p className="text-caption text-ink-muted">Moktopus — Local-first infrastructure simulator</p>
      </footer>
    </div>
  );
}
