'use client';

import { useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { api, clearAuth, type Role, type User } from '@/lib/api';

export function useAuthedUser(expectedRoles?: Role[]): User | null {
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null;
    if (!token) {
      router.replace('/login');
      return;
    }
    api<User>('/api/me')
      .then((u) => {
        if (expectedRoles && !expectedRoles.includes(u.role)) {
          router.replace(u.role === 'patient' ? '/patient' : '/care');
          return;
        }
        setUser(u);
        localStorage.setItem('user', JSON.stringify(u));
      })
      .catch(() => {
        clearAuth();
        router.replace('/login');
      });
  }, [router, expectedRoles]);

  return user;
}
