'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import { apiFetch } from '@/lib/api';

type APIKey = {
  id: string;
  orgId: string;
  name: string;
  keyPrefix: string;
  scopes: string[];
  rateLimitPerMinute: number;
  lastUsedAt: string | null;
  expiresAt: string | null;
  revokedAt: string | null;
  createdAt: string;
};

type Org = { id: string; name: string; slug: string };

export default function APIKeysPage() {
  const [orgs, setOrgs] = useState<Org[]>([]);
  const [selectedOrgId, setSelectedOrgId] = useState('');
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [newKey, setNewKey] = useState<string | null>(null);
  const [form, setForm] = useState({ name: '', scopes: 'simulation:run,topology:read' });

  useEffect(() => {
    void loadOrgs();
  }, []);

  useEffect(() => {
    if (selectedOrgId) void loadKeys();
  }, [selectedOrgId]);

  async function loadOrgs() {
    const res = await apiFetch(`/api/v1/admin/orgs`);
    if (res.ok) {
      const data = await res.json();
      setOrgs(data.data ?? []);
      if (data.data?.length > 0) setSelectedOrgId(data.data[0].id);
    }
  }

  async function loadKeys() {
    const res = await apiFetch(`/api/v1/admin/api-keys?orgId=${selectedOrgId}`);
    if (res.ok) {
      const data = await res.json();
      setKeys(data.data ?? []);
    }
  }

  async function createKey() {
    const res = await apiFetch(`/api/v1/admin/api-keys/create`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        orgId: selectedOrgId,
        name: form.name,
        scopes: form.scopes.split(',').map((s) => s.trim()),
        createdBy: 'admin',
      }),
    });
    if (res.ok) {
      const data = await res.json();
      setNewKey(data.data?.key ?? null);
      setForm({ name: '', scopes: 'simulation:run,topology:read' });
      void loadKeys();
    }
  }

  async function revokeKey(keyId: string) {
    await apiFetch(`/api/v1/admin/api-keys/revoke`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ keyId }),
    });
    void loadKeys();
  }

  return (
    <div className="min-h-screen bg-background p-8 space-y-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">API Keys</h1>
        <Link href="/admin" className="text-sm text-mute hover:text-white">← Admin Home</Link>
      </header>

      {/* Org Selector */}
      <div className="flex items-center gap-3">
        <label className="text-sm text-mute">Organization:</label>
        <select
          className="rounded border bg-black/30 px-3 py-2 text-sm"
          value={selectedOrgId}
          onChange={(e) => setSelectedOrgId(e.target.value)}
        >
          {orgs.map((org) => (
            <option key={org.id} value={org.id}>{org.name} ({org.slug})</option>
          ))}
        </select>
      </div>

      {/* Create Key */}
      <section className="rounded-xl border bg-panel p-4 space-y-3">
        <h2 className="font-semibold">Generate New API Key</h2>
        <div className="grid gap-3 md:grid-cols-2">
          <input className="rounded border bg-black/30 px-3 py-2 text-sm" placeholder="Key Name (e.g. CI Pipeline)" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          <input className="rounded border bg-black/30 px-3 py-2 text-sm" placeholder="Scopes (comma-separated)" value={form.scopes} onChange={(e) => setForm({ ...form, scopes: e.target.value })} />
        </div>
        <button onClick={createKey} disabled={!form.name} className="rounded-md bg-accent px-4 py-2 text-sm font-medium text-black disabled:opacity-50">
          Generate Key
        </button>

        {newKey && (
          <div className="mt-3 rounded-lg border border-amber-500/50 bg-amber-950/20 p-3">
            <p className="text-xs text-amber-300 font-medium mb-1">Copy this key now — it will not be shown again:</p>
            <code className="block text-sm text-white bg-black/50 rounded p-2 break-all select-all">{newKey}</code>
          </div>
        )}
      </section>

      {/* Keys Table */}
      <section>
        <h2 className="font-semibold mb-3">Active Keys ({keys.filter(k => !k.revokedAt).length})</h2>
        <div className="overflow-auto rounded-lg border">
          <table className="min-w-full text-sm">
            <thead className="bg-black/30 text-mute">
              <tr>
                <th className="px-4 py-2 text-left">Name</th>
                <th className="px-4 py-2 text-left">Prefix</th>
                <th className="px-4 py-2 text-left">Scopes</th>
                <th className="px-4 py-2 text-left">Rate Limit</th>
                <th className="px-4 py-2 text-left">Last Used</th>
                <th className="px-4 py-2 text-left">Status</th>
                <th className="px-4 py-2 text-left">Actions</th>
              </tr>
            </thead>
            <tbody>
              {keys.map((key) => (
                <tr key={key.id} className="border-t border-line/40">
                  <td className="px-4 py-2 font-medium">{key.name}</td>
                  <td className="px-4 py-2 font-mono text-xs text-mute">{key.keyPrefix}...</td>
                  <td className="px-4 py-2 text-xs text-mute">{key.scopes?.join(', ')}</td>
                  <td className="px-4 py-2 text-xs">{key.rateLimitPerMinute}/min</td>
                  <td className="px-4 py-2 text-xs text-mute">{key.lastUsedAt ? new Date(key.lastUsedAt).toLocaleString() : 'Never'}</td>
                  <td className="px-4 py-2">
                    {key.revokedAt ? (
                      <span className="text-xs text-red-400">Revoked</span>
                    ) : (
                      <span className="text-xs text-emerald-400">Active</span>
                    )}
                  </td>
                  <td className="px-4 py-2">
                    {!key.revokedAt && (
                      <button onClick={() => revokeKey(key.id)} className="text-xs text-red-400 hover:underline">Revoke</button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
