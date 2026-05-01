// Web Push subscription helper. Wraps the browser's PushManager +
// service-worker registration + base64 conversion for the VAPID key.
// Designed to be called from a UI event handler (so the permission
// prompt is allowed by browsers).
import { api } from './api';

interface PushKeyResponse {
  public_key: string;
  enabled: boolean;
}

function urlBase64ToBuffer(base64String: string): ArrayBuffer {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
  const raw = typeof window === 'undefined' ? '' : window.atob(base64);
  // Use a backing ArrayBuffer (not a SharedArrayBuffer) so the result is
  // accepted by PushManager.subscribe's BufferSource type.
  const buf = new ArrayBuffer(raw.length);
  const view = new Uint8Array(buf);
  for (let i = 0; i < raw.length; i++) view[i] = raw.charCodeAt(i);
  return buf;
}

export async function isPushSupported(): Promise<boolean> {
  return (
    typeof window !== 'undefined' &&
    'serviceWorker' in navigator &&
    'PushManager' in window &&
    'Notification' in window
  );
}

export async function getCurrentSubscription(): Promise<PushSubscription | null> {
  if (!(await isPushSupported())) return null;
  const reg = await navigator.serviceWorker.getRegistration('/sw.js');
  if (!reg) return null;
  return reg.pushManager.getSubscription();
}

export async function enablePush(): Promise<{ ok: boolean; reason?: string }> {
  if (!(await isPushSupported())) {
    return { ok: false, reason: 'not-supported' };
  }
  const keyResp = await api<PushKeyResponse>('/api/push/public-key');
  if (!keyResp.enabled || !keyResp.public_key) {
    return { ok: false, reason: 'server-disabled' };
  }
  if (Notification.permission === 'denied') {
    return { ok: false, reason: 'permission-denied' };
  }
  if (Notification.permission === 'default') {
    const granted = await Notification.requestPermission();
    if (granted !== 'granted') return { ok: false, reason: 'permission-denied' };
  }

  const reg =
    (await navigator.serviceWorker.getRegistration('/sw.js')) ||
    (await navigator.serviceWorker.register('/sw.js'));
  await navigator.serviceWorker.ready;

  let sub = await reg.pushManager.getSubscription();
  if (!sub) {
    sub = await reg.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToBuffer(keyResp.public_key),
    });
  }

  await api('/api/push/subscribe', {
    method: 'POST',
    body: JSON.stringify(sub.toJSON()),
  });
  return { ok: true };
}

export async function disablePush(): Promise<void> {
  const sub = await getCurrentSubscription();
  if (!sub) return;
  try {
    await api(`/api/push/subscribe?endpoint=${encodeURIComponent(sub.endpoint)}`, {
      method: 'DELETE',
    });
  } finally {
    await sub.unsubscribe();
  }
}
