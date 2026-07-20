<script lang="ts">
  import { onMount } from 'svelte';
  import { listDevices, type Device } from '$lib/api/devices';
  import { listJobs } from '$lib/api/jobs';

  let devices = $state<Device[]>([]);
  let jobs = $state<any[]>([]);
  let loading = $state(true);

  onMount(async () => {
    try { [devices, jobs] = await Promise.all([listDevices(), listJobs()]); } catch(e) {}
    finally { loading = false; }
  });
</script>

<div class="p-6 space-y-6">
  <h1 class="text-2xl font-bold text-white">Dashboard</h1>
  <p class="text-zinc-500 text-sm">Overview of your social media automation</p>

  {#if loading}
    <div class="grid grid-cols-4 gap-4">
      {#each [1,2,3,4] as _}<div class="skeleton h-24 rounded-xl"></div>{/each}
    </div>
  {:else}
    <div class="grid grid-cols-4 gap-4">
      <div class="bg-zinc-900 border border-emerald-500/20 rounded-xl p-4">
        <p class="text-xs text-zinc-500 font-medium">Online Devices</p>
        <p class="text-2xl font-bold text-white mt-1">{devices.filter((d: any) => d.state === 'online' || d.state === 'device').length}</p>
      </div>
      <div class="bg-zinc-900 border border-blue-500/20 rounded-xl p-4">
        <p class="text-xs text-zinc-500 font-medium">Running Jobs</p>
        <p class="text-2xl font-bold text-white mt-1">{jobs.filter((j: any) => j.status === 'running').length}</p>
      </div>
      <div class="bg-zinc-900 border border-zinc-700 rounded-xl p-4">
        <p class="text-xs text-zinc-500 font-medium">Completed Today</p>
        <p class="text-2xl font-bold text-white mt-1">{jobs.filter((j: any) => j.status === 'completed').length}</p>
      </div>
      <div class="bg-zinc-900 border border-purple-500/20 rounded-xl p-4">
        <p class="text-xs text-zinc-500 font-medium">Total Accounts</p>
        <p class="text-2xl font-bold text-white mt-1">—</p>
      </div>
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
