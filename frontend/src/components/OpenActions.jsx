// OpenActions is the pair of "take me out of JumpStart" buttons for one
// directory: show it in the OS file manager, or open a terminal sitting in
// it. It is rendered for a project root and for each subprocess working
// directory, so the labels stay generic and the directory is only named in
// the tooltip.
//
// It renders bare buttons rather than a wrapper element so it can drop
// into any existing button row. `iconOnly` swaps the labeled capsule
// buttons for the compact round icon buttons used in tight spaces like a
// process card's action row, where a full-text button pair eats too much
// width (see ProcessCard); in that mode the file-manager and terminal
// buttons show the *real* application icon (Finder, Windows Explorer,
// Nautilus, iTerm, ...) fetched from the backend via useAppIcon, falling
// back to a generic glyph only if extraction failed. `showCode` adds an
// "open in editor" button (see EditorMenu) for the project header, where
// that action is wanted but is noise on every process card. `colored`
// tints the fallback glyphs (blue/mono) for the header's colored-icon row;
// process cards leave it off and keep the neutral outline look — it has no
// effect on the real application icons, which already have their own color.
import { useState } from "react";
import { OpenInFileManager, OpenInTerminal, FileManagerIcon, TerminalIcon } from "../api";
import { fileManagerName } from "../platform";
import Icon, { ICONS } from "./Icon";
import EditorMenu from "./EditorMenu";
import useAppIcon from "../hooks/useAppIcon";

export default function OpenActions({
  dir,
  onError,
  size = "",
  iconOnly = false,
  showCode = false,
  colored = false,
}) {
  const [busy, setBusy] = useState("");
  const fmIcon = useAppIcon("fileManager", FileManagerIcon);
  const termIcon = useAppIcon("terminal", TerminalIcon);

  if (!dir) return null;

  const run = (e, kind, fn) => {
    // Process cards start/stop on card click; these buttons must not.
    e.stopPropagation();
    if (busy) return;
    setBusy(kind);
    Promise.resolve(fn(dir))
      .catch((err) => onError && onError(String(err)))
      .finally(() => setBusy(""));
  };

  if (iconOnly) {
    const cls = (color) => `icon-btn outline${colored ? ` ${color}` : ""}`;
    return (
      <>
        <button
          className={cls("blue")}
          title={`Show ${dir} in ${fmIcon?.name || fileManagerName}`}
          disabled={busy === "files"}
          onClick={(e) => run(e, "files", OpenInFileManager)}
        >
          {fmIcon?.icon ? <img src={fmIcon.icon} alt="" /> : <Icon d={ICONS.folder} />}
        </button>
        <button
          className={cls("mono")}
          title={termIcon?.name ? `Open ${termIcon.name} in ${dir}` : `Open a terminal in ${dir}`}
          disabled={busy === "terminal"}
          onClick={(e) => run(e, "terminal", OpenInTerminal)}
        >
          {termIcon?.icon ? <img src={termIcon.icon} alt="" /> : <Icon d={ICONS.terminal} />}
        </button>
        {showCode && <EditorMenu dir={dir} onError={onError} colored={colored} />}
      </>
    );
  }

  const cls = `btn ${size}`.trim();

  return (
    <>
      <button
        className={cls}
        title={`Show ${dir} in ${fileManagerName}`}
        disabled={busy === "files"}
        onClick={(e) => run(e, "files", OpenInFileManager)}
      >
        {fileManagerName}
      </button>
      <button
        className={cls}
        title={`Open a terminal in ${dir}`}
        disabled={busy === "terminal"}
        onClick={(e) => run(e, "terminal", OpenInTerminal)}
      >
        Terminal
      </button>
    </>
  );
}
