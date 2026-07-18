// Run as a pure client-side SPA: no server-side rendering (the app relies on
// localStorage-based auth and talks to the Go API at runtime), and no
// prerendering (routes like /jobs/[id] are dynamic). adapter-static serves the
// built assets with an index.html fallback.
export const ssr = false;
export const prerender = false;
