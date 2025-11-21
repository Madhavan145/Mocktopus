'use client';

import Link from 'next/link';
import { type FormEvent, useMemo, useState } from 'react';
import { ArrowLeft, Loader2, Mail, KeyRound } from 'lucide-react';

import { requestPasswordReset, resetPassword } from '@/components/auth/client';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('');
  const [token, setToken] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [generatedToken, setGeneratedToken] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [step, setStep] = useState<'request' | 'reset'>('request');

  const requestDisabled = useMemo(() => busy || email.trim() === '', [busy, email]);
  const resetDisabled = useMemo(() => busy || token.trim() === '' || newPassword.length < 12, [busy, token, newPassword]);

  const onRequest = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (requestDisabled) return;

    setBusy(true);
    setError(null);
    setMessage(null);
    try {
      const response = await requestPasswordReset({ email: email.trim().toLowerCase() });
      setGeneratedToken(response.resetToken ?? null);
      setMessage(response.status);
      setStep('reset');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to request reset token');
    } finally {
      setBusy(false);
    }
  };

  const onReset = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (resetDisabled) return;

    setBusy(true);
    setError(null);
    setMessage(null);
    try {
      const response = await resetPassword({ token: token.trim(), newPassword });
      setMessage(response.status);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to reset password');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center p-6">
      <div className="w-full max-w-sm space-y-6">
        <Link
          href="/login"
          className="inline-flex items-center gap-1.5 text-body-sm text-ink-secondary hover:text-ink transition-colors"
        >
          <ArrowLeft size={14} />
          Back to login
        </Link>

        <div className="space-y-2">
          <h1 className="text-title-lg">Reset password</h1>
          <p className="text-body-sm text-ink-secondary">
            {step === 'request'
              ? "Enter your email and we'll send you a reset token."
              : 'Enter the token and your new password.'}
          </p>
        </div>

        {step === 'request' ? (
          <form className="space-y-4" onSubmit={onRequest}>
            <Input
              id="email"
              label="Email address"
              type="email"
              placeholder="you@company.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="email"
            />
            <Button type="submit" disabled={requestDisabled} className="w-full">
              {busy ? (
                <><Loader2 size={16} className="animate-spin" /> Sending...</>
              ) : (
                <><Mail size={16} /> Send reset token</>
              )}
            </Button>
          </form>
        ) : (
          <>
            {generatedToken && (
              <div className="rounded-md border border-warning/30 bg-warning/5 p-3 space-y-1">
                <p className="text-caption font-medium text-warning">Development token</p>
                <p className="font-mono text-caption text-ink-secondary break-all select-all">{generatedToken}</p>
              </div>
            )}

            <form className="space-y-4" onSubmit={onReset}>
              <Input
                id="token"
                label="Reset token"
                placeholder="Paste your token"
                value={token}
                onChange={(e) => setToken(e.target.value)}
              />
              <Input
                id="new-password"
                label="New password"
                type="password"
                placeholder="Min 12 chars, upper/lower/number/symbol"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                autoComplete="new-password"
              />
              <Button type="submit" disabled={resetDisabled} className="w-full">
                {busy ? (
                  <><Loader2 size={16} className="animate-spin" /> Resetting...</>
                ) : (
                  <><KeyRound size={16} /> Reset password</>
                )}
              </Button>
            </form>

            <button
              type="button"
              onClick={() => setStep('request')}
              className="text-body-sm text-ink-secondary hover:text-ink transition-colors"
            >
              Request a new token
            </button>
          </>
        )}

        {error && (
          <div className="rounded-md bg-error/10 border border-error/20 px-3 py-2">
            <p className="text-caption text-error">{error}</p>
          </div>
        )}
        {message && (
          <div className="rounded-md bg-success/10 border border-success/20 px-3 py-2">
            <p className="text-caption text-success">{message}</p>
          </div>
        )}
      </div>
    </div>
  );
}
