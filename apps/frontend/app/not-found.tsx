import Link from 'next/link';

export default function NotFound() {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center space-y-4">
        <h1 className="text-6xl font-bold text-ink-muted">404</h1>
        <p className="text-lg text-ink-secondary">Page not found</p>
        <p className="text-sm text-ink-muted max-w-md">
          The page you&apos;re looking for doesn&apos;t exist or has been moved.
        </p>
        <Link
          href="/dashboard"
          className="inline-flex items-center gap-2 rounded-md bg-accent px-4 py-2 text-sm font-medium text-black hover:bg-accent/90 transition-colors"
        >
          Back to Dashboard
        </Link>
      </div>
    </div>
  );
}
