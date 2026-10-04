import { useCallback, useEffect, useRef, useState } from "react";
import {
  EventsOn,
  GitHubGetSync,
  GitHubSyncNow,
  GitHubWatch,
  GitHubSetFocused,
} from "../api";

// Live sync state for one project's board.
//
// GitHub has no push channel a desktop app can subscribe to, so the Go
// side polls: ~90s while this view is on screen, ~10min when it is not,
// and a debounced pass after local edits. Each pass emits an event that
// lands here, which is what keeps the board current without the user
// pressing anything.
export default function useGitHubSync(projectId, onTasks, onError) {
  const [sync, setSync] = useState(null);
  const [state, setState] = useState("idle"); // idle | syncing | error
  const [result, setResult] = useState(null);
  const [error, setError] = useState("");
  const [progress, setProgress] = useState(null); // { done, total } while syncing
  const tasksRef = useRef(onTasks);
  tasksRef.current = onTasks;

  // Load the project's link config, and start or stop the poll loop with
  // it. Unmounting stops polling so a background project does not keep
  // spending rate limit.
  useEffect(() => {
    let live = true;
    GitHubGetSync(projectId)
      .then((cfg) => {
        if (!live) return;
        setSync(cfg);
        return GitHubWatch(cfg?.enabled ? projectId : "");
      })
      .catch(() => {});
    return () => {
      live = false;
      GitHubWatch("").catch(() => {});
    };
  }, [projectId]);

  useEffect(() => {
    const onStart = (id) => {
      if (id !== projectId) return;
      setState("syncing");
      setProgress(null);
    };

    const onProgress = (payload) => {
      if (!payload || payload.projectId !== projectId || !payload.total) return;
      setProgress({ done: payload.done || 0, total: payload.total });
    };

    const onDone = (payload) => {
      if (!payload || payload.projectId !== projectId) return;
      setState("idle");
      setError("");
      setProgress(null);
      setResult(payload.result);
      if (payload.tasks && tasksRef.current) tasksRef.current(payload.tasks);
    };

    const onFail = (payload) => {
      if (!payload || payload.projectId !== projectId) return;
      setState("error");
      setError(payload.error || "Sync failed");
      setProgress(null);
    };

    const offStart = EventsOn("github:sync:start", onStart);
    const offProgress = EventsOn("github:sync:progress", onProgress);
    const offDone = EventsOn("github:sync:done", onDone);
    const offFail = EventsOn("github:sync:error", onFail);
    return () => {
      offStart && offStart();
      offProgress && offProgress();
      offDone && offDone();
      offFail && offFail();
    };
  }, [projectId]);

  // Slow the poll down while the window is hidden. A board nobody is
  // looking at does not need a focused-rate refresh.
  useEffect(() => {
    const onVisibility = () => GitHubSetFocused(!document.hidden).catch(() => {});
    document.addEventListener("visibilitychange", onVisibility);
    onVisibility();
    return () => {
      document.removeEventListener("visibilitychange", onVisibility);
      GitHubSetFocused(false).catch(() => {});
    };
  }, []);

  const syncNow = useCallback(async () => {
    setState("syncing");
    setProgress(null);
    try {
      const r = await GitHubSyncNow(projectId);
      setResult(r);
      setState("idle");
      setProgress(null);
    } catch (e) {
      setState("error");
      setError(String(e));
      setProgress(null);
      onError && onError(String(e));
    }
  }, [projectId, onError]);

  // Called whenever the link config changes from inside the connect
  // modal — a fresh link, a relink after the old board was deleted, or
  // an unlink. Applying the new config alone would leave a stale error
  // from the previous (broken) board sitting in the bar until the next
  // background poll tick, which can be a while, so this also clears
  // that state immediately and, for a newly-enabled link, kicks off a
  // sync right away so the bar reflects reality without the user having
  // to notice and press "Sync now" themselves.
  const applySync = useCallback(
    (cfg) => {
      setSync(cfg);
      setError("");
      GitHubWatch(cfg?.enabled ? projectId : "").catch(() => {});
      if (cfg?.enabled) {
        syncNow();
      } else {
        setState("idle");
        setResult(null);
        setProgress(null);
      }
    },
    [projectId, syncNow]
  );

  return { sync, setSync: applySync, state, result, error, progress, syncNow };
}
