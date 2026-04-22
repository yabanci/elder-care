'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { Shell } from '@/components/Shell';
import { useAuthedUser } from '@/components/AuthGate';
import { MetricEntryCard } from '@/components/MetricEntryCard';
import { SummaryGrid } from '@/components/SummaryGrid';
import { BMICard } from '@/components/BMICard';
import { api, type Alert, type MedScheduleItem, type Metric, type MetricKind, type User } from '@/lib/api';
import { METRIC_META } from '@/lib/metric-meta';
import { useI18n } from '@/lib/i18n';
import { AlertTriangle, Phone } from 'lucide-react';

const QUICK_KINDS: MetricKind[] = ['pulse', 'bp_sys', 'glucose', 'temperature'];

export default function PatientHome() {
  const authUser = useAuthedUser(['patient']);
  const { t } = useI18n();
  const [user, setUser] = useState<User | null>(null);
  const [summary, setSummary] = useState<Metric[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [schedule, setSchedule] = useState<MedScheduleItem[]>([]);
  const [busy, setBusy] = useState<string | null>(null);

  useEffect(() => {
    if (authUser) setUser(authUser);
  }, [authUser]);

  useEffect(() => {
    if (!user) return;
    refresh();
  }, [user?.id]);

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
  const greeting =
    hour < 12 ? t('greeting_morning') : hour < 18 ? t('greeting_day') : t('greeting_evening');
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
                  {unackAlerts.length} {t('alerts_new')}
                </div>
                <div className="text-sm text-ink-700">
                  {unackAlerts[0].reason} — {METRIC_META[unackAlerts[0].kind as MetricKind]?.label ?? unackAlerts[0].kind}
                </div>
              </div>
            </div>
          </Link>
        )}

        <section>
          <h2 className="text-xl font-bold mb-3">{t('recent_metrics')}</h2>
          <SummaryGrid metrics={summary} />
        </section>

        <section>
          <BMICard user={user} summary={summary} onUserChange={setUser} />
        </section>

        <section>
          <div className="flex items-baseline justify-between mb-3">
            <h2 className="text-xl font-bold">{t('today_meds')}</h2>
            <Link href="/patient/medications" className="text-primary-700 font-semibold">{t('show_all')}</Link>
          </div>
          {schedule.length === 0 ? (
            <div className="card text-ink-500">{t('today_no_meds')}</div>
          ) : (
            <div className="space-y-2">
              {schedule.map((item) => {
                const scheduled = new Date(item.scheduled_at);
                const hhmm = scheduled.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
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
                        {t('take_dose')}
                      </button>
                    ) : (
                      <div className="badge-ok">{t('dose_taken')}</div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </section>

        <section>
          <h2 className="text-xl font-bold mb-3">{t('quick_entry')}</h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {QUICK_KINDS.map((k) => (
              <MetricEntryCard key={k} kind={k} onSaved={refresh} />
            ))}
          </div>
          <div className="mt-3 text-center">
            <Link href="/patient/metrics" className="text-primary-700 font-semibold">
              {t('all_metrics')}
            </Link>
          </div>
        </section>

        {user.invite_code && (
          <section className="card bg-primary-50 border border-primary-500/20">
            <div className="text-sm font-semibold text-primary-700 mb-1">{t('invite_code_label')}</div>
            <div className="text-2xl font-bold tracking-wider">{user.invite_code}</div>
            <div className="text-sm text-ink-500 mt-2">{t('invite_code_hint')}</div>
          </section>
        )}
      </div>
    </Shell>
  );
}
