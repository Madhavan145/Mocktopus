'use client';

import { useState } from 'react';
import { LogIn, Save, Play, GitCompare, Zap } from 'lucide-react';
import Link from 'next/link';
import dynamic from 'next/dynamic';
import { Button } from '@/components/ui/button';

/**
 * Playground page — anonymous topology builder.
 * Users can build and experiment with topologies without logging in.
 * Save, Simulate, and Compare require authentication.
 */

const BuilderWorkspace = dynamic(
  () => import('@/components/builder/BuilderWorkspace'),
  { ssr: false, loading: () => <div className="flex-1 flex items-center justify-center"><p className="text-ink-muted">Loading builder...</p></div> }
);

export default function PlaygroundPage() {
  const [showAuthGate, setShowAuthGate] = useState(false);
  const [gatedAction, setGatedAction] = useState<string>('');

  const handleGatedAction = (action: string) => {
    setGatedAction(action);
    setShowAuthGate(true);
  };

  return (
    <div className="flex flex-col h-screen bg-surface">
      {/* Top bar */}
      <header className="flex items-center justify-between px-4 py-2 border-b bg-surface-elevated shrink-0">
        <div className="flex items-center gap-3">
          <Link href="/" className="flex items-center gap-2 hover:opacity-80 transition-opacity">
            <Zap size={18} className="text-accent" />
            <span className="text-body-sm font-semibold">Moktopus</span>
          </Link>
          <span className="text-caption text-ink-muted border-l pl-3 ml-1">Playground</span>
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleGatedAction('save')}
            className="gap-1.5"
          >
            <Save size={14} />
            Save
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleGatedAction('simulate')}
            className="gap-1.5"
          >
            <Play size={14} />
            Simulate
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleGatedAction('compare')}
            className="gap-1.5"
          >
            <GitCompare size={14} />
            Compare
          </Button>
          <div className="w-px h-5 bg-border mx-1" />
          <Link href="/login">
            <Button size="sm" className="gap-1.5">
              <LogIn size={14} />
              Sign In
            </Button>
          </Link>
        </div>
      </header>

      {/* Builder canvas — full BuilderWorkspace in playground mode */}
      <div className="flex-1 relative">
        <BuilderWorkspace projectId="playground" onAuthRequired={handleGatedAction} />
      </div>

      {/* Auth Gate Modal */}
      {showAuthGate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm animate-fade-in">
          <div className="w-full max-w-md rounded-xl border bg-surface-elevated p-8 shadow-2xl space-y-5">
            <div className="flex items-center gap-3">
              <div className="rounded-full bg-accent/10 p-2.5">
                <Zap size={20} className="text-accent" />
              </div>
              <div>
                <h2 className="text-title font-semibold">Sign in to {gatedAction}</h2>
                <p className="text-body-sm text-ink-secondary mt-0.5">
                  Create a free account to {gatedAction} topologies, run simulations, and collaborate with your team.
                </p>
              </div>
            </div>

            <div className="space-y-3 pt-2">
              <p className="text-caption text-ink-muted">
                Your current topology will be preserved — you can continue right where you left off.
              </p>
              <div className="flex gap-3">
                <Link href="/register" className="flex-1">
                  <Button className="w-full">Create Free Account</Button>
                </Link>
                <Link href="/login" className="flex-1">
                  <Button variant="outline" className="w-full">Sign In</Button>
                </Link>
              </div>
              <button
                type="button"
                onClick={() => setShowAuthGate(false)}
                className="w-full text-center text-caption text-ink-muted hover:text-ink-secondary transition-colors pt-1"
              >
                Continue building without saving
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
