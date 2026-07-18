import { writable } from 'svelte/store';

export interface User { id: string; username: string }
export const user = writable<User | null>(null);
export const token = writable<string | null>(null);

export function login(u: User, t: string) {
  user.set(u);
  token.set(t);
  localStorage.setItem('token', t);
  localStorage.setItem('user', JSON.stringify(u));
}

export function logout() {
  user.set(null);
  token.set(null);
  localStorage.removeItem('token');
  localStorage.removeItem('user');
}

export function init() {
  const t = localStorage.getItem('token');
  const u = localStorage.getItem('user');
  if (t && u) {
    try { token.set(t); user.set(JSON.parse(u)); } catch { logout(); }
  }
}
