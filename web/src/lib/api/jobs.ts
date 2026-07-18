import { api } from './client';

export function listJobs(params?: Record<string, string>) {
  const q = params ? '?' + new URLSearchParams(params).toString() : '';
  return api.get<any[]>('/jobs' + q);
}
export function getJob(id: string) { return api.get<any>('/jobs/' + id); }
export function getJobLogs(id: string) { return api.get<any[]>('/jobs/' + id + '/logs'); }
export function cancelJob(id: string) { return api.post<any>('/jobs/' + id + '/cancel'); }
export function createRegister(params: any) { return api.post<any>('/jobs/register', params); }
export function createLike(params: any) { return api.post<any>('/jobs/like', params); }
export function createComment(params: any) { return api.post<any>('/jobs/comment', params); }
export function createUpdateProfile(params: any) { return api.post<any>('/jobs/update-profile', params); }
export function createPost(params: any) { return api.post<any>('/jobs/create-post', params); }
export function createWatchLive(params: any) { return api.post<any>('/jobs/watch-live', params); }

export function streamJobLogs(id: string, onMessage: (entry: any) => void): EventSource {
  const t = localStorage.getItem('token');
  const es = new EventSource('/api/v1/jobs/' + id + '/stream?token=' + t);
  es.onmessage = (e) => { try { onMessage(JSON.parse(e.data)); } catch {} };
  es.onerror = () => es.close();
  return es;
}
