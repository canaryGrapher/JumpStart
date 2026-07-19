import { useCallback, useEffect, useState } from "react";
import { RunScript, StopScriptRun, ListProcessScriptRuns } from "../api";

// useScriptRuns owns the run history for one process's scripts: starting a
// run, tracking which one the log panel is showing, and clearing the
// "running" flag when the backend emits the run's exit event.
export default function useScriptRuns(projectId, procId, onError) {
  const [runs, setRuns] = useState([]);
  const [activeRunId, setActiveRunId] = useState(null);
  const [busyScriptId, setBusyScriptId] = useState(null);

  // Recover history after a remount (the backend keeps the last runs).
  useEffect(() => {
    let cancelled = false;
    ListProcessScriptRuns(procId)
      .then((list) => {
        if (cancelled || !list) return;
        setRuns(
          [...list]
            .sort((a, b) => b.startedAt - a.startedAt)
            .map((r) => ({ ...r, running: false }))
        );
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [procId]);

  const run = useCallback(
    async (script) => {
      setBusyScriptId(script.id);
      try {
        const runId = await RunScript(projectId, procId, script.id);
        setRuns((prev) => [
          {
            runId,
            scriptId: script.id,
            procId,
            name: script.name,
            command: script.command,
            startedAt: Date.now(),
            running: true,
          },
          ...prev,
        ]);
        setActiveRunId(runId);
      } catch (e) {
        onError?.(String(e));
      } finally {
        setBusyScriptId(null);
      }
    },
    [projectId, procId, onError]
  );

  const stop = useCallback(
    (runId) => StopScriptRun(runId).catch((e) => onError?.(String(e))),
    [onError]
  );

  // Called by the run log when it sees the run's exit event.
  const markFinished = useCallback((runId, exitCode) => {
    setRuns((prev) =>
      prev.map((r) => (r.runId === runId ? { ...r, running: false, exitCode } : r))
    );
  }, []);

  return { runs, activeRunId, setActiveRunId, busyScriptId, run, stop, markFinished };
}
