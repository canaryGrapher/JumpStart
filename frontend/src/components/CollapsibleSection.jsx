import { useState } from "react";

// Compact disclosure for chrome that steals board vertical space.
// Remembers open/closed per `id` in localStorage when provided.
export default function CollapsibleSection({
  id,
  title,
  summary,
  summaryClassName = "",
  defaultOpen = true,
  children,
  className = "",
  lead = null,
}) {
  const storageKey = id ? `collapse:${id}` : null;
  const [open, setOpen] = useState(() => {
    if (!storageKey) return defaultOpen;
    const saved = localStorage.getItem(storageKey);
    if (saved === "0") return false;
    if (saved === "1") return true;
    return defaultOpen;
  });

  const toggle = () => {
    setOpen((prev) => {
      const next = !prev;
      if (storageKey) localStorage.setItem(storageKey, next ? "1" : "0");
      return next;
    });
  };

  return (
    <section
      className={`collapse-section ${open ? "is-open" : "is-collapsed"} ${className}`.trim()}
    >
      <button
        type="button"
        className="collapse-section-head"
        onClick={toggle}
        aria-expanded={open}
      >
        <span className={`collapse-chevron ${open ? "open" : ""}`} aria-hidden>
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M9 6l6 6-6 6" />
          </svg>
        </span>
        {lead}
        <span className="collapse-section-title">{title}</span>
        {summary != null && summary !== "" && (
          <span className={`collapse-section-summary ${summaryClassName}`.trim()}>
            {summary}
          </span>
        )}
      </button>
      {open && <div className="collapse-section-body">{children}</div>}
    </section>
  );
}
