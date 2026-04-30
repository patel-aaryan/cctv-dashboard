// In production the SPA is served by the Go binary at the same origin, so
// relative URLs ("/api/...", "/media/...") resolve correctly.
// In `vite dev`, requests are proxied to VITE_API_URL by vite.config.ts,
// so relative URLs still work without code changes.
const API_BASE = import.meta.env.VITE_API_URL ?? '';

export interface Video {
  id: number;
  camera_name: string;
  timestamp: string;
  media_url: string;
}

export async function fetchVideos(): Promise<Video[]> {
  const res = await fetch(`${API_BASE}/api/videos`, { headers: { Accept: 'application/json' } });
  if (!res.ok) {
    throw new Error(`Failed to fetch videos (HTTP ${res.status})`);
  }
  return res.json();
}

export function mediaUrl(path: string): string {
  return `${API_BASE}${path}`;
}
