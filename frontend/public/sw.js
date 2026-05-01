// ElderCare service worker — receives Web Push notifications and renders
// them as native browser notifications. Click bounces to the URL the
// payload specifies (typically /patient/alerts).
//
// This file is intentionally framework-free (no imports, no bundler) so
// it can be served as a static asset from /sw.js and registered without
// any build step.

self.addEventListener('install', (event) => {
  // Skip waiting so updates apply on next refresh.
  event.waitUntil(self.skipWaiting());
});

self.addEventListener('activate', (event) => {
  event.waitUntil(self.clients.claim());
});

self.addEventListener('push', (event) => {
  let payload = {};
  try {
    payload = event.data ? event.data.json() : {};
  } catch (e) {
    payload = { title: 'ElderCare', body: event.data ? event.data.text() : 'New alert' };
  }
  const title = payload.title || 'ElderCare';
  const opts = {
    body: payload.body || 'New alert',
    icon: '/icon-192.png',
    badge: '/icon-192.png',
    data: { url: payload.url || '/' },
    tag: payload.alert_id || 'eldercare-alert',
    renotify: true,
  };
  event.waitUntil(self.registration.showNotification(title, opts));
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const url = event.notification.data && event.notification.data.url;
  if (!url) return;
  event.waitUntil(
    self.clients.matchAll({ type: 'window' }).then((clientList) => {
      for (const client of clientList) {
        if (client.url.includes(url) && 'focus' in client) {
          return client.focus();
        }
      }
      if (self.clients.openWindow) return self.clients.openWindow(url);
      return null;
    }),
  );
});
