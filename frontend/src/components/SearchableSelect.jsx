import { useEffect, useMemo, useRef, useState } from "react";

// Walks up from a node to find the nearest ancestor that actually
// scrolls (overflow-y auto/scroll with real overflow), so the panel's
// "is there room below?" check is against the space visible inside a
// scrolling modal body, not the whole browser window — a trigger near
// the bottom of a small scrolled panel has plenty of room in the
// window but none in the box the user can actually see.
function scrollableAncestor(node) {
  let el = node?.parentElement;
  while (el && el !== document.body) {
    const style = getComputedStyle(el);
    if (/(auto|scroll)/.test(style.overflowY) && el.scrollHeight > el.clientHeight) {
      return el;
    }
    el = el.parentElement;
  }
  return null;
}

// Searchable dropdown: type to filter, click or Enter to pick.
export default function SearchableSelect({
  value,
  options,
  onChange,
  placeholder = "Select…",
  searchPlaceholder = "Search…",
  disabled = false,
  allowCreate = false,
  onCreate,
  createLabel = (text) => `+ Create "${text}"`,
}) {
  const [open, setOpen] = useState(false);
  const [openUp, setOpenUp] = useState(false);
  const [query, setQuery] = useState("");
  const [active, setActive] = useState(0);
  const wrapRef = useRef(null);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return options;
    return options.filter((o) => o.toLowerCase().includes(q));
  }, [options, query]);

  // Close when clicking outside.
  useEffect(() => {
    const onDoc = (e) => {
      if (wrapRef.current && !wrapRef.current.contains(e.target)) setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, []);

  useEffect(() => {
    if (open) {
      setQuery("");
      setActive(0);
    }
  }, [open]);

  const pick = (o) => {
    onChange(o);
    setOpen(false);
  };

  const trimmedQuery = query.trim();
  const canCreate =
    allowCreate && !!onCreate && trimmedQuery.length > 0 && !options.some((o) => o.toLowerCase() === trimmedQuery.toLowerCase());

  const create = () => {
    if (!canCreate) return;
    onCreate(trimmedQuery);
    setOpen(false);
  };

  const onKeyDown = (e) => {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setActive((a) => Math.min(a + 1, filtered.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActive((a) => Math.max(a - 1, 0));
    } else if (e.key === "Enter") {
      e.preventDefault();
      if (filtered[active]) pick(filtered[active]);
      else if (canCreate) create();
    } else if (e.key === "Escape") {
      setOpen(false);
    }
  };

  return (
    <div className="ss-wrap" ref={wrapRef}>
      <button
        type="button"
        className="ss-trigger"
        disabled={disabled}
        onClick={() => {
          if (!open) {
            // Decide, once, whether there's room to drop the panel
            // below the trigger — inside whichever box actually
            // scrolls (a modal body, say), not just the window — and
            // open it upward instead when there isn't.
            const rect = wrapRef.current?.getBoundingClientRect();
            if (rect) {
              const boundary = scrollableAncestor(wrapRef.current)?.getBoundingClientRect();
              const bottomEdge = boundary ? boundary.bottom : window.innerHeight;
              const spaceBelow = bottomEdge - rect.bottom;
              const spaceAbove = rect.top - (boundary ? boundary.top : 0);
              setOpenUp(spaceBelow < 260 && spaceAbove > spaceBelow);
            }
          }
          setOpen((v) => !v);
        }}
      >
        <span className={value ? "" : "ss-placeholder"}>{value || placeholder}</span>
        <span className="ss-caret">▾</span>
      </button>

      {open && (() => {
        const searchBox = (
          <input
            className="ss-search"
            autoFocus
            value={query}
            placeholder={searchPlaceholder}
            onChange={(e) => {
              setQuery(e.target.value);
              setActive(0);
            }}
            onKeyDown={onKeyDown}
          />
        );
        const listBox = (
          <div className="ss-list">
            {filtered.length === 0 && !canCreate ? (
              <div className="ss-empty">No matches</div>
            ) : (
              filtered.map((o, i) => (
                <button
                  type="button"
                  key={o}
                  className={`ss-option ${o === value ? "selected" : ""} ${
                    i === active ? "active" : ""
                  }`}
                  onMouseEnter={() => setActive(i)}
                  onClick={() => pick(o)}
                >
                  {o}
                </button>
              ))
            )}
            {canCreate && (
              <button type="button" className="ss-option ss-create" onClick={create}>
                {createLabel(trimmedQuery)}
              </button>
            )}
          </div>
        );
        // Opened upward, the panel sits above the trigger — put the
        // search box at its bottom edge, right next to the trigger the
        // user just clicked, instead of leaving it stranded at the far
        // (top) end of the flipped panel.
        return (
          <div className={`ss-panel ${openUp ? "ss-panel-up" : ""}`}>
            {openUp ? (
              <>
                {listBox}
                {searchBox}
              </>
            ) : (
              <>
                {searchBox}
                {listBox}
              </>
            )}
          </div>
        );
      })()}
    </div>
  );
}
