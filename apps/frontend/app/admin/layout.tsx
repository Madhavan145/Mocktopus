'use client';

import { useEffect, useState } from 'react';
import { useAuth } from '@/components/auth/AuthProvider';

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { isAuthenticated, isAdmin } = useAuth();
  const [checked, setChecked] = useState(false);

  useEffect(() => {
    if (!isAuthenticated) {
      window.location.href = '/login';
      return;
    }
    if (!isAdmin) {
      window.location.href = '/dashboard';
      return;
    }
    setChecked(true);
  }, [isAuthenticated, isAdmin]);

  if (!checked) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <p className="text-ink-muted">Verifying access...</p>
      </div>
    );
  }

  return <>{children}</>;
}
