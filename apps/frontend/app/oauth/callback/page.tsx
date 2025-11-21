'use client';

import { useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { persistSession } from '@/components/auth/client';

/**
 * OAuth callback page — receives tokens from the backend OAuth redirect
 * and persists the session before redirecting to dashboard.
 */
export default function OAuthCallbackPage() {
  const searchParams = useSearchParams();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const accessToken = searchParams.get('accessToken');
    const refreshToken = searchParams.get('refreshToken');
    const userId = searchParams.get('userId');
    const orgId = searchParams.get('orgId');
    const role = searchParams.get('role');
    const email = searchParams.get('email');
    const isAdmin = searchParams.get('isAdmin') === 'true';

    if (!accessToken || !refreshToken || !userId) {
      setError('OAuth authentication failed. Please try again.');
      return;
    }

    persistSession({
      accessToken,
      refreshToken,
      tokenType: 'Bearer',
      expiresIn: 900,
      userId,
      orgId: orgId ?? undefined,
      role: role ?? 'engineer',
      isAdmin,
      user: {
        id: userId,
        email: email ?? '',
        role: role ?? 'engineer',
        orgId: orgId ?? undefined,
        isAdmin,
      },
    });

    // Redirect to dashboard
    window.location.href = '/dashboard';
  }, [searchParams]);

  if (error) {
    return (
      <div className="flex h-screen items-center justify-center bg-surface">
        <div className="text-center space-y-4">
          <p className="text-red-400">{error}</p>
          <a href="/login" className="text-accent underline">Return to login</a>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-screen items-center justify-center bg-surface">
      <div className="text-center space-y-2">
        <div className="animate-spin h-8 w-8 border-2 border-accent border-t-transparent rounded-full mx-auto" />
        <p className="text-ink-muted">Completing sign in...</p>
      </div>
    </div>
  );
}
