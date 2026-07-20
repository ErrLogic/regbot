<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { getJob, getJobLogs, cancelJob, streamJobLogs } from '$lib/api/jobs';

  let job = $state<any>(null);
  let logs = $state<any[]>([]);
  let loading = $state(true);
  let error = $state('');
  let es = $state<EventSource | null>(null);

  onMount(() => {
    const id = window.location.pathname.split('/').filter(Boolean).pop() || '';
    if (!id) { error = 'No job ID in URL'; loading = false; return; }

    getJob(id).then(j => {
      job = j;
      return getJobLogs(id);
    }).then(l => {
      logs = l || [];
      if (job && (job.status === 'pending' || job.status === 'running')) {
        es = streamJobLogs(id, (entry) => { logs = [...logs, entry]; });
      }
    }).catch(e => {
      error = (e && e.message) || 'Failed to load job data';
    }).finally(() => {
      loading = false;
    });
  });

  onDestroy(() => { try { es?.close(); } catch(e) {} });

  async function handleCancel() {
    if (!job) return;
    try { await cancelJob(job.id); job = { ...job, status: 'cancelled' }; try { es?.close(); } catch(e) {} }
    catch(e) {}
  }
</script>

<div class="p-6 space-y-6">
  <div class="flex items-center gap-3">
    <a href="/jobs" class="text-zinc-500 hover:text-zinc-300 transition-colors">&larr; Back</a>
    <h1 class="text-2xl font-bold text-white">Job Detail</h1>
  </div>

  {#if loading}
    <div class="skeleton h-96 rounded-xl"></div>
  {:else if error}
    <div class="bg-red-500/10 border border-red-500/20 rounded-xl p-6 text-center">
      <p class="text-red-400 text-sm">{error}</p>
      <a href="/login" class="text-blue-400 text-xs mt-2 inline-block hover:underline">Try logging in again</a>
    </div>
  {:else if job}
    <div class="grid grid-cols-3 gap-6">
      <div class="col-span-2 bg-zinc-900 border border-zinc-800 rounded-xl p-4">
        <h2 class="text-sm font-semibold text-zinc-300 mb-3">Logs</h2>
        <div class="log-viewer bg-zinc-950 rounded-lg p-3 h-96 overflow-y-auto space-y-1">
          {#if logs.length === 0}
            <span class="text-zinc-600">Waiting for logs...</span>
          {:else}
            {#each logs as log}
              <div class="flex gap-3 text-xs">
                <span class="text-zinc-600 shrink-0 w-20">{new Date(log.timestamp?.replace(' ', 'T')).toLocaleTimeString()}</span>
                <span class="shrink-0 w-10 {log.level === 'error' ? 'text-red-400' : log.level === 'warn' ? 'text-amber-400' : 'text-zinc-500'}">{log.level}</span>
                <span class="text-zinc-300">{log.message}</span>
              </div>
            {/each}
          {/if}
        </div>
      </div>
      <div class="space-y-4">
        <div class="bg-zinc-900 border border-zinc-800 rounded-xl p-4 space-y-3">
          <h2 class="text-sm font-semibold text-zinc-300">Details</h2>
          <div class="flex justify-between items-start gap-4">
            <span class="text-xs text-zinc-500">Status</span>
            <span class="text-xs px-1.5 py-0.5 rounded
              {job.status === 'completed' ? 'text-emerald-400 bg-emerald-500/10 border border-emerald-500/20' : ''}
              {job.status === 'failed' ? 'text-red-400 bg-red-500/10 border border-red-500/20' : ''}
              {job.status === 'running' ? 'text-blue-400 bg-blue-500/10 border border-blue-500/20' : ''}
              {job.status === 'pending' ? 'text-zinc-400 bg-zinc-500/10 border border-zinc-500/20' : ''}
              {job.status === 'cancelled' ? 'text-amber-400 bg-amber-500/10 border border-amber-500/20' : ''}">{job.status}</span>
          </div>
          <div class="flex justify-between items-start gap-4"><span class="text-xs text-zinc-500">Type</span><span class="text-xs text-zinc-300">{job.type}</span></div>
          <div class="flex justify-between items-start gap-4"><span class="text-xs text-zinc-500">Platform</span><span class="text-xs text-zinc-300">{job.platform}</span></div>
          <div class="flex justify-between items-start gap-4"><span class="text-xs text-zinc-500">Device</span><span class="text-xs text-zinc-300 font-mono">{job.device_serial || '—'}</span></div>
          <div class="flex justify-between items-start gap-4"><span class="text-xs text-zinc-500">Created</span><span class="text-xs text-zinc-300">{new Date(job.created_at).toLocaleString()}</span></div>
          {#if job.error_message}
            <div class="flex justify-between items-start gap-4"><span class="text-xs text-zinc-500">Error</span><span class="text-xs text-red-400 text-right break-all">{job.error_message}</span></div>
          {/if}
        </div>
        {#if job.status === 'pending' || job.status === 'running'}
          <button onclick={handleCancel}
            class="w-full bg-red-600/10 hover:bg-red-600/20 text-red-400 text-sm border border-red-500/20
                   rounded-lg px-4 py-2.5 transition-colors font-medium">Cancel Job</button>
        {/if}
      </div>
    </div>
  {/if}
</div>
