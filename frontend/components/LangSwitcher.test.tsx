import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { LangSwitcher } from './LangSwitcher';
import { I18nProvider, useI18n } from '@/lib/i18n';

function CurrentLang() {
  const { lang } = useI18n();
  return <span data-testid="lang">{lang}</span>;
}

function renderSwitcher() {
  return render(
    <I18nProvider>
      <LangSwitcher />
      <CurrentLang />
    </I18nProvider>,
  );
}

describe('LangSwitcher', () => {
  it('shows three language buttons', () => {
    renderSwitcher();
    expect(screen.getByRole('button', { name: 'Русский' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Қазақша' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'English' })).toBeInTheDocument();
  });

  it('clicking a language updates the active lang and persists to localStorage', () => {
    renderSwitcher();
    expect(screen.getByTestId('lang')).toHaveTextContent('ru');

    fireEvent.click(screen.getByRole('button', { name: 'English' }));

    expect(screen.getByTestId('lang')).toHaveTextContent('en');
    expect(localStorage.getItem('lang')).toBe('en');
  });

  it('marks the active language with primary styling', () => {
    renderSwitcher();
    fireEvent.click(screen.getByRole('button', { name: 'Қазақша' }));
    const kk = screen.getByRole('button', { name: 'Қазақша' });
    expect(kk.className).toMatch(/border-primary-600/);
  });
});
