'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import { apiFetch } from '@/lib/api';

type Org = {
  id: string;
  name: string;
  slug: string;
  plan: string;
  maxSeats: number;
  maxProjects: number;
  maxSimulationsPerDay: number;
  billingEmail: string;
  suspendedAt: string | null;
  createdAt: string;
};

type Member = {
  id: string;
  orgId: string;
  userId: string;
  role: string;
  email: string;
  fullName: string;
  joinedAt: string;
};

export default function OrgsPage() {
  const [orgs, setOrgs] = useState<Org[]>([]);
  const [selectedOrg, setSelectedOrg] = useState<Org | null>(null);
  const [members, setMembers] = useState<Member[]>([]);
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({ name: '', slug: '', plan: 'starter', billingEmail: '' });

  useEffect(() => { void loadOrgs(); }, []);

  async function loadOrgs() {
    const res = await apiFetch(`/api/v1/admin/orgs`);
    if (res.ok) {
      const data = await res.json();
      setOrgs(data.data ?? []);
    }
  }

  async function loadMembers(orgId: string) {
    const res = await apiFetch(`/api/v1/admin/orgs/members?orgId=${orgId}`);
    if (res.ok) {
      const data = await res.json();
      setMembers(data.data ?? []);
    }
  }

  async function createOrg() {
    const res = await apiFetch(`/api/v1/admin/orgs`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(form),
    });
    if (res.ok) {
      setShowCreate(false);
      setForm({ name: '', slug: '', plan: 'starter', billingEmail: '' });
      void loadOrgs();
    }
  }

  async function removeMember(orgId: string, userId: string) {
    await apiFetch(`/api/v1/admin/orgs/members/remove`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ orgId, userId }),
    });
    void loadMembers(orgId);
  }

  return (
    <div className="min-h-screen bg-background p-8 space-y-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Organizations</h1>
        <div className="flex gap-3">
          <button onClick={() => setShowCreate(!showCreate)} className="rounded-md bg-accent px-4 py-2 text-sm font-medium text-black">
            + New Organization
          </button>
          <Link href="/admin" className="text-sm text-mute hover:text-white self-center">← Admin Home</Link>
        </div>
      </header>

      {/* Create Form */}
      {showCreate && (
        <section className="rounded-xl border bg-panel p-4 space-y-3">
          <h2 className="font-semibold">Create Organization</h2>
          <div className="grid gap-3 md:grid-cols-2">
            <input className="rounded border bg-black/30 px-3 py-2 text-sm" placeholder="Organization Name" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            <input className="rounded border bg-black/30 px-3 py-2 text-sm" placeholder="slug (url-friendly)" value={form.slug} onChange={(e) => setForm({ ...form, slug: e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, '') })} />
            <select className="rounded border bg-black/30 px-3 py-2 text-sm" value={form.plan} onChange={(e) => setForm({ ...form, plan: e.target.value })}>
              <option value="trial">Trial</option>
              <option value="starter">Starter</option>
              <option value="professional">Professional</option>
              <option value="enterprise">Enterprise</option>
            </select>
            <input className="rounded border bg-black/30 px-3 py-2 text-sm" placeholder="Billing Email" value={form.billingEmail} onChange={(e) => setForm({ ...form, billingEmail: e.target.value })} />
          </div>
          <button onClick={createOrg} className="rounded-md bg-emerald-600 px-4 py-2 text-sm font-medium text-white">Create</button>
        </section>
      )}

      {/* Org List */}
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {orgs.map((org) => (
          <div
            key={org.id}
            className={`rounded-xl border p-4 cursor-pointer transition-all ${selectedOrg?.id === org.id ? 'border-accent bg-accent/5' : 'bg-panel hover:border-accent/40'}`}
            onClick={() => { setSelectedOrg(org); void loadMembers(org.id); }}
          >
            <div className="flex justify-between items-start">
              <div>
                <h3 className="font-semibold">{org.name}</h3>
                <p className="text-xs text-mute">{org.slug}</p>
              </div>
              <span className={`px-2 py-0.5 rounded text-xs font-medium ${
                org.plan === 'enterprise' ? 'bg-purple-500/20 text-purple-300' :
                org.plan === 'professional' ? 'bg-blue-500/20 text-blue-300' :
                org.plan === 'starter' ? 'bg-green-500/20 text-green-300' :
                'bg-gray-500/20 text-gray-300'
              }`}>{org.plan}</span>
            </div>
            <div className="mt-2 flex gap-4 text-xs text-mute">
              <span>{org.maxSeats} seats</span>
              <span>{org.maxProjects} projects</span>
              <span>{org.maxSimulationsPerDay} sims/day</span>
            </div>
            {org.suspendedAt && <p className="mt-1 text-xs text-red-400">Suspended</p>}
          </div>
        ))}
      </div>

      {/* Selected Org Members */}
      {selectedOrg && (
        <section className="rounded-xl border bg-panel p-4">
          <h2 className="font-semibold mb-3">Members of {selectedOrg.name}</h2>
          {members.length === 0 ? (
            <p className="text-sm text-mute">No members yet.</p>
          ) : (
            <div className="overflow-auto rounded border">
              <table className="min-w-full text-sm">
                <thead className="bg-black/30 text-mute">
                  <tr>
                    <th className="px-3 py-2 text-left">Email</th>
                    <th className="px-3 py-2 text-left">Name</th>
                    <th className="px-3 py-2 text-left">Role</th>
                    <th className="px-3 py-2 text-left">Joined</th>
                    <th className="px-3 py-2 text-left">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {members.map((m) => (
                    <tr key={m.id} className="border-t border-line/40">
                      <td className="px-3 py-2">{m.email}</td>
                      <td className="px-3 py-2 text-mute">{m.fullName || '—'}</td>
                      <td className="px-3 py-2">
                        <span className={`px-2 py-0.5 rounded text-xs ${
                          m.role === 'owner' ? 'bg-amber-500/20 text-amber-300' :
                          m.role === 'admin' ? 'bg-blue-500/20 text-blue-300' :
                          'bg-gray-500/20 text-gray-300'
                        }`}>{m.role}</span>
                      </td>
                      <td className="px-3 py-2 text-mute text-xs">{new Date(m.joinedAt).toLocaleDateString()}</td>
                      <td className="px-3 py-2">
                        <button onClick={() => removeMember(selectedOrg.id, m.userId)} className="text-xs text-red-400 hover:underline">Remove</button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      )}
    </div>
  );
}
