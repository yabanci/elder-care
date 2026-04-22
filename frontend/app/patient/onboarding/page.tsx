'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { api, type User } from '@/lib/api';
import { useAuthedUser } from '@/components/AuthGate';
import { useI18n } from '@/lib/i18n';
import { LangSwitcher } from '@/components/LangSwitcher';
import { ClipboardList } from 'lucide-react';

export default function OnboardingPage() {
  const router = useRouter();
  const user = useAuthedUser(['patient'], { skipOnboardingRedirect: true });
  const { t } = useI18n();
  const [form, setForm] = useState({
    height: '',
    weight: '',
    chronic: '',
    bp_norm: '',
    meds: '',
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    const height = parseInt(form.height, 10);
    const weight = parseFloat(form.weight);
    if (!height || height < 80 || height > 250) {
      setError(t('onboard_bad_height'));
      return;
    }
    if (!weight || weight < 20 || weight > 300) {
      setError(t('onboard_bad_weight'));
      return;
    }
    setSaving(true);
    setError(null);
    try {
      const updated = await api<User>('/api/me', {
        method: 'PATCH',
        body: JSON.stringify({
          height_cm: height,
          chronic_conditions: form.chronic || null,
          bp_norm: form.bp_norm || null,
          prescribed_meds: form.meds || null,
          onboarded: true,
        }),
      });
      await api('/api/metrics', {
        method: 'POST',
        body: JSON.stringify({
          kind: 'weight',
          value: weight,
          measured_at: new Date().toISOString(),
        }),
      });
      localStorage.setItem('user', JSON.stringify(updated));
      router.replace('/patient');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка сохранения');
    } finally {
      setSaving(false);
    }
  }

  if (!user) return null;

  return (
    <div className="min-h-screen flex items-start justify-center p-4 bg-gradient-to-br from-primary-50 to-white">
      <div className="w-full max-w-md py-6">
        <LangSwitcher className="mb-4" />
        <div className="text-center mb-6">
          <div className="inline-flex w-14 h-14 items-center justify-center rounded-2xl bg-primary-100 text-primary-700 mb-3">
            <ClipboardList className="w-7 h-7" />
          </div>
          <h1 className="text-2xl font-bold">{t('onboard_title')}</h1>
          <p className="text-ink-500 mt-1">{t('onboard_sub')}</p>
        </div>

        <form onSubmit={submit} className="card space-y-4">
          <div>
            <label className="label" htmlFor="h">{t('onboard_height')}</label>
            <input
              id="h"
              type="number"
              inputMode="numeric"
              className="field"
              placeholder="170"
              value={form.height}
              onChange={(e) => setForm({ ...form, height: e.target.value })}
              required
            />
          </div>
          <div>
            <label className="label" htmlFor="w">{t('onboard_weight')}</label>
            <input
              id="w"
              type="number"
              inputMode="decimal"
              step="0.1"
              className="field"
              placeholder="70"
              value={form.weight}
              onChange={(e) => setForm({ ...form, weight: e.target.value })}
              required
            />
          </div>
          <div>
            <label className="label" htmlFor="c">{t('onboard_chronic')}</label>
            <input
              id="c"
              className="field"
              value={form.chronic}
              onChange={(e) => setForm({ ...form, chronic: e.target.value })}
            />
          </div>
          <div>
            <label className="label" htmlFor="bp">{t('onboard_bp_norm')}</label>
            <input
              id="bp"
              className="field"
              placeholder="120/80"
              value={form.bp_norm}
              onChange={(e) => setForm({ ...form, bp_norm: e.target.value })}
            />
          </div>
          <div>
            <label className="label" htmlFor="m">{t('onboard_meds')}</label>
            <input
              id="m"
              className="field"
              value={form.meds}
              onChange={(e) => setForm({ ...form, meds: e.target.value })}
            />
          </div>

          {error && <div className="text-danger-500 font-semibold">{error}</div>}

          <button type="submit" className="btn-primary w-full" disabled={saving}>
            {saving ? t('onboard_saving') : t('onboard_submit')}
          </button>
        </form>
      </div>
    </div>
  );
}
