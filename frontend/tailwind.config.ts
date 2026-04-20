import type { Config } from 'tailwindcss';

const config: Config = {
  content: ['./app/**/*.{ts,tsx}', './components/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#f0fdfa',
          100: '#ccfbf1',
          500: '#14b8a6',
          600: '#0d9488',
          700: '#0f766e',
        },
        accent: {
          500: '#f97316',
          600: '#ea580c',
        },
        danger: {
          500: '#e11d48',
          600: '#be123c',
        },
        warn: {
          500: '#f59e0b',
        },
        ok: {
          500: '#10b981',
        },
        ink: {
          900: '#0f172a',
          700: '#334155',
          500: '#64748b',
          300: '#cbd5e1',
          100: '#f1f5f9',
        },
      },
      fontSize: {
        base: ['1.125rem', { lineHeight: '1.6' }],
        lg: ['1.25rem', { lineHeight: '1.6' }],
        xl: ['1.5rem', { lineHeight: '1.4' }],
        '2xl': ['1.875rem', { lineHeight: '1.3' }],
        '3xl': ['2.25rem', { lineHeight: '1.2' }],
      },
      borderRadius: {
        xl: '1rem',
        '2xl': '1.25rem',
      },
      boxShadow: {
        card: '0 1px 3px rgba(15,23,42,.06), 0 8px 24px rgba(15,23,42,.05)',
      },
    },
  },
  plugins: [],
};

export default config;
