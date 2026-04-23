import '@testing-library/jest-dom/vitest';
import { afterEach, beforeEach } from 'vitest';
import { cleanup } from '@testing-library/react';

// happy-dom + Node 25 can leave localStorage undefined; provide a minimal in-memory shim.
class MemoryStorage implements Storage {
  private store = new Map<string, string>();
  get length() {
    return this.store.size;
  }
  clear() {
    this.store.clear();
  }
  getItem(key: string) {
    return this.store.has(key) ? (this.store.get(key) as string) : null;
  }
  key(i: number) {
    return Array.from(this.store.keys())[i] ?? null;
  }
  removeItem(key: string) {
    this.store.delete(key);
  }
  setItem(key: string, value: string) {
    this.store.set(key, String(value));
  }
}

function ensureStorage() {
  const w = globalThis as unknown as { localStorage?: Storage; sessionStorage?: Storage };
  if (!w.localStorage || typeof w.localStorage.getItem !== 'function') {
    w.localStorage = new MemoryStorage();
  }
  if (!w.sessionStorage || typeof w.sessionStorage.getItem !== 'function') {
    w.sessionStorage = new MemoryStorage();
  }
}

beforeEach(() => {
  ensureStorage();
  localStorage.clear();
  // Force an unsupported browser lang so the i18n default is deterministic.
  Object.defineProperty(globalThis.navigator, 'language', {
    configurable: true,
    get: () => 'xx-XX',
  });
});

afterEach(() => {
  cleanup();
  localStorage.clear();
});
