import { useEffect, useState } from "react";
import { GetRemoteBanner } from "../api";

const DISMISS_KEY = "dismissedBannerIds";

const dismissedIds = () => {
  try {
    return JSON.parse(localStorage.getItem(DISMISS_KEY)) || [];
  } catch {
    return [];
  }
};

// Fetches the remote announcements once and exposes the first banner the
// user has not already dismissed. Dismissals persist in localStorage, so a
// closed banner never reappears (across restarts, polls, or reopens).
export default function useRemoteBanner() {
  const [banner, setBanner] = useState(null);

  useEffect(() => {
    GetRemoteBanner()
      .then((list) => {
        const seen = dismissedIds();
        const next = (list || []).find(
          (b) => b && b.enabled && b.id && !seen.includes(b.id)
        );
        setBanner(next || null);
      })
      .catch(() => {}); // offline: stay quiet
  }, []);

  const dismiss = () => {
    if (banner) {
      const ids = dismissedIds();
      ids.push(banner.id);
      localStorage.setItem(DISMISS_KEY, JSON.stringify(ids.slice(-50)));
    }
    setBanner(null);
  };

  return { banner, dismiss };
}
