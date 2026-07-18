import { writable } from 'svelte/store';

export interface Device {
  id: string; serial: string; model: string; state: string;
  android_version: string; last_seen_at: string;
}

export const devices = writable<Device[]>([]);
export const activeDevice = writable<string | null>(null);

export function setActive(serial: string) {
  activeDevice.set(serial);
  localStorage.setItem('activeDevice', serial);
}

export function initDevice() {
  const s = localStorage.getItem('activeDevice');
  if (s) activeDevice.set(s);
}
