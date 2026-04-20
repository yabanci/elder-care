'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { Shell } from '@/components/Shell';
import { useAuthedUser } from '@/components/AuthGate';
import { api, type Alert, type LinkedPatient, type MetricKind } from '@/lib/api';
import { METRIC_META, formatValue } from '@/lib/metric-meta';

interface Row {
  alert: Alert;
  patient: LinkedPatient;
}

export default function CareAlerts() {
  const user = useAuthedUser(['doctor', 'family']);
  const [rows, setRows] = useState<Row[]>([]);

  useEffect(() => {
    if (!user) return;
    (async () => {
      const patients = await api<LinkedPatient[]>('/api/patients');
      const all: Row[] = [];
      for (const p of patients) {
        const alerts = await api<Alert[]>(`/api/patients/${p.patient_id}/alerts`);
        for (const a of alerts) {
          if (!a.acknowledged) all.push({ alert: a, patient: p });
        }
      }
      all.sort((a, b) => +new Date(b.alert.created_at) - +new Date(a.alert.created_at));
      setRows(all);
    })();
  }, [user]);

  if (!user) return null;

  return (
    <Shell user={user}>
      <h1 className="text-2xl font-bold mb-4">Активные оповещения</h1>
      <div className="space-y-2">
        {rows.map(({ alert: a, patient: p }) => {
          const meta = METRIC_META[a.kind as MetricKind];
          const color = a.severity === 'critical'
            ? 'border-danger-500/30 bg-danger-500/5'
            : 'border-warn-500/30 bg-warn-500/5';
          return (
            <Link
              key={a.id}
              href={`/care/patient/${p.patient_id}`}
              className={`card !p-4 block border ${color} hover:shadow-lg`}
            >
              <div className="flex items-start gap-3">
                <div className="text-2xl">{meta?.emoji ?? '⚠'}</div>
                <div className="flex-1">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="font-bold">{p.full_name}</span>
                    <span className={a.severity === 'critical' ? 'badge-danger' : 'badge-warn'}>
                      {a.severity === 'critical' ? 'Критично' : 'Внимание'}
                    </span>
                    <span className="text-ink-500">· {meta?.label ?? a.kind}</span>
                  </div>
                  <div className="text-sm mt-1">{a.reason}</div>
                  {a.value != null && meta && (
                    <div className="text-sm text-ink-500 mt-1">
                      {formatValue(a.kind as MetricKind, a.value)} {meta.unit}
                      {a.baseline_mean != null && (
                        <> · норма ≈ {formatValue(a.kind as MetricKind, a.baseline_mean)}</>
                      )}
                    </div>
                  )}
                  <div className="text-xs text-ink-500 mt-1">
                    {new Date(a.created_at).toLocaleString('ru-RU')}
                  </div>
                </div>
              </div>
            </Link>
          );
        })}
        {rows.length === 0 && <div className="card text-ink-500">Активных оповещений нет.</div>}
      </div>
    </Shell>
  );
}
