import { useEffect, useState } from "react";
import { CheckForUpdate } from "../api";

const CHECK_INTERVAL = 6 * 60 * 60 * 1000; // 6 hours
const SNOOZE_KEY = "updateSnoozedVersion";

// Polls GitHub Releases (via the Go backend) and exposes the update info
// when a newer version exists. Dismissing snoozes that specific version.
export default function useUpdateCheck() {
  const [info, setInfo] = useState(null);

  useEffect(() => {
    const check = () =>
      CheckForUpdate()
        .then((res) => {
          if (!res || !res.available) return;
          if (localStorage.getItem(SNOOZE_KEY) === res.latestVersion) return;
          setInfo(res);
        })
        .catch(() => {}); // offline or rate-limited: stay quiet
    check();
    const t = setInterval(check, CHECK_INTERVAL);
    return () => clearInterval(t);
  }, []);

  const dismiss = () => {
    if (info) localStorage.setItem(SNOOZE_KEY, info.latestVersion);
    setInfo(null);
  };

  return { update: info, dismiss };
}
