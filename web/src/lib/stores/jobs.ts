import { writable } from 'svelte/store';

export interface Job {
  id: string; type: string; platform: string; status: string;
  device_serial: string; params: any; result: any;
  error_message: string; created_at: string; started_at: string; completed_at: string;
}
export const jobs = writable<Job[]>([]);
export const activeJobId = writable<string | null>(null);
