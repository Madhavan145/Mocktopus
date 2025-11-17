'use client';

import { useEffect } from 'react';

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error('Application error:', error);
  }, [error]);

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center space-y-4 max-w-md">
        <h1 className="text-2xl font-bold text-red-400">Something went wrong</h1>
        <p className="text-sm text-ink-muted">
          An unexpected error occurred. Please try again or contact support if the issue persists.
        </p>
        <div className="flex items-center justify-center gap-3">
          <button
            onClick={reset}
            className="rounded-md bg-accent px-4 py-2 text-sm font-medium text-black hover:bg-accent/90 transition-colors"
          >
            Try Again
          </button>
          <a
            href="/dashboard"
            className="rounded-md border px-4 py-2 text-sm hover:bg-white/5 transition-colors"
          >
            Back to Dashboard
          </a>
        </div>
      </div>
    </div>
  );
}
