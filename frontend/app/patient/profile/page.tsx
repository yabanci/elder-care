'use client';

import { useEffect, useState } from 'react';
import { Shell } from '@/components/Shell';
import { useAuthedUser } from '@/components/AuthGate';
import { Watch, Gauge, Thermometer, Globe } from 'lucide-react';
import { LANGS, useI18n, type Lang } from '@/lib/i18n';
import { api, type User } from '@/lib/api';

type DeviceKey = 'watch' | 'bp' | 'thermo';

interface Device {
  key: DeviceKey;
  nameKey: string;
  Icon: typeof Watch;
}

const DEVICES: Device[] = [
  { key: 'watch', nameKey: 'device_watch', Icon: Watch },
  { key: 'bp', nameKey: 'device_bp', Icon: Gauge },
  { key: 'thermo', nameKey: 'device_thermo', Icon: Thermometer },
];

const STORAGE_KEY = 'devices';

function loadState(): Record<DeviceKey, boolean> {
  if (typeof window === 'undefined') return { watch: false, bp: false, thermo: false };
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) return { watch: false, bp: false, thermo: false, ...JSON.parse(raw) };
  } catch {}
  return { watch: false, bp: false, thermo: false };
}

export default function ProfilePage() {
  const user = useAuthedUser(['patient']);
  const { t, lang, setLang } = useI18n();
  const [state, setState] = useState<Record<DeviceKey, boolean>>({
    watch: false,
    bp: false,
    thermo: false,
  });

  useEffect(() => {
    setState(loadState());
  }, []);

  function toggle(key: DeviceKey) {
    setState((prev) => {
      const next = { ...prev, [key]: !prev[key] };
      localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
      return next;
    });
  }

  async function changeLang(next: Lang) {
    setLang(next);
    try {
      const updated = await api<User>('/api/me', {
        method: 'PATCH',
        body: JSON.stringify({ lang: next }),
      });
      localStorage.setItem('user', JSON.stringify(updated));
    } catch {
      // Non-fatal: localStorage has the lang, backend sync can retry later.
    }
  }

  if (!user) return null;

  return (
    <Shell user={user}>
      <h1 className="text-2xl font-bold mb-4">{t('profile_title')}</h1>

      <div className="card mb-4">
        <div className="font-semibold text-ink-700">{user.full_name}</div>
        <div className="text-ink-500 text-sm">{user.email}</div>
        {user.invite_code && (
          <div className="mt-3 text-sm">
            {t('profile_invite')}:{' '}
            <span className="font-mono font-bold">{user.invite_code}</span>
          </div>
        )}
      </div>

      <div className="card mb-4">
        <div className="flex items-center gap-3 mb-3">
          <Globe className="w-5 h-5 text-primary-700" />
          <div className="font-bold">{t('profile_language')}</div>
        </div>
        <div className="grid grid-cols-3 gap-2">
          {LANGS.map((l) => (
            <button
              key={l.code}
              onClick={() => changeLang(l.code as Lang)}
              className={`min-h-12 rounded-xl border-2 font-semibold text-base ${
                lang === l.code
                  ? 'border-primary-600 bg-primary-50 text-primary-700'
                  : 'border-ink-300 bg-white text-ink-700'
              }`}
            >
              <span className="mr-2">{l.flag}</span>
              {l.label}
            </button>
          ))}
        </div>
      </div>

      <h2 className="text-xl font-bold mb-2">{t('profile_devices')}</h2>
      <div className="space-y-2">
        {DEVICES.map(({ key, nameKey, Icon }) => {
          const connected = state[key];
          return (
            <div key={key} className="card !p-4 flex items-center gap-4">
              <div className="w-11 h-11 rounded-xl bg-ink-100 flex items-center justify-center">
                <Icon className="w-5 h-5 text-primary-700" />
              </div>
              <div className="flex-1">
                <div className="font-bold">{t(nameKey)}</div>
                <div className={`text-sm ${connected ? 'text-ok-500' : 'text-ink-500'}`}>
                  {connected ? t('device_connected') : t('device_disconnected')}
                </div>
              </div>
              <button
                onClick={() => toggle(key)}
                className={connected ? 'btn-ghost !min-h-10 !px-4 !text-sm' : 'btn-primary !min-h-10 !px-4 !text-sm'}
              >
                {connected ? t('device_disconnect') : t('device_connect')}
              </button>
            </div>
          );
        })}
      </div>
    </Shell>
  );
}
