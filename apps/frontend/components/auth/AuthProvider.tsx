'use client';

import { createContext, useContext, useEffect, useState, useCallback, useRef, type ReactNode } from 'react';
import { getSession, clearSession, GATEWAY_URL } from '@/lib/api';
import { persistSession, destroySession } from '@/components/auth/client';

type User = {
  id: string;
  email: string;
  role: string;
  orgId?: string;
  isAdmin?: boolean;
};

type AuthContextValue = {
  user: User | null;
  isAuthenticated: boolean;
  isAdmin: boolean;
  logout: () => void;
  refreshSession: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue>({
  user: null,
  isAuthenticated: false,
  isAdmin: false,
  logout: () => {},
  refreshSession: async () => {},
});

export function useAuth() {
  return useContext(AuthContext);
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const refreshTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const logout = useCallback(() => {
    // Invalidate session on server (best-effort)
    const session = getSession();
    if (session?.refreshToken) {
      fetch(`${GATEWAY_URL}/api/v1/auth/logout`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${session.accessToken}` },
        body: JSON.stringify({ refreshToken: session.refreshToken }),
      }).catch(() => {});
    }

    destroySession();
    clearSession();
    setUser(null);
    if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current);
    window.location.href = '/login';
  }, []);

  const refreshSession = useCallback(async () => {
    const session = getSession();
    if (!session?.refreshToken || !session?.user?.id) {
      logout();
      return;
    }

    const maxRetries = 3;
    let lastError: unknown;

    for (let attempt = 0; attempt < maxRetries; attempt++) {
      try {
        const response = await fetch(`${GATEWAY_URL}/api/v1/auth/refresh`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ userId: session.user.id, refreshToken: session.refreshToken }),
        });

        if (!response.ok) {
          if (attempt < maxRetries - 1) {
            // Wait with exponential backoff before retrying
            await new Promise(resolve => setTimeout(resolve, Math.pow(2, attempt + 1) * 1000));
            continue;
          }
          logout();
          return;
        }

        const payload = await response.json();
        const data = payload.data ?? payload;

        persistSession({
          accessToken: data.accessToken,
          refreshToken: data.refreshToken,
          tokenType: 'Bearer',
          expiresIn: data.expiresIn ?? 900,
          userId: data.userId ?? session.user.id,
          orgId: data.orgId ?? session.user.orgId,
          role: data.role ?? session.user.role,
          isAdmin: data.isAdmin ?? session.user.isAdmin,
          user: {
            id: data.userId ?? session.user.id,
            email: session.user.email,
            role: data.role ?? session.user.role,
            orgId: data.orgId ?? session.user.orgId,
            isAdmin: data.isAdmin ?? session.user.isAdmin,
          },
        });

        setUser({
          id: data.userId ?? session.user.id,
          email: session.user.email,
          role: data.role ?? session.user.role,
          orgId: data.orgId ?? session.user.orgId,
          isAdmin: data.isAdmin ?? session.user.isAdmin,
        });

        // Schedule next refresh via timer ref
        if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current);
        const nextRefreshMs = Math.max(((data.expiresIn ?? 900) - 60) * 1000, 10_000);
        refreshTimerRef.current = setTimeout(() => { void refreshSession(); }, nextRefreshMs);
        return;
      } catch (err) {
        lastError = err;
        if (attempt < maxRetries - 1) {
          await new Promise(resolve => setTimeout(resolve, Math.pow(2, attempt + 1) * 1000));
          continue;
        }
      }
    }

    // All retries failed
    console.error('Token refresh failed after retries:', lastError);
    logout();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [logout]);

  // Schedule automatic token refresh 1 minute before expiry
  const scheduleRefresh = useCallback((expiresIn: number) => {
    if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current);
    const refreshInMs = Math.max((expiresIn - 60) * 1000, 10_000);
    refreshTimerRef.current = setTimeout(() => {
      void refreshSession();
    }, refreshInMs);
  }, [refreshSession]);

  useEffect(() => {
    const session = getSession();
    if (session?.user) {
      setUser(session.user);
      scheduleRefresh(session.expiresIn ?? 900);
    }

    return () => {
      if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current);
    };
  }, [scheduleRefresh]);

  return (
    <AuthContext.Provider
      value={{
        user,
        isAuthenticated: !!user,
        isAdmin: user?.isAdmin === true,
        logout,
        refreshSession,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}
