<script lang="ts">
  import { onMount } from 'svelte';
  import { listJobs } from '$lib/api/jobs';
  let jobs = $state<any[]>([]);
  let loading = $state(true);
  let filter = $state('');

  onMount(async () => {
    try { jobs = await listJobs(); } catch(e) {} finally { loading = false; }
  });

  const statusColors: Record<string, string> = {
    completed: 'text-emerald-400 bg-emerald-500/10', failed: 'text-red-400 bg-red-500/10',
    running: 'text-blue-400 bg-blue-500/10', pending: 'text-zinc-400 bg-zinc-500/10',
    cancelled: 'text-amber-400 bg-amber-500/10'
  };

  let filtered = $derived(filter ? jobs.filter(j => j.status === filter) : jobs);
</script>

<div class="p-6 space-y-6">
  <div class="flex items-center justify-between">
    <div><h1 class="text-2xl font-bold text-white">Jobs</h1><p class="text-zinc-500 text-sm mt-1">Automation task history</p></div>
  </div>

  <div class="flex gap-2">
    {#each ['','pending','running','completed','failed'] as f}
      <button onclick={() => filter = f}
        class="px-3 py-1.5 rounded-lg text-xs font-medium transition-colors
               {filter === f ? 'bg-blue-600 text-white' : 'bg-zinc-900 text-zinc-400 border border-zinc-800 hover:border-zinc-700'}">
        {f || 'All'}
      </button>
    {/each}
  </div>

  {#if loading}<div class="skeleton h-64 rounded-xl"></div>
  {:else if filtered.length === 0}
    <div class="text-center py-16"><p class="text-zinc-500">No jobs found.</p></div>
  {:else}
    <div class="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
      <table class="w-full">
        <thead><tr class="border-b border-zinc-800 text-left text-xs text-zinc-500">
          <th class="p-3 font-medium">Type</th><th class="p-3 font-medium">Platform</th>
          <th class="p-3 font-medium">Status</th><th class="p-3 font-medium">Device</th><th class="p-3 font-medium">Created</th>
        </tr></thead>
        <tbody>
          {#each filtered as job}
            <tr class="border-b border-zinc-800/50 hover:bg-zinc-800/30 transition-colors">
              <td class="p-3"><a href="/jobs/{job.id}" class="text-sm text-blue-400 hover:underline">{job.type}</a></td>
              <td class="p-3"><span class="text-xs px-1.5 py-0.5 rounded {job.platform === 'instagram' ? 'bg-pink-500/10 text-pink-400' : 'bg-cyan-500/10 text-cyan-400'}">{job.platform}</span></td>
              <td class="p-3"><span class="text-xs px-1.5 py-0.5 rounded {statusColors[job.status] || ''}">{job.status}</span></td>
              <td class="p-3 text-xs text-zinc-500 font-mono">{job.device_serial || '—'}</td>
              <td class="p-3 text-xs text-zinc-500">{new Date(job.created_at).toLocaleString()}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
