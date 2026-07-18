<script lang="ts">
  import { goto } from '$app/navigation';
  import { createPost } from '$lib/api/jobs';
  import { api } from '$lib/api/client';
  import { activeDevice } from '$lib/stores/devices';
  let platform = $state('instagram'); let caption = $state(''); let loading = $state(false); let error = $state('');
  let mediaIds = $state<string[]>([]); let uploading = $state(false);

  async function handleUpload(e: Event) {
    const f = (e.target as HTMLInputElement).files?.[0]; if (!f) return;
    uploading = true;
    const fd = new FormData(); fd.append('file', f);
    try {
      const t = localStorage.getItem('token') || '';
      const res = await fetch('/api/v1/media/upload', { method: 'POST', headers: { Authorization: 'Bearer ' + t }, body: fd });
      const json = await res.json();
      mediaIds = [...mediaIds, json.data.id];
    } catch {} finally { uploading = false; }
  }

  async function submit(e: Event) {
    e.preventDefault(); error = '';
    if (!$activeDevice) { error = 'Select a device first.'; return; }
    if (mediaIds.length === 0) { error = 'Upload at least one media file.'; return; }
    loading = true;
    try { const r = await createPost({ platform, device_serial: $activeDevice, caption, media_ids: mediaIds }); goto('/jobs/' + r.id); }
    catch(err: any) { error = err.message; loading = false; }
  }
</script>
<div class="p-6 max-w-lg mx-auto space-y-6">
  <div><h1 class="text-2xl font-bold text-white">Create Post</h1><p class="text-zinc-500 text-sm mt-1">Publish a new post with media</p></div>
  <form onsubmit={submit} class="space-y-4">
    <div class="flex gap-2">
      <button type="button" onclick={() => platform = 'instagram'} class="flex-1 py-2.5 rounded-lg text-sm font-medium {platform === 'instagram' ? 'bg-pink-500/10 text-pink-400 border border-pink-500/20' : 'bg-zinc-900 text-zinc-400 border border-zinc-800'}">Instagram</button>
      <button type="button" onclick={() => platform = 'tiktok'} class="flex-1 py-2.5 rounded-lg text-sm font-medium {platform === 'tiktok' ? 'bg-cyan-500/10 text-cyan-400 border border-cyan-500/20' : 'bg-zinc-900 text-zinc-400 border border-zinc-800'}">TikTok</button>
    </div>
    <div>
      <label class="block text-xs font-medium text-zinc-400 mb-1.5">Media</label>
      <label class="block border-2 border-dashed border-zinc-700 hover:border-zinc-500 rounded-lg p-8 text-center cursor-pointer transition-colors">
        <input type="file" accept="image/*,video/*" onchange={handleUpload} class="hidden" />
        <span class="text-zinc-500 text-sm">{uploading ? 'Uploading...' : 'Click to upload photo or video'}</span>
      </label>
      {#if mediaIds.length > 0}<p class="text-xs text-emerald-400 mt-1">{mediaIds.length} file(s) uploaded</p>{/if}
    </div>
    <div><label class="block text-xs font-medium text-zinc-400 mb-1.5">Caption</label><textarea bind:value={caption} rows={3} class="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-white placeholder-zinc-600 focus:outline-none focus:ring-2 focus:ring-blue-500/50 resize-none" placeholder="Write a caption..."></textarea></div>
    {#if error}<p class="text-red-400 text-xs">{error}</p>{/if}
    <button type="submit" disabled={loading || !$activeDevice} class="w-full bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white font-medium rounded-lg px-4 py-2.5 text-sm">Publish Post</button>
  </form>
</div>
