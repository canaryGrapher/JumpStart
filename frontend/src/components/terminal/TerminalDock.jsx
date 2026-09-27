import { useEffect } from "react";
import useTerminalDock from "../../hooks/useTerminalDock";
import TerminalWindow from "./TerminalWindow";
import TerminalTaskbar from "./TerminalTaskbar";

// Mounted once at the app root. Process cards and script/test runners open
// terminals here instead of rendering logs inline, so the bottom of the
// screen — not the card — is where output actually lives: a taskbar of
// tabs, and a floating window for whichever of them aren't minimized.
export default function TerminalDock() {
  const { windows } = useTerminalDock();

  // Lets CSS reserve space above the taskbar (see .main in _base.scss)
  // only while there's actually a dock to avoid covering content.
  useEffect(() => {
    document.documentElement.classList.toggle("has-terminal-dock", windows.length > 0);
  }, [windows.length]);

  if (windows.length === 0) return null;

  const visible = windows.filter((w) => !w.minimized).sort((a, b) => a.order - b.order);

  return (
    <>
      {visible.length > 0 && (
        <div className="terminal-windows">
          {visible.map((w) => (
            <TerminalWindow key={w.id} win={w} />
          ))}
        </div>
      )}
      <TerminalTaskbar windows={windows} />
    </>
  );
}
