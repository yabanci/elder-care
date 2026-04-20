'use client';

import { useRouter } from 'next/navigation';
import { useEffect } from 'react';

export default function RootRedirect() {
  const router = useRouter();
  useEffect(() => {
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
  }, [router]);

  return (
    <div className="min-h-screen flex items-center justify-center text-ink-500">
      Загружаем...
    </div>
  );
}
