import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BMICard } from './BMICard';
import { I18nProvider } from '@/lib/i18n';
import type { Metric, User } from '@/lib/api';

const baseUser: User = {
  id: 'u1',
  email: 'p@test.kz',
  full_name: 'Patient',
  role: 'patient',
  onboarded: true,
};

const weightMetric: Metric = {
  id: 'm1',
  patient_id: 'u1',
  kind: 'weight',
  value: 70,
  measured_at: '2026-01-01T00:00:00Z',
};

function renderBMI(props: Parameters<typeof BMICard>[0]) {
  return render(
    <I18nProvider>
      <BMICard {...props} />
    </I18nProvider>,
  );
}

describe('BMICard', () => {
  beforeEach(() => vi.restoreAllMocks());

  it('prompts for height when not set', () => {
    renderBMI({ user: baseUser, summary: [], onUserChange: vi.fn() });
    expect(screen.getByText(/Укажите свой рост/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Указать' })).toBeInTheDocument();
  });

  it('saves height via PATCH and calls onUserChange', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ ...baseUser, height_cm: 175 }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }) as unknown as Response,
    );
    const onUserChange = vi.fn();

    renderBMI({ user: baseUser, summary: [], onUserChange });

    fireEvent.click(screen.getByRole('button', { name: 'Указать' }));
    fireEvent.change(screen.getByPlaceholderText('170'), { target: { value: '175' } });
    fireEvent.click(screen.getByRole('button', { name: 'OK' }));

    await waitFor(() => expect(onUserChange).toHaveBeenCalled());
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringMatching(/\/api\/me$/),
      expect.objectContaining({ method: 'PATCH' }),
    );
    const body = JSON.parse((fetchMock.mock.calls[0][1] as RequestInit).body as string);
    expect(body).toEqual({ height_cm: 175 });
    expect(onUserChange.mock.calls[0][0].height_cm).toBe(175);
  });

  it('rejects out-of-range heights without calling fetch', () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch');
    renderBMI({ user: baseUser, summary: [], onUserChange: vi.fn() });

    fireEvent.click(screen.getByRole('button', { name: 'Указать' }));
    fireEvent.change(screen.getByPlaceholderText('170'), { target: { value: '50' } });
    fireEvent.click(screen.getByRole('button', { name: 'OK' }));

    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('shows "need weight" prompt when only height is known', () => {
    renderBMI({
      user: { ...baseUser, height_cm: 170 },
      summary: [],
      onUserChange: vi.fn(),
    });
    expect(screen.getByText(/добавьте замер веса/)).toBeInTheDocument();
  });

  it('renders BMI and Norma badge for normal weight', () => {
    renderBMI({
      user: { ...baseUser, height_cm: 170 },
      summary: [weightMetric],
      onUserChange: vi.fn(),
    });
    // 70 / 1.7^2 ≈ 24.2
    expect(screen.getByText('24.2')).toBeInTheDocument();
    expect(screen.getByText('Норма')).toBeInTheDocument();
  });

  it('renders Obese badge for high BMI', () => {
    renderBMI({
      user: { ...baseUser, height_cm: 160 },
      summary: [{ ...weightMetric, value: 95 }],
      onUserChange: vi.fn(),
    });
    // 95 / 1.6^2 ≈ 37.1
    expect(screen.getByText('37.1')).toBeInTheDocument();
    expect(screen.getByText('Ожирение')).toBeInTheDocument();
  });
});
