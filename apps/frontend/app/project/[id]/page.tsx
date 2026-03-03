'use client';

import Link from 'next/link';
import { Layers, Play, RotateCcw, GitCompare, DollarSign, Lightbulb, BarChart3, ArrowRight, Cloud, FileCode, LayoutTemplate } from 'lucide-react';
import { AppShell } from '@/components/layout/app-shell';

type PageProps = {
  params: { id: string };
};

const modules = [
  { key: 'builder', label: 'Topology Builder', desc: 'Design your architecture graph', icon: Layers, color: 'text-accent' },
  { key: 'simulate', label: 'Simulate', desc: 'Run load and failure scenarios', icon: Play, color: 'text-success' },
  { key: 'results', label: 'Results', desc: 'Analyze simulation metrics', icon: BarChart3, color: 'text-info' },
  { key: 'replay', label: 'Replay', desc: 'Step through simulation timeline', icon: RotateCcw, color: 'text-purple-400' },
  { key: 'compare', label: 'Compare', desc: 'A/B architecture comparison', icon: GitCompare, color: 'text-cyan-400' },
  { key: 'cost', label: 'Cost Analysis', desc: 'Infrastructure cost breakdown', icon: DollarSign, color: 'text-warning' },
  { key: 'recommendations', label: 'Recommendations', desc: 'AI-driven optimization hints', icon: Lightbulb, color: 'text-amber-300' },
  { key: 'catalog', label: 'Cloud Catalog', desc: 'Browse AWS, GCP, Azure instances', icon: Cloud, color: 'text-blue-400' },
  { key: 'iac', label: 'IaC Import/Export', desc: 'Import Terraform, CloudFormation, Pulumi', icon: FileCode, color: 'text-green-400' },
  { key: 'templates', label: 'Templates', desc: 'Production architecture patterns', icon: LayoutTemplate, color: 'text-pink-400' },
];

export default function ProjectPage({ params }: PageProps) {
  const base = `/project/${params.id}`;

  return (
    <AppShell projectId={params.id}>
      <div className="space-y-8 animate-fade-in">
        <div className="space-y-2">
          <div className="flex items-center gap-2 text-caption text-ink-muted">
            <span>Projects</span>
            <span>/</span>
            <span className="text-ink-secondary">{params.id}</span>
          </div>
          <h1 className="text-title-lg">Project: {params.id}</h1>
          <p className="text-body-sm text-ink-secondary">
            Infrastructure simulation workspace with builder, simulation, cost analysis, and cloud integrations.
          </p>
        </div>

        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {modules.map((mod) => (
            <Link
              key={mod.key}
              href={`${base}/${mod.key}`}
              className="group relative rounded-lg border bg-surface p-5 hover:border-ink-muted transition-all duration-200 hover:shadow-md"
            >
              <div className="flex items-start justify-between">
                <mod.icon size={20} className={mod.color} />
                <ArrowRight size={14} className="text-ink-muted opacity-0 group-hover:opacity-100 transition-opacity" />
              </div>
              <h3 className="mt-3 text-body-sm font-medium">{mod.label}</h3>
              <p className="mt-1 text-caption text-ink-muted">{mod.desc}</p>
            </Link>
          ))}
        </div>
      </div>
    </AppShell>
  );
}
