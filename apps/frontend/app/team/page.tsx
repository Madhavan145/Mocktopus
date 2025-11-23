'use client';

import { useEffect, useState } from 'react';
import { UserPlus, Users, Mail, Shield } from 'lucide-react';

import { AppShell } from '@/components/layout/app-shell';
import { Button } from '@/components/ui/button';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { apiFetch } from '@/lib/api';

type TeamMember = { email: string; role: string };
type V1Envelope<T> = { success: boolean; data: T; error: { code: string; message: string } | null };

export default function TeamPage() {
  const [projectId] = useState('demo');
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteRole, setInviteRole] = useState('viewer');
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const loadMembers = async () => {
    setError(null);
    try {
      const response = await apiFetch(`/api/v1/team/members?projectId=${encodeURIComponent(projectId)}`);
      if (!response.ok) throw new Error('unable to load team members');
      const payload = await parseV1Data<{ members: TeamMember[] }>(response);
      setMembers(payload.members);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'unable to load team members');
    }
  };

  useEffect(() => { void loadMembers(); }, [projectId]);

  const invite = async () => {
    if (inviteEmail.trim() === '') { setError('Email is required'); return; }
    setError(null);
    setMessage(null);
    try {
      const response = await apiFetch(`/api/v1/team/invite`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ projectId, email: inviteEmail.trim().toLowerCase(), role: inviteRole }),
      });
      if (!response.ok) throw new Error('invite failed');
      await parseV1Data<{ member: TeamMember }>(response);
      setInviteEmail('');
      setMessage('Invitation sent successfully');
      await loadMembers();
      setTimeout(() => setMessage(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'invite failed');
    }
  };

  const roleColor = (role: string) => {
    switch (role) {
      case 'owner': return 'default' as const;
      case 'engineer': return 'info' as const;
      default: return 'secondary' as const;
    }
  };

  return (
    <AppShell>
      <div className="space-y-6 max-w-5xl animate-fade-in">
        <div>
          <h1 className="text-title-lg">Team</h1>
          <p className="text-body-sm text-ink-secondary mt-1">
            Manage project collaborators and role-based access.
          </p>
        </div>

        {/* Invite */}
        <Card>
          <CardHeader>
            <CardTitle>Invite Member</CardTitle>
            <CardDescription>Add a collaborator to the {projectId} project.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col md:flex-row gap-3">
              <div className="flex-1">
                <Input
                  placeholder="member@company.com"
                  type="email"
                  value={inviteEmail}
                  onChange={(e) => setInviteEmail(e.target.value)}
                />
              </div>
              <div className="space-y-1.5">
                <select
                  className="flex h-9 w-full rounded-md border bg-surface px-3 py-2 text-body-sm text-ink transition-colors hover:border-ink-muted focus:border-accent focus:ring-1 focus:ring-accent focus:outline-none"
                  value={inviteRole}
                  onChange={(e) => setInviteRole(e.target.value)}
                >
                  <option value="owner">Owner</option>
                  <option value="engineer">Engineer</option>
                  <option value="viewer">Viewer</option>
                </select>
              </div>
              <Button onClick={invite}>
                <UserPlus size={14} /> Invite
              </Button>
            </div>

            {message && (
              <div className="mt-3 rounded-md bg-success/10 border border-success/20 px-3 py-2">
                <p className="text-caption text-success">{message}</p>
              </div>
            )}
            {error && (
              <div className="mt-3 rounded-md bg-error/10 border border-error/20 px-3 py-2">
                <p className="text-caption text-error">{error}</p>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Members List */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>Members</CardTitle>
                <CardDescription>{members.length} team member{members.length !== 1 ? 's' : ''}</CardDescription>
              </div>
              <Users size={18} className="text-ink-muted" />
            </div>
          </CardHeader>
          <CardContent>
            {members.length === 0 ? (
              <div className="flex flex-col items-center py-8 text-center">
                <Users size={28} className="text-ink-muted mb-2" />
                <p className="text-body-sm text-ink-secondary">No team members yet.</p>
              </div>
            ) : (
              <div className="space-y-2">
                {members.map((member) => (
                  <div
                    key={member.email}
                    className="flex items-center justify-between rounded-lg border bg-surface-elevated p-3"
                  >
                    <div className="flex items-center gap-3">
                      <div className="flex h-8 w-8 items-center justify-center rounded-full bg-accent-muted text-accent text-caption font-semibold">
                        {member.email.charAt(0).toUpperCase()}
                      </div>
                      <div>
                        <p className="text-body-sm font-medium">{member.email}</p>
                      </div>
                    </div>
                    <Badge variant={roleColor(member.role)}>
                      <Shield size={10} className="mr-1" />
                      {member.role}
                    </Badge>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </AppShell>
  );
}

async function parseV1Data<T>(response: Response): Promise<T> {
  const text = await response.text();
  if (!text || !text.trim()) {
    throw new Error(`Empty response body (status ${response.status})`);
  }
  let payload: Partial<V1Envelope<T>>;
  try {
    payload = JSON.parse(text) as Partial<V1Envelope<T>>;
  } catch {
    throw new Error(`Invalid JSON response (status ${response.status})`);
  }
  if (typeof payload !== 'object' || payload === null || payload.success !== true) {
    throw new Error(payload?.error?.message ?? 'Invalid response envelope');
  }
  return payload.data as T;
}
