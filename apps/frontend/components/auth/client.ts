export const GATEWAY_URL = process.env.NEXT_PUBLIC_GATEWAY_URL ?? 'http://localhost:8080';

type V1Envelope<T> = {
  success: boolean;
  data: T;
  error: { code: string; message: string } | null;
};

type AuthResponse = {
  accessToken: string;
  refreshToken: string;
  tokenType?: string;
  expiresIn?: number;
  userId: string;
  orgId?: string;
  role: string;
  isAdmin?: boolean;
  user?: {
    id: string;
    email: string;
    role: string;
    orgId?: string;
    isAdmin?: boolean;
  };
};

type ForgotPasswordResponse = {
  status: string;
  resetToken?: string;
};

type ResetPasswordResponse = {
  status: string;
};

export async function registerUser(payload: { email: string; password: string; role: string }): Promise<AuthResponse> {
  const response = await fetch(`${GATEWAY_URL}/api/v1/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });

  return parseV1Response<AuthResponse>(response);
}

export async function loginUser(payload: { email: string; password: string }): Promise<AuthResponse> {
  const response = await fetch(`${GATEWAY_URL}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });

  return parseV1Response<AuthResponse>(response);
}

export async function requestPasswordReset(payload: { email: string }): Promise<ForgotPasswordResponse> {
  const response = await fetch(`${GATEWAY_URL}/api/v1/auth/password/forgot`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });

  return parseV1Response<ForgotPasswordResponse>(response);
}

export async function resetPassword(payload: { token: string; newPassword: string }): Promise<ResetPasswordResponse> {
  const response = await fetch(`${GATEWAY_URL}/api/v1/auth/password/reset`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });

  return parseV1Response<ResetPasswordResponse>(response);
}

export function persistSession(auth: AuthResponse): void {
  if (typeof window === 'undefined') {
    return;
  }

  // Normalize to consistent storage shape
  const session = {
    accessToken: auth.accessToken,
    refreshToken: auth.refreshToken,
    tokenType: auth.tokenType ?? 'Bearer',
    expiresIn: auth.expiresIn ?? 3600,
    user: auth.user ?? {
      id: auth.userId,
      email: '',
      role: auth.role,
      orgId: auth.orgId,
      isAdmin: auth.isAdmin ?? false,
    },
  };

  window.localStorage.setItem('moktopus.auth', JSON.stringify(session));

  // Set cookie for Next.js middleware (readable server-side for route protection)
  const maxAge = session.expiresIn || 3600;
  document.cookie = `moktopus.session=${auth.accessToken}; path=/; max-age=${maxAge}; SameSite=Lax`;
}

export function destroySession(): void {
  if (typeof window === 'undefined') return;
  window.localStorage.removeItem('moktopus.auth');
  document.cookie = 'moktopus.session=; path=/; max-age=0; SameSite=Lax';
}

async function parseV1Response<T>(response: Response): Promise<T> {
  const payload = (await response.json()) as Partial<V1Envelope<T>>;
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
