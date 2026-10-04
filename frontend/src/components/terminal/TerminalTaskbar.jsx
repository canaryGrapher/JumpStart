import Icon, { ICONS } from "../Icon";
import TerminalTaskbarTab from "./TerminalTaskbarTab";

// Fixed bar along the bottom of the window, one tab per open or minimized
// terminal — a Windows-style taskbar for process logs and script/test runs.
export default function TerminalTaskbar({ windows }) {
  const ordered = [...windows].sort((a, b) => a.order - b.order);

  return (
    <div className="terminal-taskbar">
      <Icon d={ICONS.terminal} />
      <div className="terminal-taskbar-tabs">
        {ordered.map((w) => (
          <TerminalTaskbarTab key={w.id} win={w} />
        ))}
      </div>
    </div>
  );
}
