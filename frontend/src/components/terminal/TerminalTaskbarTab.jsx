import { closeTerminal, toggleTerminal } from "../../terminalDock";

// One tab on the bottom terminal taskbar. Clicking it minimizes or
// restores its window (same as clicking a taskbar button in a desktop OS);
// its own close button dismisses the run for good.
export default function TerminalTaskbarTab({ win }) {
  const dotClass =
    win.source === "process" ? "" : win.running ? "on" : win.exitCode ? "bad" : "";

  return (
    <div
      className={`terminal-tab ${win.minimized ? "" : "active"}`}
      onClick={() => toggleTerminal(win.id)}
      title={win.minimized ? "Restore" : "Minimize"}
    >
      {dotClass !== "" || win.source !== "process" ? (
        <span className={`run-dot ${dotClass}`} />
      ) : null}
      <span className="terminal-tab-title">{win.title}</span>
      <button
        className="terminal-tab-close"
        title="Close"
        onClick={(e) => {
          e.stopPropagation();
          closeTerminal(win.id);
        }}
      >
        ✕
      </button>
    </div>
  );
}
