import type { Metadata, Viewport } from 'next';
import { AuthProvider } from '@/components/auth/AuthProvider';
import './globals.css';

export const metadata: Metadata = {
  title: {
    default: 'Moktopus — Infrastructure Simulation Platform',
    template: '%s | Moktopus',
  },
  description: 'Simulate, analyze, and optimize your cloud infrastructure architecture before deployment.',
  applicationName: 'Moktopus',
  keywords: ['infrastructure', 'simulation', 'cloud', 'architecture', 'DevOps', 'IaC'],
  authors: [{ name: 'Moktopus' }],
  robots: 'noindex, nofollow',
};

export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  maximumScale: 5,
  themeColor: '#0f172a',
  colorScheme: 'dark',
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en" className="dark" suppressHydrationWarning>
      <body className="font-sans antialiased bg-surface text-ink min-h-screen">
        <a href="#main-content" className="sr-only focus:not-sr-only focus:absolute focus:top-2 focus:left-2 focus:z-[9999] focus:rounded-md focus:bg-accent focus:px-4 focus:py-2 focus:text-white">
          Skip to main content
        </a>
        <main id="main-content">
          <AuthProvider>
            {children}
          </AuthProvider>
        </main>
      </body>
    </html>
  );
}
