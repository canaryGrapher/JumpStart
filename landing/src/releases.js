// Full release history for the downloads page. downloads.js only resolves the
// single latest build for the hero/footer buttons; this module fetches every
// published release so users can grab an older version or a beta.
import { createSignal } from "solid-js";
import { OWNER, REPO } from "./downloads";

const CACHE_KEY = "jumpstart_releases";
const CACHE_TTL = 60 * 60 * 1000; // 1 hour
const PER_PAGE = 50;

// Each platform is matched against the asset filename produced by the release
// workflow (jumpstart_<tag>_<suffix>).
export const PLATFORMS = [
  { id: "macos", label: "macOS", match: "macos" },
  { id: "windows", label: "Windows", match: "windows" },
  { id: "linux", label: "Linux", match: "linux" },
];

export const [releases, setReleases] = createSignal([]);
export const [status, setStatus] = createSignal("loading"); // loading | ready | error

// isBeta treats anything GitHub flags as a pre-release, or any tag carrying a
// pre-release suffix, as the beta channel. Matches the in-app updater's rule.
function isBeta(rel) {
  const tag = (rel.tag_name || "").toLowerCase();
  return !!rel.prerelease || tag.includes("-beta") || tag.includes("-rc") || tag.includes("-alpha");
}

function bytesLabel(n) {
  if (!n) return "";
  const mb = n / 1024 / 1024;
  return mb >= 1 ? `${mb.toFixed(1)} MB` : `${Math.round(n / 1024)} KB`;
}

function normalize(rel) {
  const assets = (rel.assets || []).map((a) => ({
    name: a.name,
    url: a.browser_download_url,
    size: bytesLabel(a.size),
    downloads: a.download_count || 0,
  }));

  const byPlatform = {};
  for (const p of PLATFORMS) {
    byPlatform[p.id] = assets.find((a) => a.name.toLowerCase().includes(p.match)) || null;
  }

  return {
    tag: rel.tag_name,
    name: rel.name || rel.tag_name,
    beta: isBeta(rel),
    publishedAt: rel.published_at,
    url: rel.html_url,
    body: rel.body || "",
    assets: byPlatform,
    totalDownloads: assets.reduce((sum, a) => sum + a.downloads, 0),
  };
}

// load fetches the release list once per hour and caches the normalized shape
// so navigating away and back doesn't re-hit the API (which is rate limited to
// 60 requests/hour for anonymous callers).
export function loadReleases() {
  try {
    const cached = JSON.parse(localStorage.getItem(CACHE_KEY) || "null");
    if (cached && cached.data && Date.now() - cached.at < CACHE_TTL) {
      setReleases(cached.data);
      setStatus("ready");
      return;
    }
  } catch (e) {
    /* ignore cache errors */
  }

  setStatus("loading");
  fetch(`https://api.github.com/repos/${OWNER}/${REPO}/releases?per_page=${PER_PAGE}`, {
    headers: { Accept: "application/vnd.github+json" },
  })
    .then((r) => (r.ok ? r.json() : Promise.reject(r.status)))
    .then((list) => {
      const data = (list || []).filter((r) => !r.draft).map(normalize);
      setReleases(data);
      setStatus(data.length ? "ready" : "error");
      try {
        localStorage.setItem(CACHE_KEY, JSON.stringify({ at: Date.now(), data }));
      } catch (e) {
        /* ignore */
      }
    })
    .catch(() => setStatus("error"));
}

// detectPlatform guesses the visitor's OS so their build can be highlighted.
// Returns null when it can't tell, in which case nothing is highlighted.
export function detectPlatform() {
  const ua = (navigator.userAgent || "").toLowerCase();
  if (ua.includes("mac")) return "macos";
  if (ua.includes("win")) return "windows";
  if (ua.includes("linux") || ua.includes("x11")) return "linux";
  return null;
}

export function formatDate(iso) {
  if (!iso) return "";
  try {
    return new Date(iso).toLocaleDateString(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  } catch (e) {
    return "";
  }
}
