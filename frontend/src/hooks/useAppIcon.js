import { useEffect, useState } from "react";

// Every OpenActions instance (one per process card, plus the project
// header) wants the same file-manager/terminal icon, and it never changes
// while the app is running — so each fetcher is called once and its result
// (or in-flight promise) is shared across every hook call, rather than
// firing one IPC round-trip per card.
const cache = new Map();

function shared(key, fetcher) {
  if (!cache.has(key)) {
    cache.set(
      key,
      fetcher().catch(() => null)
    );
  }
  return cache.get(key);
}

// useAppIcon resolves a single, dir-independent AppIcon (the real icon of
// the file manager or terminal JumpStart would launch) via fetcher, sharing
// one in-flight/resolved request across every component that asks for the
// same key. Returns null until resolved, or if extraction failed / the
// platform binding isn't available — callers should fall back to a
// generic glyph in that case rather than show a broken image.
export default function useAppIcon(key, fetcher) {
  const [icon, setIcon] = useState(null);

  useEffect(() => {
    let active = true;
    shared(key, fetcher).then((result) => {
      if (active && result && result.icon) setIcon(result);
    });
    return () => {
      active = false;
    };
  }, [key]);

  return icon;
}
