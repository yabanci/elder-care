'use client';

import { use, useEffect, useState } from 'react';
import Link from 'next/link';
import { Shell } from '@/components/Shell';
import { useAuthedUser } from '@/components/AuthGate';
import { SummaryGrid } from '@/components/SummaryGrid';
import { MetricChart } from '@/components/MetricChart';
import { api, type Alert, type LinkedPatient, type Metric, type MetricKind } from '@/lib/api';
import { METRIC_META } from '@/lib/metric-meta';
import { useI18n } from '@/lib/i18n';
import { MessageSquare } from 'lucide-react';

const CHART_KINDS: MetricKind[] = ['pulse', 'bp_sys', 'glucose', 'spo2'];

export default function PatientDetail({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const user = useAuthedUser(['doctor', 'family']);
  const { t } = useI18n();
  const [patient, setPatient] = useState<LinkedPatient | null>(null);
  const [summary, setSummary] = useState<Metric[]>([]);
  const [metrics, setMetrics] = useState<Metric[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);

  useEffect(() => {
    if (!user) return;
    (async () => {
      const patients = await api<LinkedPatient[]>('/api/patients');
      setPatient(patients.find((p) => p.patient_id === id) ?? null);

      const [s, m, a] = await Promise.all([
        api<Metric[]>(`/api/patients/${id}/metrics/summary`),
        api<Metric[]>(`/api/patients/${id}/metrics`),
        api<Alert[]>(`/api/patients/${id}/alerts`),
      ]);
      setSummary(s);
      setMetrics(m);
      setAlerts(a);
    })();
  }, [user, id]);

  if (!user) return null;

  return (
    <Shell user={user}>
      <div className="flex items-start justify-between gap-3 mb-4">
        <div>
          <Link href="/care" className="text-primary-700 font-semibold">← {t('nav_patients')}</Link>
          <h1 className="text-2xl font-bold mt-1">{patient?.full_name ?? '...'}</h1>
          <div className="text-ink-500">
            {patient?.email}
            {patient?.phone && ` · ${patient.phone}`}
          </div>
        </div>
        <Link href={`/care/messages/${id}`} className="btn-primary">
          <MessageSquare className="w-5 h-5" /> {t('send')}
        </Link>
      </div>

      {alerts.filter((a) => !a.acknowledged).length > 0 && (
        <div className="card !bg-danger-500/5 border border-danger-500/20 mb-4">
          <div className="font-bold text-danger-500 mb-2">
            {t('care_alerts_title')} ({alerts.filter((a) => !a.acknowledged).length})
          </div>
          <ul className="space-y-1">
            {alerts
              .filter((a) => !a.acknowledged)
              .slice(0, 5)
              .map((a) => (
                <li key={a.id} className="text-sm">
                  <span className={a.severity === 'critical' ? 'badge-danger' : 'badge-warn'}>
                    {a.severity === 'critical' ? t('severity_critical') : t('severity_warning')}
                  </span>
                  {' '}
                  <b>{METRIC_META[a.kind as MetricKind]?.label ?? a.kind}</b> — {a.reason}
                </li>
              ))}
          </ul>
        </div>
      )}

      <h2 className="text-xl font-bold mb-2">{t('recent_metrics')}</h2>
      <SummaryGrid metrics={summary} />

      <h2 className="text-xl font-bold mt-6 mb-2">{t('metrics_title')}</h2>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {CHART_KINDS.map((k) => (
          <MetricChart key={k} kind={k} metrics={metrics} />
        ))}
      </div>
    </Shell>
  );
}
