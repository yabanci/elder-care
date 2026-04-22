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
    const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null;
    if (!token) {
      router.replace('/login');
      return;
    }
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
        localStorage.setItem('user', JSON.stringify(u));
      })
      .catch((err) => {
        if (cancelled) return;
        if (err instanceof ApiError && err.status === 401) {
          clearAuth();
          router.replace('/login');
        }
      });
    return () => {
      cancelled = true;
    };
  }, [router, rolesKey, pathname, skipOnboarding]);

  return user;
}
