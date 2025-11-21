/**
 * Authenticated API client — wraps fetch with Bearer token from localStorage.
 * Automatically handles 401 by clearing session and redirecting to /login.
 */

export const GATEWAY_URL = process.env.NEXT_PUBLIC_GATEWAY_URL ?? 'http://localhost:8080';

const SESSION_KEY = 'moktopus.auth';

type StoredSession = {
  accessToken: string;
  refreshToken: string;
  tokenType: string;
  expiresIn: number;
  user: { id: string; email: string; role: string; orgId?: string; isAdmin?: boolean };
};

export function getSession(): StoredSession | null {
  if (typeof window === 'undefined') return null;
  const raw = window.localStorage.getItem(SESSION_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as StoredSession;
  } catch {
    return null;
  }
}

export function clearSession(): void {
  if (typeof window === 'undefined') return;
  window.localStorage.removeItem(SESSION_KEY);
  document.cookie = 'moktopus.session=; path=/; max-age=0; SameSite=Lax';
}

/**
 * Authenticated fetch wrapper.
 * - Attaches Authorization header from stored session
 * - Redirects to /login on 401
 */
export async function apiFetch(path: string, init?: RequestInit): Promise<Response> {
  const session = getSession();
  const headers = new Headers(init?.headers);

  if (session?.accessToken) {
    headers.set('Authorization', `Bearer ${session.accessToken}`);
  }

  if (!headers.has('Content-Type') && init?.body && typeof init.body === 'string') {
    headers.set('Content-Type', 'application/json');
  }

  const url = path.startsWith('http') ? path : `${GATEWAY_URL}${path}`;

  const response = await fetch(url, { ...init, headers });

  if (response.status === 401) {
    clearSession();
    if (typeof window !== 'undefined') {
      window.location.href = '/login';
    }
  }

  if (response.status === 403) {
    const text = await response.text().catch(() => '');
    let message = 'You do not have permission to access this resource';
    try {
      const parsed = JSON.parse(text);
      if (parsed?.error?.message) message = parsed.error.message;
    } catch {}
    throw new ApiPermissionError(message);
  }

  return response;
}

/**
 * Error thrown when user is authenticated but lacks permission (403).
 */
export class ApiPermissionError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'ApiPermissionError';
  }
}

type V1Envelope<T> = {
  success: boolean;
  data: T;
  error: { code: string; message: string } | null;
};

/**
 * Parse a v1 JSON envelope response.
 */
export async function parseV1<T>(response: Response): Promise<T> {
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
  if (typeof payload !== 'object' || payload === null || typeof payload.success !== 'boolean') {
    throw new Error('Invalid API response envelope');
  }
  if (!payload.success) {
    throw new Error(payload.error?.message ?? `Request failed with status ${response.status}`);
  }
  if (!('data' in payload)) {
    throw new Error('API response missing data');
  }
  return payload.data as T;
}

/**
 * Convenience: apiFetch + parse v1 envelope in one call.
 */
export async function apiGet<T>(path: string): Promise<T> {
  const res = await apiFetch(path);
  if (!res.ok) {
    const text = await res.text().catch(() => '');
    throw new Error(text || `Request failed with status ${res.status}`);
  }
  return parseV1<T>(res);
}

export async function apiPost<T>(path: string, body: unknown): Promise<T> {
  const res = await apiFetch(path, {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const text = await res.text().catch(() => '');
    throw new Error(text || `Request failed with status ${res.status}`);
  }
  return parseV1<T>(res);
}

export async function apiPut<T>(path: string, body: unknown): Promise<T> {
  const res = await apiFetch(path, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const text = await res.text().catch(() => '');
    throw new Error(text || `Request failed with status ${res.status}`);
  }
  return parseV1<T>(res);
}

export async function apiDelete(path: string): Promise<void> {
  const res = await apiFetch(path, { method: 'DELETE' });
  if (!res.ok) {
    const text = await res.text().catch(() => '');
    throw new Error(text || `Request failed with status ${res.status}`);
  }
}
