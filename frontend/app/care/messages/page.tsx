'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { Shell } from '@/components/Shell';
import { useAuthedUser } from '@/components/AuthGate';
import { api, type LinkedPatient } from '@/lib/api';
import { useI18n } from '@/lib/i18n';

export default function CareMessages() {
  const user = useAuthedUser(['doctor', 'family']);
  const { t } = useI18n();
  const [patients, setPatients] = useState<LinkedPatient[]>([]);

  useEffect(() => {
    if (!user) return;
    api<LinkedPatient[]>('/api/patients').then(setPatients);
  }, [user]);

  if (!user) return null;

  return (
    <Shell user={user}>
      <h1 className="text-2xl font-bold mb-4">{t('messages_title')}</h1>
      <div className="space-y-2">
        {patients.map((p) => (
          <Link
            key={p.patient_id}
            href={`/care/messages/${p.patient_id}`}
            className="card !p-4 flex items-center gap-4 hover:bg-ink-100"
          >
            <div className="w-12 h-12 rounded-full bg-primary-100 flex items-center justify-center text-xl font-bold text-primary-700">
              {p.full_name.split(' ').map((w) => w[0]).slice(0, 2).join('')}
            </div>
            <div className="flex-1">
              <div className="font-bold">{p.full_name}</div>
              <div className="text-sm text-ink-500">
                {p.email}
                {p.phone && ` · ${p.phone}`}
              </div>
            </div>
          </Link>
        ))}
        {patients.length === 0 && <div className="card text-ink-500">{t('care_no_patients')}</div>}
      </div>
    </Shell>
  );
}
