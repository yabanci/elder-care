'use client';

import { useEffect, useState } from 'react';
import { Shell } from '@/components/Shell';
import { useAuthedUser } from '@/components/AuthGate';
import { api, type Medication } from '@/lib/api';
import { Plus, Trash2 } from 'lucide-react';

export default function MedicationsPage() {
  const user = useAuthedUser(['patient']);
  const [meds, setMeds] = useState<Medication[]>([]);
  const [adding, setAdding] = useState(false);
  const [form, setForm] = useState({ name: '', dosage: '', times: '08:00' });

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
    if (!confirm('Удалить это лекарство?')) return;
    await api(`/api/medications/${id}`, { method: 'DELETE' });
    refresh();
  }

  if (!user) return null;

  return (
    <Shell user={user}>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold">Мои лекарства</h1>
        <button onClick={() => setAdding((v) => !v)} className="btn-primary">
          <Plus className="w-5 h-5" /> Добавить
        </button>
      </div>

      {adding && (
        <form onSubmit={add} className="card space-y-3 mb-4">
          <div>
            <label className="label">Название</label>
            <input
              className="field"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              required
            />
          </div>
          <div>
            <label className="label">Дозировка</label>
            <input
              className="field"
              placeholder="10 мг"
              value={form.dosage}
              onChange={(e) => setForm({ ...form, dosage: e.target.value })}
            />
          </div>
          <div>
            <label className="label">Время приёма (через запятую)</label>
            <input
              className="field"
              placeholder="08:00, 20:00"
              value={form.times}
              onChange={(e) => setForm({ ...form, times: e.target.value })}
              required
            />
          </div>
          <button type="submit" className="btn-primary w-full">Сохранить</button>
        </form>
      )}

      <div className="space-y-2">
        {meds.map((m) => (
          <div key={m.id} className="card !p-4 flex items-center gap-4">
            <div className="text-3xl">💊</div>
            <div className="flex-1">
              <div className="font-bold text-lg">{m.name}</div>
              <div className="text-ink-500">
                {m.dosage ? `${m.dosage} · ` : ''}
                {m.times_of_day.join(', ')}
              </div>
              {m.notes && <div className="text-sm text-ink-500 mt-1">{m.notes}</div>}
            </div>
            <button onClick={() => remove(m.id)} className="btn-ghost !min-h-12 !px-3">
              <Trash2 className="w-5 h-5 text-danger-500" />
            </button>
          </div>
        ))}
        {meds.length === 0 && (
          <div className="card text-ink-500">Пока нет активных лекарств.</div>
        )}
      </div>
    </Shell>
  );
}
