<script lang="ts">
  import { listDevices, type Device } from '$lib/api/devices';
  import { activeDevice, setActive } from '$lib/stores/devices';
  import { onMount } from 'svelte';

  let devices = $state<Device[]>([]);
  let loading = $state(true);
  let error = $state('');

  async function refresh() {
    loading = true; error = '';
    try { devices = await listDevices(); } catch(e: any) { error = e.message; }
    finally { loading = false; }
  }

  onMount(refresh);
</script>

<div class="p-6 space-y-6">
  <div class="flex items-center justify-between">
    <div><h1 class="text-2xl font-bold text-white">Devices</h1><p class="text-zinc-500 text-sm mt-1">Connected ADB devices</p></div>
    <button onclick={refresh} disabled={loading}
      class="bg-zinc-800 hover:bg-zinc-700 text-zinc-300 px-4 py-2 rounded-lg text-sm transition-colors">
      {loading ? 'Scanning...' : 'Refresh'}
    </button>
  </div>

  {#if error}<p class="text-red-400 text-sm">{error}</p>{/if}

  {#if loading}
    <div class="grid grid-cols-3 gap-4">{#each Array(3) as _}<div class="skeleton h-32 rounded-xl"></div>{/each}</div>
  {:else if devices.length === 0}
    <div class="text-center py-16">
      <p class="text-zinc-500">No devices found. Connect an Android device via ADB.</p>
    </div>
  {:else}
    <div class="grid grid-cols-3 gap-4">
      {#each devices as d}
        <button onclick={() => setActive(d.serial)}
          class="text-left bg-zinc-900 border rounded-xl p-4 transition-all
                 {$activeDevice === d.serial ? 'border-blue-500/50 ring-1 ring-blue-500/20' : 'border-zinc-800 hover:border-zinc-700'}">
          <div class="flex items-center justify-between mb-3">
            <span class="text-sm font-medium text-white font-mono">{d.serial}</span>
            <span class="w-2 h-2 rounded-full
              {d.state === 'online' || d.state === 'device' ? 'bg-emerald-500' : d.state === 'busy' ? 'bg-amber-500' : 'bg-zinc-600'}">
            </span>
          </div>
          <div class="space-y-1 text-xs text-zinc-500">
            <p>{d.model || 'Unknown model'}</p>
            <p>Android {d.android_version || 'Unknown'}</p>
            <p class="capitalize">{d.state}</p>
          </div>
          {#if $activeDevice === d.serial}
            <p class="text-blue-400 text-xs mt-2 font-medium">● Active</p>
          {/if}
        </button>
      {/each}
    </div>
  {/if}
</div>
