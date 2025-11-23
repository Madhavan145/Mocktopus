'use client';

import { useEffect, useState, useCallback } from 'react';
import { Users, Mail, Shield, Crown, Eye, Wrench, UserMinus, Plus, Building2 } from 'lucide-react';
import { AppShell } from '@/components/layout/app-shell';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { apiFetch, parseV1 } from '@/lib/api';
import { useAuth } from '@/components/auth/AuthProvider';

type Member = {
  userId: string;
  email: string;
  role: string;
  joinedAt: string;
};

type Invite = {
  id: string;
  email: string;
  role: string;
  expiresAt: string;
};

type OrgInfo = {
  id: string;
  name: string;
  slug: string;
  plan: string;
  maxSeats: number;
  maxProjects: number;
  maxSimulationsPerDay: number;
};

const ROLE_ICONS: Record<string, React.ReactNode> = {
  owner: <Crown size={14} className="text-amber-400" />,
  admin: <Shield size={14} className="text-blue-400" />,
  engineer: <Wrench size={14} className="text-green-400" />,
  viewer: <Eye size={14} className="text-ink-muted" />,
};

const ROLE_OPTIONS = ['admin', 'engineer', 'viewer'] as const;

export default function OrgSettingsPage() {
  const { user } = useAuth();
  const [org, setOrg] = useState<OrgInfo | null>(null);
  const [members, setMembers] = useState<Member[]>([]);
  const [invites, setInvites] = useState<Invite[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Invite form
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteRole, setInviteRole] = useState<string>('engineer');
  const [inviting, setInviting] = useState(false);
  const [inviteSuccess, setInviteSuccess] = useState<string | null>(null);

  // Org name edit
  const [editingName, setEditingName] = useState(false);
  const [newName, setNewName] = useState('');

  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      const [orgRes, membersRes, invitesRes] = await Promise.all([
        apiFetch('/api/v1/org'),
        apiFetch('/api/v1/org/members'),
        apiFetch('/api/v1/org/invites'),
      ]);

      if (orgRes.ok) setOrg(await parseV1<OrgInfo>(orgRes));
      if (membersRes.ok) {
        const data = await parseV1<{ members: Member[]; callerRole: string } | Member[]>(membersRes);
        // Support both old format (array) and new format (object with callerRole)
        if (Array.isArray(data)) {
          setMembers(data);
        } else {
          setMembers(data.members ?? []);
        }
      }
      if (invitesRes.ok) {
        setInvites(await parseV1<Invite[]>(invitesRes));
      } else {
        setInvites([]);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load org data');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void loadData(); }, [loadData]);

  const currentUserRole = members.find((m) => m.userId === user?.id)?.role ?? 'viewer';
  const canManage = currentUserRole === 'owner' || currentUserRole === 'admin';
  const isOwner = currentUserRole === 'owner';

  const handleInvite = async () => {
    if (!inviteEmail.trim()) return;
    setInviting(true);
    setInviteSuccess(null);
    try {
      const res = await apiFetch('/api/v1/org/invite', {
        method: 'POST',
        body: JSON.stringify({ email: inviteEmail.trim(), role: inviteRole }),
      });
      if (!res.ok) throw new Error((await res.text()) || 'Invite failed');
      const data = await parseV1<{ token: string; email: string }>(res);
      setInviteSuccess(`Invite sent to ${data.email}`);
      setInviteEmail('');
      void loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Invite failed');
    } finally {
      setInviting(false);
    }
  };

  const handleRoleChange = async (userId: string, newRole: string) => {
    try {
      const res = await apiFetch('/api/v1/org/members/role', {
        method: 'PUT',
        body: JSON.stringify({ userId, role: newRole }),
      });
      if (!res.ok) throw new Error((await res.text()) || 'Role update failed');
      void loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update role');
    }
  };

  const handleRemove = async (userId: string, email: string) => {
    if (!confirm(`Remove ${email} from the organization?`)) return;
    try {
      const res = await apiFetch(`/api/v1/org/members?userId=${userId}`, { method: 'DELETE' });
      if (!res.ok) throw new Error((await res.text()) || 'Remove failed');
      void loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove member');
    }
  };

  const handleUpdateName = async () => {
    if (!newName.trim()) return;
    try {
      const res = await apiFetch('/api/v1/org', {
        method: 'PUT',
        body: JSON.stringify({ name: newName.trim() }),
      });
      if (!res.ok) throw new Error((await res.text()) || 'Update failed');
      setEditingName(false);
      void loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update name');
    }
  };

  if (loading) {
    return (
      <AppShell>
        <div className="flex items-center justify-center min-h-[400px]">
          <p className="text-ink-muted">Loading organization settings...</p>
        </div>
      </AppShell>
    );
  }

  return (
    <AppShell>
      <div className="space-y-8 max-w-4xl">
        <div className="space-y-2">
          <h1 className="text-title-lg flex items-center gap-3">
            <Building2 size={24} className="text-accent" />
            Organization Settings
          </h1>
          <p className="text-body-sm text-ink-secondary">
            Manage your organization, members, and roles.
          </p>
        </div>

        {error && (
          <div className="rounded-md bg-error/10 border border-error/20 px-4 py-3">
            <p className="text-body-sm text-error">{error}</p>
            <button onClick={() => setError(null)} className="text-caption text-error/70 hover:text-error mt-1">Dismiss</button>
          </div>
        )}

        {/* Org Info Card */}
        {org && (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center justify-between">
                <span>Organization</span>
                <Badge variant="secondary">{org.plan}</Badge>
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center gap-4">
                {editingName ? (
                  <div className="flex items-center gap-2 flex-1">
                    <Input
                      value={newName}
                      onChange={(e) => setNewName(e.target.value)}
                      placeholder="Organization name"
                      className="max-w-xs"
                    />
                    <Button size="sm" onClick={handleUpdateName}>Save</Button>
                    <Button size="sm" variant="outline" onClick={() => setEditingName(false)}>Cancel</Button>
                  </div>
                ) : (
                  <div className="flex items-center gap-3">
                    <span className="text-body font-medium">{org.name}</span>
                    <span className="text-caption text-ink-muted">({org.slug})</span>
                    {canManage && (
                      <button
                        onClick={() => { setNewName(org.name); setEditingName(true); }}
                        className="text-caption text-accent hover:text-accent-hover"
                      >
                        Edit
                      </button>
                    )}
                  </div>
                )}
              </div>
              <div className="grid grid-cols-3 gap-4 pt-2">
                <div className="text-center p-3 rounded-md bg-surface">
                  <p className="text-title font-semibold">{members.length}/{org.maxSeats}</p>
                  <p className="text-caption text-ink-muted">Seats</p>
                </div>
                <div className="text-center p-3 rounded-md bg-surface">
                  <p className="text-title font-semibold">{org.maxProjects}</p>
                  <p className="text-caption text-ink-muted">Max Projects</p>
                </div>
                <div className="text-center p-3 rounded-md bg-surface">
                  <p className="text-title font-semibold">{org.maxSimulationsPerDay}</p>
                  <p className="text-caption text-ink-muted">Sims/Day</p>
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Members */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Users size={18} />
              Members ({members.length})
            </CardTitle>
          </CardHeader>
          <CardContent>
            {!canManage && (
              <div className="mb-3 rounded-md bg-surface px-3 py-2 text-caption text-ink-muted">
                You have view-only access. Contact your org owner or admin to manage members.
              </div>
            )}
            <div className="divide-y">
              {members.map((member) => (
                <div key={member.userId} className="flex items-center justify-between py-3">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-full bg-accent/10 flex items-center justify-center text-caption font-semibold text-accent">
                      {member.email.charAt(0).toUpperCase()}
                    </div>
                    <div>
                      <p className="text-body-sm font-medium">{member.email}</p>
                      <p className="text-caption text-ink-muted">Joined {new Date(member.joinedAt).toLocaleDateString()}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    <div className="flex items-center gap-1.5">
                      {ROLE_ICONS[member.role]}
                      {isOwner && member.userId !== user?.id ? (
                        <select
                          value={member.role}
                          onChange={(e) => handleRoleChange(member.userId, e.target.value)}
                          className="text-caption bg-surface border rounded px-1.5 py-0.5"
                        >
                          <option value="admin">Admin</option>
                          <option value="engineer">Engineer</option>
                          <option value="viewer">Viewer</option>
                        </select>
                      ) : (
                        <span className="text-caption capitalize">{member.role}</span>
                      )}
                    </div>
                    {canManage && member.userId !== user?.id && member.role !== 'owner' && (
                      <button
                        onClick={() => handleRemove(member.userId, member.email)}
                        className="p-1 rounded text-ink-muted hover:text-error transition-colors"
                        title="Remove member"
                      >
                        <UserMinus size={14} />
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Invite Member */}
        {canManage && (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Mail size={18} />
                Invite Member
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {inviteSuccess && (
                <div className="rounded-md bg-success/10 border border-success/20 px-3 py-2">
                  <p className="text-caption text-success">{inviteSuccess}</p>
                </div>
              )}
              <div className="flex items-end gap-3">
                <div className="flex-1">
                  <Input
                    label="Email address"
                    type="email"
                    placeholder="colleague@company.com"
                    value={inviteEmail}
                    onChange={(e) => setInviteEmail(e.target.value)}
                  />
                </div>
                <div className="w-32">
                  <label className="text-caption font-medium text-ink-secondary block mb-1.5">Role</label>
                  <select
                    value={inviteRole}
                    onChange={(e) => setInviteRole(e.target.value)}
                    className="w-full rounded-md border bg-surface px-2.5 py-2 text-body-sm"
                  >
                    {ROLE_OPTIONS.map((r) => (
                      <option key={r} value={r}>{r.charAt(0).toUpperCase() + r.slice(1)}</option>
                    ))}
                  </select>
                </div>
                <Button onClick={handleInvite} disabled={inviting || !inviteEmail.trim()} className="gap-1.5">
                  <Plus size={14} />
                  {inviting ? 'Sending...' : 'Invite'}
                </Button>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Pending Invites */}
        {invites.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle>Pending Invites ({invites.length})</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="divide-y">
                {invites.map((inv) => (
                  <div key={inv.id} className="flex items-center justify-between py-2.5">
                    <div>
                      <p className="text-body-sm">{inv.email}</p>
                      <p className="text-caption text-ink-muted">Expires {new Date(inv.expiresAt).toLocaleDateString()}</p>
                    </div>
                    <Badge variant="secondary">{inv.role}</Badge>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        )}

        {/* Roles Explanation */}
        <Card>
          <CardHeader>
            <CardTitle>Role Permissions</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-2 p-3 rounded-md bg-surface">
                <div className="flex items-center gap-2">{ROLE_ICONS.owner} <span className="text-body-sm font-medium">Owner</span></div>
                <p className="text-caption text-ink-muted">Full control. Manage billing, members, and all settings. Transfer ownership.</p>
              </div>
              <div className="space-y-2 p-3 rounded-md bg-surface">
                <div className="flex items-center gap-2">{ROLE_ICONS.admin} <span className="text-body-sm font-medium">Admin</span></div>
                <p className="text-caption text-ink-muted">Invite/remove members. Manage projects and org settings.</p>
              </div>
              <div className="space-y-2 p-3 rounded-md bg-surface">
                <div className="flex items-center gap-2">{ROLE_ICONS.engineer} <span className="text-body-sm font-medium">Engineer</span></div>
                <p className="text-caption text-ink-muted">Create projects, build topologies, run simulations. Cannot manage members.</p>
              </div>
              <div className="space-y-2 p-3 rounded-md bg-surface">
                <div className="flex items-center gap-2">{ROLE_ICONS.viewer} <span className="text-body-sm font-medium">Viewer</span></div>
                <p className="text-caption text-ink-muted">Read-only access to projects and simulation results.</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </AppShell>
  );
}
