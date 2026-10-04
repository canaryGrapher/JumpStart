import { useCallback, useEffect, useState } from "react";
import { RunScript, ListProcessScriptRuns, EventsOn } from "../api";
import { openTerminal } from "../terminalDock";

// useScriptRuns owns the run history for one process's scripts: starting a
// run, opening its terminal window in the dock, and clearing the
// "running" flag when the backend emits the run's exit event.
export default function useScriptRuns(projectId, procId, onError) {
  const [runs, setRuns] = useState([]);
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

  const markFinished = useCallback((runId, exitCode) => {
    setRuns((prev) =>
      prev.map((r) => (r.runId === runId ? { ...r, running: false, exitCode } : r))
    );
  }, []);

  // Keeps the run history chips current even while a run's terminal window
  // is closed or minimized in the dock.
  useEffect(() => {
    const running = runs.filter((r) => r.running);
    if (running.length === 0) return;
    const offs = running.map((r) =>
      EventsOn(`exit:${r.runId}`, (code) => markFinished(r.runId, code))
    );
    return () => offs.forEach((off) => off());
  }, [runs, markFinished]);

  const openRun = useCallback((run) => {
    openTerminal({
      id: `script:${run.runId}`,
      title: run.name,
      procId: run.runId,
      source: "script",
      command: run.command,
      running: run.running,
      exitCode: run.exitCode,
    });
  }, []);

  const run = useCallback(
    async (script) => {
      setBusyScriptId(script.id);
      try {
        const runId = await RunScript(projectId, procId, script.id);
        const newRun = {
          runId,
          scriptId: script.id,
          procId,
          name: script.name,
          command: script.command,
          startedAt: Date.now(),
          running: true,
        };
        setRuns((prev) => [newRun, ...prev]);
        openRun(newRun);
      } catch (e) {
        onError?.(String(e));
      } finally {
        setBusyScriptId(null);
      }
    },
    [projectId, procId, onError, openRun]
  );

  return { runs, busyScriptId, run, openRun };
}
