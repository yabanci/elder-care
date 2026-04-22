'use client';

import { useEffect, useState } from 'react';
import { Shell } from '@/components/Shell';
import { useAuthedUser } from '@/components/AuthGate';
import { api, type Medication } from '@/lib/api';
import { useI18n } from '@/lib/i18n';
import { Plus, Trash2 } from 'lucide-react';

export default function MedicationsPage() {
  const user = useAuthedUser(['patient']);
  const { t } = useI18n();
  const [meds, setMeds] = useState<Medication[]>([]);
  const [adding, setAdding] = useState(false);
  const [form, setForm] = useState({ name: '', dosage: '', times: '08:00' });
  const [confirmId, setConfirmId] = useState<string | null>(null);

  useEffect(() => {
    if (!user) return;
    refresh();
  }, [user]);

  async function refresh() {
    setMeds(await api<Medication[]>('/api/medications'));
  }

  async function add(e: React.FormEvent) {
    e.preventDefault();
    const times_of_day = form.times
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean);
    await api('/api/medications', {
      method: 'POST',
      body: JSON.stringify({
        name: form.name,
        dosage: form.dosage,
        times_of_day,
      }),
    });
    setForm({ name: '', dosage: '', times: '08:00' });
    setAdding(false);
    refresh();
  }

  async function remove(id: string) {
    await api(`/api/medications/${id}`, { method: 'DELETE' });
    setConfirmId(null);
    refresh();
  }

  if (!user) return null;

  return (
    <Shell user={user}>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold">{t('meds_title')}</h1>
        <button onClick={() => setAdding((v) => !v)} className="btn-primary">
          <Plus className="w-5 h-5" /> {t('add')}
        </button>
      </div>

      {adding && (
        <form onSubmit={add} className="card space-y-3 mb-4">
          <div>
            <label className="label">{t('meds_add_title')}</label>
            <input
              className="field"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              required
            />
          </div>
          <div>
            <label className="label">{t('meds_dosage')}</label>
            <input
              className="field"
              placeholder="10 мг"
              value={form.dosage}
              onChange={(e) => setForm({ ...form, dosage: e.target.value })}
            />
          </div>
          <div>
            <label className="label">{t('meds_times')}</label>
            <input
              className="field"
              placeholder="08:00, 20:00"
              value={form.times}
              onChange={(e) => setForm({ ...form, times: e.target.value })}
              required
            />
          </div>
          <button type="submit" className="btn-primary w-full">{t('save')}</button>
        </form>
      )}

      <div className="space-y-2">
        {meds.map((m) => (
          <div key={m.id} className="card !p-4">
            <div className="flex items-center gap-4">
              <div className="text-3xl">💊</div>
              <div className="flex-1">
                <div className="font-bold text-lg">{m.name}</div>
                <div className="text-ink-500">
                  {m.dosage ? `${m.dosage} · ` : ''}
                  {m.times_of_day.join(', ')}
                </div>
                {m.notes && <div className="text-sm text-ink-500 mt-1">{m.notes}</div>}
              </div>
              <button
                onClick={() => setConfirmId(m.id)}
                className="btn-ghost !min-h-12 !px-3"
                aria-label={t('delete')}
              >
                <Trash2 className="w-5 h-5 text-danger-500" />
              </button>
            </div>
            {confirmId === m.id && (
              <div className="mt-3 flex items-center gap-2 rounded-xl bg-danger-500/10 p-3">
                <span className="flex-1 font-semibold text-danger-500">
                  {t('meds_confirm')}
                </span>
                <button
                  onClick={() => remove(m.id)}
                  className="btn-danger !min-h-10 !px-4 !text-sm"
                >
                  {t('yes')}
                </button>
                <button
                  onClick={() => setConfirmId(null)}
                  className="btn-ghost !min-h-10 !px-4 !text-sm"
                >
                  {t('cancel')}
                </button>
              </div>
            )}
          </div>
        ))}
        {meds.length === 0 && (
          <div className="card text-ink-500">{t('meds_empty')}</div>
        )}
      </div>
    </Shell>
  );
}
