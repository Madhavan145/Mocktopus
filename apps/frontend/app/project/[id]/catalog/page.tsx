'use client';

import { useState, useEffect } from 'react';
import { AppShell } from '@/components/layout/app-shell';
import { apiFetch } from '@/lib/api';
import { Cloud, Database, Server, HardDrive, Wifi, Search, Filter } from 'lucide-react';

interface CacheInstance {
  id: string;
  provider: string;
  service: string;
  engine: string;
  nodeType: string;
  vcpus: number;
  memoryGb: number;
  maxConnections: number;
  pricePerMonth: number;
  region: string;
  clusterSupport: boolean;
}

interface ComputeInstance {
  id: string;
  provider: string;
  service: string;
  family: string;
  name: string;
  displayName: string;
  vcpus: number;
  memoryGb: number;
  pricePerMonth: number;
  region: string;
}

interface DatabaseInstance {
  id: string;
  provider: string;
  service: string;
  engine: string;
  instanceClass: string;
  vcpus: number;
  memoryGb: number;
  iops: number;
  pricePerMonth: number;
  multiAz: boolean;
}

type TabType = 'compute' | 'cache' | 'database' | 'queue';

export default function CatalogPage({ params }: { params: { id: string } }) {
  const [activeTab, setActiveTab] = useState<TabType>('compute');
  const [provider, setProvider] = useState<string>('');
  const [searchQuery, setSearchQuery] = useState('');
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    fetchData();
  }, [activeTab, provider]);

  const fetchData = async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (provider) params.set('provider', provider);
      const res = await apiFetch(`/api/v1/catalog/${activeTab}?${params}`);
      const json = await res.json();
      if (json.success) setData(json.data);
    } catch (e) {
      console.error('Failed to fetch catalog:', e);
    } finally {
      setLoading(false);
    }
  };

  const tabs = [
    { id: 'compute' as TabType, label: 'Compute', icon: <Server size={16} /> },
    { id: 'cache' as TabType, label: 'Cache', icon: <HardDrive size={16} /> },
    { id: 'database' as TabType, label: 'Database', icon: <Database size={16} /> },
    { id: 'queue' as TabType, label: 'Queue', icon: <Wifi size={16} /> },
  ];

  const providers = [
    { id: '', label: 'All Providers' },
    { id: 'aws', label: 'AWS' },
    { id: 'gcp', label: 'GCP' },
    { id: 'azure', label: 'Azure' },
  ];

  const items: any[] = data?.items || [];
  const filtered = searchQuery
    ? items.filter((item: any) =>
        JSON.stringify(item).toLowerCase().includes(searchQuery.toLowerCase())
      )
    : items;

  return (
    <AppShell projectId={params.id}>
      <div className="space-y-6">
        {/* Header */}
        <div>
          <h1 className="text-heading flex items-center gap-2">
            <Cloud size={24} className="text-accent" />
            Cloud Service Catalog
          </h1>
          <p className="text-body text-ink-muted mt-1">
            Browse real AWS, GCP, and Azure service configurations with actual pricing. Select instances for your topology nodes.
          </p>
        </div>

        {/* Filters Bar */}
        <div className="flex items-center gap-4 flex-wrap">
          {/* Tabs */}
          <div className="flex items-center gap-1 rounded-lg bg-surface p-1 border">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm transition-colors ${
                  activeTab === tab.id
                    ? 'bg-accent text-white'
                    : 'text-ink-muted hover:text-ink hover:bg-surface-elevated'
                }`}
              >
                {tab.icon}
                {tab.label}
              </button>
            ))}
          </div>

          {/* Provider filter */}
          <div className="flex items-center gap-1 rounded-lg bg-surface p-1 border">
            {providers.map((p) => (
              <button
                key={p.id}
                onClick={() => setProvider(p.id)}
                className={`rounded-md px-3 py-1.5 text-sm transition-colors ${
                  provider === p.id
                    ? 'bg-surface-elevated text-ink font-medium'
                    : 'text-ink-muted hover:text-ink'
                }`}
              >
                {p.label}
              </button>
            ))}
          </div>

          {/* Search */}
          <div className="relative flex-1 min-w-[200px]">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-muted" />
            <input
              type="text"
              placeholder="Search instances..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full rounded-lg border bg-surface px-3 py-2 pl-8 text-sm text-ink placeholder:text-ink-muted focus:border-accent focus:outline-none"
            />
          </div>
        </div>

        {/* Results Count */}
        <div className="flex items-center justify-between">
          <span className="text-caption text-ink-muted">
            {filtered.length} {activeTab} configurations found
          </span>
        </div>

        {/* Results Table */}
        {loading ? (
          <div className="flex items-center justify-center h-48">
            <div className="animate-spin h-6 w-6 border-2 border-accent border-t-transparent rounded-full" />
          </div>
        ) : (
          <div className="overflow-x-auto rounded-xl border bg-surface">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-surface-elevated/50">
                  {activeTab === 'compute' && (
                    <>
                      <th className="px-4 py-3 text-left font-medium text-ink-muted">Instance</th>
                      <th className="px-4 py-3 text-left font-medium text-ink-muted">Provider</th>
                      <th className="px-4 py-3 text-left font-medium text-ink-muted">Service</th>
                      <th className="px-4 py-3 text-right font-medium text-ink-muted">vCPUs</th>
                      <th className="px-4 py-3 text-right font-medium text-ink-muted">Memory</th>
                      <th className="px-4 py-3 text-right font-medium text-ink-muted">$/month</th>
                    </>
                  )}
                  {activeTab === 'cache' && (
                    <>
                      <th className="px-4 py-3 text-left font-medium text-ink-muted">Node Type</th>
                      <th className="px-4 py-3 text-left font-medium text-ink-muted">Provider</th>
                      <th className="px-4 py-3 text-left font-medium text-ink-muted">Service</th>
                      <th className="px-4 py-3 text-right font-medium text-ink-muted">Memory</th>
                      <th className="px-4 py-3 text-right font-medium text-ink-muted">Max Conn</th>
                      <th className="px-4 py-3 text-center font-medium text-ink-muted">Cluster</th>
                      <th className="px-4 py-3 text-right font-medium text-ink-muted">$/month</th>
                    </>
                  )}
                  {activeTab === 'database' && (
                    <>
                      <th className="px-4 py-3 text-left font-medium text-ink-muted">Instance</th>
                      <th className="px-4 py-3 text-left font-medium text-ink-muted">Provider</th>
                      <th className="px-4 py-3 text-left font-medium text-ink-muted">Engine</th>
                      <th className="px-4 py-3 text-right font-medium text-ink-muted">vCPUs</th>
                      <th className="px-4 py-3 text-right font-medium text-ink-muted">Memory</th>
                      <th className="px-4 py-3 text-right font-medium text-ink-muted">IOPS</th>
                      <th className="px-4 py-3 text-center font-medium text-ink-muted">Multi-AZ</th>
                      <th className="px-4 py-3 text-right font-medium text-ink-muted">$/month</th>
                    </>
                  )}
                  {activeTab === 'queue' && (
                    <>
                      <th className="px-4 py-3 text-left font-medium text-ink-muted">Service</th>
                      <th className="px-4 py-3 text-left font-medium text-ink-muted">Provider</th>
                      <th className="px-4 py-3 text-left font-medium text-ink-muted">Type</th>
                      <th className="px-4 py-3 text-right font-medium text-ink-muted">Throughput/s</th>
                      <th className="px-4 py-3 text-right font-medium text-ink-muted">$/million</th>
                    </>
                  )}
                </tr>
              </thead>
              <tbody>
                {filtered.map((item: any, i: number) => (
                  <tr key={item.id || i} className="border-b border-border/50 hover:bg-surface-elevated/30 transition-colors cursor-pointer">
                    {activeTab === 'compute' && (
                      <>
                        <td className="px-4 py-3 font-mono text-xs">{item.name || item.displayName}</td>
                        <td className="px-4 py-3"><ProviderBadge provider={item.provider} /></td>
                        <td className="px-4 py-3 text-ink-muted">{item.service}</td>
                        <td className="px-4 py-3 text-right">{item.vcpus}</td>
                        <td className="px-4 py-3 text-right">{item.memoryGb} GB</td>
                        <td className="px-4 py-3 text-right font-medium text-accent">${item.pricePerMonth?.toFixed(2)}</td>
                      </>
                    )}
                    {activeTab === 'cache' && (
                      <>
                        <td className="px-4 py-3 font-mono text-xs">{item.nodeType}</td>
                        <td className="px-4 py-3"><ProviderBadge provider={item.provider} /></td>
                        <td className="px-4 py-3 text-ink-muted">{item.service}</td>
                        <td className="px-4 py-3 text-right">{item.memoryGb} GB</td>
                        <td className="px-4 py-3 text-right">{item.maxConnections?.toLocaleString()}</td>
                        <td className="px-4 py-3 text-center">{item.clusterSupport ? '✓' : '—'}</td>
                        <td className="px-4 py-3 text-right font-medium text-accent">${item.pricePerMonth?.toFixed(2)}</td>
                      </>
                    )}
                    {activeTab === 'database' && (
                      <>
                        <td className="px-4 py-3 font-mono text-xs">{item.instanceClass}</td>
                        <td className="px-4 py-3"><ProviderBadge provider={item.provider} /></td>
                        <td className="px-4 py-3 text-ink-muted">{item.engine}</td>
                        <td className="px-4 py-3 text-right">{item.vcpus}</td>
                        <td className="px-4 py-3 text-right">{item.memoryGb} GB</td>
                        <td className="px-4 py-3 text-right">{item.iops?.toLocaleString()}</td>
                        <td className="px-4 py-3 text-center">{item.multiAz ? '✓' : '—'}</td>
                        <td className="px-4 py-3 text-right font-medium text-accent">${item.pricePerMonth?.toFixed(2)}</td>
                      </>
                    )}
                    {activeTab === 'queue' && (
                      <>
                        <td className="px-4 py-3 font-mono text-xs">{item.service}</td>
                        <td className="px-4 py-3"><ProviderBadge provider={item.provider} /></td>
                        <td className="px-4 py-3 text-ink-muted">{item.type}</td>
                        <td className="px-4 py-3 text-right">{item.throughputPerSec?.toLocaleString()}</td>
                        <td className="px-4 py-3 text-right font-medium text-accent">${item.pricePerMillion?.toFixed(3)}</td>
                      </>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
            {filtered.length === 0 && (
              <div className="flex items-center justify-center h-32 text-ink-muted text-sm">
                No results found
              </div>
            )}
          </div>
        )}
      </div>
    </AppShell>
  );
}

function ProviderBadge({ provider }: { provider: string }) {
  const colors: Record<string, string> = {
    aws: 'bg-orange-500/10 text-orange-400 border-orange-500/20',
    gcp: 'bg-blue-500/10 text-blue-400 border-blue-500/20',
    azure: 'bg-cyan-500/10 text-cyan-400 border-cyan-500/20',
  };
  return (
    <span className={`inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-medium ${colors[provider] || 'bg-surface-elevated text-ink-muted'}`}>
      {provider?.toUpperCase()}
    </span>
  );
}
