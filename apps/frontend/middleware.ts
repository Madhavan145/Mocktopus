import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

/**
 * Next.js Edge Middleware — server-side route protection.
 * Checks for the presence of the auth cookie (set on login).
 * Redirects unauthenticated users to /login for protected routes.
 */

const PUBLIC_PATHS = [
  '/',
  '/login',
  '/register',
  '/forgot-password',
  '/reset-password',
  '/playground',
  '/oauth/callback',
];

const PUBLIC_PREFIXES = [
  '/api/',
  '/_next/',
  '/favicon',
  '/images/',
  '/fonts/',
];

function isPublic(pathname: string): boolean {
  if (PUBLIC_PATHS.includes(pathname)) return true;
  return PUBLIC_PREFIXES.some((prefix) => pathname.startsWith(prefix));
}

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // If authenticated user visits the landing page, redirect to dashboard
  if (pathname === '/') {
    const authCookie = request.cookies.get('moktopus.session');
    if (authCookie?.value) {
      try {
        const payload = JSON.parse(
          Buffer.from(authCookie.value.split('.')[1], 'base64url').toString()
        );
        if (payload.exp && payload.exp * 1000 > Date.now()) {
          return NextResponse.redirect(new URL('/dashboard', request.url));
        }
      } catch {
        // Malformed token — let them see the landing page
      }
    }
    return NextResponse.next();
  }

  if (isPublic(pathname)) {
    return NextResponse.next();
  }

  // Check for auth cookie (set during login/register)
  const authCookie = request.cookies.get('moktopus.session');

  if (!authCookie?.value) {
    const loginUrl = new URL('/login', request.url);
    loginUrl.searchParams.set('redirect', pathname);
    return NextResponse.redirect(loginUrl);
  }

  // Verify token is not expired by checking the exp claim
  try {
    const payload = JSON.parse(
      Buffer.from(authCookie.value.split('.')[1], 'base64url').toString()
    );
    if (payload.exp && payload.exp * 1000 < Date.now()) {
      // Token expired — redirect to login
      const response = NextResponse.redirect(new URL('/login', request.url));
      response.cookies.delete('moktopus.session');
      return response;
    }
  } catch {
    // Malformed token — redirect to login
    const response = NextResponse.redirect(new URL('/login', request.url));
    response.cookies.delete('moktopus.session');
    return response;
  }

  return NextResponse.next();
}

export const config = {
  matcher: ['/((?!_next/static|_next/image|favicon.ico).*)'],
};
