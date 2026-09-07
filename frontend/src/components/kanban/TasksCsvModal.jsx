import { useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import {
  EventsOn,
  ExportTasksCSV,
  ImportTasksCSVText,
} from "../../api";
import { capture } from "../../analytics";
import { COLUMNS, TYPES } from "./columns";

const PRIORITIES = [
  { id: "low", label: "Low" },
  { id: "medium", label: "Medium" },
  { id: "high", label: "High" },
];

const toggleIn = (list, id) =>
  list.includes(id) ? list.filter((x) => x !== id) : [...list, id];

const readFileText = (file) =>
  new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result || ""));
    reader.onerror = () => reject(reader.error || new Error("Could not read file"));
    reader.readAsText(file);
  });

const importSummary = (res) => {
  const parts = [
    `${res.total} row${res.total === 1 ? "" : "s"}`,
    `${res.created} created`,
    `${res.updated} updated`,
  ];
  if (res.removed > 0) parts.push(`${res.removed} removed`);
  if (res.sprintsCreated > 0) {
    parts.push(
      `${res.sprintsCreated} sprint${res.sprintsCreated === 1 ? "" : "s"} added`
    );
  }
  return `Imported ${parts.join(" · ")}`;
};

// Bulk import / export sheet for the kanban. Import accepts a file
// picker or drag-and-drop with Add (incremental) or Replace modes;
// export can ship every task or a filtered subset (columns, sprints,
// types, labels, priorities). Both sides show a live progress meter
// driven by Go events.
export default function TasksCsvModal({
  projectId,
  projectName,
  tasks,
  sprints,
  onImported,
  onClose,
  onError,
}) {
  const [tab, setTab] = useState("export");
  const [busy, setBusy] = useState(false); // "import" | "export" | false
  const [progress, setProgress] = useState(null); // { done, total }
  const [resultMsg, setResultMsg] = useState("");
  const [localErr, setLocalErr] = useState("");
  const [dragOver, setDragOver] = useState(false);
  const [pendingFile, setPendingFile] = useState(null); // { name, text }
  const [importMode, setImportMode] = useState("add"); // "add" | "replace"

  const [exportAll, setExportAll] = useState(true);
  const [statuses, setStatuses] = useState([]);
  const [sprintIds, setSprintIds] = useState([]);
  const [types, setTypes] = useState([]);
  const [labels, setLabels] = useState([]);
  const [priorities, setPriorities] = useState([]);

  const fileRef = useRef(null);

  const allLabels = useMemo(() => {
    const set = new Set();
    for (const t of tasks || []) {
      for (const l of t.labels || []) if (l) set.add(l);
    }
    return [...set].sort((a, b) => a.localeCompare(b));
  }, [tasks]);

  const matchedCount = useMemo(() => {
    if (exportAll) return (tasks || []).length;
    const statusOK = new Set(statuses);
    const sprintOK = new Set(sprintIds);
    const typeOK = new Set(types);
    const labelOK = new Set(labels);
    const prioOK = new Set(priorities);
    return (tasks || []).filter((t) => {
      if (statusOK.size && !statusOK.has(t.status || "todo")) return false;
      if (sprintOK.size && !sprintOK.has(t.sprintId || "")) return false;
      if (typeOK.size && !typeOK.has(t.type || "task")) return false;
      if (prioOK.size && !prioOK.has(t.priority || "")) return false;
      if (labelOK.size && !(t.labels || []).some((l) => labelOK.has(l))) return false;
      return true;
    }).length;
  }, [exportAll, tasks, statuses, sprintIds, types, labels, priorities]);

  const pct =
    progress?.total > 0 ? Math.round((progress.done / progress.total) * 100) : 0;

  const listenProgress = (event) => {
    const off = EventsOn(event, (p) => {
      if (!p || p.projectId !== projectId || !p.total) return;
      setProgress({ done: p.done || 0, total: p.total });
    });
    return off;
  };

  const acceptFile = async (file) => {
    if (!file) return;
    const name = file.name || "tasks.csv";
    if (!/\.csv$/i.test(name) && file.type && !file.type.includes("csv") && !file.type.includes("text")) {
      setLocalErr("Please choose a .csv file.");
      return;
    }
    try {
      const text = await readFileText(file);
      setPendingFile({ name, text });
      setLocalErr("");
      setResultMsg("");
    } catch (e) {
      setLocalErr(String(e));
    }
  };

  const runImport = async () => {
    if (busy || !pendingFile?.text) return;
    setBusy("import");
    setProgress(null);
    setLocalErr("");
    setResultMsg("");
    const off = listenProgress("tasks:csv:progress");
    try {
      const res = await ImportTasksCSVText(projectId, pendingFile.text, importMode);
      if (!res) return;
      onImported && onImported(res.tasks, res.sprints);
      capture("tasks_csv_imported", {
        mode: importMode,
        updated: res.updated,
        created: res.created,
        removed: res.removed || 0,
        sprintsCreated: res.sprintsCreated || 0,
        total: res.total,
      });
      setResultMsg(importSummary(res));
      setPendingFile(null);
    } catch (e) {
      setLocalErr(String(e));
      onError && onError(String(e));
    } finally {
      off && off();
      setBusy(false);
      setProgress(null);
    }
  };

  const runExport = async () => {
    if (busy) return;
    if (!exportAll && matchedCount === 0) {
      setLocalErr("No tasks match the selected filters.");
      return;
    }
    setBusy("export");
    setProgress(null);
    setLocalErr("");
    setResultMsg("");
    const off = listenProgress("tasks:csv:export:progress");
    try {
      const filter = exportAll
        ? { statuses: [], sprintIds: [], types: [], labels: [], priorities: [] }
        : {
            statuses,
            sprintIds,
            types,
            labels,
            priorities,
          };
      const res = await ExportTasksCSV(projectId, filter);
      if (!res?.path) return; // cancelled folder picker
      capture("tasks_csv_exported", { filename: res.filename, count: res.count });
      setResultMsg(`Exported ${res.count} task${res.count === 1 ? "" : "s"} → ${res.filename}`);
    } catch (e) {
      setLocalErr(String(e));
      onError && onError(String(e));
    } finally {
      off && off();
      setBusy(false);
      setProgress(null);
    }
  };

  const ChipGroup = ({ title, options, selected, onToggle, emptyHint }) => (
    <div className="csv-filter-group">
      <div className="csv-filter-label">{title}</div>
      {options.length === 0 ? (
        <p className="csv-filter-empty">{emptyHint}</p>
      ) : (
        <div className="csv-chip-row">
          {options.map((o) => (
            <button
              key={o.id}
              type="button"
              className={`csv-chip ${selected.includes(o.id) ? "on" : ""}`}
              onClick={() => onToggle(o.id)}
              disabled={busy === "export"}
            >
              {o.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );

  return createPortal(
    <div className="modal-overlay" onClick={() => !busy && onClose()}>
      <div
        className="modal pinned-actions csv-modal"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="modal-scroll-body">
          <h2>Import / Export tasks</h2>
          <p className="csv-modal-sub">
            Move {projectName || "board"} items in or out as CSV. Use the{" "}
            <code>sprint</code> column for sprint names — unknown names create
            the sprint locally (and on GitHub when the board is linked).
          </p>

          <div className="seg csv-tabs">
            <button
              type="button"
              className={tab === "export" ? "on" : ""}
              disabled={!!busy}
              onClick={() => {
                setTab("export");
                setLocalErr("");
                setResultMsg("");
              }}
            >
              Export
            </button>
            <button
              type="button"
              className={tab === "import" ? "on" : ""}
              disabled={!!busy}
              onClick={() => {
                setTab("import");
                setLocalErr("");
                setResultMsg("");
              }}
            >
              Import
            </button>
          </div>

          {tab === "export" && (
            <div className="csv-tab-body">
              <div className="csv-scope">
                <label className="csv-radio">
                  <input
                    type="radio"
                    checked={exportAll}
                    disabled={!!busy}
                    onChange={() => setExportAll(true)}
                  />
                  <span>
                    Export all <strong>{(tasks || []).length}</strong> tasks
                  </span>
                </label>
                <label className="csv-radio">
                  <input
                    type="radio"
                    checked={!exportAll}
                    disabled={!!busy}
                    onChange={() => setExportAll(false)}
                  />
                  <span>
                    Export a selection{" "}
                    <strong className={!exportAll ? "" : "csv-muted"}>
                      ({matchedCount} matching)
                    </strong>
                  </span>
                </label>
              </div>

              {!exportAll && (
                <div className="csv-filters">
                  <ChipGroup
                    title="Columns (boards)"
                    options={COLUMNS.map((c) => ({ id: c.id, label: c.label }))}
                    selected={statuses}
                    onToggle={(id) => setStatuses((s) => toggleIn(s, id))}
                  />
                  <ChipGroup
                    title="Sprints"
                    options={[
                      { id: "", label: "Backlog" },
                      ...(sprints || []).map((s) => ({
                        id: s.id,
                        label: s.name || "Untitled sprint",
                      })),
                    ]}
                    selected={sprintIds}
                    onToggle={(id) => setSprintIds((s) => toggleIn(s, id))}
                    emptyHint="No sprints yet — only Backlog is available."
                  />
                  <ChipGroup
                    title="Types"
                    options={TYPES.map((t) => ({ id: t.id, label: t.label }))}
                    selected={types}
                    onToggle={(id) => setTypes((s) => toggleIn(s, id))}
                  />
                  <ChipGroup
                    title="Labels / tags"
                    options={allLabels.map((l) => ({ id: l, label: l }))}
                    selected={labels}
                    onToggle={(id) => setLabels((s) => toggleIn(s, id))}
                    emptyHint="No labels on this board yet."
                  />
                  <ChipGroup
                    title="Priority"
                    options={PRIORITIES}
                    selected={priorities}
                    onToggle={(id) => setPriorities((s) => toggleIn(s, id))}
                  />
                  <p className="csv-filter-hint">
                    Leave a group untouched to include every value in that group. Selection uses AND across groups.
                  </p>
                </div>
              )}
            </div>
          )}

          {tab === "import" && (
            <div className="csv-tab-body">
              <div className="csv-scope">
                <label className="csv-radio">
                  <input
                    type="radio"
                    name="csv-import-mode"
                    checked={importMode === "add"}
                    disabled={!!busy}
                    onChange={() => setImportMode("add")}
                  />
                  <span>
                    <strong>Add</strong>
                    <span className="csv-mode-desc">
                      {" "}
                      — merge into the board. Matching ids update; new rows create tasks. Existing cards
                      not in the file stay put (use this for an incremental CSV).
                    </span>
                  </span>
                </label>
                <label className="csv-radio">
                  <input
                    type="radio"
                    name="csv-import-mode"
                    checked={importMode === "replace"}
                    disabled={!!busy}
                    onChange={() => setImportMode("replace")}
                  />
                  <span>
                    <strong>Replace</strong>
                    <span className="csv-mode-desc">
                      {" "}
                      — treat the CSV as the full board. Matching ids update; rows not in the file are
                      removed.
                    </span>
                  </span>
                </label>
              </div>

              <div
                className={`csv-dropzone ${dragOver ? "over" : ""} ${pendingFile ? "has-file" : ""}`}
                onDragEnter={(e) => {
                  e.preventDefault();
                  setDragOver(true);
                }}
                onDragOver={(e) => {
                  e.preventDefault();
                  setDragOver(true);
                }}
                onDragLeave={() => setDragOver(false)}
                onDrop={(e) => {
                  e.preventDefault();
                  setDragOver(false);
                  const file = e.dataTransfer?.files?.[0];
                  acceptFile(file);
                }}
                onClick={() => !busy && fileRef.current?.click()}
                role="button"
                tabIndex={0}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    fileRef.current?.click();
                  }
                }}
              >
                <input
                  ref={fileRef}
                  type="file"
                  accept=".csv,text/csv"
                  hidden
                  disabled={!!busy}
                  onChange={(e) => {
                    acceptFile(e.target.files?.[0]);
                    e.target.value = "";
                  }}
                />
                {pendingFile ? (
                  <>
                    <div className="csv-drop-title">{pendingFile.name}</div>
                    <div className="csv-drop-desc">
                      Ready to {importMode === "replace" ? "replace" : "add"}. Click Import, or drop
                      another file to swap.
                    </div>
                  </>
                ) : (
                  <>
                    <div className="csv-drop-title">Drop a CSV here</div>
                    <div className="csv-drop-desc">
                      or click to choose a file. Put sprint names in the{" "}
                      <code>sprint</code> column.
                    </div>
                  </>
                )}
              </div>
            </div>
          )}

          {busy && (
            <div className="csv-progress">
              <span>
                {busy === "import"
                  ? progress?.total
                    ? `Importing ${progress.done}/${progress.total}…`
                    : "Importing…"
                  : progress?.total
                    ? `Exporting ${progress.done}/${progress.total}…`
                    : "Choose a folder…"}
              </span>
              <div className={`meter ${progress?.total ? "" : "indeterminate"}`}>
                <div style={progress?.total ? { width: `${pct}%` } : undefined} />
              </div>
              {progress?.total > 0 && <span>{pct}%</span>}
            </div>
          )}

          {resultMsg && !busy && <div className="csv-result ok">{resultMsg}</div>}
          {localErr && <div className="csv-result err">{localErr}</div>}
        </div>

        <div className="modal-actions">
          <button type="button" className="btn" disabled={!!busy} onClick={onClose}>
            Close
          </button>
          {tab === "export" ? (
            <button
              type="button"
              className="btn primary"
              disabled={!!busy || matchedCount === 0}
              onClick={runExport}
            >
              {busy === "export" ? "Exporting…" : `Export ${matchedCount} task${matchedCount === 1 ? "" : "s"}`}
            </button>
          ) : (
            <button
              type="button"
              className="btn primary"
              disabled={!!busy || !pendingFile}
              onClick={runImport}
            >
              {busy === "import"
                ? "Importing…"
                : importMode === "replace"
                  ? "Replace with CSV"
                  : "Add from CSV"}
            </button>
          )}
        </div>
      </div>
    </div>,
    document.body
  );
}
