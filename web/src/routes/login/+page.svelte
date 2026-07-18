<script lang="ts">
  import { login as apiLogin } from '$lib/api/auth';
  import { login as storeLogin } from '$lib/stores/auth';
  import { goto } from '$app/navigation';

  let username = $state('');
  let password = $state('');
  let error = $state('');
  let loading = $state(false);

  async function handleSubmit(e: Event) {
    e.preventDefault();
    error = '';
    if (!username || !password) { error = 'All fields are required.'; return; }
    loading = true;
    try {
      const result = await apiLogin(username, password);
      storeLogin({ id: '', username }, result.token);
      goto('/');
    } catch (err: any) {
      error = err.message || 'Login failed';
    } finally { loading = false; }
  }
</script>

<div class="min-h-screen flex items-center justify-center bg-zinc-950 p-4">
  <div class="w-full max-w-sm">
    <div class="text-center mb-8">
      <h1 class="text-3xl font-bold text-white"><span class="text-blue-500">Reg</span>Bot</h1>
      <p class="text-zinc-500 mt-2 text-sm">Social Media Manager</p>
    </div>
    <form onsubmit={handleSubmit} class="bg-zinc-900 border border-zinc-800 rounded-xl p-6 space-y-4">
      <div>
        <label class="block text-xs font-medium text-zinc-400 mb-1.5">Username</label>
        <input type="text" bind:value={username}
          class="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-white
                 placeholder-zinc-500 focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500
                 transition-colors" placeholder="Enter your username" />
      </div>
      <div>
        <label class="block text-xs font-medium text-zinc-400 mb-1.5">Password</label>
        <input type="password" bind:value={password}
          class="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-white
                 placeholder-zinc-500 focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500
                 transition-colors" placeholder="Enter your password" />
      </div>
      {#if error}
        <p class="text-red-400 text-xs">{error}</p>
      {/if}
      <button type="submit" disabled={loading}
        class="w-full bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white font-medium rounded-lg
               px-4 py-2.5 text-sm transition-colors">
        {loading ? 'Signing in...' : 'Sign In'}
      </button>
    </form>
  </div>
</div>
