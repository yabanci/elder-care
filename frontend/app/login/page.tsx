'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { api, setToken, type User } from '@/lib/api';
import { useI18n } from '@/lib/i18n';

export default function LoginPage() {
  const router = useRouter();
  const { t } = useI18n();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const res = await api<{ token: string; user: User }>('/api/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email, password }),
      });
      setToken(res.token);
      localStorage.setItem('user', JSON.stringify(res.user));
      router.push(res.user.role === 'patient' ? '/patient' : '/care');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка входа');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4 bg-gradient-to-br from-primary-50 to-white">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-primary-600 text-white text-3xl mb-4">
            ❤
          </div>
          <h1 className="text-3xl font-bold">ElderCare</h1>
          <p className="text-ink-500 mt-2">{t('login_tagline')}</p>
        </div>

        <form onSubmit={submit} className="card space-y-5">
          <h2 className="text-2xl font-bold">{t('login_title')}</h2>

          <div>
            <label className="label" htmlFor="email">{t('login_email')}</label>
            <input
              id="email"
              type="email"
              className="field"
              placeholder="patient@demo.kz"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              autoComplete="email"
            />
          </div>

          <div>
            <label className="label" htmlFor="pw">{t('login_password')}</label>
            <input
              id="pw"
              type="password"
              className="field"
              placeholder="••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              autoComplete="current-password"
            />
          </div>

          {error && <div className="text-danger-500 font-semibold">{error}</div>}

          <button type="submit" className="btn-primary w-full" disabled={loading}>
            {loading ? t('loading') : t('login_submit')}
          </button>

          <div className="text-center text-ink-500">
            {t('login_no_account')}{' '}
            <Link href="/register" className="text-primary-700 font-semibold hover:underline">
              {t('login_register')}
            </Link>
          </div>

          <div className="border-t border-ink-300 pt-4 text-sm text-ink-500">
            <div className="font-semibold mb-2 text-ink-700">{t('login_demo')}</div>
            <div>{t('role_patient')}: patient@demo.kz / demo1234</div>
            <div>{t('role_doctor')}: doctor@demo.kz / demo1234</div>
            <div>{t('role_family')}: family@demo.kz / demo1234</div>
          </div>
        </form>
      </div>
    </div>
  );
}
