<script lang="ts">
  import { onMount } from 'svelte';
  import { fetchVideos, mediaUrl, type Video } from '$lib/api';

  let videos = $state<Video[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  function formatTimestamp(iso: string): string {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return iso;
    return d.toLocaleString(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    });
  }

  async function load() {
    loading = true;
    error = null;
    try {
      videos = await fetchVideos();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  onMount(load);
</script>

<header class="topbar">
  <div class="brand">
    <span class="dot" aria-hidden="true"></span>
    <h1>CCTV Dashboard</h1>
  </div>
  <div class="meta">
    <span class="count">{videos.length} clip{videos.length === 1 ? '' : 's'}</span>
    <button onclick={load} disabled={loading}>
      {loading ? 'Refreshing…' : 'Refresh'}
    </button>
  </div>
</header>

<main>
  {#if loading && videos.length === 0}
    <p class="status">Loading videos…</p>
  {:else if error}
    <p class="status error">Error: {error}</p>
  {:else if videos.length === 0}
    <p class="status">No videos yet. Drop an .mp4 into the archive to see it appear here.</p>
  {:else}
    <div class="grid">
      {#each videos as v (v.id)}
        <article class="card">
          <header class="card-header">
            <h2>{v.camera_name}</h2>
            <time datetime={v.timestamp}>{formatTimestamp(v.timestamp)}</time>
          </header>
          <!-- svelte-ignore a11y_media_has_caption -->
          <video controls preload="metadata" src={mediaUrl(v.media_url)}></video>
        </article>
      {/each}
    </div>
  {/if}
</main>

<style>
  :global(html, body) {
    margin: 0;
    background: #0d1117;
    color: #e6edf3;
    font-family: system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif;
  }

  .topbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.9rem 1.25rem;
    border-bottom: 1px solid #30363d;
    position: sticky;
    top: 0;
    background: rgba(13, 17, 23, 0.92);
    backdrop-filter: blur(8px);
    z-index: 10;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 0.6rem;
  }

  .brand h1 {
    margin: 0;
    font-size: 1.05rem;
    font-weight: 600;
    letter-spacing: 0.01em;
  }

  .dot {
    width: 0.55rem;
    height: 0.55rem;
    border-radius: 50%;
    background: #f85149;
    box-shadow: 0 0 8px rgba(248, 81, 73, 0.7);
  }

  .meta {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    font-size: 0.875rem;
    color: #8b949e;
  }

  button {
    background: #238636;
    color: #fff;
    border: 1px solid #2ea043;
    padding: 0.4rem 0.9rem;
    border-radius: 6px;
    cursor: pointer;
    font: inherit;
    font-size: 0.875rem;
  }

  button:hover:not(:disabled) {
    background: #2ea043;
  }

  button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  main {
    padding: 1.25rem;
  }

  .status {
    text-align: center;
    color: #8b949e;
    padding: 3rem 1rem;
  }

  .status.error {
    color: #f85149;
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: 1rem;
  }

  .card {
    background: #161b22;
    border: 1px solid #30363d;
    border-radius: 8px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .card-header {
    padding: 0.7rem 0.95rem;
    border-bottom: 1px solid #30363d;
  }

  .card-header h2 {
    margin: 0 0 0.2rem;
    font-size: 0.95rem;
    font-weight: 600;
  }

  .card-header time {
    font-size: 0.8rem;
    color: #8b949e;
  }

  video {
    width: 100%;
    background: #000;
    aspect-ratio: 16 / 9;
    display: block;
  }
</style>
