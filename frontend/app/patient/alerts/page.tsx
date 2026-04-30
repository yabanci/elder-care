'use client';

import { useEffect, useState } from 'react';
import { Shell } from '@/components/Shell';
import { useAuthedUser } from '@/components/AuthGate';
import { api, type Alert, type MetricKind } from '@/lib/api';
import { METRIC_META, formatValue } from '@/lib/metric-meta';
import { useI18n } from '@/lib/i18n';

// localizedReason returns the i18n-mapped reason string for an alert,
// falling back to the legacy `reason` field for pre-v2 alerts whose
// reason_code is "legacy" (or unknown to the dictionary).
function localizedReason(t: (k: string) => string, a: Alert): string {
  if (!a.reason_code || a.reason_code === 'legacy') {
    return a.reason || t('reason_legacy');
  }
  const key = `reason_${a.reason_code}`;
  const localized = t(key);
  // useI18n's t() returns the key itself when no translation exists.
  if (localized === key) return a.reason || a.reason_code;
  return localized;
}

export default function AlertsPage() {
  const user = useAuthedUser(['patient']);
  const { t, lang } = useI18n();
  const [alerts, setAlerts] = useState<Alert[]>([]);

  useEffect(() => {
    if (!user) return;
    refresh();
  }, [user]);

  async function refresh() {
    const a = await api<Alert[]>('/api/alerts');
    setAlerts(a);
    const unack = a.filter((x) => !x.acknowledged).length;
    localStorage.setItem('alertCount', String(unack));
  }

  async function ack(id: string) {
    await api(`/api/alerts/${id}/acknowledge`, { method: 'POST' });
    refresh();
  }

  if (!user) return null;

  return (
    <Shell user={user}>
      <h1 className="text-2xl font-bold mb-4">{t('alerts_title')}</h1>
      <div className="space-y-2">
        {alerts.map((a) => {
          const meta = METRIC_META[a.kind as MetricKind];
          const color =
            a.severity === 'critical'
              ? 'border-danger-500/30 bg-danger-500/5'
              : 'border-warn-500/30 bg-warn-500/5';
          const localeTag = lang === 'kk' ? 'kk-KZ' : lang === 'en' ? 'en-US' : 'ru-RU';
          return (
            <div key={a.id} className={`card !p-4 border ${color}`}>
              <div className="flex items-start gap-3">
                <div className="text-2xl">{meta?.emoji ?? '⚠'}</div>
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <span className="font-bold">{meta?.label ?? a.kind}</span>
                    <span
                      className={
                        a.severity === 'critical' ? 'badge-danger' : 'badge-warn'
                      }
                    >
                      {a.severity === 'critical' ? t('severity_critical') : t('severity_warning')}
                    </span>
                    {a.acknowledged && <span className="badge-ok">✓</span>}
                  </div>
                  <div className="text-ink-700 mt-1">{localizedReason(t, a)}</div>
                  {a.value != null && (
                    <div className="text-sm text-ink-500 mt-1">
                      {t('alert_value')}: <b>{formatValue(a.kind as MetricKind, a.value)} {meta?.unit}</b>
                      {a.baseline_mean != null && (
                        <>
                          {' '}· {t('alert_baseline')} ≈ {formatValue(a.kind as MetricKind, a.baseline_mean)}
                          {a.baseline_std ? ` ± ${a.baseline_std.toFixed(1)}` : ''}
                        </>
                      )}
                    </div>
                  )}
                  <div className="text-xs text-ink-500 mt-1">
                    {new Date(a.created_at).toLocaleString(localeTag)}
                  </div>
                </div>
                {!a.acknowledged && (
                  <button onClick={() => ack(a.id)} className="btn-ghost !min-h-10 !px-3">
                    {t('alerts_ack')}
                  </button>
                )}
              </div>
            </div>
          );
        })}
        {alerts.length === 0 && <div className="card text-ink-500">{t('alerts_empty')}</div>}
      </div>
    </Shell>
  );
}
