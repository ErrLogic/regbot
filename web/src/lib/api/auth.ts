import { api } from './client';

export function login(username: string, password: string) {
  return api.post<{token: string; expires_at: string}>('/auth/login', { username, password });
}

export function register(username: string, password: string) {
  return api.post<{id: string; username: string}>('/auth/register', { username, password });
}
