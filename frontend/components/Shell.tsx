'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { Activity, Pill, Users, MessageSquare, LogOut, AlertTriangle, Calendar } from 'lucide-react';
import { clearAuth, type User } from '@/lib/api';
import { useI18n } from '@/lib/i18n';

export function Shell({
  children,
  user,
}: {
  children: React.ReactNode;
  user: User;
}) {
  const pathname = usePathname();
  const router = useRouter();
  const { t } = useI18n();
  const [alertCount, setAlertCount] = useState(0);

  useEffect(() => {
    const raw = localStorage.getItem('alertCount');
    if (raw) setAlertCount(parseInt(raw, 10) || 0);
  }, [pathname]);

  const nav = navFor(user.role, t);

  function logout() {
    clearAuth();
    router.push('/login');
  }

  const roleLabel = {
    patient: t('role_patient'),
    doctor: t('role_doctor'),
    family: t('role_family'),
  }[user.role];

  return (
    <div className="min-h-screen flex flex-col">
      <header className="bg-white border-b border-ink-300 sticky top-0 z-20">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 flex items-center justify-between h-16">
          <Link href="/" className="flex items-center gap-2 font-bold text-xl text-primary-700">
            <span className="inline-flex items-center justify-center w-10 h-10 rounded-xl bg-primary-600 text-white">
              ❤
            </span>
            ElderCare
          </Link>
          <div className="flex items-center gap-3">
            {user.role === 'patient' ? (
              <Link href="/patient/profile" className="hidden sm:block text-right hover:opacity-80">
                <div className="font-semibold">{user.full_name}</div>
                <div className="text-sm text-ink-500">{roleLabel}</div>
              </Link>
            ) : (
              <div className="hidden sm:block text-right">
                <div className="font-semibold">{user.full_name}</div>
                <div className="text-sm text-ink-500">{roleLabel}</div>
              </div>
            )}
            <button onClick={logout} className="btn-ghost !min-h-12 !px-4" aria-label={t('logout')}>
              <LogOut className="w-5 h-5" /> <span className="hidden sm:inline">{t('logout')}</span>
            </button>
          </div>
        </div>
      </header>

      <main className="flex-1 max-w-6xl mx-auto w-full p-4 sm:p-6 pb-28">{children}</main>

      <nav className="fixed bottom-0 inset-x-0 bg-white border-t border-ink-300 z-20">
        <div
          className="max-w-6xl mx-auto grid"
          style={{ gridTemplateColumns: `repeat(${nav.length}, minmax(0, 1fr))` }}
        >
          {nav.map((item) => {
            const active = pathname === item.href || pathname.startsWith(item.href + '/');
            return (
              <Link
                key={item.href}
                href={item.href}
                className={`flex flex-col items-center justify-center py-3 gap-1 ${
                  active ? 'text-primary-700' : 'text-ink-500'
                }`}
              >
                <item.Icon className="w-6 h-6" />
                <span className="text-sm font-semibold">{item.label}</span>
                {item.href === '/alerts' && alertCount > 0 && (
                  <span className="absolute mt-0 ml-7 text-xs bg-danger-500 text-white rounded-full px-1.5">
                    {alertCount}
                  </span>
                )}
              </Link>
            );
          })}
        </div>
      </nav>
    </div>
  );
}

function navFor(role: string, t: (k: string) => string) {
  if (role === 'patient') {
    return [
      { href: '/patient', label: t('nav_home'), Icon: Activity },
      { href: '/patient/medications', label: t('nav_meds'), Icon: Pill },
      { href: '/patient/plans', label: t('nav_plans'), Icon: Calendar },
      { href: '/patient/alerts', label: t('nav_alerts'), Icon: AlertTriangle },
      { href: '/patient/messages', label: t('nav_messages'), Icon: MessageSquare },
    ];
  }
  return [
    { href: '/care', label: t('nav_patients'), Icon: Users },
    { href: '/care/alerts', label: t('nav_alerts'), Icon: AlertTriangle },
    { href: '/care/messages', label: t('nav_messages'), Icon: MessageSquare },
    { href: '/care/link', label: t('nav_add'), Icon: Activity },
  ];
}
