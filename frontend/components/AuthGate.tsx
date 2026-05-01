'use client';

import { usePathname, useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { ApiError, api, clearAuth, type Role, type User } from '@/lib/api';

export function useAuthedUser(
  expectedRoles?: Role[],
  opts?: { skipOnboardingRedirect?: boolean },
): User | null {
  const router = useRouter();
  const pathname = usePathname();
  const [user, setUser] = useState<User | null>(null);
  const rolesKey = expectedRoles?.join(',') ?? '';
  const skipOnboarding = opts?.skipOnboardingRedirect ?? false;

  useEffect(() => {
    // Auth is now cookie-based: we cannot read the HttpOnly cookie from
    // JS, so we just attempt /api/me and let the server be the
    // authority. 401 → bounce to login. This also handles the "expired
    // token" case naturally without any client-side TTL bookkeeping.
    let cancelled = false;
    api<User>('/api/me')
      .then((u) => {
        if (cancelled) return;
        const roles = rolesKey ? (rolesKey.split(',') as Role[]) : undefined;
        if (roles && !roles.includes(u.role)) {
          router.replace(u.role === 'patient' ? '/patient' : '/care');
          return;
        }
        if (
          !skipOnboarding &&
          u.role === 'patient' &&
          !u.onboarded &&
          pathname !== '/patient/onboarding'
        ) {
          router.replace('/patient/onboarding');
          return;
        }
        setUser(u);
        if (typeof window !== 'undefined') {
          localStorage.setItem('user', JSON.stringify(u));
        }
      })
      .catch((err) => {
        if (cancelled) return;
        if (err instanceof ApiError && err.status === 401) {
          void clearAuth();
          router.replace('/login');
        }
      });
    return () => {
      cancelled = true;
    };
  }, [router, rolesKey, pathname, skipOnboarding]);

  return user;
}
