'use client';

import { useCallback, useEffect, useState } from 'react';
import { ChevronDown, Building2, Check } from 'lucide-react';
import { apiFetch, parseV1, getSession, clearSession } from '@/lib/api';

type OrgOption = {
  id: string;
  name: string;
  slug: string;
  plan: string;
};

export function OrgSwitcher() {
  const [orgs, setOrgs] = useState<OrgOption[]>([]);
  const [currentOrgId, setCurrentOrgId] = useState<string>('');
  const [open, setOpen] = useState(false);
  const [switching, setSwitching] = useState(false);

  useEffect(() => {
    const session = getSession();
    if (session?.user?.orgId) {
      setCurrentOrgId(session.user.orgId);
    }

    const loadOrgs = async () => {
      try {
        const res = await apiFetch('/api/v1/org/list');
        if (res.ok) {
          const data = await parseV1<OrgOption[]>(res);
          setOrgs(data);
        }
      } catch { /* ignore */ }
    };
    void loadOrgs();
  }, []);

  const currentOrg = orgs.find((o) => o.id === currentOrgId);

  const handleSwitch = useCallback(async (orgId: string) => {
    if (orgId === currentOrgId) {
      setOpen(false);
      return;
    }

    setSwitching(true);
    try {
      const res = await apiFetch('/api/v1/org/switch', {
        method: 'POST',
        body: JSON.stringify({ orgId }),
      });
      if (!res.ok) throw new Error('switch failed');
      const data = await parseV1<{ accessToken: string; refreshToken: string; orgId: string; role: string }>(res);

      // Update stored session with new token
      const session = getSession();
      if (session) {
        const updated = {
          ...session,
          accessToken: data.accessToken,
          refreshToken: data.refreshToken,
          user: {
            ...session.user,
            orgId: data.orgId,
            role: data.role,
          },
        };
        window.localStorage.setItem('moktopus.auth', JSON.stringify(updated));
        // Update cookie for middleware
        document.cookie = `moktopus.session=${data.accessToken}; path=/; max-age=${session.expiresIn || 900}; SameSite=Lax`;
      }

      // Reload page to reflect new org context
      window.location.reload();
    } catch {
      // Ignore switch errors
    } finally {
      setSwitching(false);
      setOpen(false);
    }
  }, [currentOrgId]);

  if (orgs.length <= 1) {
    return currentOrg ? (
      <div className="flex items-center gap-2 px-3 py-2 text-caption text-ink-secondary">
        <Building2 size={14} />
        <span className="truncate">{currentOrg.name}</span>
      </div>
    ) : null;
  }

  return (
    <div className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-2 w-full px-3 py-2 rounded-md text-caption text-ink-secondary hover:bg-surface-elevated transition-colors"
      >
        <Building2 size={14} className="shrink-0" />
        <span className="truncate flex-1 text-left">{currentOrg?.name ?? 'Select org'}</span>
        <ChevronDown size={12} className={`shrink-0 transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>

      {open && (
        <div className="absolute left-0 right-0 top-full mt-1 z-50 rounded-md border bg-surface-elevated shadow-lg py-1">
          {orgs.map((org) => (
            <button
              key={org.id}
              onClick={() => handleSwitch(org.id)}
              disabled={switching}
              className="flex items-center gap-2 w-full px-3 py-2 text-caption hover:bg-surface transition-colors text-left"
            >
              <span className="flex-1 truncate">{org.name}</span>
              {org.id === currentOrgId && <Check size={12} className="text-accent shrink-0" />}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
