import { useCallback, useEffect, useRef, useState } from "react";
import {
  buildContext,
  getContextStatus,
  clearContext,
  onIndexProgress,
} from "../../../codeContext";

// Owns the code index for one project: its saved status, rebuilding it,
// and live progress while a scan runs.
export default function useCodeContext(projectId, onError) {
  const [status, setStatus] = useState(null);
  const [building, setBuilding] = useState(false);
  const [progress, setProgress] = useState(null);

  // See useChatSessions: keeping the callback in a ref stops an inline
  // arrow from invalidating the load effect on every render.
  const errRef = useRef(onError);
  errRef.current = onError;
  const fail = (e) => errRef.current && errRef.current(String(e));

  const refresh = useCallback(async () => {
    if (!projectId) return;
    try {
      setStatus(await getContextStatus(projectId));
    } catch (e) {
      fail(e);
    }
  }, [projectId]);

  useEffect(() => {
    setStatus(null);
    refresh();
  }, [projectId, refresh]);

  useEffect(() => {
    if (!building || !projectId) return;
    return onIndexProgress(projectId, setProgress);
  }, [building, projectId]);

  const build = useCallback(async () => {
    if (!projectId || building) return;
    setBuilding(true);
    setProgress({ done: 0, total: 0, path: "" });
    try {
      setStatus(await buildContext(projectId));
    } catch (e) {
      fail(e);
    } finally {
      setBuilding(false);
      setProgress(null);
    }
  }, [projectId, building]);

  const clear = useCallback(async () => {
    try {
      await clearContext(projectId);
      await refresh();
    } catch (e) {
      fail(e);
    }
  }, [projectId, refresh]);

  return { status, building, progress, build, clear, refresh };
}
