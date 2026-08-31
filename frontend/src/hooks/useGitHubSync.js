import { useCallback, useEffect, useRef, useState } from "react";
import {
  EventsOn,
  EventsOff,
  GitHubGetSync,
  GitHubSyncNow,
  GitHubWatch,
  GitHubSetFocused,
} from "../api";

// Live sync state for one project's board.
//
// GitHub has no push channel a desktop app can subscribe to, so the Go
// side polls: fast while this view is on screen, slow when it is not,
// and immediately after any local edit. Each pass emits an event that
// lands here, which is what keeps the board current without the user
// pressing anything.
export default function useGitHubSync(projectId, onTasks, onError) {
  const [sync, setSync] = useState(null);
  const [state, setState] = useState("idle"); // idle | syncing | error
  const [result, setResult] = useState(null);
  const [error, setError] = useState("");
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
    const onStart = (id) => id === projectId && setState("syncing");

    const onDone = (payload) => {
      if (!payload || payload.projectId !== projectId) return;
      setState("idle");
      setError("");
      setResult(payload.result);
      if (payload.tasks && tasksRef.current) tasksRef.current(payload.tasks);
    };

    const onFail = (payload) => {
      if (!payload || payload.projectId !== projectId) return;
      setState("error");
      setError(payload.error || "Sync failed");
    };

    EventsOn("github:sync:start", onStart);
    EventsOn("github:sync:done", onDone);
    EventsOn("github:sync:error", onFail);
    return () => {
      EventsOff("github:sync:start");
      EventsOff("github:sync:done");
      EventsOff("github:sync:error");
    };
  }, [projectId]);

  // Slow the poll down while the window is hidden. A board nobody is
  // looking at does not need a ten-second refresh.
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
    try {
      const r = await GitHubSyncNow(projectId);
      setResult(r);
      setState("idle");
    } catch (e) {
      setState("error");
      setError(String(e));
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
      }
    },
    [projectId, syncNow]
  );

  return { sync, setSync: applySync, state, result, error, syncNow };
}
