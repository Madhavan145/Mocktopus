'use client';

import { Sidebar } from './sidebar';

export function AppShell({ children, projectId }: { children: React.ReactNode; projectId?: string }) {
  return (
    <div className="flex min-h-screen">
      <Sidebar projectId={projectId} />
      <main className="ml-[var(--sidebar-width)] flex-1 p-6 lg:p-8 w-0">
        <div className="mx-auto w-full max-w-[1800px]">
          {children}
        </div>
      </main>
    </div>
  );
}
