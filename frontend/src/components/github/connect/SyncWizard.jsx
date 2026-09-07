import { useEffect, useMemo, useState } from "react";
import {
  EventsOn,
  GitHubListOwners,
  GitHubListProjects,
  GitHubBoardPresets,
  GitHubCreateProject,
  GitHubImportRepoItems,
  GitHubLinkProject,
} from "../../../api";
import SearchableSelect from "../../SearchableSelect";
import { ghColumnColor } from "./ghColors";

const ownerLabel = (o) => `${o.login} ${o.type === "organization" ? "(organisation)" : "(personal account)"}`;
const boardLabel = (b) => `#${b.number} ${b.title}${b.closed ? " (closed)" : ""}`;

// A single, linear path for the one decision this modal ultimately
// exists to make: which GitHub Projects board this repository syncs
// with. Earlier this was scattered across a "Sync Now" shortcut, a
// collapsible "sync options" accordion, and a separate multi-step
// wizard nested inside it — three different affordances stacked on top
// of each other for what is, underneath, six small questions asked in
// order: account, board, repo link, import, layout, review. Asking them
// one at a time (instead of all at once, or hidden behind a toggle)
// is the whole point of this component.
export default function SyncWizard({ projectId, owner, repo, fullName, initialTitle, onLinked, onCancel, onError }) {
  const [stepIndex, setStepIndex] = useState(0);

  // --- Step 1: account/organisation ---
  const [owners, setOwners] = useState(null);
  const [ownerLogin, setOwnerLogin] = useState("");
  const [ownerErr, setOwnerErr] = useState(null);

  // --- Step 2: board (create new, or use an existing one) ---
  const [boardMode, setBoardMode] = useState("new"); // "new" | "existing"
  const [title, setTitle] = useState(initialTitle || repo || "New board");
  const [boards, setBoards] = useState(null);
  const [boardId, setBoardId] = useState("");

  // --- Step 3: connect to this repository ---
  const [connectNow, setConnectNow] = useState(true);

  // --- Step 4: import existing items ---
  const [importItems, setImportItems] = useState(true);
  const [importScope, setImportScope] = useState("both"); // "issues" | "prs" | "both"
  const [asIssue, setAsIssue] = useState(false);

  // --- Step 5: starting layout (new boards only) ---
  const [presets, setPresets] = useState(null);
  const [presetKey, setPresetKey] = useState("basic");

  const [busy, setBusy] = useState(false);
  // Live counter while import/sync walk items one GraphQL call at a time —
  // without it the review CTA just says "Working…" for the whole pass.
  const [progress, setProgress] = useState(null); // { kind: "import"|"sync", done, total }
  const [localErr, setLocalErr] = useState(null);
  const [done, setDone] = useState(null); // { project } once created/selected but left unlinked

  useEffect(() => {
    if (!busy) {
      setProgress(null);
      return;
    }
    const onImport = (p) => {
      if (!p || !p.total) return;
      setProgress({ kind: "import", done: p.done || 0, total: p.total });
    };
    const onSync = (p) => {
      if (!p || p.projectId !== projectId || !p.total) return;
      setProgress({ kind: "sync", done: p.done || 0, total: p.total });
    };
    // EventsOn returns an unsubscribe — use that so we don't EventsOff the
    // shared github:sync:progress channel the board hook also listens on.
    const offImport = EventsOn("github:import:progress", onImport);
    const offSync = EventsOn("github:sync:progress", onSync);
    return () => {
      offImport && offImport();
      offSync && offSync();
    };
  }, [busy, projectId]);

  useEffect(() => {
    GitHubListOwners()
      .then((list) => {
        setOwners(list || []);
        const match = (list || []).find((o) => o.login.toLowerCase() === (owner || "").toLowerCase());
        if (match) setOwnerLogin(match.login);
        else if ((list || []).length) setOwnerLogin(list[0].login);
        else setOwnerErr(`Could not find "${owner}" among your accessible GitHub accounts.`);
      })
      .catch((e) => setOwnerErr(String(e)));
    GitHubBoardPresets()
      .then((p) => {
        setPresets(p || []);
        if (p && p.length && !p.find((x) => x.key === presetKey)) setPresetKey(p[0].key);
      })
      .catch((e) => setLocalErr(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!ownerLogin) return;
    setBoards(null);
    setBoardId("");
    GitHubListProjects(ownerLogin)
      .then((list) => {
        setBoards(list || []);
        setBoardId((list || [])[0]?.id || "");
      })
      .catch((e) => setLocalErr(String(e)));
  }, [ownerLogin]);

  const owner_ = (owners || []).find((o) => o.login === ownerLogin);
  const selectedOwnerLabel = owner_ ? ownerLabel(owner_) : "";
  const selectedBoard = (boards || []).find((b) => b.id === boardId);

  const steps = useMemo(() => {
    const base = ["org", "board", "connect", "import"];
    if (boardMode === "new") base.push("layout");
    base.push("review");
    return base;
  }, [boardMode]);

  const step = steps[stepIndex];
  const stepMeta = {
    org: { label: "Account", title: "Where should this board live?", desc: "Boards belong to a GitHub account or organization." },
    board: { label: "Board", title: "Name the board", desc: "Start a new board, or connect to one that already exists." },
    connect: { label: "Repository", title: "Connect a repository?", desc: `Link this board to ${fullName || `${owner}/${repo}`} so items sync automatically.` },
    import: { label: "Import", title: "Bring in existing work?", desc: `Optionally seed the board from ${fullName || `${owner}/${repo}`}.` },
    layout: { label: "Layout", title: "Choose a starting layout", desc: "Sets the board's initial status columns — you can rename or add to these later." },
    review: { label: "Review", title: "Ready to go", desc: "Double-check the details, then create the board." },
  }[step];

  const canAdvance = {
    org: !!ownerLogin && !ownerErr,
    board: boardMode === "new" ? title.trim().length > 0 : !!boardId,
    connect: true,
    import: true,
    layout: !!presetKey,
    review: true,
  }[step];

  const goNext = () => setStepIndex((i) => Math.min(i + 1, steps.length - 1));
  const goBack = () => {
    if (stepIndex === 0) {
      onCancel && onCancel();
      return;
    }
    setStepIndex((i) => Math.max(i - 1, 0));
  };

  const busyLabel = (() => {
    if (!busy) return null;
    if (progress?.kind === "import") {
      return progress.done < 1 ? "Importing…" : `Importing ${progress.done}/${progress.total}…`;
    }
    if (progress?.kind === "sync") {
      return progress.done < 1 ? "Syncing…" : `Syncing ${progress.done}/${progress.total} tasks`;
    }
    return "Working…";
  })();

  const submit = async () => {
    setBusy(true);
    setProgress(null);
    setLocalErr(null);
    try {
      let project;
      if (boardMode === "new") {
        project = await GitHubCreateProject(owner_?.id, title.trim(), presetKey);
      } else {
        project = selectedBoard;
      }
      if (importItems) {
        try {
          await GitHubImportRepoItems(
            project.id,
            owner,
            repo,
            importScope === "issues" || importScope === "both",
            importScope === "prs" || importScope === "both"
          );
        } catch (e) {
          // The board itself is fine; a failed import shouldn't strand
          // the user back at square one.
          onError && onError(`Board ready, but importing items failed: ${e}`);
        }
      }
      if (connectNow) {
        const cfg = await GitHubLinkProject(projectId, project.id, fullName, asIssue);
        onLinked && onLinked(cfg);
      } else {
        setDone({ project });
      }
    } catch (e) {
      setLocalErr(String(e));
    } finally {
      setBusy(false);
      setProgress(null);
    }
  };

  const connectLater = async () => {
    if (!done) return;
    setBusy(true);
    try {
      const cfg = await GitHubLinkProject(projectId, done.project.id, fullName, asIssue);
      onLinked && onLinked(cfg);
    } catch (e) {
      setLocalErr(String(e));
    } finally {
      setBusy(false);
    }
  };

  if (done) {
    return (
      <div className="gh-wizard">
        <div className="gh-wiz-done">
          <div className="gh-wiz-done-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round">
              <path d="M20 6L9 17l-5-5" />
            </svg>
          </div>
          <h4>"{done.project.title}" is ready on GitHub</h4>
          <p className="gh-muted">
            It isn't connected to this repository yet, so items won't sync until you link it.
            {!onCancel && " You can close this window with the X above whenever you're done."}
          </p>
          {localErr && <div className="gh-warn-inline">{localErr}</div>}
          <div className="gh-wiz-actions">
            {onCancel ? (
              <button className="btn small ghost" disabled={busy} onClick={onCancel}>
                Close
              </button>
            ) : (
              <span />
            )}
            <button className="btn primary gh-cta" disabled={busy} onClick={connectLater}>
              {busy ? "Connecting…" : `Connect to ${fullName || `${owner}/${repo}`}`}
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="gh-wizard">
      <div className="gh-wiz-progress">
        <div className="gh-wiz-progress-track">
          <div className="gh-wiz-progress-fill" style={{ width: `${(stepIndex / (steps.length - 1)) * 100}%` }} />
        </div>
        <div className="gh-wiz-progress-label">
          Step {stepIndex + 1} of {steps.length} · {stepMeta.label}
        </div>
      </div>

      <div className="gh-step-head">
        <h3>{stepMeta.title}</h3>
        <p className="gh-muted">{stepMeta.desc}</p>
      </div>

      <div className="gh-wiz-body">
        {step === "org" && (
          <div className="field">
            <label>Account / organisation</label>
            {owners === null ? (
              <span className="gh-muted">Loading accounts…</span>
            ) : (
              <SearchableSelect
                value={selectedOwnerLabel}
                options={(owners || []).map(ownerLabel)}
                onChange={(label) => {
                  const o = (owners || []).find((x) => ownerLabel(x) === label);
                  if (o) setOwnerLogin(o.login);
                }}
                placeholder="Choose an account…"
                searchPlaceholder="Search accounts…"
              />
            )}
            {ownerErr && <div className="gh-warn-inline">{ownerErr}</div>}
          </div>
        )}

        {step === "board" && (
          <div className="gh-wiz-board">
            <div className="gh-seg">
              <button
                type="button"
                className={`gh-seg-opt ${boardMode === "new" ? "on" : ""}`}
                onClick={() => setBoardMode("new")}
              >
                Create new
              </button>
              <button
                type="button"
                className={`gh-seg-opt ${boardMode === "existing" ? "on" : ""}`}
                onClick={() => setBoardMode("existing")}
                disabled={boards !== null && boards.length === 0}
              >
                Use existing
              </button>
            </div>

            {boardMode === "new" ? (
              <div className="field">
                <label>Board name</label>
                <input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Board name" />
              </div>
            ) : boards === null ? (
              <span className="gh-muted">Loading boards for {ownerLogin}…</span>
            ) : boards.length === 0 ? (
              <span className="gh-muted">No boards found for {ownerLogin} yet — create a new one instead.</span>
            ) : (
              <div className="field">
                <label>Board</label>
                <SearchableSelect
                  value={selectedBoard ? boardLabel(selectedBoard) : ""}
                  options={boards.map(boardLabel)}
                  onChange={(label) => {
                    const b = boards.find((x) => boardLabel(x) === label);
                    if (b) setBoardId(b.id);
                  }}
                  placeholder="Choose a board…"
                  searchPlaceholder="Search boards…"
                />
              </div>
            )}
          </div>
        )}

        {step === "connect" && (
          <div className="gh-wiz-choice-grid">
            <button type="button" className={`gh-choice-card ${connectNow ? "selected" : ""}`} onClick={() => setConnectNow(true)}>
              <span className="gh-choice-title">Connect now</span>
              <span className="gh-choice-desc">Link this board to {fullName || `${owner}/${repo}`} and start syncing immediately.</span>
            </button>
            <button type="button" className={`gh-choice-card ${!connectNow ? "selected" : ""}`} onClick={() => setConnectNow(false)}>
              <span className="gh-choice-title">Skip for now</span>
              <span className="gh-choice-desc">Set up the board without linking it. You can connect it later.</span>
            </button>
          </div>
        )}

        {step === "import" && (
          <div className="gh-wiz-import">
            <label className="gh-check">
              <input type="checkbox" checked={importItems} onChange={(e) => setImportItems(e.target.checked)} />
              Import items from {fullName || `${owner}/${repo}`}
            </label>

            {importItems && (
              <div className="gh-seg gh-seg-3">
                <button type="button" className={`gh-seg-opt ${importScope === "issues" ? "on" : ""}`} onClick={() => setImportScope("issues")}>
                  Open issues
                </button>
                <button type="button" className={`gh-seg-opt ${importScope === "prs" ? "on" : ""}`} onClick={() => setImportScope("prs")}>
                  Open pull requests
                </button>
                <button type="button" className={`gh-seg-opt ${importScope === "both" ? "on" : ""}`} onClick={() => setImportScope("both")}>
                  Both
                </button>
              </div>
            )}

            {connectNow && (
              <label className="gh-check">
                <input type="checkbox" checked={asIssue} onChange={(e) => setAsIssue(e.target.checked)} />
                Create real issues instead of drafts for new items
              </label>
            )}
          </div>
        )}

        {step === "layout" && (
          <div className="gh-wiz-layout">
            {presets === null ? (
              <span className="gh-muted">Loading layouts…</span>
            ) : (
              <div className="gh-preset-grid">
                {presets.map((p) => (
                  <button
                    type="button"
                    key={p.key}
                    className={`gh-preset-card ${p.key === presetKey ? "selected" : ""}`}
                    onClick={() => setPresetKey(p.key)}
                  >
                    <span className="gh-preset-title">{p.label}</span>
                    <span className="gh-preset-desc">{p.description}</span>
                    <span className="gh-preset-cols">
                      {p.columns.map((c) => (
                        <span key={c.name} className="gh-preset-col" style={{ "--col-color": ghColumnColor(c.color) }}>
                          {c.name}
                        </span>
                      ))}
                    </span>
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        {step === "review" && (
          <div className="gh-wiz-review">
            <div className="gh-review-row">
              <span className="gh-review-key">Account</span>
              <span className="gh-review-val">{ownerLogin}</span>
            </div>
            <div className="gh-review-row">
              <span className="gh-review-key">Board</span>
              <span className="gh-review-val">
                {boardMode === "new" ? `${title.trim()} (new)` : boardLabel(selectedBoard || {})}
              </span>
            </div>
            {boardMode === "new" && (
              <div className="gh-review-row">
                <span className="gh-review-key">Layout</span>
                <span className="gh-review-val">{presets?.find((p) => p.key === presetKey)?.label || presetKey}</span>
              </div>
            )}
            <div className="gh-review-row">
              <span className="gh-review-key">Repository</span>
              <span className="gh-review-val">{connectNow ? `Connected to ${fullName || `${owner}/${repo}`}` : "Not connected"}</span>
            </div>
            <div className="gh-review-row">
              <span className="gh-review-key">Import</span>
              <span className="gh-review-val">
                {importItems ? (importScope === "both" ? "Open issues & pull requests" : importScope === "issues" ? "Open issues" : "Open pull requests") : "Nothing"}
              </span>
            </div>
            {connectNow && (
              <div className="gh-review-row">
                <span className="gh-review-key">New items sync as</span>
                <span className="gh-review-val">{asIssue ? "Real issues" : "Draft items"}</span>
              </div>
            )}
          </div>
        )}

        {localErr && <div className="gh-warn-inline">{localErr}</div>}
      </div>

      <div className="gh-wiz-actions">
        {stepIndex > 0 || onCancel ? (
          <button className="btn small ghost" disabled={busy} onClick={goBack}>
            {stepIndex === 0 ? "Cancel" : "Back"}
          </button>
        ) : (
          <span />
        )}
        {step === "review" ? (
          <button className="btn primary gh-cta" disabled={busy} onClick={submit}>
            {busyLabel || (boardMode === "new" ? (connectNow ? "Create board & sync" : "Create board") : connectNow ? "Connect & sync" : "Save")}
          </button>
        ) : (
          <button className="btn primary gh-cta" disabled={!canAdvance} onClick={goNext}>
            Continue
          </button>
        )}
      </div>
    </div>
  );
}
