import { createSignal } from "solid-js";

// Total downloads = sum of download_count across every asset of every release.
// GitHub only reports counts per-asset, so we must page through all releases.

const OWNER = "canaryGrapher";
const REPO = "JumpStart";

const [totalDownloads, setTotalDownloads] = createSignal(0);
export { totalDownloads };

const CACHE_KEY = "jumpstart_total_downloads";
const CACHE_TTL = 60 * 60 * 1000; // 1 hour

function sumReleases(releases) {
  return (releases || []).reduce(
    (total, rel) =>
      total +
      (rel.assets || []).reduce((s, a) => s + (a.download_count || 0), 0),
    0
  );
}

export function loadTotalDownloads() {
  try {
    const cached = JSON.parse(localStorage.getItem(CACHE_KEY) || "null");
    if (cached && Date.now() - cached.at < CACHE_TTL) {
      setTotalDownloads(cached.value);
      return;
    }
  } catch (e) {
    /* ignore cache errors */
  }

  fetch(`https://api.github.com/repos/${OWNER}/${REPO}/releases?per_page=100`, {
    headers: { Accept: "application/vnd.github+json" },
  })
    .then((r) => (r.ok ? r.json() : Promise.reject(r.status)))
    .then((releases) => {
      const value = sumReleases(releases);
      setTotalDownloads(value);
      try {
        localStorage.setItem(CACHE_KEY, JSON.stringify({ at: Date.now(), value }));
      } catch (e) {
        /* ignore */
      }
    })
    .catch(() => {
      /* keep 0 */
    });
}
