import './globals.css';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'ElderCare — здоровье под контролем',
  description: 'Система мониторинга здоровья для пожилых людей',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ru">
      <body>{children}</body>
    </html>
  );
}
