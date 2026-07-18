import { api } from './client';

export interface Device {
  id: string; serial: string; model: string; state: string;
  android_version: string; last_seen_at: string;
}

export function listDevices() { return api.get<Device[]>('/devices'); }
export function refreshDevices() { return api.post<Device[]>('/devices/refresh'); }
