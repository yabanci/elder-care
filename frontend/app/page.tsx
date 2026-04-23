'use client';

import { useRouter } from 'next/navigation';
import { useEffect } from 'react';
import { useAuthedUser } from '@/components/AuthGate';
import { Splash } from '@/components/Splash';

export default function RootRedirect() {
  const router = useRouter();
  const user = useAuthedUser();

  useEffect(() => {
    if (!user) return;
    router.replace(user.role === 'patient' ? '/patient' : '/care');
  }, [user, router]);

  return <Splash />;
}
