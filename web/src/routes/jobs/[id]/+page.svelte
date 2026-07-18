<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { page } from '$app/stores';
  import { getJob, getJobLogs, cancelJob, streamJobLogs } from '$lib/api/jobs';

  let job = $state<any>(null);
  let logs = $state<any[]>([]);
  let loading = $state(true);
  let es = $state<EventSource | null>(null);

  onMount(async () => {
    const id = $page.params.id;
    try {
      job = await getJob(id);
      logs = await getJobLogs(id);
      if (job.status === 'pending' || job.status === 'running') {
        es = streamJobLogs(id, (entry) => { logs = [...logs, entry]; });
      }
    } catch(e) {} finally { loading = false; }
  });

  onDestroy(() => es?.close());

  async function handleCancel() {
    if (!job) return;
    try { await cancelJob(job.id); job = { ...job, status: 'cancelled' }; es?.close(); }
    catch(e) {}
  }

  const statusColors: Record<string, string> = {
    completed: 'text-emerald-400 bg-emerald-500/10 border-emerald-500/20',
    failed: 'text-red-400 bg-red-500/10 border-red-500/20',
    running: 'text-blue-400 bg-blue-500/10 border-blue-500/20',
    pending: 'text-zinc-400 bg-zinc-500/10 border-zinc-500/20',
    cancelled: 'text-amber-400 bg-amber-500/10 border-amber-500/20'
  };
</script>

<div class="p-6 space-y-6">
  <div class="flex items-center gap-3">
    <a href="/jobs" class="text-zinc-500 hover:text-zinc-300 transition-colors">&larr; Back</a>
    <h1 class="text-2xl font-bold text-white">Job Detail</h1>
  </div>

  {#if loading}<div class="skeleton h-96 rounded-xl"></div>
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
                <span class="text-zinc-600 shrink-0 w-20">{new Date(log.timestamp).toLocaleTimeString()}</span>
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
          <DetailRow label="Status">
            <span class="text-xs px-1.5 py-0.5 rounded {statusColors[job.status] || ''}">{job.status}</span>
          </DetailRow>
          <DetailRow label="Type" value={job.type} />
          <DetailRow label="Platform" value={job.platform} />
          <DetailRow label="Device" value={job.device_serial || '—'} />
          <DetailRow label="Created" value={new Date(job.created_at).toLocaleString()} />
          {#if job.error_message}
            <DetailRow label="Error" value={job.error_message} cls="text-red-400" />
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

{#snippet DetailRow(details: { label: string; value?: string; cls?: string; children?: any })}
  <div class="flex justify-between items-start gap-4">
    <span class="text-xs text-zinc-500">{details.label}</span>
    {#if details.children}
      {@render details.children()}
    {:else}
      <span class="text-xs text-zinc-300 text-right break-all {details.cls || ''}">{details.value || '—'}</span>
    {/if}
  </div>
{/snippet}
