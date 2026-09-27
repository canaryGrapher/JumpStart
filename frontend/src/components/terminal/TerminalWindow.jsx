import { useState } from "react";
import Icon, { ICONS } from "../Icon";
import LogPanel from "../LogPanel";
import { StopScriptRun } from "../../api";
import { closeTerminal, minimizeTerminal } from "../../terminalDock";

// One floating window in the terminal dock, showing the live log for a
// process, script run, or test run. The dock owns minimize/restore/close,
// so a window can vanish from here and reappear from the taskbar tab
// without losing its place or its buffered output (LogPanel refetches it).
export default function TerminalWindow({ win }) {
  const [stopping, setStopping] = useState(false);

  const outcome =
    win.source === "process"
      ? null
      : win.running
      ? { text: "Running", kind: "on" }
      : win.exitCode === 0
      ? { text: "Succeeded", kind: "on" }
      : win.exitCode === undefined
      ? { text: "Finished", kind: "off" }
      : { text: `Exit ${win.exitCode}`, kind: "bad" };

  const stop = async () => {
    setStopping(true);
    try {
      await StopScriptRun(win.procId);
    } finally {
      setStopping(false);
    }
  };

  return (
    <div className="terminal-window">
      <div
        className="terminal-window-head"
        onClick={() => minimizeTerminal(win.id)}
        title="Minimize"
      >
        <div className="terminal-window-title">
          <Icon d={ICONS.terminal} />
          <span>{win.title}</span>
          {win.command && <code>{win.command}</code>}
        </div>
        <div className="terminal-window-actions" onClick={(e) => e.stopPropagation()}>
          {outcome && (
            <span className={`status-pill ${outcome.kind}`}>
              <span className="dot" />
              {outcome.text}
            </span>
          )}
          {win.source === "script" && win.running && (
            <button className="btn small danger" disabled={stopping} onClick={stop}>
              Stop
            </button>
          )}
          <button
            className="terminal-window-btn"
            title="Minimize"
            onClick={() => minimizeTerminal(win.id)}
          >
            &#8211;
          </button>
          <button
            className="terminal-window-btn"
            title="Close"
            onClick={() => closeTerminal(win.id)}
          >
            ✕
          </button>
        </div>
      </div>
      <div className="terminal-window-body">
        <LogPanel procId={win.procId} source={win.source} />
      </div>
    </div>
  );
}
