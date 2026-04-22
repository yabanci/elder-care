'use client';

import { useState } from 'react';
import { api, type Metric, type User } from '@/lib/api';
import { useI18n } from '@/lib/i18n';
import { Activity } from 'lucide-react';

function bmiStatus(bmi: number, t: (k: string) => string): { label: string; cls: string } {
  if (bmi < 18.5) return { label: t('bmi_under'), cls: 'badge-warn' };
  if (bmi < 25) return { label: t('bmi_normal'), cls: 'badge-ok' };
  if (bmi < 30) return { label: t('bmi_over'), cls: 'badge-warn' };
  return { label: t('bmi_obese'), cls: 'badge-danger' };
}

export function BMICard({
  user,
  summary,
  onUserChange,
}: {
  user: User;
  summary: Metric[];
  onUserChange: (u: User) => void;
}) {
  const { t } = useI18n();
  const [editing, setEditing] = useState(false);
  const [value, setValue] = useState(user.height_cm?.toString() ?? '');
  const [saving, setSaving] = useState(false);

  const weight = summary.find((m) => m.kind === 'weight')?.value;
  const height = user.height_cm;

  async function save(e: React.FormEvent) {
    e.preventDefault();
    const h = parseInt(value, 10);
    if (!h || h < 80 || h > 250) return;
    setSaving(true);
    try {
      const updated = await api<User>('/api/me', {
        method: 'PATCH',
        body: JSON.stringify({ height_cm: h }),
      });
      onUserChange(updated);
      localStorage.setItem('user', JSON.stringify(updated));
      setEditing(false);
    } finally {
      setSaving(false);
    }
  }

  if (!height) {
    return (
      <div className="card flex items-center gap-4">
        <div className="w-11 h-11 rounded-xl bg-ink-100 flex items-center justify-center">
          <Activity className="w-5 h-5 text-primary-700" />
        </div>
        <div className="flex-1">
          <div className="font-bold">{t('bmi_label')}</div>
          <div className="text-ink-500 text-sm">{t('bmi_set_height')}</div>
        </div>
        {editing ? (
          <form onSubmit={save} className="flex items-center gap-2">
            <input
              type="number"
              min={80}
              max={250}
              autoFocus
              placeholder="170"
              value={value}
              onChange={(e) => setValue(e.target.value)}
              className="field !min-h-10 !px-3 w-24"
            />
            <button type="submit" className="btn-primary !min-h-10 !px-4 !text-sm" disabled={saving}>
              OK
            </button>
          </form>
        ) : (
          <button
            onClick={() => setEditing(true)}
            className="btn-primary !min-h-10 !px-4 !text-sm"
          >
            {t('bmi_set_btn')}
          </button>
        )}
      </div>
    );
  }

  if (!weight) {
    return (
      <div className="card flex items-center gap-4">
        <div className="w-11 h-11 rounded-xl bg-ink-100 flex items-center justify-center">
          <Activity className="w-5 h-5 text-primary-700" />
        </div>
        <div className="flex-1">
          <div className="font-bold">{t('bmi_label')}</div>
          <div className="text-ink-500 text-sm">
            {height} · {t('bmi_need_weight')}
          </div>
        </div>
      </div>
    );
  }

  const bmi = weight / (height / 100) ** 2;
  const status = bmiStatus(bmi, t);

  return (
    <div className="card flex items-center gap-4">
      <div className="w-11 h-11 rounded-xl bg-ink-100 flex items-center justify-center">
        <Activity className="w-5 h-5 text-primary-700" />
      </div>
      <div className="flex-1">
        <div className="font-bold text-2xl">{bmi.toFixed(1)}</div>
        <div className="text-ink-500 text-sm">
          {t('bmi_label')} · {height} · {weight}
        </div>
      </div>
      <div className={status.cls}>{status.label}</div>
    </div>
  );
}
