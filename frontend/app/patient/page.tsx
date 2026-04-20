'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { Shell } from '@/components/Shell';
import { useAuthedUser } from '@/components/AuthGate';
import { MetricEntryCard } from '@/components/MetricEntryCard';
import { SummaryGrid } from '@/components/SummaryGrid';
import { api, type Alert, type MedScheduleItem, type Metric, type MetricKind } from '@/lib/api';
import { METRIC_META } from '@/lib/metric-meta';
import { AlertTriangle, Phone } from 'lucide-react';

const QUICK_KINDS: MetricKind[] = ['pulse', 'bp_sys', 'glucose', 'temperature'];

export default function PatientHome() {
  const user = useAuthedUser(['patient']);
  const [summary, setSummary] = useState<Metric[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [schedule, setSchedule] = useState<MedScheduleItem[]>([]);
  const [busy, setBusy] = useState<string | null>(null);

  useEffect(() => {
    if (!user) return;
    refresh();
  }, [user]);

  async function refresh() {
    const [s, a, sch] = await Promise.all([
      api<Metric[]>('/api/metrics/summary'),
      api<Alert[]>('/api/alerts'),
      api<MedScheduleItem[]>('/api/medications/today'),
    ]);
    setSummary(s);
    setAlerts(a);
    setSchedule(sch);
    const unack = a.filter((x) => !x.acknowledged).length;
    localStorage.setItem('alertCount', String(unack));
  }

  async function takeDose(item: MedScheduleItem) {
    setBusy(item.medication_id + item.scheduled_at);
    try {
      await api(`/api/medications/${item.medication_id}/log`, {
        method: 'POST',
        body: JSON.stringify({ scheduled_at: item.scheduled_at, status: 'taken' }),
      });
      await refresh();
    } finally {
      setBusy(null);
    }
  }

  if (!user) return null;

  const hour = new Date().getHours();
  const greeting = hour < 12 ? 'Доброе утро' : hour < 18 ? 'Добрый день' : 'Добрый вечер';
  const unackAlerts = alerts.filter((a) => !a.acknowledged);

  return (
    <Shell user={user}>
      <div className="space-y-6">
        <div className="flex items-end justify-between">
          <div>
            <div className="text-ink-500">{greeting},</div>
            <h1 className="text-3xl font-bold">{user.full_name.split(' ')[0]}</h1>
          </div>
          <a href="tel:103" className="btn-accent" aria-label="Вызов скорой">
            <Phone className="w-5 h-5" /> 103
          </a>
        </div>

        {unackAlerts.length > 0 && (
          <Link href="/patient/alerts" className="block">
            <div className="card !bg-danger-500/5 border border-danger-500/20 flex items-start gap-3">
              <AlertTriangle className="w-6 h-6 text-danger-500 shrink-0 mt-1" />
              <div>
                <div className="font-bold text-danger-500">
                  {unackAlerts.length} новых оповещений
                </div>
                <div className="text-sm text-ink-700">
                  {unackAlerts[0].reason} — {METRIC_META[unackAlerts[0].kind as MetricKind]?.label ?? unackAlerts[0].kind}
                </div>
              </div>
            </div>
          </Link>
        )}

        <section>
          <h2 className="text-xl font-bold mb-3">Последние показатели</h2>
          <SummaryGrid metrics={summary} />
        </section>

        <section>
          <div className="flex items-baseline justify-between mb-3">
            <h2 className="text-xl font-bold">Лекарства сегодня</h2>
            <Link href="/patient/medications" className="text-primary-700 font-semibold">Все →</Link>
          </div>
          {schedule.length === 0 ? (
            <div className="card text-ink-500">На сегодня лекарств нет.</div>
          ) : (
            <div className="space-y-2">
              {schedule.map((item) => {
                const t = new Date(item.scheduled_at);
                const hhmm = t.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
                const pending = item.status === 'pending' || item.status === 'missed';
                return (
                  <div
                    key={item.medication_id + item.scheduled_at}
                    className={`card !p-4 flex items-center gap-4 ${
                      item.status === 'missed' ? 'border border-warn-500/30' : ''
                    }`}
                  >
                    <div className="text-3xl font-bold w-20 text-primary-700">{hhmm}</div>
                    <div className="flex-1">
                      <div className="font-bold text-lg">{item.name}</div>
                      {item.dosage && <div className="text-ink-500 text-sm">{item.dosage}</div>}
                    </div>
                    {pending ? (
                      <button
                        className="btn-primary !min-h-12 !px-4"
                        disabled={busy === item.medication_id + item.scheduled_at}
                        onClick={() => takeDose(item)}
                      >
                        Принял ✓
                      </button>
                    ) : (
                      <div className="badge-ok">✓ Принято</div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </section>

        <section>
          <h2 className="text-xl font-bold mb-3">Быстрый ввод</h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {QUICK_KINDS.map((k) => (
              <MetricEntryCard key={k} kind={k} onSaved={refresh} />
            ))}
          </div>
          <div className="mt-3 text-center">
            <Link href="/patient/metrics" className="text-primary-700 font-semibold">
              Показать все метрики и графики →
            </Link>
          </div>
        </section>

        {user.invite_code && (
          <section className="card bg-primary-50 border border-primary-500/20">
            <div className="text-sm font-semibold text-primary-700 mb-1">Код приглашения</div>
            <div className="text-2xl font-bold tracking-wider">{user.invite_code}</div>
            <div className="text-sm text-ink-500 mt-2">
              Сообщите этот код врачу или родственнику, чтобы они могли видеть ваши показатели.
            </div>
          </section>
        )}
      </div>
    </Shell>
  );
}
