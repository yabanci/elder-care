'use client';

import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, ReferenceLine } from 'recharts';
import type { Metric, MetricKind } from '@/lib/api';
import { METRIC_META } from '@/lib/metric-meta';

export function MetricChart({ kind, metrics }: { kind: MetricKind; metrics: Metric[] }) {
  const meta = METRIC_META[kind];
  const data = [...metrics]
    .filter((m) => m.kind === kind)
    .sort((a, b) => +new Date(a.measured_at) - +new Date(b.measured_at))
    .map((m) => ({
      t: new Date(m.measured_at).toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit' }),
      v: m.value,
    }));

  const mean = data.length ? data.reduce((s, d) => s + d.v, 0) / data.length : 0;

  return (
    <div className="card">
      <div className="flex items-center gap-2 mb-4">
        <span className="text-2xl">{meta.emoji}</span>
        <div className="text-lg font-bold">{meta.label}</div>
        <div className="text-sm text-ink-500 ml-auto">{meta.unit}</div>
      </div>
      <div className="h-56">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={data} margin={{ top: 5, right: 10, bottom: 5, left: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
            <XAxis dataKey="t" tick={{ fontSize: 12 }} />
            <YAxis tick={{ fontSize: 12 }} domain={['auto', 'auto']} />
            <Tooltip contentStyle={{ borderRadius: 12, border: '1px solid #e2e8f0' }} />
            {mean > 0 && (
              <ReferenceLine y={mean} stroke="#94a3b8" strokeDasharray="4 4" label={{ value: `норма ${mean.toFixed(1)}`, fontSize: 11, fill: '#64748b', position: 'right' }} />
            )}
            <Line type="monotone" dataKey="v" stroke={meta.color} strokeWidth={3} dot={{ r: 4 }} activeDot={{ r: 6 }} />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
