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
  suspendedAt: string | null;
  createdAt: string;
};

type UserSummary = {
  id: string;
  email: string;
  fullName: string;
  role: string;
  isPlatformAdmin: boolean;
  createdAt: string;
};

type UsageStats = {
  simulationsToday: number;
  simulationsLimit: number;
  activeSeats: number;
  seatsLimit: number;
  activeProjects: number;
  projectsLimit: number;
  llmCallsThisMonth: number;
};

export default function AdminPage() {
  const [orgs, setOrgs] = useState<Org[]>([]);
  const [users, setUsers] = useState<UserSummary[]>([]);
  const [stats, setStats] = useState<UsageStats | null>(null);
  const [selectedOrgId, setSelectedOrgId] = useState<string>('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void loadData();
  }, []);

  useEffect(() => {
    if (selectedOrgId) {
      void loadUsage(selectedOrgId);
    }
  }, [selectedOrgId]);

  async function loadData() {
    setLoading(true);
    try {
      const [orgsRes, usersRes] = await Promise.all([
        apiFetch(`/api/v1/admin/orgs`),
        apiFetch(`/api/v1/admin/users`),
      ]);
      if (orgsRes.ok) {
        const data = await orgsRes.json();
        setOrgs(data.data ?? []);
        if (data.data?.length > 0 && !selectedOrgId) {
          setSelectedOrgId(data.data[0].id);
        }
      }
      if (usersRes.ok) {
        const data = await usersRes.json();
        setUsers(data.data ?? []);
      }
    } catch {
      // API unavailable
    } finally {
      setLoading(false);
    }
  }

  async function loadUsage(orgId: string) {
    try {
      const res = await apiFetch(`/api/v1/admin/usage?orgId=${orgId}`);
      if (res.ok) {
        const data = await res.json();
        setStats(data.data ?? null);
      }
    } catch { /* ignore */ }
  }

  async function suspendOrg(orgId: string) {
    await apiFetch(`/api/v1/admin/orgs/suspend`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ orgId }),
    });
    void loadData();
  }

  async function unsuspendOrg(orgId: string) {
    await apiFetch(`/api/v1/admin/orgs/unsuspend`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ orgId }),
    });
    void loadData();
  }

  async function toggleAdmin(userId: string, current: boolean) {
    await apiFetch(`/api/v1/admin/users/promote`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ userId, isPlatformAdmin: !current }),
    });
    void loadData();
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-background p-8">
        <p className="text-mute animate-pulse">Loading admin panel...</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background p-8 space-y-8">
      <header className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Platform Admin</h1>
          <p className="text-sm text-mute">Manage organizations, users, API keys, and platform settings</p>
        </div>
        <Link href="/dashboard" className="text-sm text-accent underline">
          ← Back to Dashboard
        </Link>
      </header>

      {/* Navigation */}
      <nav className="flex gap-4 border-b pb-2">
        <Link href="/admin" className="text-sm font-semibold text-accent border-b-2 border-accent pb-1">Overview</Link>
        <Link href="/admin/orgs" className="text-sm text-mute hover:text-white">Organizations</Link>
        <Link href="/admin/users" className="text-sm text-mute hover:text-white">Users</Link>
        <Link href="/admin/api-keys" className="text-sm text-mute hover:text-white">API Keys</Link>
        <Link href="/admin/audit" className="text-sm text-mute hover:text-white">Audit Log</Link>
      </nav>

      {/* Usage Overview */}
      {stats && (
        <section className="grid gap-4 md:grid-cols-4">
          <StatCard label="Simulations Today" value={`${stats.simulationsToday}/${stats.simulationsLimit}`} />
          <StatCard label="Active Seats" value={`${stats.activeSeats}/${stats.seatsLimit}`} />
          <StatCard label="Active Projects" value={`${stats.activeProjects}/${stats.projectsLimit}`} />
          <StatCard label="LLM Calls (Month)" value={String(stats.llmCallsThisMonth)} />
        </section>
      )}

      {/* Organizations Table */}
      <section>
        <h2 className="text-lg font-semibold mb-3">Organizations ({orgs.length})</h2>
        <div className="overflow-auto rounded-lg border">
          <table className="min-w-full text-sm">
            <thead className="bg-black/30 text-mute">
              <tr>
                <th className="px-4 py-2 text-left">Name</th>
                <th className="px-4 py-2 text-left">Slug</th>
                <th className="px-4 py-2 text-left">Plan</th>
                <th className="px-4 py-2 text-left">Seats</th>
                <th className="px-4 py-2 text-left">Status</th>
                <th className="px-4 py-2 text-left">Actions</th>
              </tr>
            </thead>
            <tbody>
              {orgs.map((org) => (
                <tr key={org.id} className="border-t border-line/40 hover:bg-black/20">
                  <td className="px-4 py-2 font-medium">{org.name}</td>
                  <td className="px-4 py-2 text-mute">{org.slug}</td>
                  <td className="px-4 py-2">
                    <span className={`px-2 py-0.5 rounded text-xs font-medium ${
                      org.plan === 'enterprise' ? 'bg-purple-500/20 text-purple-300' :
                      org.plan === 'professional' ? 'bg-blue-500/20 text-blue-300' :
                      org.plan === 'starter' ? 'bg-green-500/20 text-green-300' :
                      'bg-gray-500/20 text-gray-300'
                    }`}>{org.plan}</span>
                  </td>
                  <td className="px-4 py-2">{org.maxSeats}</td>
                  <td className="px-4 py-2">
                    {org.suspendedAt ? (
                      <span className="text-red-400 text-xs font-medium">Suspended</span>
                    ) : (
                      <span className="text-emerald-400 text-xs font-medium">Active</span>
                    )}
                  </td>
                  <td className="px-4 py-2">
                    {org.suspendedAt ? (
                      <button onClick={() => unsuspendOrg(org.id)} className="text-xs text-emerald-400 hover:underline">Reactivate</button>
                    ) : (
                      <button onClick={() => suspendOrg(org.id)} className="text-xs text-red-400 hover:underline">Suspend</button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      {/* Users Table */}
      <section>
        <h2 className="text-lg font-semibold mb-3">Users ({users.length})</h2>
        <div className="overflow-auto rounded-lg border">
          <table className="min-w-full text-sm">
            <thead className="bg-black/30 text-mute">
              <tr>
                <th className="px-4 py-2 text-left">Email</th>
                <th className="px-4 py-2 text-left">Name</th>
                <th className="px-4 py-2 text-left">Role</th>
                <th className="px-4 py-2 text-left">Admin</th>
                <th className="px-4 py-2 text-left">Actions</th>
              </tr>
            </thead>
            <tbody>
              {users.map((user) => (
                <tr key={user.id} className="border-t border-line/40 hover:bg-black/20">
                  <td className="px-4 py-2">{user.email}</td>
                  <td className="px-4 py-2 text-mute">{user.fullName || '—'}</td>
                  <td className="px-4 py-2">{user.role}</td>
                  <td className="px-4 py-2">
                    {user.isPlatformAdmin ? (
                      <span className="text-amber-300 text-xs font-medium">Platform Admin</span>
                    ) : (
                      <span className="text-mute text-xs">—</span>
                    )}
                  </td>
                  <td className="px-4 py-2">
                    <button
                      onClick={() => toggleAdmin(user.id, user.isPlatformAdmin)}
                      className="text-xs text-accent hover:underline"
                    >
                      {user.isPlatformAdmin ? 'Revoke Admin' : 'Grant Admin'}
                    </button>
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

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border bg-panel p-4">
      <p className="text-xs text-mute">{label}</p>
      <p className="mt-1 text-xl font-bold">{value}</p>
    </div>
  );
}
