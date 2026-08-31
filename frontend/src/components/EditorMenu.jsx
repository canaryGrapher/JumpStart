import { useEffect, useRef, useState } from "react";
import { OpenInEditor } from "../api";
import useEditors from "../hooks/useEditors";

// EditorMenu is the "open in editor" header button. With one editor
// installed it behaves like any other icon button — one click opens it.
// With several, the same click opens a small popover so the user can pick
// which one, each row showing that editor's own real application icon
// rather than a generic code glyph. Renders nothing when no supported
// editor is installed, the same way the GitHub button hides itself when
// there's no remote to link to.
export default function EditorMenu({ dir, onError, colored = false }) {
  const editors = useEditors();
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const wrapRef = useRef(null);

  useEffect(() => {
    const onDoc = (e) => {
      if (wrapRef.current && !wrapRef.current.contains(e.target)) setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, []);

  if (!editors.length) return null;

  const openWith = (id) => {
    if (busy) return;
    setOpen(false);
    setBusy(true);
    OpenInEditor(id, dir)
      .catch((err) => onError && onError(String(err)))
      .finally(() => setBusy(false));
  };

  const primary = editors[0];
  const cls = `icon-btn outline${colored ? " vscode-blue" : ""}`;

  return (
    <div className="editor-menu" ref={wrapRef}>
      <button
        className={cls}
        title={editors.length > 1 ? "Open in editor" : `Open in ${primary.name}`}
        disabled={busy}
        onClick={(e) => {
          e.stopPropagation();
          if (editors.length > 1) setOpen((v) => !v);
          else openWith(primary.id);
        }}
      >
        {primary.icon ? <img src={primary.icon} alt="" /> : <span className="editor-menu-dot" />}
      </button>

      {open && (
        <div className="editor-menu-panel">
          {editors.map((ed) => (
            <button key={ed.id} className="editor-menu-item" onClick={(e) => { e.stopPropagation(); openWith(ed.id); }}>
              {ed.icon ? <img src={ed.icon} alt="" /> : <span className="editor-menu-dot" />}
              <span>{ed.name}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
