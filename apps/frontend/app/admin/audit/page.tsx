'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import { apiFetch } from '@/lib/api';

type AuditEntry = {
  id: number;
  orgId: string;
  userId: string;
  action: string;
  resourceType: string;
  resourceId: string;
  metadata: Record<string, unknown>;
  ipAddress: string;
  createdAt: string;
};

type Org = { id: string; name: string; slug: string };

export default function AuditPage() {
  const [orgs, setOrgs] = useState<Org[]>([]);
  const [selectedOrgId, setSelectedOrgId] = useState('');
  const [entries, setEntries] = useState<AuditEntry[]>([]);

  useEffect(() => { void loadOrgs(); }, []);
  useEffect(() => { if (selectedOrgId) void loadAudit(); }, [selectedOrgId]);

  async function loadOrgs() {
    const res = await apiFetch(`/api/v1/admin/orgs`);
    if (res.ok) {
      const data = await res.json();
      setOrgs(data.data ?? []);
      if (data.data?.length > 0) setSelectedOrgId(data.data[0].id);
    }
  }

  async function loadAudit() {
    const res = await apiFetch(`/api/v1/admin/audit?orgId=${selectedOrgId}`);
    if (res.ok) {
      const data = await res.json();
      setEntries(data.data ?? []);
    }
  }

  return (
    <div className="min-h-screen bg-background p-8 space-y-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Audit Log</h1>
        <Link href="/admin" className="text-sm text-mute hover:text-white">← Admin Home</Link>
      </header>

      <div className="flex items-center gap-3">
        <label className="text-sm text-mute">Organization:</label>
        <select className="rounded border bg-black/30 px-3 py-2 text-sm" value={selectedOrgId} onChange={(e) => setSelectedOrgId(e.target.value)}>
          {orgs.map((org) => <option key={org.id} value={org.id}>{org.name}</option>)}
        </select>
        <button onClick={loadAudit} className="rounded border px-3 py-2 text-xs hover:bg-black/20">Refresh</button>
      </div>

      {entries.length === 0 ? (
        <p className="text-sm text-mute">No audit entries recorded yet.</p>
      ) : (
        <div className="overflow-auto rounded-lg border">
          <table className="min-w-full text-sm">
            <thead className="bg-black/30 text-mute">
              <tr>
                <th className="px-4 py-2 text-left">Timestamp</th>
                <th className="px-4 py-2 text-left">Action</th>
                <th className="px-4 py-2 text-left">Resource</th>
                <th className="px-4 py-2 text-left">User</th>
                <th className="px-4 py-2 text-left">IP</th>
                <th className="px-4 py-2 text-left">Details</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((entry) => (
                <tr key={entry.id} className="border-t border-line/40">
                  <td className="px-4 py-2 text-xs text-mute whitespace-nowrap">{new Date(entry.createdAt).toLocaleString()}</td>
                  <td className="px-4 py-2">
                    <span className={`px-2 py-0.5 rounded text-xs font-medium ${
                      entry.action.includes('delete') || entry.action.includes('revoke') ? 'bg-red-500/20 text-red-300' :
                      entry.action.includes('create') || entry.action.includes('add') ? 'bg-green-500/20 text-green-300' :
                      'bg-blue-500/20 text-blue-300'
                    }`}>{entry.action}</span>
                  </td>
                  <td className="px-4 py-2 text-xs">{entry.resourceType}{entry.resourceId ? ` (${entry.resourceId.slice(0, 8)}...)` : ''}</td>
                  <td className="px-4 py-2 text-xs text-mute">{entry.userId ? entry.userId.slice(0, 8) + '...' : '—'}</td>
                  <td className="px-4 py-2 text-xs text-mute">{entry.ipAddress || '—'}</td>
                  <td className="px-4 py-2 text-xs text-mute max-w-[200px] truncate">{JSON.stringify(entry.metadata)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
