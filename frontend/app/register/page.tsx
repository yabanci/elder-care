'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { api, setToken, type Role, type User } from '@/lib/api';

export default function RegisterPage() {
  const router = useRouter();
  const [form, setForm] = useState({
    email: '',
    password: '',
    full_name: '',
    role: 'patient' as Role,
    phone: '',
    birth_date: '',
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const res = await api<{ token: string; user: User }>('/api/auth/register', {
        method: 'POST',
        body: JSON.stringify(form),
      });
      setToken(res.token);
      localStorage.setItem('user', JSON.stringify(res.user));
      router.push(res.user.role === 'patient' ? '/patient' : '/care');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка регистрации');
    } finally {
      setLoading(false);
    }
  }

  function up<K extends keyof typeof form>(k: K, v: (typeof form)[K]) {
    setForm((f) => ({ ...f, [k]: v }));
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4 bg-gradient-to-br from-primary-50 to-white">
      <div className="w-full max-w-md">
        <form onSubmit={submit} className="card space-y-5">
          <h1 className="text-2xl font-bold">Регистрация</h1>

          <div>
            <label className="label">Я —</label>
            <div className="grid grid-cols-3 gap-2">
              {([
                { v: 'patient', t: 'Пациент' },
                { v: 'doctor', t: 'Врач' },
                { v: 'family', t: 'Родственник' },
              ] as const).map((r) => (
                <button
                  key={r.v}
                  type="button"
                  onClick={() => up('role', r.v)}
                  className={`min-h-14 px-3 rounded-xl border-2 font-semibold text-base ${
                    form.role === r.v
                      ? 'border-primary-600 bg-primary-50 text-primary-700'
                      : 'border-ink-300 bg-white text-ink-700'
                  }`}
                >
                  {r.t}
                </button>
              ))}
            </div>
          </div>

          <div>
            <label className="label" htmlFor="name">ФИО</label>
            <input
              id="name"
              className="field"
              value={form.full_name}
              onChange={(e) => up('full_name', e.target.value)}
              required
            />
          </div>

          <div>
            <label className="label" htmlFor="email">Email</label>
            <input
              id="email"
              type="email"
              className="field"
              value={form.email}
              onChange={(e) => up('email', e.target.value)}
              required
              autoComplete="email"
            />
          </div>

          <div>
            <label className="label" htmlFor="pw">Пароль (минимум 6 символов)</label>
            <input
              id="pw"
              type="password"
              className="field"
              value={form.password}
              onChange={(e) => up('password', e.target.value)}
              minLength={6}
              required
              autoComplete="new-password"
            />
          </div>

          <div>
            <label className="label" htmlFor="phone">Телефон</label>
            <input
              id="phone"
              className="field"
              placeholder="+7..."
              value={form.phone}
              onChange={(e) => up('phone', e.target.value)}
            />
          </div>

          {form.role === 'patient' && (
            <div>
              <label className="label" htmlFor="bd">Дата рождения</label>
              <input
                id="bd"
                type="date"
                className="field"
                value={form.birth_date}
                onChange={(e) => up('birth_date', e.target.value)}
              />
            </div>
          )}

          {error && <div className="text-danger-500 font-semibold">{error}</div>}

          <button type="submit" className="btn-primary w-full" disabled={loading}>
            {loading ? 'Создаём...' : 'Создать аккаунт'}
          </button>

          <div className="text-center text-ink-500">
            Уже есть аккаунт?{' '}
            <Link href="/login" className="text-primary-700 font-semibold hover:underline">
              Войти
            </Link>
          </div>
        </form>
      </div>
    </div>
  );
}
