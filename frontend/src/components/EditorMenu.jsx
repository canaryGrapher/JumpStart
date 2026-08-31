import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { OpenInEditor } from "../api";
import useEditors from "../hooks/useEditors";

// EditorMenu is the "open in editor" header button. With one editor
// installed it behaves like any other icon button — one click opens it.
// With several, the same click opens a small popover so the user can pick
// which one, each row showing that editor's own real application icon
// rather than a generic code glyph. Renders nothing when no supported
// editor is installed, the same way the GitHub button hides itself when
// there's no remote to link to.
//
// The popover is portaled to <body> and positioned from the button's own
// bounding rect rather than living inside .editor-menu via `position:
// absolute`. It used to do the latter, but the button sits in
// .main-header, which needs its own low z-index stacking context so the
// sticky .tabs-bar right below it can paint over *that* (see .tabs-bar's
// z-index) as it scrolls underneath. That trapped the popover's z-index
// inside .main-header's context too, so .tabs-bar's higher z-index — and
// then the process cards further down, plain content with no stacking
// context of their own to lose to — both painted over it regardless of
// the popover's own z-index. Escaping to <body> sidesteps the whole
// ancestor-stacking-context chase for good.
export default function EditorMenu({ dir, onError, colored = false }) {
  const editors = useEditors();
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [pos, setPos] = useState(null);
  const btnRef = useRef(null);
  const panelRef = useRef(null);

  const reposition = () => {
    const btn = btnRef.current;
    if (!btn) return;
    const r = btn.getBoundingClientRect();
    setPos({ top: r.bottom + 6, right: window.innerWidth - r.right });
  };

  useEffect(() => {
    if (!open) return;
    reposition();
    // The button can move under the popover (window resize, or the page
    // scrolling behind a sticky header) while it's open; keep it glued to
    // the button rather than left floating over whatever used to be there.
    window.addEventListener("resize", reposition);
    window.addEventListener("scroll", reposition, true);
    return () => {
      window.removeEventListener("resize", reposition);
      window.removeEventListener("scroll", reposition, true);
    };
  }, [open]);

  useEffect(() => {
    const onDoc = (e) => {
      if (btnRef.current?.contains(e.target)) return;
      if (panelRef.current?.contains(e.target)) return;
      setOpen(false);
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
    <div className="editor-menu">
      <button
        ref={btnRef}
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

      {open && pos &&
        createPortal(
          <div
            className="editor-menu-panel"
            ref={panelRef}
            style={{ position: "fixed", top: pos.top, right: pos.right }}
          >
            {editors.map((ed) => (
              <button key={ed.id} className="editor-menu-item" onClick={(e) => { e.stopPropagation(); openWith(ed.id); }}>
                {ed.icon ? <img src={ed.icon} alt="" /> : <span className="editor-menu-dot" />}
                <span>{ed.name}</span>
              </button>
            ))}
          </div>,
          document.body
        )}
    </div>
  );
}
