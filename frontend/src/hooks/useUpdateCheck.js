import { useCallback, useEffect, useState } from "react";
import { CheckForUpdate } from "../api";
import { isBetaEnabled, onChannelChange } from "../updateChannel";

const CHECK_INTERVAL = 6 * 60 * 60 * 1000; // 6 hours
const SNOOZE_KEY = "updateSnoozedVersion";

// Polls GitHub Releases (via the Go backend) and exposes the update info
// when a newer version exists on the active channel. Dismissing snoozes that
// specific version. Toggling the beta channel re-checks immediately, so a
// pending pre-release shows up without waiting for the next poll.
export default function useUpdateCheck() {
  const [info, setInfo] = useState(null);
  const [beta, setBeta] = useState(isBetaEnabled);

  const run = useCallback(
    (useBeta) =>
      CheckForUpdate(useBeta)
        .then((res) => {
          if (!res || !res.available) {
            setInfo(null);
            return;
          }
          if (localStorage.getItem(SNOOZE_KEY) === res.latestVersion) return;
          setInfo(res);
        })
        .catch(() => {}), // offline or rate-limited: stay quiet
    []
  );

  useEffect(() => {
    run(beta);
    const t = setInterval(() => run(beta), CHECK_INTERVAL);
    return () => clearInterval(t);
  }, [beta, run]);

  useEffect(() => onChannelChange(setBeta), []);

  const dismiss = () => {
    if (info) localStorage.setItem(SNOOZE_KEY, info.latestVersion);
    setInfo(null);
  };

  return { update: info, dismiss };
}
