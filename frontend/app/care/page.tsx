'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { Shell } from '@/components/Shell';
import { useAuthedUser } from '@/components/AuthGate';
import { api, type LinkedPatient } from '@/lib/api';
import { Users, Plus } from 'lucide-react';

export default function CareHome() {
  const user = useAuthedUser(['doctor', 'family']);
  const [patients, setPatients] = useState<LinkedPatient[]>([]);

  useEffect(() => {
    if (!user) return;
    api<LinkedPatient[]>('/api/patients').then(setPatients);
  }, [user]);

  if (!user) return null;

  return (
    <Shell user={user}>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold">Мои пациенты</h1>
        <Link href="/care/link" className="btn-primary">
          <Plus className="w-5 h-5" /> Добавить
        </Link>
      </div>
      <div className="space-y-2">
        {patients.map((p) => (
          <Link
            key={p.patient_id}
            href={`/care/patient/${p.patient_id}`}
            className="card !p-4 flex items-center gap-4 hover:bg-ink-100"
          >
            <div className="w-14 h-14 rounded-full bg-primary-100 flex items-center justify-center text-xl font-bold text-primary-700">
              {p.full_name.split(' ').map((w) => w[0]).slice(0, 2).join('')}
            </div>
            <div className="flex-1">
              <div className="font-bold text-lg">{p.full_name}</div>
              <div className="text-sm text-ink-500">
                {p.email}
                {p.phone && ` · ${p.phone}`}
              </div>
            </div>
          </Link>
        ))}
        {patients.length === 0 && (
          <div className="card flex items-start gap-3">
            <Users className="w-6 h-6 text-ink-500" />
            <div>
              <div className="font-bold">Пациентов пока нет</div>
              <div className="text-ink-500 mt-1">
                Попросите пациента сообщить вам код приглашения и нажмите «Добавить».
              </div>
            </div>
          </div>
        )}
      </div>
    </Shell>
  );
}
