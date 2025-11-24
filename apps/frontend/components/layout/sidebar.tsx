'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import {
  LayoutDashboard,
  Layers,
  Play,
  RotateCcw,
  GitCompare,
  DollarSign,
  Lightbulb,
  Settings,
  Users,
  CreditCard,
  Activity,
  Menu,
  X,
  Zap,
  Cloud,
  FileCode,
  LayoutTemplate,
  Building2,
  LogOut,
  User,
} from 'lucide-react';
import { useState, useCallback } from 'react';
import { cn } from '@/components/ui/cn';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
import { OrgSwitcher } from './OrgSwitcher';
import { useAuth } from '@/components/auth/AuthProvider';

interface NavItem {
  href: string;
  label: string;
  icon: React.ReactNode;
  section?: string;
}

export function Sidebar({ projectId }: { projectId?: string }) {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);
  const [showLogoutConfirm, setShowLogoutConfirm] = useState(false);
  const { user, logout } = useAuth();

  const handleSignOut = useCallback(() => {
    setShowLogoutConfirm(true);
  }, []);

  const confirmLogout = useCallback(() => {
    setShowLogoutConfirm(false);
    logout();
  }, [logout]);

  const pid = projectId ?? 'demo';

  const mainNav: NavItem[] = [
    { href: '/dashboard', label: 'Dashboard', icon: <LayoutDashboard size={18} /> },
    { href: '/observability', label: 'Observability', icon: <Activity size={18} /> },
  ];

  const projectNav: NavItem[] = [
    { href: `/project/${pid}/builder`, label: 'Builder', icon: <Layers size={18} />, section: 'Project' },
    { href: `/project/${pid}/simulate`, label: 'Simulate', icon: <Play size={18} /> },
    { href: `/project/${pid}/replay`, label: 'Replay', icon: <RotateCcw size={18} /> },
    { href: `/project/${pid}/compare`, label: 'Compare', icon: <GitCompare size={18} /> },
    { href: `/project/${pid}/cost`, label: 'Cost', icon: <DollarSign size={18} /> },
    { href: `/project/${pid}/recommendations`, label: 'Insights', icon: <Lightbulb size={18} /> },
    { href: `/project/${pid}/catalog`, label: 'Cloud Catalog', icon: <Cloud size={18} /> },
    { href: `/project/${pid}/iac`, label: 'IaC Import', icon: <FileCode size={18} /> },
    { href: `/project/${pid}/templates`, label: 'Templates', icon: <LayoutTemplate size={18} /> },
  ];

  const settingsNav: NavItem[] = [
    { href: '/settings', label: 'Settings', icon: <Settings size={18} />, section: 'Workspace' },
    { href: '/settings/org', label: 'Organization', icon: <Building2 size={18} /> },
    { href: '/team', label: 'Team', icon: <Users size={18} /> },
    { href: '/billing', label: 'Billing', icon: <CreditCard size={18} /> },
  ];

  const isActive = (href: string) => pathname === href || pathname.startsWith(href + '/');

  const renderNavItem = (item: NavItem) => (
    <Link
      key={item.href}
      href={item.href}
      className={cn(
        'nav-item',
        isActive(item.href) && 'active'
      )}
    >
      {item.icon}
      {!collapsed && <span>{item.label}</span>}
    </Link>
  );

  const renderSection = (label: string) =>
    !collapsed && (
      <p className="mb-1 mt-5 px-3 text-[11px] font-semibold uppercase tracking-wider text-ink-muted">{label}</p>
    );

  return (
    <aside
      className={cn(
        'fixed left-0 top-0 z-40 flex h-full flex-col border-r bg-surface transition-all duration-200',
        collapsed ? 'w-[60px]' : 'w-[var(--sidebar-width)]'
      )}
    >
      {/* Logo */}
      <div className="flex h-14 items-center justify-between border-b px-4">
        {!collapsed && (
          <div className="flex items-center gap-2">
            <Zap size={20} className="text-accent" />
            <span className="text-body font-semibold">Moktopus</span>
          </div>
        )}
        <button
          onClick={() => setCollapsed(!collapsed)}
          className="rounded-md p-1.5 text-ink-muted hover:bg-surface-elevated hover:text-ink transition-colors"
        >
          {collapsed ? <Menu size={16} /> : <X size={16} />}
        </button>
      </div>

      {/* Org Switcher */}
      {!collapsed && (
        <div className="border-b px-2 py-2">
          <OrgSwitcher />
        </div>
      )}

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto px-2 py-3">
        {renderSection('Overview')}
        {mainNav.map(renderNavItem)}
        {renderSection('Project')}
        {projectNav.map(renderNavItem)}
        {renderSection('Workspace')}
        {settingsNav.map(renderNavItem)}
      </nav>

      {/* User profile & Logout */}
      <div className="border-t px-3 py-3 space-y-2">
        {!collapsed && user && (
          <div className="flex items-center gap-2 px-1">
            <div className="flex h-7 w-7 items-center justify-center rounded-full bg-accent/20 text-accent">
              <User size={14} />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-caption font-medium truncate">{user.email}</p>
              <p className="text-[10px] text-ink-muted capitalize">{user.role}</p>
            </div>
          </div>
        )}
        <button
          onClick={handleSignOut}
          className={cn(
            'flex items-center gap-2 w-full rounded-md px-3 py-2 text-caption text-ink-muted hover:text-red-400 hover:bg-red-500/10 transition-colors',
            collapsed && 'justify-center px-0'
          )}
        >
          <LogOut size={16} />
          {!collapsed && <span>Sign out</span>}
        </button>
      </div>

      <ConfirmDialog
        open={showLogoutConfirm}
        onConfirm={confirmLogout}
        onCancel={() => setShowLogoutConfirm(false)}
        title="Sign out?"
        description="You will be redirected to the login page. Any unsaved changes will be lost."
        confirmLabel="Sign out"
        cancelLabel="Cancel"
        variant="destructive"
      />
    </aside>
  );
}
