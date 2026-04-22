const API_BASE =
  process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090';

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
  }
}

function getToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('token');
}

export function setToken(token: string) {
  localStorage.setItem('token', token);
}

export function clearAuth() {
  localStorage.removeItem('token');
  localStorage.removeItem('user');
}

export async function api<T>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set('Content-Type', 'application/json');
  const token = getToken();
  if (token) headers.set('Authorization', `Bearer ${token}`);

  const res = await fetch(`${API_BASE}${path}`, { ...init, headers });
  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try {
      const body = await res.json();
      if (body.error) msg = body.error;
    } catch {}
    throw new ApiError(res.status, msg);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

export type Role = 'patient' | 'doctor' | 'family';

export interface User {
  id: string;
  email: string;
  full_name: string;
  role: Role;
  phone?: string;
  birth_date?: string;
  invite_code?: string;
  height_cm?: number;
  chronic_conditions?: string;
  bp_norm?: string;
  prescribed_meds?: string;
  onboarded: boolean;
  lang?: string;
}

export interface Metric {
  id: string;
  patient_id: string;
  kind: MetricKind;
  value: number;
  value_2?: number;
  note?: string;
  measured_at: string;
}

export type MetricKind =
  | 'pulse'
  | 'bp_sys'
  | 'bp_dia'
  | 'glucose'
  | 'temperature'
  | 'weight'
  | 'spo2';

export interface Alert {
  id: string;
  patient_id: string;
  metric_id?: string;
  severity: 'info' | 'warning' | 'critical';
  reason: string;
  kind: string;
  value?: number;
  baseline_mean?: number;
  baseline_std?: number;
  acknowledged: boolean;
  created_at: string;
}

export interface Medication {
  id: string;
  patient_id: string;
  name: string;
  dosage?: string;
  times_of_day: string[];
  start_date: string;
  end_date?: string;
  active: boolean;
  notes?: string;
}

export interface MedScheduleItem {
  medication_id: string;
  name: string;
  dosage?: string;
  scheduled_at: string;
  status: 'pending' | 'taken' | 'missed' | 'skipped';
}

export interface LinkedPatient {
  patient_id: string;
  full_name: string;
  email: string;
  phone?: string;
  relation: 'doctor' | 'family';
}

export interface Caregiver {
  id: string;
  full_name: string;
  email: string;
  phone?: string;
  relation: 'doctor' | 'family';
}

export interface Message {
  id: string;
  sender_id: string;
  recipient_id: string;
  body: string;
  read_at?: string;
  created_at: string;
  sender_name?: string;
}

export type PlanType = 'doctor_visit' | 'take_med' | 'rest' | 'other';

export interface Plan {
  id: string;
  patient_id: string;
  day_of_week: number;
  title: string;
  plan_type: PlanType;
  time_of_day?: string;
  created_at: string;
}
