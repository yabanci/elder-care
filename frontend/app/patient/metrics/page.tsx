'use client';

import { useEffect, useState } from 'react';
import { Shell } from '@/components/Shell';
import { useAuthedUser } from '@/components/AuthGate';
import { MetricChart } from '@/components/MetricChart';
import { MetricEntryCard } from '@/components/MetricEntryCard';
import { api, type Metric, type MetricKind } from '@/lib/api';

const ALL: MetricKind[] = ['pulse', 'bp_sys', 'bp_dia', 'glucose', 'spo2', 'temperature', 'weight'];

export default function MetricsPage() {
  const user = useAuthedUser(['patient']);
  const [metrics, setMetrics] = useState<Metric[]>([]);

  useEffect(() => {
    if (!user) return;
    api<Metric[]>('/api/metrics').then(setMetrics);
  }, [user]);

  if (!user) return null;

  return (
    <Shell user={user}>
      <h1 className="text-2xl font-bold mb-4">Мои показатели</h1>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {ALL.map((k) => (
          <MetricChart key={k} kind={k} metrics={metrics} />
        ))}
      </div>
      <h2 className="text-xl font-bold mt-8 mb-3">Добавить измерение</h2>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        {ALL.map((k) => (
          <MetricEntryCard
            key={k}
            kind={k}
            onSaved={() => api<Metric[]>('/api/metrics').then(setMetrics)}
          />
        ))}
      </div>
    </Shell>
  );
}
