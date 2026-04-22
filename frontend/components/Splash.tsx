'use client';

import { useEffect, useState } from 'react';
import { Heart } from 'lucide-react';
import { useI18n } from '@/lib/i18n';

export function Splash({ durationMs = 1200 }: { durationMs?: number }) {
  const { t } = useI18n();
  const [hidden, setHidden] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => setHidden(true), durationMs);
    return () => clearTimeout(timer);
  }, [durationMs]);

  return (
    <div
      className={`fixed inset-0 z-[999] flex flex-col items-center justify-center text-white transition-opacity duration-500 ${
        hidden ? 'opacity-0 pointer-events-none' : 'opacity-100'
      }`}
      style={{ background: 'linear-gradient(135deg,#2563eb,#1e40af)' }}
    >
      <div className="w-20 h-20 rounded-3xl bg-white/15 flex items-center justify-center">
        <Heart className="w-10 h-10 fill-white text-white" />
      </div>
      <h1 className="mt-4 text-3xl font-extrabold">{t('app_name')}</h1>
      <p className="mt-1.5 text-base opacity-80">{t('app_tagline')}</p>
    </div>
  );
}
