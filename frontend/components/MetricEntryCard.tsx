'use client';

import { useState } from 'react';
import { api, type Alert, type Metric, type MetricKind } from '@/lib/api';
import { METRIC_META } from '@/lib/metric-meta';

interface Props {
  kind: MetricKind;
  onSaved?: (m: Metric, alert: Alert | null) => void;
}

export function MetricEntryCard({ kind, onSaved }: Props) {
  const meta = METRIC_META[kind];
  const [value, setValue] = useState('');
  const [saving, setSaving] = useState(false);
  const [result, setResult] = useState<{ ok: boolean; alert: Alert | null; msg?: string } | null>(
    null,
  );

  async function save() {
    const num = parseFloat(value.replace(',', '.'));
    if (!isFinite(num)) return;
    setSaving(true);
    setResult(null);
    try {
      const res = await api<{ metric: Metric; alert: Alert | null }>('/api/metrics', {
        method: 'POST',
        body: JSON.stringify({ kind, value: num }),
      });
      setResult({ ok: true, alert: res.alert });
      setValue('');
      onSaved?.(res.metric, res.alert);
    } catch (err) {
      setResult({ ok: false, alert: null, msg: err instanceof Error ? err.message : 'Ошибка' });
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="card space-y-3">
      <div className="flex items-center gap-3">
        <div className="text-3xl">{meta.emoji}</div>
        <div>
          <div className="text-xl font-bold">{meta.label}</div>
          <div className="text-sm text-ink-500">{meta.unit}</div>
        </div>
      </div>
      <div className="flex gap-2">
        <input
          className="field !min-h-16 text-2xl font-bold"
          inputMode="decimal"
          placeholder="—"
          value={value}
          onChange={(e) => setValue(e.target.value)}
        />
        <button onClick={save} className="btn-primary !px-4" disabled={saving || !value}>
          {saving ? '...' : 'Сохранить'}
        </button>
      </div>
      {result?.ok && !result.alert && (
        <div className="badge-ok">✓ В норме</div>
      )}
      {result?.ok && result.alert && (
        <div
          className={
            result.alert.severity === 'critical' ? 'badge-danger' : 'badge-warn'
          }
        >
          ⚠ {result.alert.reason}
        </div>
      )}
      {result && !result.ok && <div className="text-danger-500 text-sm">{result.msg}</div>}
    </div>
  );
}
