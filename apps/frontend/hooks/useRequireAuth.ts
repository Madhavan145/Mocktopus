'use client';

import { useEffect, useState } from 'react';
import { getSession } from '@/lib/api';

/**
 * Hook that checks if user is authenticated.
 * Redirects to /login if no session exists.
 * Returns true when check is done and user is authenticated.
 */
export function useRequireAuth(): boolean {
  const [ready, setReady] = useState(false);

  useEffect(() => {
    const session = getSession();
    if (!session?.accessToken) {
      window.location.href = '/login';
      return;
    }
    setReady(true);
  }, []);

  return ready;
}
