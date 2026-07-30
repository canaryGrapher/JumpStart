// Build classification for the running binary.
//
// Release builds stamp main.Version via ldflags (-X main.Version=<tag>), so
// the version string alone tells us which kind of build is running:
//
//   "dev" / ""        -> development (local `wails dev` or an unstamped build)
//   "1.5.0-beta.2"    -> beta (any pre-release tail starting with "beta")
//   "1.5.0-rc.1"      -> prerelease (other pre-release tails)
//   "1.5.0"           -> production
//
// Mirrors internal/update/semver.go so both sides agree on what "beta" means.

import { useEffect, useState } from "react";
import { GetAppVersion } from "./api";

// preTail returns the pre-release part of a semver tag ("beta.2"), or "".
function preTail(version) {
  const v = String(version || "")
    .trim()
    .replace(/^v/, "")
    .split("+")[0];
  const i = v.indexOf("-");
  return i >= 0 ? v.slice(i + 1) : "";
}

export const DEV = "dev";
export const BETA = "beta";
export const PRERELEASE = "prerelease";
export const PRODUCTION = "production";

// classify maps a version string to one of the build kinds above.
export function classify(version) {
  const v = String(version || "").trim();
  if (v === "" || v.toLowerCase() === "dev") return DEV;
  const pre = preTail(v).toLowerCase();
  if (pre === "") return PRODUCTION;
  return pre.startsWith("beta") ? BETA : PRERELEASE;
}

// LABELS are the persistent on-screen indicators. Production is intentionally
// absent: shipped builds show no badge.
export const LABELS = {
  [DEV]: "Development build",
  [BETA]: "Beta build",
  [PRERELEASE]: "Pre-release build",
};

// useBuildInfo resolves the running build once at mount. Both fields stay
// null until the binding answers, so a production build never flashes a
// "Development build" badge on the first paint.
export default function useBuildInfo() {
  const [version, setVersion] = useState(null);

  useEffect(() => {
    GetAppVersion()
      .then((v) => setVersion(v || DEV))
      .catch(() => setVersion(DEV));
  }, []);

  return { version, kind: version === null ? null : classify(version) };
}
