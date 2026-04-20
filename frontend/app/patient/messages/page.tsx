'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { Shell } from '@/components/Shell';
import { useAuthedUser } from '@/components/AuthGate';
import { api, type Caregiver } from '@/lib/api';

export default function PatientMessages() {
  const user = useAuthedUser(['patient']);
  const [cg, setCg] = useState<Caregiver[]>([]);

  useEffect(() => {
    if (!user) return;
    api<Caregiver[]>('/api/caregivers').then(setCg);
  }, [user]);

  if (!user) return null;

  return (
    <Shell user={user}>
      <h1 className="text-2xl font-bold mb-4">Сообщения</h1>
      <div className="space-y-2">
        {cg.map((c) => (
          <Link
            key={c.id}
            href={`/patient/messages/${c.id}`}
            className="card !p-4 flex items-center gap-4 hover:bg-ink-100"
          >
            <div className="w-12 h-12 rounded-full bg-primary-100 flex items-center justify-center text-xl font-bold text-primary-700">
              {c.full_name.split(' ').map((w) => w[0]).slice(0, 2).join('')}
            </div>
            <div className="flex-1">
              <div className="font-bold">{c.full_name}</div>
              <div className="text-sm text-ink-500">
                {c.relation === 'doctor' ? 'Врач' : 'Родственник'}
                {c.phone && ` · ${c.phone}`}
              </div>
            </div>
          </Link>
        ))}
        {cg.length === 0 && (
          <div className="card text-ink-500">
            Пока никто не подключён. Передайте ваш код приглашения врачу или родственнику.
          </div>
        )}
      </div>
    </Shell>
  );
}
