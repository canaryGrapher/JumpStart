import { Fragment, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AISearch, EventsOn, Search } from "../api";
import { aiConfigured, getAISettings } from "../ai";
import useAIProgress, { formatElapsed } from "../hooks/useAIProgress";
import { formatDue } from "../dueDates";
import { openInApp } from "../navigate";

const HIDE_DONE_KEY = "jumpstart.palette.hideDone";
const GROUPS = [
  { kind: "project", label: "Projects" },
  { kind: "task", label: "Tasks" },
  { kind: "file", label: "Files" },
];
const FIELD_LABELS = {
  description: "Description",
  labels: "Labels",
  assignee: "Assignee",
  "acceptance criteria": "Acceptance criteria",
  subtasks: "Subtasks",
  links: "Links",
  milestone: "Milestone",
  "file names": "Files",
  fields: "Fields",
  "file text": "Text in file",
  "file name": "File name",
};

const readBool = (k) => {
  try {
    return localStorage.getItem(k) === "1";
  } catch {
    return false;
  }
};

// Highlights the searched words (at word starts, like the backend's prefix
// matching) without using innerHTML.
function Highlight({ text, terms }) {
  if (!text || !terms?.length) return text || null;
  const escaped = terms.map((t) => t.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"));
  const re = new RegExp(`(^|[^\\p{L}\\p{N}])(${escaped.join("|")})`, "giu");
  const parts = [];
  let last = 0;
  for (const m of text.matchAll(re)) {
    const start = m.index + m[1].length;
    parts.push(text.slice(last, start));
    parts.push(<mark key={start}>{m[2]}</mark>);
    last = start + m[2].length;
  }
  parts.push(text.slice(last));
  return <>{parts}</>;
}

function ResultRow({ r, terms, active, onPick, onHover, id }) {
  const meta = [];
  if (r.kind !== "project") meta.push(r.projectName);
  if (r.kind === "file") meta.push(`in “${r.taskTitle}”`);
  if (r.kind === "task" && r.statusLabel) meta.push(r.statusLabel);
  return (
    <li
      id={id}
      role="option"
      aria-selected={active}
      className={`palette-row ${active ? "active" : ""} ${r.done ? "done" : ""}`}
      onMouseMove={onHover}
      onMouseDown={(e) => {
        e.preventDefault();
        onPick(r);
      }}
    >
      <span className={`palette-kind kind-${r.kind === "task" ? r.type || "task" : r.kind}`} aria-hidden>
        {r.kind === "project" ? "P" : r.kind === "file" ? "F" : (r.type || "task")[0].toUpperCase()}
      </span>
      <span className="palette-main">
        <span className="palette-title">
          <Highlight text={r.title} terms={terms} />
        </span>
        {r.snippet && (
          <span className="palette-snippet">
            {r.field && <em>{FIELD_LABELS[r.field] || r.field}: </em>}
            <Highlight text={r.snippet} terms={terms} />
          </span>
        )}
        <span className="palette-meta">{meta.filter(Boolean).join(" · ")}</span>
      </span>
      {r.dueDate && (
        <span className={`palette-due ${r.pastDue ? "late" : ""}`} title={r.pastDue ? "Past due" : "Due"}>
          {formatDue(r.dueDate)}
        </span>
      )}
    </li>
  );
}

// Spotlight-style search over every project: ⌘K (or the sidebar search,
// or the optional system shortcut) opens it. Type to search; Tab switches
// to Ask, where the local AI turns a question into a filter.
export default function CommandPalette({ open, initialQuery = "", currentProjectId, onClose }) {
  const [mode, setMode] = useState("search"); // search | ask
  const [q, setQ] = useState(initialQuery);
  const [res, setRes] = useState(null);
  const [error, setError] = useState("");
  const [sel, setSel] = useState(0);
  const [hideDone, setHideDone] = useState(() => readBool(HIDE_DONE_KEY));
  const [thisProject, setThisProject] = useState(false);
  const [ask, setAsk] = useState(null); // AISearchResult
  const [asking, setAsking] = useState(false);
  const progress = useAIProgress();
  const inputRef = useRef(null);
  const listRef = useRef(null);
  const seq = useRef(0);

  useEffect(() => {
    if (open) {
      setQ(initialQuery);
      setSel(0);
      setError("");
      setTimeout(() => inputRef.current?.focus(), 0);
    }
  }, [open, initialQuery]);

  const scope = thisProject && currentProjectId ? currentProjectId : "";

  const run = useCallback(async () => {
    if (mode !== "search") return;
    const mine = ++seq.current;
    if (!q.trim()) {
      setRes(null);
      return;
    }
    try {
      const r = await Search({ query: q, projectId: scope, hideDone, limit: 60 });
      if (mine === seq.current) {
        setRes(r);
        setError("");
        setSel(0);
      }
    } catch (e) {
      if (mine === seq.current) setError(String(e));
    }
  }, [q, scope, hideDone, mode]);

  useEffect(() => {
    if (!open) return undefined;
    const t = setTimeout(run, 120);
    return () => clearTimeout(t);
  }, [open, run]);

  // OCR finishing in the background can add results; refresh quietly.
  useEffect(() => {
    if (!open) return undefined;
    return EventsOn("search:updated", () => run());
  }, [open, run]);

  const results = mode === "ask" ? ask?.results || [] : res?.results || [];
  const ordered = useMemo(() => {
    if (mode === "ask") return results;
    return GROUPS.flatMap((g) => results.filter((r) => r.kind === g.kind));
  }, [results, mode]);

  useEffect(() => {
    listRef.current?.querySelector(".palette-row.active")?.scrollIntoView({ block: "nearest" });
  }, [sel]);

  if (!open) return null;

  const pick = (r) => {
    onClose();
    openInApp(r.projectId, r.kind === "project" ? "" : r.taskId);
  };

  const doAsk = async () => {
    if (!q.trim() || asking) return;
    const { host, model } = getAISettings();
    setAsking(true);
    setError("");
    const requestId = progress.begin();
    try {
      const out = await AISearch(host, model, q, scope, { requestId, think: "" });
      if (!out.stopped) {
        setAsk(out);
        setSel(0);
      }
    } catch (e) {
      setError(String(e));
      setAsk(null);
    } finally {
      progress.end();
      setAsking(false);
    }
  };

  const onKey = (e) => {
    if (e.key === "Escape") {
      e.preventDefault();
      if (asking) progress.stop();
      else onClose();
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      setSel((s) => Math.min(ordered.length - 1, s + 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setSel((s) => Math.max(0, s - 1));
    } else if (e.key === "Tab") {
      e.preventDefault();
      setMode((m) => (m === "search" ? "ask" : "search"));
      setSel(0);
    } else if (e.key === "Enter") {
      e.preventDefault();
      if (mode === "ask" && (!ask || e.metaKey)) doAsk();
      else if (ordered[sel]) pick(ordered[sel]);
    }
  };

  const terms = mode === "search" ? res?.terms || [] : [];
  const q1 = q.trim();
  let idx = -1;

  return (
    <div className="palette-backdrop" onMouseDown={onClose}>
      <div className="palette" role="dialog" aria-label="Search JumpStart" onMouseDown={(e) => e.stopPropagation()}>
        <div className="palette-input-row">
          <div className="palette-modes" role="tablist" aria-label="Search mode">
            <button type="button" role="tab" aria-selected={mode === "search"} className={mode === "search" ? "on" : ""} onClick={() => setMode("search")}>
              Search
            </button>
            <button type="button" role="tab" aria-selected={mode === "ask"} className={mode === "ask" ? "on" : ""} onClick={() => setMode("ask")}>
              Ask AI
            </button>
          </div>
          <input
            ref={inputRef}
            className="palette-input"
            aria-label={mode === "ask" ? "Ask about your tasks" : "Search everything"}
            aria-controls="palette-results"
            aria-activedescendant={ordered[sel] ? `palette-opt-${sel}` : undefined}
            placeholder={mode === "ask" ? "Ask: what is overdue for Sam this week?" : "Search tasks, files, dates (Oct 9, next week)…"}
            value={q}
            onChange={(e) => {
              setQ(e.target.value);
              if (mode === "ask") setAsk(null);
            }}
            onKeyDown={onKey}
          />
        </div>

        <div className="palette-options">
          <label>
            <input
              type="checkbox"
              checked={hideDone}
              onChange={(e) => {
                setHideDone(e.target.checked);
                try {
                  localStorage.setItem(HIDE_DONE_KEY, e.target.checked ? "1" : "0");
                } catch {
                  // not persisted
                }
              }}
            />
            Hide done
          </label>
          {currentProjectId && (
            <label>
              <input type="checkbox" checked={thisProject} onChange={(e) => setThisProject(e.target.checked)} />
              This project only
            </label>
          )}
          {mode === "search" && res?.notes?.length > 0 && <span className="palette-notes">Searching: {res.notes.join(" · ")}</span>}
          {mode === "search" && res?.pending > 0 && <span className="palette-notes">Reading text from {res.pending} file{res.pending === 1 ? "" : "s"}…</span>}
        </div>

        {mode === "ask" && (
          <div className="palette-ask">
            {!aiConfigured() ? (
              <p className="palette-empty">Pick a local model in Settings &gt; AI to ask questions.</p>
            ) : asking ? (
              <div className="palette-thinking" role="status">
                Thinking… {formatElapsed(progress.elapsed || 0)}
                <button type="button" className="btn small" onClick={progress.stop}>Stop</button>
              </div>
            ) : ask ? (
              <div className="palette-plan">
                <strong>{ask.plan.explain || "Filter"}</strong>
                <code>{JSON.stringify(ask.plan.query)}</code>
                {ask.plan.projects?.length > 0 && <span>Projects: {ask.plan.projects.join(", ")}</span>}
                {ask.plan.warnings?.map((w) => (
                  <span key={w} className="palette-warn">{w}</span>
                ))}
                <span className="palette-notes">{ask.total} matching task{ask.total === 1 ? "" : "s"} · ⌘↵ to ask again</span>
              </div>
            ) : (
              q1 && <p className="palette-empty">Press Enter to ask. The model only picks a filter; the results are your real tasks.</p>
            )}
          </div>
        )}

        {error && <p className="palette-error" role="alert">{error}</p>}
        {mode === "search" && res?.errors?.length > 0 && <p className="palette-error">Not understood: {res.errors.join(", ")}</p>}

        <ul className="palette-results" id="palette-results" role="listbox" ref={listRef} aria-label="Results">
          {mode === "search" && q1 && res && ordered.length === 0 && <li className="palette-empty">No matches for “{q1}”.</li>}
          {mode === "search"
            ? GROUPS.map((g) => {
                const rows = ordered.filter((r) => r.kind === g.kind);
                if (!rows.length) return null;
                return (
                  <Fragment key={g.kind}>
                    <li className="palette-group" role="presentation">
                      {g.label}
                      {g.kind === "task" && res.total > res.results.length && <span> · showing {res.results.length} of {res.total}</span>}
                    </li>
                    {rows.map((r) => {
                      idx += 1;
                      const i = idx;
                      return (
                        <ResultRow key={`${r.kind}-${r.projectId}-${r.taskId}-${r.attachmentId}`} id={`palette-opt-${i}`} r={r} terms={terms} active={i === sel} onPick={pick} onHover={() => setSel(i)} />
                      );
                    })}
                  </Fragment>
                );
              })
            : ordered.map((r, i) => (
                <ResultRow key={`${r.projectId}-${r.taskId}`} id={`palette-opt-${i}`} r={r} terms={[]} active={i === sel} onPick={pick} onHover={() => setSel(i)} />
              ))}
        </ul>

        <footer className="palette-foot">
          <span><kbd>↑</kbd><kbd>↓</kbd> move</span>
          <span><kbd>↵</kbd> open</span>
          <span><kbd>Tab</kbd> {mode === "search" ? "ask AI" : "search"}</span>
          <span><kbd>Esc</kbd> close</span>
          <span className="palette-help">status:todo · @name · #label · due:next week · has:file · is:overdue</span>
        </footer>
      </div>
    </div>
  );
}
