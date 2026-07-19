import ScriptRunLog from "./ScriptRunLog";

const time = (ms) =>
  new Date(ms).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });

// ScriptRunsPanel lists the recent runs of a process's scripts and shows
// the log of whichever one is selected. History is in-memory on the
// backend (last 10 per script), so it resets when the app restarts.
export default function ScriptRunsPanel({
  runs,
  activeRunId,
  onSelect,
  onStop,
  onFinished,
}) {
  const active = runs.find((r) => r.runId === activeRunId) || runs[0];

  return (
    <div className="script-runs" onClick={(e) => e.stopPropagation()}>
      {runs.length === 0 ? (
        <div className="script-runs-empty">
          No script runs yet. Click a script above to run it.
        </div>
      ) : (
        <>
          <div className="script-run-list">
            {runs.map((r) => (
              <button
                key={r.runId}
                className={`script-run-item ${r.runId === active?.runId ? "active" : ""}`}
                onClick={() => onSelect(r.runId)}
              >
                <span className={`run-dot ${r.running ? "on" : r.exitCode ? "bad" : ""}`} />
                {r.name}
                <span className="run-time">{time(r.startedAt)}</span>
              </button>
            ))}
          </div>
          {active && (
            <ScriptRunLog run={active} onStop={onStop} onFinished={onFinished} />
          )}
        </>
      )}
    </div>
  );
}
