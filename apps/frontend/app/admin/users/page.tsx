'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import { apiFetch } from '@/lib/api';

type UserSummary = {
  id: string;
  email: string;
  fullName: string;
  role: string;
  isPlatformAdmin: boolean;
  createdAt: string;
};

export default function UsersPage() {
  const [users, setUsers] = useState<UserSummary[]>([]);
  const [search, setSearch] = useState('');

  useEffect(() => { void loadUsers(); }, []);

  async function loadUsers() {
    const res = await apiFetch(`/api/v1/admin/users`);
    if (res.ok) {
      const data = await res.json();
      setUsers(data.data ?? []);
    }
  }

  async function toggleAdmin(userId: string, current: boolean) {
    await apiFetch(`/api/v1/admin/users/promote`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ userId, isPlatformAdmin: !current }),
    });
    void loadUsers();
  }

  const filtered = users.filter(
    (u) => u.email.toLowerCase().includes(search.toLowerCase()) || u.fullName.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="min-h-screen bg-background p-8 space-y-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Users</h1>
        <Link href="/admin" className="text-sm text-mute hover:text-white">← Admin Home</Link>
      </header>

      <input
        className="w-full max-w-md rounded border bg-black/30 px-4 py-2 text-sm"
        placeholder="Search by email or name..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
      />

      <div className="overflow-auto rounded-lg border">
        <table className="min-w-full text-sm">
          <thead className="bg-black/30 text-mute">
            <tr>
              <th className="px-4 py-2 text-left">Email</th>
              <th className="px-4 py-2 text-left">Name</th>
              <th className="px-4 py-2 text-left">Role</th>
              <th className="px-4 py-2 text-left">Platform Admin</th>
              <th className="px-4 py-2 text-left">Created</th>
              <th className="px-4 py-2 text-left">Actions</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((user) => (
              <tr key={user.id} className="border-t border-line/40 hover:bg-black/20">
                <td className="px-4 py-2">{user.email}</td>
                <td className="px-4 py-2 text-mute">{user.fullName || '—'}</td>
                <td className="px-4 py-2">{user.role}</td>
                <td className="px-4 py-2">
                  {user.isPlatformAdmin ? (
                    <span className="text-amber-300 text-xs font-medium px-2 py-0.5 rounded bg-amber-500/20">Admin</span>
                  ) : (
                    <span className="text-mute text-xs">—</span>
                  )}
                </td>
                <td className="px-4 py-2 text-xs text-mute">{new Date(user.createdAt).toLocaleDateString()}</td>
                <td className="px-4 py-2">
                  <button
                    onClick={() => toggleAdmin(user.id, user.isPlatformAdmin)}
                    className={`text-xs hover:underline ${user.isPlatformAdmin ? 'text-red-400' : 'text-accent'}`}
                  >
                    {user.isPlatformAdmin ? 'Revoke Admin' : 'Grant Admin'}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <p className="text-xs text-mute">{filtered.length} of {users.length} users shown</p>
    </div>
  );
}
