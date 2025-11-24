'use client';

import { useState } from 'react';
import { User, Lock, Shield, LogOut } from 'lucide-react';

import { AppShell } from '@/components/layout/app-shell';
import { Button } from '@/components/ui/button';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { useAuth } from '@/components/auth/AuthProvider';
import { apiFetch } from '@/lib/api';

export default function AccountPage() {
  const { user, logout } = useAuth();
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [passwordError, setPasswordError] = useState('');
  const [passwordSuccess, setPasswordSuccess] = useState('');
  const [changing, setChanging] = useState(false);

  const handlePasswordChange = async (e: React.FormEvent) => {
    e.preventDefault();
    setPasswordError('');
    setPasswordSuccess('');

    if (newPassword !== confirmPassword) {
      setPasswordError('New passwords do not match');
      return;
    }
    if (newPassword.length < 12) {
      setPasswordError('Password must be at least 12 characters');
      return;
    }

    setChanging(true);
    try {
      const res = await apiFetch('/api/v1/auth/password/change', {
        method: 'POST',
        body: JSON.stringify({ currentPassword, newPassword }),
      });

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setPasswordError(data?.error?.message || 'Failed to change password');
        return;
      }

      setPasswordSuccess('Password changed successfully. Please log in again.');
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
      // Redirect to login after password change since sessions are invalidated
      setTimeout(() => logout(), 2000);
    } catch (err: any) {
      setPasswordError(err?.message || 'An error occurred');
    } finally {
      setChanging(false);
    }
  };

  return (
    <AppShell>
      <div className="space-y-6 max-w-3xl animate-fade-in">
        <div>
          <h1 className="text-title-lg">Account</h1>
          <p className="text-body-sm text-ink-secondary mt-1">
            Manage your profile and security settings.
          </p>
        </div>

        {/* Profile Card */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <User className="h-5 w-5" />
              Profile
            </CardTitle>
            <CardDescription>Your account information</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-body-sm font-medium text-ink-secondary">Email</label>
                <p className="text-body-md mt-1">{user?.email || '—'}</p>
              </div>
              <div>
                <label className="text-body-sm font-medium text-ink-secondary">Role</label>
                <div className="mt-1">
                  <Badge variant="secondary">{user?.role || 'member'}</Badge>
                </div>
              </div>
              <div>
                <label className="text-body-sm font-medium text-ink-secondary">User ID</label>
                <p className="text-body-sm text-ink-tertiary mt-1 font-mono">{user?.id || '—'}</p>
              </div>
              <div>
                <label className="text-body-sm font-medium text-ink-secondary">Organization</label>
                <p className="text-body-sm text-ink-tertiary mt-1 font-mono">{user?.orgId || '—'}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Password Change Card */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Lock className="h-5 w-5" />
              Change Password
            </CardTitle>
            <CardDescription>Update your password. You will be logged out after changing.</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handlePasswordChange} className="space-y-4">
              <div>
                <label className="text-body-sm font-medium">Current Password</label>
                <Input
                  type="password"
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  placeholder="Enter current password"
                  required
                  className="mt-1"
                />
              </div>
              <div>
                <label className="text-body-sm font-medium">New Password</label>
                <Input
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  placeholder="At least 12 characters with upper/lower/number/symbol"
                  required
                  minLength={12}
                  className="mt-1"
                />
              </div>
              <div>
                <label className="text-body-sm font-medium">Confirm New Password</label>
                <Input
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  placeholder="Repeat new password"
                  required
                  className="mt-1"
                />
              </div>

              {passwordError && (
                <p className="text-sm text-red-500">{passwordError}</p>
              )}
              {passwordSuccess && (
                <p className="text-sm text-green-600">{passwordSuccess}</p>
              )}

              <Button type="submit" disabled={changing}>
                {changing ? 'Changing...' : 'Change Password'}
              </Button>
            </form>
          </CardContent>
        </Card>

        {/* Security Card */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Shield className="h-5 w-5" />
              Security
            </CardTitle>
            <CardDescription>Session and security management</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-body-md font-medium">Active Session</p>
                <p className="text-body-sm text-ink-secondary">Currently logged in from this browser</p>
              </div>
              <Badge variant="success">Active</Badge>
            </div>
            <div className="border-t pt-4">
              <Button variant="destructive" onClick={logout} className="flex items-center gap-2">
                <LogOut className="h-4 w-4" />
                Log Out
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </AppShell>
  );
}
