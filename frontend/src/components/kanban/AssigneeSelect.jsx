import { useEffect, useMemo, useRef, useState } from "react";

// Split / join the Task.Assignee string the same way Go does, so a
// round-trip through sync does not reshuffle chips for no reason.
export function parseAssignees(value) {
  if (!value || !String(value).trim()) return [];
  const seen = new Set();
  const out = [];
  for (const part of String(value).split(",")) {
    const login = part.trim();
    if (!login) continue;
    const key = login.toLowerCase();
    if (seen.has(key)) continue;
    seen.add(key);
    out.push(login);
  }
  return out;
}

export function formatAssignees(logins) {
  const cleaned = parseAssignees((logins || []).join(","));
  return [...cleaned].sort((a, b) => a.toLowerCase().localeCompare(b.toLowerCase())).join(", ");
}

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

// GitHub-style multi-select for task assignees: chips for the selected
// people, searchable checklist of team members underneath.
export default function AssigneeSelect({
  value = "",
  users = [],
  onChange,
  placeholder = "Select assignees…",
  disabled = false,
  allowCreate = true,
}) {
  const [open, setOpen] = useState(false);
  const [openUp, setOpenUp] = useState(false);
  const [query, setQuery] = useState("");
  const [active, setActive] = useState(0);
  const wrapRef = useRef(null);

  const selected = useMemo(() => parseAssignees(value), [value]);
  const selectedSet = useMemo(
    () => new Set(selected.map((s) => s.toLowerCase())),
    [selected]
  );

  const byLogin = useMemo(() => {
    const m = new Map();
    for (const u of users) {
      if (u?.login) m.set(u.login.toLowerCase(), u);
    }
    return m;
  }, [users]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    const list = users.length
      ? users
      : selected.map((login) => ({ login, name: "", avatarUrl: "" }));
    // Always surface selected people who are not in the team list (custom
    // names typed while offline) so they can be unchecked.
    const extras = selected
      .filter((login) => !byLogin.has(login.toLowerCase()))
      .map((login) => ({ login, name: "", avatarUrl: "" }));
    const merged = [...list];
    for (const e of extras) {
      if (!merged.some((u) => u.login.toLowerCase() === e.login.toLowerCase())) {
        merged.push(e);
      }
    }
    if (!q) return merged;
    return merged.filter(
      (u) =>
        u.login.toLowerCase().includes(q) ||
        (u.name || "").toLowerCase().includes(q)
    );
  }, [users, selected, byLogin, query]);

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

  const emit = (next) => onChange(formatAssignees(next));

  const toggle = (login) => {
    const key = login.toLowerCase();
    if (selectedSet.has(key)) {
      emit(selected.filter((s) => s.toLowerCase() !== key));
    } else {
      emit([...selected, login]);
    }
  };

  const remove = (login, e) => {
    e.stopPropagation();
    toggle(login);
  };

  const trimmedQuery = query.trim().replace(/^@/, "");
  const canCreate =
    allowCreate &&
    trimmedQuery.length > 0 &&
    !selectedSet.has(trimmedQuery.toLowerCase()) &&
    !users.some((u) => u.login.toLowerCase() === trimmedQuery.toLowerCase());

  const create = () => {
    if (!canCreate) return;
    emit([...selected, trimmedQuery]);
    setQuery("");
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
      if (filtered[active]) toggle(filtered[active].login);
      else if (canCreate) create();
    } else if (e.key === "Escape") {
      setOpen(false);
    } else if (e.key === "Backspace" && !query && selected.length) {
      emit(selected.slice(0, -1));
    }
  };

  const labelFor = (login) => {
    const u = byLogin.get(login.toLowerCase());
    return u?.name ? `${u.login} (${u.name})` : login;
  };

  return (
    <div className="ss-wrap assignee-select" ref={wrapRef}>
      <button
        type="button"
        className="ss-trigger assignee-trigger"
        disabled={disabled}
        onClick={() => {
          if (!open) {
            const rect = wrapRef.current?.getBoundingClientRect();
            if (rect) {
              const boundary = scrollableAncestor(wrapRef.current)?.getBoundingClientRect();
              const bottomEdge = boundary ? boundary.bottom : window.innerHeight;
              const spaceBelow = bottomEdge - rect.bottom;
              const spaceAbove = rect.top - (boundary ? boundary.top : 0);
              setOpenUp(spaceBelow < 280 && spaceAbove > spaceBelow);
            }
          }
          setOpen((v) => !v);
        }}
      >
        <div className="assignee-chips">
          {selected.length === 0 ? (
            <span className="ss-placeholder">{placeholder}</span>
          ) : (
            selected.map((login) => {
              const u = byLogin.get(login.toLowerCase());
              return (
                <span className="assignee-chip" key={login} title={labelFor(login)}>
                  {u?.avatarUrl ? (
                    <img className="assignee-avatar" src={u.avatarUrl} alt="" />
                  ) : (
                    <span className="assignee-avatar fallback">
                      {login.slice(0, 1).toUpperCase()}
                    </span>
                  )}
                  <span className="assignee-login">{login}</span>
                  <span
                    className="assignee-remove"
                    role="button"
                    tabIndex={-1}
                    onClick={(e) => remove(login, e)}
                  >
                    ×
                  </span>
                </span>
              );
            })
          )}
        </div>
        <span className="ss-caret">▾</span>
      </button>

      {open && (
        <div className={`ss-panel ${openUp ? "ss-panel-up" : ""}`}>
          {!openUp && (
            <input
              className="ss-search"
              autoFocus
              value={query}
              placeholder="Search people…"
              onChange={(e) => {
                setQuery(e.target.value);
                setActive(0);
              }}
              onKeyDown={onKeyDown}
            />
          )}
          <div className="ss-list">
            {filtered.length === 0 && !canCreate ? (
              <div className="ss-empty">
                {users.length === 0
                  ? "Type a name and press Enter"
                  : "No matches"}
              </div>
            ) : (
              filtered.map((u, i) => {
                const on = selectedSet.has(u.login.toLowerCase());
                return (
                  <button
                    type="button"
                    key={u.login}
                    className={`ss-option assignee-option ${on ? "selected" : ""} ${
                      i === active ? "active" : ""
                    }`}
                    onMouseEnter={() => setActive(i)}
                    onClick={() => toggle(u.login)}
                  >
                    <span className={`assignee-check ${on ? "on" : ""}`}>
                      {on ? "✓" : ""}
                    </span>
                    {u.avatarUrl ? (
                      <img className="assignee-avatar" src={u.avatarUrl} alt="" />
                    ) : (
                      <span className="assignee-avatar fallback">
                        {u.login.slice(0, 1).toUpperCase()}
                      </span>
                    )}
                    <span className="assignee-meta">
                      <span className="assignee-login">{u.login}</span>
                      {u.name ? <span className="assignee-name">{u.name}</span> : null}
                    </span>
                  </button>
                );
              })
            )}
            {canCreate && (
              <button type="button" className="ss-option ss-create" onClick={create}>
                + Add “{trimmedQuery}”
              </button>
            )}
          </div>
          {openUp && (
            <input
              className="ss-search"
              autoFocus
              value={query}
              placeholder="Search people…"
              onChange={(e) => {
                setQuery(e.target.value);
                setActive(0);
              }}
              onKeyDown={onKeyDown}
            />
          )}
        </div>
      )}
    </div>
  );
}
