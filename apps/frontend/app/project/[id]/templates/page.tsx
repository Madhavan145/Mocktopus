'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { AppShell } from '@/components/layout/app-shell';
import { apiFetch } from '@/lib/api';
import { LayoutTemplate, Layers, Zap, Globe, Server, GitBranch, Brain, ArrowRight } from 'lucide-react';

interface Template {
  id: string;
  name: string;
  description: string;
  category: string;
  provider: string;
  complexity: string;
  estimatedCost: string;
  tags: string[];
  nodeCount: number;
  edgeCount: number;
}

const categoryIcons: Record<string, any> = {
  microservices: <Layers size={16} />,
  'event-driven': <Zap size={16} />,
  serverless: <Zap size={16} />,
  traditional: <Server size={16} />,
  data: <GitBranch size={16} />,
  distributed: <Globe size={16} />,
  'service-mesh': <GitBranch size={16} />,
  ml: <Brain size={16} />,
};

const complexityColors: Record<string, string> = {
  beginner: 'bg-green-500/10 text-green-400 border-green-500/20',
  intermediate: 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20',
  advanced: 'bg-orange-500/10 text-orange-400 border-orange-500/20',
  expert: 'bg-red-500/10 text-red-400 border-red-500/20',
};

export default function TemplatesPage({ params }: { params: { id: string } }) {
  const router = useRouter();
  const [templates, setTemplates] = useState<Template[]>([]);
  const [categories, setCategories] = useState<string[]>([]);
  const [selectedCategory, setSelectedCategory] = useState('');
  const [selectedTemplate, setSelectedTemplate] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchTemplates();
    fetchCategories();
  }, []);

  const fetchTemplates = async () => {
    try {
      const res = await apiFetch(`/api/v1/templates`);
      const json = await res.json();
      if (json.success) setTemplates(json.data.templates || []);
    } catch (e) {
      console.error('Failed to fetch templates:', e);
    } finally {
      setLoading(false);
    }
  };

  const fetchCategories = async () => {
    try {
      const res = await apiFetch(`/api/v1/templates/categories`);
      const json = await res.json();
      if (json.success) setCategories(json.data.categories || []);
    } catch (e) {
      console.error('Failed to fetch categories:', e);
    }
  };

  const fetchTemplateDetail = async (id: string) => {
    try {
      const res = await apiFetch(`/api/v1/templates/${id}`);
      const json = await res.json();
      if (json.success) setSelectedTemplate(json.data);
    } catch (e) {
      console.error('Failed to fetch template detail:', e);
    }
  };

  const filtered = selectedCategory
    ? templates.filter((t) => t.category === selectedCategory)
    : templates;

  return (
    <AppShell projectId={params.id}>
      <div className="space-y-6">
        {/* Header */}
        <div>
          <h1 className="text-heading flex items-center gap-2">
            <LayoutTemplate size={24} className="text-accent" />
            Architecture Templates
          </h1>
          <p className="text-body text-ink-muted mt-1">
            Start from pre-built production architecture patterns. Each template includes a full topology graph and recommended simulation parameters.
          </p>
        </div>

        {/* Category Filter */}
        <div className="flex items-center gap-2 flex-wrap">
          <button
            onClick={() => setSelectedCategory('')}
            className={`rounded-full px-3 py-1.5 text-xs border transition-colors ${
              !selectedCategory
                ? 'bg-accent text-white border-accent'
                : 'text-ink-muted border-border hover:border-accent/50'
            }`}
          >
            All
          </button>
          {categories.map((cat) => (
            <button
              key={cat}
              onClick={() => setSelectedCategory(cat)}
              className={`flex items-center gap-1.5 rounded-full px-3 py-1.5 text-xs border transition-colors ${
                selectedCategory === cat
                  ? 'bg-accent text-white border-accent'
                  : 'text-ink-muted border-border hover:border-accent/50'
              }`}
            >
              {categoryIcons[cat]}
              {cat}
            </button>
          ))}
        </div>

        {/* Template Grid */}
        {loading ? (
          <div className="flex items-center justify-center h-48">
            <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
          </div>
        ) : selectedTemplate ? (
          /* Template Detail View */
          <div className="space-y-6">
            <button
              onClick={() => setSelectedTemplate(null)}
              className="text-sm text-accent hover:underline"
            >
              ← Back to templates
            </button>
            <div className="rounded-xl border bg-surface p-6 space-y-6">
              <div className="flex items-start justify-between">
                <div>
                  <h2 className="text-lg font-semibold text-ink">{selectedTemplate.name}</h2>
                  <p className="text-sm text-ink-muted mt-1">{selectedTemplate.description}</p>
                </div>
                <span className={`inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-medium ${complexityColors[selectedTemplate.complexity]}`}>
                  {selectedTemplate.complexity}
                </span>
              </div>

              {/* Stats */}
              <div className="grid grid-cols-4 gap-4">
                <div className="rounded-lg border bg-surface-elevated/50 p-3 text-center">
                  <p className="text-xl font-bold text-accent">{selectedTemplate.graph?.nodes?.length || 0}</p>
                  <p className="text-xs text-ink-muted">Nodes</p>
                </div>
                <div className="rounded-lg border bg-surface-elevated/50 p-3 text-center">
                  <p className="text-xl font-bold text-ink">{selectedTemplate.graph?.edges?.length || 0}</p>
                  <p className="text-xs text-ink-muted">Edges</p>
                </div>
                <div className="rounded-lg border bg-surface-elevated/50 p-3 text-center">
                  <p className="text-xl font-bold text-ink">{selectedTemplate.provider}</p>
                  <p className="text-xs text-ink-muted">Provider</p>
                </div>
                <div className="rounded-lg border bg-surface-elevated/50 p-3 text-center">
                  <p className="text-xl font-bold text-ink">{selectedTemplate.estimatedCost}</p>
                  <p className="text-xs text-ink-muted">Est. Cost</p>
                </div>
              </div>

              {/* Nodes List */}
              <div>
                <h3 className="text-sm font-medium text-ink mb-3">Architecture Components</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
                  {selectedTemplate.graph?.nodes?.map((node: any) => (
                    <div key={node.id} className="flex items-center gap-3 rounded-lg border p-3">
                      <div className="h-8 w-8 rounded-lg bg-accent/10 flex items-center justify-center text-accent">
                        <Server size={14} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium text-ink truncate">{node.id}</p>
                        <p className="text-xs text-ink-muted">{node.type}</p>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Traffic Preset */}
              {selectedTemplate.trafficPreset && (
                <div>
                  <h3 className="text-sm font-medium text-ink mb-3">Recommended Simulation Config</h3>
                  <div className="grid grid-cols-3 gap-3">
                    <div className="rounded-lg border p-3">
                      <p className="text-xs text-ink-muted">RPS</p>
                      <p className="text-sm font-medium text-ink">{selectedTemplate.trafficPreset.rps}</p>
                    </div>
                    <div className="rounded-lg border p-3">
                      <p className="text-xs text-ink-muted">Duration</p>
                      <p className="text-sm font-medium text-ink">{selectedTemplate.trafficPreset.durationSec}s</p>
                    </div>
                    <div className="rounded-lg border p-3">
                      <p className="text-xs text-ink-muted">Pattern</p>
                      <p className="text-sm font-medium text-ink">{selectedTemplate.trafficPreset.pattern}</p>
                    </div>
                  </div>
                </div>
              )}

              {/* Apply Button */}
              <div className="flex gap-3">
                <button
                  onClick={async () => {
                    if (!selectedTemplate?.graph) return;
                    // Create a new project named after the template
                    let newProjectId: string;
                    try {
                      const res = await apiFetch(`/api/v1/projects`, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ name: selectedTemplate.name, description: selectedTemplate.description }),
                      });
                      const json = await res.json();
                      if (json.success && json.data?.id) {
                        newProjectId = json.data.id;
                      } else {
                        newProjectId = `tmpl-${crypto.randomUUID().slice(0, 8)}`;
                      }
                    } catch {
                      newProjectId = `tmpl-${crypto.randomUUID().slice(0, 8)}`;
                    }

                    const nodes = (selectedTemplate.graph.nodes || []).map((node: any, index: number) => ({
                      id: node.id,
                      type: 'custom',
                      position: { x: 140 + (index % 4) * 260, y: 100 + Math.floor(index / 4) * 160 },
                      data: {
                        label: node.id.replace(/-/g, ' ').replace(/\b\w/g, (c: string) => c.toUpperCase()),
                        kind: node.type || 'service',
                        config: node.config || {},
                      },
                    }));
                    const edges = (selectedTemplate.graph.edges || []).map((edge: any) => ({
                      id: edge.id,
                      source: edge.source,
                      target: edge.target,
                      animated: false,
                    }));
                    const payload = { nodes, edges, updatedAt: new Date().toISOString() };
                    localStorage.setItem(`moktopus:project:${newProjectId}:builder`, JSON.stringify(payload));
                    router.push(`/project/${newProjectId}/builder`);
                  }}
                  className="flex-1 rounded-lg bg-accent px-4 py-3 text-sm font-medium text-white hover:bg-accent/90 transition-colors flex items-center justify-center gap-2"
                >
                  Create Project &amp; Open in Builder <ArrowRight size={14} />
                </button>
                <button className="rounded-lg border px-4 py-3 text-sm text-ink hover:bg-surface-elevated transition-colors">
                  Export as Terraform
                </button>
              </div>
            </div>
          </div>
        ) : (
          /* Grid View */
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
            {filtered.map((template) => (
              <div
                key={template.id}
                onClick={() => fetchTemplateDetail(template.id)}
                className="group rounded-xl border bg-surface p-5 hover:border-accent/50 hover:shadow-lg transition-all cursor-pointer"
              >
                <div className="flex items-start justify-between mb-3">
                  <div className="h-10 w-10 rounded-lg bg-accent/10 flex items-center justify-center text-accent">
                    {categoryIcons[template.category] || <Layers size={18} />}
                  </div>
                  <span className={`inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-medium ${complexityColors[template.complexity]}`}>
                    {template.complexity}
                  </span>
                </div>
                <h3 className="text-sm font-semibold text-ink group-hover:text-accent transition-colors">
                  {template.name}
                </h3>
                <p className="text-xs text-ink-muted mt-1 line-clamp-2">{template.description}</p>
                <div className="flex items-center gap-3 mt-4 text-xs text-ink-muted">
                  <span>{template.nodeCount} nodes</span>
                  <span>•</span>
                  <span>{template.edgeCount} edges</span>
                  <span>•</span>
                  <span>{template.estimatedCost}</span>
                </div>
                <div className="flex flex-wrap gap-1 mt-3">
                  {template.tags?.slice(0, 3).map((tag) => (
                    <span key={tag} className="rounded-full bg-surface-elevated px-2 py-0.5 text-[10px] text-ink-muted">
                      {tag}
                    </span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </AppShell>
  );
}
