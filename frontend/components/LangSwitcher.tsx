'use client';

import { LANGS, useI18n, type Lang } from '@/lib/i18n';

export function LangSwitcher({ className = '' }: { className?: string }) {
  const { lang, setLang } = useI18n();
  return (
    <div className={`flex items-center justify-center gap-2 ${className}`}>
      {LANGS.map((l) => (
        <button
          key={l.code}
          type="button"
          onClick={() => setLang(l.code as Lang)}
          aria-label={l.label}
          className={`min-h-10 px-3 rounded-xl border-2 font-semibold text-sm ${
            lang === l.code
              ? 'border-primary-600 bg-primary-50 text-primary-700'
              : 'border-ink-300 bg-white text-ink-500'
          }`}
        >
          <span className="mr-1">{l.flag}</span>
          {l.code.toUpperCase()}
        </button>
      ))}
    </div>
  );
}
