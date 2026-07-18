<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { user, token, init } from '$lib/stores/auth';
  import { initDevice } from '$lib/stores/devices';
  import '../app.css';

  let { children } = $props();

  let ready = $state(false);

  $effect(() => {
    init();
    initDevice();
    ready = true;
  });

  // Redirect to /login when not authenticated (except on the login page itself).
  $effect(() => {
    if (ready && !$token && page.url.pathname !== '/login') {
      goto('/login');
    }
  });

  const nav = [
    { section: 'Main', items: [
      { href: '/', icon: 'grid', label: 'Dashboard' },
      { href: '/devices', icon: 'smartphone', label: 'Devices' }
    ] },
    { section: 'Automation', items: [
      { href: '/register', icon: 'user-plus', label: 'Register' },
      { href: '/like', icon: 'heart', label: 'Like Post' },
      { href: '/comment', icon: 'message-circle', label: 'Comment' },
      { href: '/profile', icon: 'user', label: 'Update Profile' },
      { href: '/post', icon: 'plus-square', label: 'Create Post' },
      { href: '/live', icon: 'video', label: 'Watch Live' }
    ] },
    { section: 'History', items: [
      { href: '/jobs', icon: 'activity', label: 'Jobs' },
      { href: '/accounts', icon: 'users', label: 'Accounts' }
    ] }
  ];

  const iconMap: Record<string, string> = {
    grid: '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/></svg>',
    smartphone: '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="5" y="2" width="14" height="20" rx="2" ry="2"/><line x1="12" y1="18" x2="12.01" y2="18"/></svg>',
    'user-plus': '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><line x1="19" y1="8" x2="19" y2="14"/><line x1="22" y1="11" x2="16" y2="11"/></svg>',
    heart: '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>',
    'message-circle': '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>',
    user: '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>',
    'plus-square': '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><line x1="12" y1="8" x2="12" y2="16"/><line x1="8" y1="12" x2="16" y2="12"/></svg>',
    video: '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2" ry="2"/></svg>',
    activity: '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>',
    users: '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>'
  };
</script>

{#if ready}
  {#if $token && $user}
    <div class="flex h-screen overflow-hidden bg-zinc-950">
      <aside class="w-64 border-r border-zinc-800 bg-zinc-900 flex flex-col shrink-0">
        <div class="h-14 flex items-center px-4 border-b border-zinc-800">
          <a href="/" class="text-lg font-bold tracking-tight text-white">
            <span class="text-blue-500">Reg</span>Bot
          </a>
        </div>
        <nav class="flex-1 p-3 space-y-1 overflow-y-auto">
          {#each nav as group}
            <div class="pt-3 pb-1 px-2">
              <span class="text-[10px] font-semibold uppercase tracking-widest text-zinc-500">{group.section}</span>
            </div>
            {#each group.items as item}
              <a
                href={item.href}
                class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors
                  {page.url.pathname === item.href
                    ? 'bg-blue-600/10 text-blue-400 border border-blue-500/20'
                    : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/50'}"
              >
                <span class="w-4 h-4 flex items-center justify-center">{@html iconMap[item.icon]}</span>
                {item.label}
              </a>
            {/each}
          {/each}
        </nav>
        <div class="p-3 border-t border-zinc-800">
          <span class="text-xs text-zinc-500">{$user?.username}</span>
        </div>
      </aside>
      <main class="flex-1 overflow-y-auto">
        {@render children()}
      </main>
    </div>
  {:else}
    {@render children()}
  {/if}
{/if}
