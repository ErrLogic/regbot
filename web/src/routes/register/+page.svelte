<script lang="ts">
  import { goto } from '$app/navigation';
  import { createRegister } from '$lib/api/jobs';
  import { activeDevice } from '$lib/stores/devices';

  let platform = $state('instagram');
  let email = $state('');
  let dryRun = $state(false);
  let loading = $state(false);
  let error = $state('');

  async function handleSubmit(e: Event) {
    e.preventDefault();
    error = '';
    if (!$activeDevice) { error = 'Select a device first.'; return; }
    if (!email) { error = 'Email is required.'; return; }
    loading = true;
    try {
      const result = await createRegister({
        platform, email, dry_run: dryRun, device_serial: $activeDevice
      });
      goto('/jobs/' + result.id);
    } catch (err: any) { error = err.message; loading = false; }
  }
</script>

<div class="p-6 max-w-lg mx-auto space-y-6">
  <div><h1 class="text-2xl font-bold text-white">Register Account</h1><p class="text-zinc-500 text-sm mt-1">Create a new social media account</p></div>

  <form onsubmit={handleSubmit} class="space-y-4">
    <div class="flex gap-2">
      <button type="button" onclick={() => platform = 'instagram'}
        class="flex-1 py-2.5 rounded-lg text-sm font-medium transition-colors
               {platform === 'instagram' ? 'bg-pink-500/10 text-pink-400 border border-pink-500/20' : 'bg-zinc-900 text-zinc-400 border border-zinc-800'}">
        Instagram
      </button>
      <button type="button" onclick={() => platform = 'tiktok'}
        class="flex-1 py-2.5 rounded-lg text-sm font-medium transition-colors
               {platform === 'tiktok' ? 'bg-cyan-500/10 text-cyan-400 border border-cyan-500/20' : 'bg-zinc-900 text-zinc-400 border border-zinc-800'}">
        TikTok
      </button>
    </div>

    <div>
      <label class="block text-xs font-medium text-zinc-400 mb-1.5">Device</label>
      <input type="text" value={$activeDevice || 'No device selected'} disabled
        class="w-full bg-zinc-800/50 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-500" />
    </div>

    <div>
      <label class="block text-xs font-medium text-zinc-400 mb-1.5">Email</label>
      <input type="email" bind:value={email}
        class="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-white
               placeholder-zinc-600 focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500"
        placeholder="user@gmail.com" />
    </div>

    <label class="flex items-center gap-3 text-sm text-zinc-400">
      <input type="checkbox" bind:checked={dryRun} class="rounded bg-zinc-800 border-zinc-700" />
      Dry run (validate but don't submit)
    </label>

    {#if error}<p class="text-red-400 text-xs">{error}</p>{/if}

    <button type="submit" disabled={loading || !$activeDevice}
      class="w-full bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white font-medium rounded-lg
             px-4 py-2.5 text-sm transition-colors">
      {loading ? 'Creating...' : 'Start Registration'}
    </button>
  </form>
</div>
