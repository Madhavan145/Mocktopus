'use client';

import Link from 'next/link';
import { type FormEvent, useMemo, useState } from 'react';
import { Zap, ArrowRight, Loader2, Check, X } from 'lucide-react';

import { persistSession, registerUser } from '@/components/auth/client';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

export default function RegisterPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const checks = useMemo(() => ({
    length: password.length >= 12,
    upper: /[A-Z]/.test(password),
    lower: /[a-z]/.test(password),
    number: /[0-9]/.test(password),
    symbol: /[^A-Za-z0-9]/.test(password),
  }), [password]);

  const allPassed = Object.values(checks).every(Boolean);
  const disabled = useMemo(() => busy || email.trim() === '' || !allPassed, [busy, email, allPassed]);

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (disabled) return;

    setBusy(true);
    setError(null);
    try {
      const auth = await registerUser({
        email: email.trim().toLowerCase(),
        password,
        role: 'engineer',
      });
      persistSession(auth);
      window.location.href = '/dashboard';
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Registration failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex min-h-screen">
      {/* Left - Branding */}
      <div className="hidden lg:flex lg:w-1/2 flex-col justify-between bg-surface border-r p-12">
        <div className="flex items-center gap-2">
          <Zap size={24} className="text-accent" />
          <span className="text-title font-semibold">Moktopus</span>
        </div>
        <div className="space-y-4">
          <h2 className="text-display leading-tight">
            Build resilient<br />
            <span className="text-accent">architectures</span><br />
            with confidence.
          </h2>
          <p className="text-body text-ink-secondary max-w-md">
            Design, simulate, and validate your distributed systems
            in a safe sandbox environment.
          </p>
        </div>
        <p className="text-caption text-ink-muted">
          Start free. Scale when you're ready.
        </p>
      </div>

      {/* Right - Form */}
      <div className="flex flex-1 items-center justify-center p-6">
        <div className="w-full max-w-sm space-y-6">
          <div className="flex items-center gap-2 lg:hidden">
            <Zap size={20} className="text-accent" />
            <span className="text-body font-semibold">Moktopus</span>
          </div>

          <div className="space-y-2">
            <h1 className="text-title-lg">Create your account</h1>
            <p className="text-body-sm text-ink-secondary">
              Set up your workspace and start simulating.
            </p>
          </div>

          <form className="space-y-4" onSubmit={onSubmit}>
            <Input
              id="email"
              label="Work email"
              type="email"
              placeholder="you@company.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="email"
            />
            <Input
              id="password"
              label="Password"
              type="password"
              placeholder="Create a strong password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="new-password"
            />

            {/* Password strength indicator */}
            {password.length > 0 && (
              <div className="space-y-1.5 rounded-md bg-surface-elevated border p-3">
                {[
                  { key: 'length', label: '12+ characters' },
                  { key: 'upper', label: 'Uppercase letter' },
                  { key: 'lower', label: 'Lowercase letter' },
                  { key: 'number', label: 'Number' },
                  { key: 'symbol', label: 'Special character' },
                ].map(({ key, label }) => (
                  <div key={key} className="flex items-center gap-2 text-caption">
                    {checks[key as keyof typeof checks] ? (
                      <Check size={12} className="text-success" />
                    ) : (
                      <X size={12} className="text-ink-muted" />
                    )}
                    <span className={checks[key as keyof typeof checks] ? 'text-ink-secondary' : 'text-ink-muted'}>
                      {label}
                    </span>
                  </div>
                ))}
              </div>
            )}

            {error && (
              <div className="rounded-md bg-error/10 border border-error/20 px-3 py-2">
                <p className="text-caption text-error">{error}</p>
              </div>
            )}

            <Button type="submit" disabled={disabled} className="w-full">
              {busy ? (
                <>
                  <Loader2 size={16} className="animate-spin" />
                  Creating account...
                </>
              ) : (
                <>
                  Create account
                  <ArrowRight size={16} />
                </>
              )}
            </Button>
          </form>

          <p className="text-body-sm text-ink-secondary">
            Already have an account?{' '}
            <Link href="/login" className="text-accent hover:text-accent-hover transition-colors">
              Sign in
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}
