import Icon, { ICONS } from "../Icon";

// ScriptBar is the Scripts section of a process card: one run button per
// custom script (e.g. "Migrate" running `go run . --migrate`) and a History
// toggle for recent runs. Running a script opens its log in the history.
export default function ScriptBar({
  scripts,
  busyScriptId,
  runCount = 0,
  runsOpen,
  onRun,
  onToggleRuns,
}) {
  if (!scripts || scripts.length === 0) return null;

  return (
    <section className="script-bar" onClick={(e) => e.stopPropagation()}>
      <header className="script-bar-head">
        <span className="script-bar-label">Scripts</span>
        <button
          className={`chip-toggle small ${runsOpen ? "active" : ""}`}
          aria-pressed={runsOpen}
          title="Show recent script runs and their logs"
          onClick={onToggleRuns}
        >
          <Icon d={ICONS.clock} />
          History
          {runCount > 0 && <span className="count">{runCount}</span>}
        </button>
      </header>
      <div className="script-chips">
        {scripts.map((s) => {
          const busy = busyScriptId === s.id;
          return (
            <button
              key={s.id}
              className={`script-chip ${busy ? "busy" : ""}`}
              title={s.command}
              disabled={busy}
              onClick={() => onRun(s)}
            >
              <span className="script-chip-icon">
                {busy ? <span className="spinner" /> : <Icon d={ICONS.play} filled />}
              </span>
              <span className="script-chip-name">{s.name}</span>
            </button>
          );
        })}
      </div>
    </section>
  );
}
