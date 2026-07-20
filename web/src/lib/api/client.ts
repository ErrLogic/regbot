import { token, logout } from '$lib/stores/auth';
import { get } from 'svelte/store';

const BASE = '/api/v1';

function getToken(): string | null {
  // Svelte store (fast after init), fall back to localStorage (survives
  // hard-reload / store-not-yet-hydrated race).
  return get(token) || localStorage.getItem('token');
}

async function request<T>(method: string, path: string, body?: any): Promise<T> {
  const t = getToken();
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (t) headers['Authorization'] = `Bearer ${t}`;

  const res = await fetch(BASE + path, {
    method, headers,
    body: body ? JSON.stringify(body) : undefined
  });

  if (res.status === 401) { logout(); throw new Error('Unauthorized'); }

  const json = await res.json();
  if (json.error) throw new Error(json.error);
  return json.data as T;
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: any) => request<T>('POST', path, body),
  del: <T>(path: string) => request<T>('DELETE', path)
};
