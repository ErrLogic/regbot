<script lang="ts">
  import { goto } from '$app/navigation';
  import { createComment } from '$lib/api/jobs';
  import { activeDevice } from '$lib/stores/devices';
  let platform = $state('instagram'); let postUrl = $state(''); let text = $state('');
  let loading = $state(false); let error = $state('');

  async function submit(e: Event) {
    e.preventDefault(); error = '';
    if (!$activeDevice) { error = 'Select a device first.'; return; }
    loading = true;
    try { const r = await createComment({ platform, device_serial: $activeDevice, post_url: postUrl, text }); goto('/jobs/' + r.id); }
    catch(err: any) { error = err.message; loading = false; }
  }
</script>
<div class="p-6 max-w-lg mx-auto space-y-6">
  <div><h1 class="text-2xl font-bold text-white">Comment on Post</h1><p class="text-zinc-500 text-sm mt-1">Post a comment by URL</p></div>
  <form onsubmit={submit} class="space-y-4">
    <div class="flex gap-2">
      <button type="button" onclick={() => platform = 'instagram'} class="flex-1 py-2.5 rounded-lg text-sm font-medium {platform === 'instagram' ? 'bg-pink-500/10 text-pink-400 border border-pink-500/20' : 'bg-zinc-900 text-zinc-400 border border-zinc-800'}">Instagram</button>
      <button type="button" onclick={() => platform = 'tiktok'} class="flex-1 py-2.5 rounded-lg text-sm font-medium {platform === 'tiktok' ? 'bg-cyan-500/10 text-cyan-400 border border-cyan-500/20' : 'bg-zinc-900 text-zinc-400 border border-zinc-800'}">TikTok</button>
    </div>
    <div><label class="block text-xs font-medium text-zinc-400 mb-1.5">Post URL</label><input type="url" bind:value={postUrl} class="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-white placeholder-zinc-600 focus:outline-none focus:ring-2 focus:ring-blue-500/50" /></div>
    <div><label class="block text-xs font-medium text-zinc-400 mb-1.5">Comment</label><textarea bind:value={text} rows={3} class="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-white placeholder-zinc-600 focus:outline-none focus:ring-2 focus:ring-blue-500/50 resize-none" placeholder="Write a comment..."></textarea></div>
    {#if error}<p class="text-red-400 text-xs">{error}</p>{/if}
    <button type="submit" disabled={loading || !$activeDevice} class="w-full bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white font-medium rounded-lg px-4 py-2.5 text-sm">Post Comment</button>
  </form>
</div>
