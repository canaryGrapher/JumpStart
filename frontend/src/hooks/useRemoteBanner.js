import { useEffect, useState } from "react";
import { GetRemoteBanner } from "../api";

const POLL_INTERVAL = 30 * 60 * 1000; // 30 minutes
const DISMISS_KEY = "dismissedBannerIds";
export const BANNER_URL_KEY = "bannerUrl"; // Preferences override

const dismissedIds = () => {
  try {
    return JSON.parse(localStorage.getItem(DISMISS_KEY)) || [];
  } catch {
    return [];
  }
};

// Fetches the remote overlay banner config and exposes it unless the user
// already dismissed that banner id.
export default function useRemoteBanner() {
  const [banner, setBanner] = useState(null);

  useEffect(() => {
    const fetchBanner = () =>
      GetRemoteBanner(localStorage.getItem(BANNER_URL_KEY) || "")
        .then((b) => {
          if (!b || !b.enabled || !b.id) return setBanner(null);
          if (dismissedIds().includes(b.id)) return;
          setBanner(b);
        })
        .catch(() => {}); // offline: stay quiet
    fetchBanner();
    const t = setInterval(fetchBanner, POLL_INTERVAL);
    return () => clearInterval(t);
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
