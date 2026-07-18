<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  let accounts = $state<any[]>([]); let loading = $state(true);

  onMount(async () => {
    try { accounts = await api.get<any[]>('/accounts'); } catch(e) {} finally { loading = false; }
  });
</script>
<div class="p-6 space-y-6">
  <div><h1 class="text-2xl font-bold text-white">Accounts</h1><p class="text-zinc-500 text-sm mt-1">Registered social media accounts</p></div>
  {#if loading}<div class="skeleton h-64 rounded-xl"></div>
  {:else if accounts.length === 0}
    <div class="text-center py-16"><p class="text-zinc-500">No accounts registered yet.</p></div>
  {:else}
    <div class="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
      <table class="w-full">
        <thead><tr class="border-b border-zinc-800 text-left text-xs text-zinc-500"><th class="p-3 font-medium">Platform</th><th class="p-3 font-medium">Email</th><th class="p-3 font-medium">Username</th><th class="p-3 font-medium">Status</th><th class="p-3 font-medium">Created</th></tr></thead>
        <tbody>
          {#each accounts as a}
            <tr class="border-b border-zinc-800/50 hover:bg-zinc-800/30">
              <td class="p-3"><span class="text-xs px-1.5 py-0.5 rounded {a.platform === 'instagram' ? 'bg-pink-500/10 text-pink-400' : 'bg-cyan-500/10 text-cyan-400'}">{a.platform}</span></td>
              <td class="p-3 text-sm text-zinc-300">{a.email}</td>
              <td class="p-3 text-sm text-zinc-300">{a.username}</td>
              <td class="p-3 text-xs text-zinc-400">{a.status}</td>
              <td class="p-3 text-xs text-zinc-500">{new Date(a.created_at).toLocaleDateString()}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
