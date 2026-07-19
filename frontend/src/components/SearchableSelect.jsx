import { useEffect, useMemo, useRef, useState } from "react";

// Searchable dropdown: type to filter, click or Enter to pick.
export default function SearchableSelect({
  value,
  options,
  onChange,
  placeholder = "Select…",
  searchPlaceholder = "Search…",
  disabled = false,
}) {
  const [open, setOpen] = useState(false);
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
        onClick={() => setOpen((v) => !v)}
      >
        <span className={value ? "" : "ss-placeholder"}>{value || placeholder}</span>
        <span className="ss-caret">▾</span>
      </button>

      {open && (
        <div className="ss-panel">
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
          <div className="ss-list">
            {filtered.length === 0 ? (
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
          </div>
        </div>
      )}
    </div>
  );
}
