'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { Shell } from '@/components/Shell';
import { useAuthedUser } from '@/components/AuthGate';
import { api } from '@/lib/api';

export default function LinkPage() {
  const user = useAuthedUser(['doctor', 'family']);
  const router = useRouter();
  const [code, setCode] = useState('');
  const [err, setErr] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setErr(null);
    try {
      await api('/api/patients/link', {
        method: 'POST',
        body: JSON.stringify({ invite_code: code.trim() }),
      });
      router.push('/care');
    } catch (x) {
      setErr(x instanceof Error ? x.message : 'Ошибка');
    } finally {
      setLoading(false);
    }
  }

  if (!user) return null;

  return (
    <Shell user={user}>
      <h1 className="text-2xl font-bold mb-4">Добавить пациента</h1>
      <form onSubmit={submit} className="card space-y-4 max-w-md">
        <div>
          <label className="label">Код приглашения</label>
          <input
            className="field uppercase tracking-widest text-2xl font-bold text-center"
            placeholder="ABCD1234"
            value={code}
            onChange={(e) => setCode(e.target.value.toUpperCase())}
            maxLength={16}
            required
          />
          <div className="text-sm text-ink-500 mt-2">
            Код можно получить у пациента на главном экране.
          </div>
        </div>
        {err && <div className="text-danger-500 font-semibold">{err}</div>}
        <button className="btn-primary w-full" disabled={loading}>
          {loading ? '...' : 'Подключиться'}
        </button>
      </form>
    </Shell>
  );
}
