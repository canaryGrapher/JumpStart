import { useEffect } from "react";
import { EventsOn } from "../../api";
import LogPanel from "../LogPanel";

const time = (ms) =>
  new Date(ms).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });

// ScriptRunLog shows the output of one script run plus its live state.
// A run is just a process in the manager keyed by runId, so the existing
// LogPanel and "exit:<id>" event work unchanged.
export default function ScriptRunLog({ run, onStop, onFinished }) {
  useEffect(() => {
    if (!run.running) return;
    return EventsOn(`exit:${run.runId}`, (code) => onFinished(run.runId, code));
  }, [run.runId, run.running, onFinished]);

  const outcome = run.running
    ? { text: "Running", kind: "on" }
    : run.exitCode === 0
    ? { text: "Succeeded", kind: "on" }
    : run.exitCode === undefined
    ? { text: "Finished", kind: "off" }
    : { text: `Exit ${run.exitCode}`, kind: "bad" };

  return (
    <div className="script-run-log">
      <div className="script-run-head">
        <div className="script-run-meta">
          <strong>{run.name}</strong>
          <code>{run.command}</code>
        </div>
        <div className="script-run-actions">
          <span className={`status-pill ${outcome.kind}`}>
            <span className="dot" />
            {outcome.text}
          </span>
          <span className="pid">{time(run.startedAt)}</span>
          {run.running && (
            <button className="btn small danger" onClick={() => onStop(run.runId)}>
              Stop
            </button>
          )}
        </div>
      </div>
      <LogPanel procId={run.runId} />
    </div>
  );
}
