const time = (ms) =>
  new Date(ms).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });

// ScriptRunsPanel lists the recent runs of a process's scripts. Their
// output no longer lives inline here — clicking a run opens (or refocuses)
// its terminal window at the bottom of the screen, the same place a
// script's log pops up the moment it starts running.
export default function ScriptRunsPanel({ runs, onOpen }) {
  return (
    <div className="script-runs" onClick={(e) => e.stopPropagation()}>
      {runs.length === 0 ? (
        <div className="script-runs-empty">
          No script runs yet. Click a script above to run it.
        </div>
      ) : (
        <div className="script-run-list">
          {runs.map((r) => (
            <button
              key={r.runId}
              className="script-run-item"
              title="Open this run's terminal"
              onClick={() => onOpen(r)}
            >
              <span className={`run-dot ${r.running ? "on" : r.exitCode ? "bad" : ""}`} />
              {r.name}
              <span className="run-time">{time(r.startedAt)}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
