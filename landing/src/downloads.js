import { createSignal } from "solid-js";

// Download links resolve to the LATEST GitHub release at runtime, so the
// landing page never needs a manual tag bump. GitHub's asset URLs already
// include the version in the filename (jumpstart_vX.Y.Z_<platform>...), so
// downloads land with a versioned name.

const OWNER = "canaryGrapher";
const REPO = "JumpStart";

export const RELEASES_PAGE = `https://github.com/${OWNER}/${REPO}/releases`;

// Offline/rate-limited fallback tag. The live API fetch overrides this; bump
// it only if you want the fallback to match a specific release.
const FALLBACK_TAG = "v1.2.0";

function assetURL(tag, suffix) {
  return `https://github.com/${OWNER}/${REPO}/releases/download/${tag}/jumpstart_${tag}_${suffix}`;
}

function urlsForTag(tag) {
  return {
    tag,
    macos: assetURL(tag, "macos-universal.zip"),
    windows: assetURL(tag, "windows-amd64.zip"),
    linux: assetURL(tag, "linux-amd64.tar.gz"),
  };
}

const [downloads, setDownloads] = createSignal(urlsForTag(FALLBACK_TAG));
export { downloads };

const CACHE_KEY = "jumpstart_latest_release";
const CACHE_TTL = 60 * 60 * 1000; // 1 hour

function pickAsset(assets, keyword, tag, fallbackSuffix) {
  const hit = (assets || []).find((a) =>
    a.name.toLowerCase().includes(keyword)
  );
  return hit ? hit.browser_download_url : assetURL(tag, fallbackSuffix);
}

// Fetch the newest release once (cached in localStorage) and update the
// download signal. Falls back silently to FALLBACK_TAG on any error.
export function loadLatestRelease() {
  try {
    const cached = JSON.parse(localStorage.getItem(CACHE_KEY) || "null");
    if (cached && cached.data && Date.now() - cached.at < CACHE_TTL) {
      setDownloads(cached.data);
      return;
    }
  } catch (e) {
    /* ignore cache errors */
  }

  fetch(`https://api.github.com/repos/${OWNER}/${REPO}/releases/latest`, {
    headers: { Accept: "application/vnd.github+json" },
  })
    .then((r) => (r.ok ? r.json() : Promise.reject(r.status)))
    .then((rel) => {
      const tag = rel.tag_name || FALLBACK_TAG;
      const data = {
        tag,
        macos: pickAsset(rel.assets, "macos", tag, "macos-universal.zip"),
        windows: pickAsset(rel.assets, "windows", tag, "windows-amd64.zip"),
        linux: pickAsset(rel.assets, "linux", tag, "linux-amd64.tar.gz"),
      };
      setDownloads(data);
      try {
        localStorage.setItem(CACHE_KEY, JSON.stringify({ at: Date.now(), data }));
      } catch (e) {
        /* ignore */
      }
    })
    .catch(() => {
      /* keep fallback */
    });
}
