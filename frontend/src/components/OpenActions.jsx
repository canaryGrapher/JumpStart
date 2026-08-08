// OpenActions is the pair of "take me out of JumpStart" buttons for one
// directory: show it in the OS file manager, or open a terminal sitting in
// it. It is rendered for a project root and for each subprocess working
// directory, so the labels stay generic and the directory is only named in
// the tooltip.
//
// It renders two bare buttons rather than a wrapper element so it can drop
// into any existing button row.
import { useState } from "react";
import { OpenInFileManager, OpenInTerminal } from "../api";
import { fileManagerName } from "../platform";

export default function OpenActions({ dir, onError, size = "" }) {
  const [busy, setBusy] = useState("");

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
