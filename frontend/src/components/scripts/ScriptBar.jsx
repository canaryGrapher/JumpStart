// ScriptBar renders one button per custom script on a process card, e.g.
// "Migrate" running `go run . --migrate`. Clicking runs the script and
// opens its log; the chevron toggles the run history panel.
export default function ScriptBar({ scripts, busyScriptId, runsOpen, onRun, onToggleRuns }) {
  if (!scripts || scripts.length === 0) return null;

  return (
    <div className="script-bar" onClick={(e) => e.stopPropagation()}>
      <div className="script-bar-label">Scripts</div>
      <div className="script-chips">
        {scripts.map((s) => (
          <button
            key={s.id}
            className="script-chip"
            title={s.command}
            disabled={busyScriptId === s.id}
            onClick={() => onRun(s)}
          >
            {busyScriptId === s.id ? "Running…" : s.name}
          </button>
        ))}
        <button
          className="script-chip ghost"
          title="Show recent script runs and their logs"
          onClick={onToggleRuns}
        >
          {runsOpen ? "Hide runs" : "Runs"}
        </button>
      </div>
    </div>
  );
}
