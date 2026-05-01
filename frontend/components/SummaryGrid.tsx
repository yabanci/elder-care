'use client';

import { Metric, MetricKind } from '@/lib/api';
import { METRIC_META, formatValue } from '@/lib/metric-meta';
import { localeTag, useI18n } from '@/lib/i18n';

const ORDER: MetricKind[] = ['pulse', 'bp_sys', 'bp_dia', 'glucose', 'spo2', 'temperature', 'weight'];

export function SummaryGrid({ metrics }: { metrics: Metric[] }) {
  const { lang } = useI18n();
  const byKind = new Map(metrics.map((m) => [m.kind, m]));
  const tag = localeTag(lang);

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
      {ORDER.map((k) => {
        const m = byKind.get(k);
        const meta = METRIC_META[k];
        return (
          <div key={k} className="card !p-4">
            <div className="flex items-center gap-2 text-sm font-semibold text-ink-500">
              <span className="text-xl">{meta.emoji}</span>
              {meta.short}
            </div>
            <div className="mt-2 flex items-baseline gap-1">
              <div className="text-3xl font-bold">
                {m ? formatValue(k, m.value) : '—'}
              </div>
              <div className="text-sm text-ink-500">{meta.unit}</div>
            </div>
            {m && (
              <div className="text-xs text-ink-500 mt-1">
                {new Date(m.measured_at).toLocaleDateString(tag, {
                  day: 'numeric',
                  month: 'short',
                  hour: '2-digit',
                  minute: '2-digit',
                })}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
