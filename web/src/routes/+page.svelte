<script lang="ts">
  import { onMount } from 'svelte';
  import { listDevices, type Device } from '$lib/api/devices';
  import { listJobs } from '$lib/api/jobs';
  import { api } from '$lib/api/client';

  let devices = $state<Device[]>([]);
  let jobs = $state<any[]>([]);
  let loading = $state(true);

  onMount(async () => {
    try {
      const [d, j] = await Promise.all([listDevices(), listJobs()]);
      devices = d;
      jobs = j;
    } catch(e) {} finally { loading = false; }
  });
</script>

<div class="p-6 space-y-6">
  <div>
    <h1 class="text-2xl font-bold text-white">Dashboard</h1>
    <p class="text-zinc-500 text-sm mt-1">Overview of your social media automation</p>
  </div>

  {#if loading}
    <div class="grid grid-cols-4 gap-4">
      {#each Array(4) as _}
        <div class="skeleton h-24 rounded-xl"></div>
      {/each}
    </div>
  {:else}
    <div class="grid grid-cols-4 gap-4">
      <StatCard label="Online Devices" value={devices.filter(d => d.state === 'online' || d.state === 'device').length} color="green" />
      <StatCard label="Running Jobs" value={jobs.filter(j => j.status === 'running').length} color="blue" />
      <StatCard label="Completed Today" value={jobs.filter(j => j.status === 'completed').length} color="zinc" />
      <StatCard label="Total Accounts" value="—" color="purple" />
    </div>

    <div class="grid grid-cols-2 gap-6">
      <div class="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
        <h2 class="text-sm font-semibold text-zinc-300 mb-3">Recent Jobs</h2>
        {#if jobs.length === 0}
          <p class="text-zinc-600 text-sm py-8 text-center">No jobs yet. Start by registering an account.</p>
        {:else}
          <div class="space-y-2">
            {#each jobs.slice(0, 8) as job}
              <a href="/jobs/{job.id}" class="flex items-center justify-between p-2 rounded-lg hover:bg-zinc-800/50 transition-colors">
                <div class="flex items-center gap-3">
                  <span class="text-xs px-1.5 py-0.5 rounded font-mono {job.platform === 'instagram' ? 'bg-pink-500/10 text-pink-400' : 'bg-cyan-500/10 text-cyan-400'}">{job.platform}</span>
                  <span class="text-sm text-zinc-300">{job.type}</span>
                </div>
                <span class="text-xs {job.status === 'completed' ? 'text-emerald-400' : job.status === 'failed' ? 'text-red-400' : job.status === 'running' ? 'text-blue-400' : 'text-zinc-500'}">{job.status}</span>
              </a>
            {/each}
          </div>
        {/if}
      </div>
      <div class="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
        <h2 class="text-sm font-semibold text-zinc-300 mb-3">Quick Actions</h2>
        <div class="grid grid-cols-2 gap-2">
          <a href="/register" class="p-3 rounded-lg border border-zinc-800 hover:bg-zinc-800/50 transition-colors text-sm text-zinc-300">Register Account</a>
          <a href="/like" class="p-3 rounded-lg border border-zinc-800 hover:bg-zinc-800/50 transition-colors text-sm text-zinc-300">Like Post</a>
          <a href="/comment" class="p-3 rounded-lg border border-zinc-800 hover:bg-zinc-800/50 transition-colors text-sm text-zinc-300">Comment</a>
          <a href="/post" class="p-3 rounded-lg border border-zinc-800 hover:bg-zinc-800/50 transition-colors text-sm text-zinc-300">Create Post</a>
        </div>
      </div>
    </div>
  {/if}
</div>

{#snippet StatCard(details: { label: string; value: any; color: string })}
  {@const colors: Record<string, string> = {
    green: 'border-emerald-500/20', blue: 'border-blue-500/20', zinc: 'border-zinc-700', purple: 'border-purple-500/20'
  }}
  <div class="bg-zinc-900 border {colors[details.color]} rounded-xl p-4">
    <p class="text-xs text-zinc-500 font-medium">{details.label}</p>
    <p class="text-2xl font-bold text-white mt-1">{details.value}</p>
  </div>
{/snippet}
