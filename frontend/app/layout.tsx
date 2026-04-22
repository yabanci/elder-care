import './globals.css';
import type { Metadata } from 'next';
import { I18nProvider } from '@/lib/i18n';

export const metadata: Metadata = {
  title: 'ElderCare — здоровье под контролем',
  description: 'Система мониторинга здоровья для пожилых людей',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ru" suppressHydrationWarning>
      <body>
        <I18nProvider>{children}</I18nProvider>
      </body>
    </html>
  );
}
