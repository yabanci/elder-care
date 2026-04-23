import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { I18nProvider, persistLangLocally, useI18n } from './i18n';
import { createElement } from 'react';

const wrapper = ({ children }: { children: React.ReactNode }) =>
  createElement(I18nProvider, null, children);

describe('useI18n', () => {
  beforeEach(() => {
    // Force navigator.language to an unsupported value so the default falls through to 'ru'
    // unless a test sets localStorage explicitly.
    vi.stubGlobal('navigator', { ...navigator, language: 'fr-FR' });
  });

  it('defaults to ru when nothing is stored and browser lang is unsupported', () => {
    const { result } = renderHook(() => useI18n(), { wrapper });
    expect(result.current.lang).toBe('ru');
    expect(result.current.t('save')).toBe('Сохранить');
  });

  it('detects browser language when supported', () => {
    vi.stubGlobal('navigator', { ...navigator, language: 'kk-KZ' });
    const { result } = renderHook(() => useI18n(), { wrapper });
    expect(result.current.lang).toBe('kk');
  });

  it('reads stored lang on init (overrides browser)', () => {
    vi.stubGlobal('navigator', { ...navigator, language: 'kk-KZ' });
    localStorage.setItem('lang', 'en');
    const { result } = renderHook(() => useI18n(), { wrapper });
    expect(result.current.lang).toBe('en');
    expect(result.current.t('save')).toBe('Save');
  });

  it('falls back to ru dict when key missing in current lang', () => {
    localStorage.setItem('lang', 'en');
    const { result } = renderHook(() => useI18n(), { wrapper });
    // A non-existent key returns the key itself (final fallback).
    expect(result.current.t('definitely_not_a_key')).toBe('definitely_not_a_key');
  });

  it('changes language and persists to localStorage', () => {
    const { result } = renderHook(() => useI18n(), { wrapper });
    act(() => result.current.setLang('kk'));
    expect(result.current.lang).toBe('kk');
    expect(result.current.t('save')).toBe('Сақтау');
    expect(localStorage.getItem('lang')).toBe('kk');
  });

  it('ignores invalid stored lang and falls back to browser/ru', () => {
    localStorage.setItem('lang', 'de');
    const { result } = renderHook(() => useI18n(), { wrapper });
    expect(result.current.lang).toBe('ru');
  });
});

describe('persistLangLocally', () => {
  it('stores supported langs', () => {
    persistLangLocally('en');
    expect(localStorage.getItem('lang')).toBe('en');
  });

  it('uppercases & lowercases consistently', () => {
    persistLangLocally('KK');
    expect(localStorage.getItem('lang')).toBe('kk');
  });

  it('ignores unknown values', () => {
    localStorage.setItem('lang', 'en');
    persistLangLocally('xx');
    expect(localStorage.getItem('lang')).toBe('en');
  });

  it('ignores null/undefined', () => {
    localStorage.setItem('lang', 'en');
    persistLangLocally(null);
    persistLangLocally(undefined);
    expect(localStorage.getItem('lang')).toBe('en');
  });
});
