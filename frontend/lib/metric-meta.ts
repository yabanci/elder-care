import type { MetricKind } from './api';

export const METRIC_META: Record<
  MetricKind,
  { label: string; unit: string; short: string; emoji: string; color: string }
> = {
  pulse: { label: 'Пульс', short: 'Пульс', unit: 'уд/мин', emoji: '💓', color: '#e11d48' },
  bp_sys: { label: 'Давление (верхнее)', short: 'АД верх.', unit: 'мм рт.ст.', emoji: '🩺', color: '#0d9488' },
  bp_dia: { label: 'Давление (нижнее)', short: 'АД ниж.', unit: 'мм рт.ст.', emoji: '🩺', color: '#0f766e' },
  glucose: { label: 'Глюкоза', short: 'Сахар', unit: 'ммоль/л', emoji: '🩸', color: '#ea580c' },
  temperature: { label: 'Температура', short: 'Темп.', unit: '°C', emoji: '🌡️', color: '#f59e0b' },
  weight: { label: 'Вес', short: 'Вес', unit: 'кг', emoji: '⚖️', color: '#64748b' },
  spo2: { label: 'Сатурация', short: 'SpO₂', unit: '%', emoji: '🫁', color: '#14b8a6' },
};

export function formatValue(kind: MetricKind, v: number): string {
  if (kind === 'glucose') return v.toFixed(1);
  if (kind === 'temperature') return v.toFixed(1);
  if (kind === 'weight') return v.toFixed(1);
  return Math.round(v).toString();
}
