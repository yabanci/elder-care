'use client';

import { useRouter } from 'next/navigation';
import { useEffect } from 'react';
import { Splash } from '@/components/Splash';

export default function RootRedirect() {
  const router = useRouter();
  useEffect(() => {
    const redirect = () => {
      const userRaw = typeof window !== 'undefined' ? localStorage.getItem('user') : null;
      if (!userRaw) {
        router.replace('/login');
        return;
      }
      try {
        const user = JSON.parse(userRaw);
        router.replace(user.role === 'patient' ? '/patient' : '/care');
      } catch {
        router.replace('/login');
      }
    };
    const t = setTimeout(redirect, 1200);
    return () => clearTimeout(t);
  }, [router]);

  return <Splash />;
}
