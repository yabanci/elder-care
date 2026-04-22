'use client';

import { useEffect, useState } from 'react';
import { Shell } from '@/components/Shell';
import { useAuthedUser } from '@/components/AuthGate';
import { api, type Plan, type PlanType } from '@/lib/api';
import { useI18n, type Lang } from '@/lib/i18n';
import { Plus, Pencil, Trash2, Stethoscope, Pill, Coffee, Clipboard } from 'lucide-react';

const DAYS_BY_LANG: Record<Lang, { full: string[]; short: string[] }> = {
  ru: {
    full: ['Понедельник', 'Вторник', 'Среда', 'Четверг', 'Пятница', 'Суббота', 'Воскресенье'],
    short: ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'],
  },
  kk: {
    full: ['Дүйсенбі', 'Сейсенбі', 'Сәрсенбі', 'Бейсенбі', 'Жұма', 'Сенбі', 'Жексенбі'],
    short: ['Дс', 'Сс', 'Ср', 'Бс', 'Жм', 'Сн', 'Жк'],
  },
  en: {
    full: ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'],
    short: ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'],
  },
};

const TYPE_ICONS: Record<PlanType, { Icon: typeof Stethoscope; color: string; key: string }> = {
  doctor_visit: { Icon: Stethoscope, color: 'text-primary-700', key: 'plan_type_doctor_visit' },
  take_med: { Icon: Pill, color: 'text-ok-500', key: 'plan_type_take_med' },
  rest: { Icon: Coffee, color: 'text-warn-500', key: 'plan_type_rest' },
  other: { Icon: Clipboard, color: 'text-ink-500', key: 'plan_type_other' },
};

function todayIdx() {
  const d = new Date().getDay();
  return (d + 6) % 7;
}

interface FormState {
  day_of_week: number;
  title: string;
  plan_type: PlanType;
  time_of_day: string;
}

const EMPTY_FORM: FormState = {
  day_of_week: todayIdx(),
  title: '',
  plan_type: 'other',
  time_of_day: '',
};

export default function PlansPage() {
  const user = useAuthedUser(['patient']);
  const { t, lang } = useI18n();
  const days = DAYS_BY_LANG[lang];
  const [plans, setPlans] = useState<Plan[]>([]);
  const [selectedDay, setSelectedDay] = useState(todayIdx());
  const [editId, setEditId] = useState<string | null>(null);
  const [adding, setAdding] = useState(false);
  const [form, setForm] = useState<FormState>(EMPTY_FORM);
  const [confirmId, setConfirmId] = useState<string | null>(null);

  useEffect(() => {
    if (!user) return;
    refresh();
  }, [user]);

  async function refresh() {
    setPlans(await api<Plan[]>('/api/plans'));
  }

  function startAdd() {
    setForm({ ...EMPTY_FORM, day_of_week: selectedDay });
    setEditId(null);
    setAdding(true);
  }

  function startEdit(p: Plan) {
    setForm({
      day_of_week: p.day_of_week,
      title: p.title,
      plan_type: p.plan_type,
      time_of_day: p.time_of_day ?? '',
    });
    setEditId(p.id);
    setAdding(true);
  }

  async function save(e: React.FormEvent) {
    e.preventDefault();
    const body = {
      day_of_week: form.day_of_week,
      title: form.title,
      plan_type: form.plan_type,
      time_of_day: form.time_of_day || null,
    };
    if (editId) {
      await api(`/api/plans/${editId}`, { method: 'PATCH', body: JSON.stringify(body) });
    } else {
      await api('/api/plans', { method: 'POST', body: JSON.stringify(body) });
    }
    setAdding(false);
    setEditId(null);
    setForm(EMPTY_FORM);
    refresh();
  }

  async function remove(id: string) {
    await api(`/api/plans/${id}`, { method: 'DELETE' });
    setConfirmId(null);
    refresh();
  }

  if (!user) return null;

  const dayPlans = plans.filter((p) => p.day_of_week === selectedDay);

  return (
    <Shell user={user}>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold">{t('plans_title')}</h1>
        <button onClick={startAdd} className="btn-primary">
          <Plus className="w-5 h-5" /> {t('plans_add')}
        </button>
      </div>

      <div className="flex gap-2 overflow-x-auto pb-2 mb-4">
        {days.short.map((d, i) => {
          const active = i === selectedDay;
          const count = plans.filter((p) => p.day_of_week === i).length;
          return (
            <button
              key={i}
              onClick={() => setSelectedDay(i)}
              className={`min-w-[56px] px-3 py-2 rounded-xl border-2 font-semibold ${
                active
                  ? 'border-primary-600 bg-primary-600 text-white'
                  : 'border-ink-300 bg-white text-ink-700'
              }`}
            >
              <div>{d}</div>
              {count > 0 && (
                <div className={`text-xs font-normal ${active ? 'text-white/80' : 'text-ink-500'}`}>
                  {count}
                </div>
              )}
            </button>
          );
        })}
      </div>

      {adding && (
        <form onSubmit={save} className="card space-y-3 mb-4">
          <h2 className="font-bold">{editId ? t('plans_edit_title') : t('plans_new_title')}</h2>
          <div>
            <label className="label">{t('plan_name')}</label>
            <input
              className="field"
              value={form.title}
              onChange={(e) => setForm({ ...form, title: e.target.value })}
              required
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">{t('plan_day')}</label>
              <select
                className="field"
                value={form.day_of_week}
                onChange={(e) => setForm({ ...form, day_of_week: parseInt(e.target.value, 10) })}
              >
                {days.full.map((d, i) => (
                  <option key={i} value={i}>{d}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="label">{t('plan_time')}</label>
              <input
                type="time"
                className="field"
                value={form.time_of_day}
                onChange={(e) => setForm({ ...form, time_of_day: e.target.value })}
              />
            </div>
          </div>
          <div>
            <label className="label">{t('plan_type')}</label>
            <select
              className="field"
              value={form.plan_type}
              onChange={(e) => setForm({ ...form, plan_type: e.target.value as PlanType })}
            >
              {(Object.keys(TYPE_ICONS) as PlanType[]).map((k) => (
                <option key={k} value={k}>{t(TYPE_ICONS[k].key)}</option>
              ))}
            </select>
          </div>
          <div className="flex gap-2">
            <button type="button" onClick={() => { setAdding(false); setEditId(null); }} className="btn-ghost flex-1">
              {t('cancel')}
            </button>
            <button type="submit" className="btn-primary flex-1">{t('save')}</button>
          </div>
        </form>
      )}

      <div className="mb-2 font-semibold text-ink-700">{days.full[selectedDay]}</div>

      <div className="space-y-2">
        {dayPlans.length === 0 && (
          <div className="card text-ink-500">{t('plan_empty')}</div>
        )}
        {dayPlans.map((p) => {
          const meta = TYPE_ICONS[p.plan_type];
          const Icon = meta.Icon;
          return (
            <div key={p.id} className="card !p-4">
              <div className="flex items-center gap-4">
                <div className="w-11 h-11 rounded-xl bg-ink-100 flex items-center justify-center">
                  <Icon className={`w-5 h-5 ${meta.color}`} />
                </div>
                <div className="flex-1">
                  <div className="font-bold">{p.title}</div>
                  <div className="text-ink-500 text-sm">
                    {p.time_of_day ? `${p.time_of_day} · ` : ''}
                    {t(meta.key)}
                  </div>
                </div>
                <button onClick={() => startEdit(p)} className="btn-ghost !min-h-10 !px-3" aria-label={t('edit')}>
                  <Pencil className="w-4 h-4" />
                </button>
                <button onClick={() => setConfirmId(p.id)} className="btn-ghost !min-h-10 !px-3" aria-label={t('delete')}>
                  <Trash2 className="w-4 h-4 text-danger-500" />
                </button>
              </div>
              {confirmId === p.id && (
                <div className="mt-3 flex items-center gap-2 rounded-xl bg-danger-500/10 p-3">
                  <span className="flex-1 font-semibold text-danger-500">{t('plan_confirm')}</span>
                  <button onClick={() => remove(p.id)} className="btn-danger !min-h-10 !px-4 !text-sm">
                    {t('yes')}
                  </button>
                  <button onClick={() => setConfirmId(null)} className="btn-ghost !min-h-10 !px-4 !text-sm">
                    {t('cancel')}
                  </button>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </Shell>
  );
}
