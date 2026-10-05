import { useMemo, useRef, useState } from "react";
import {
  Button,
  Dialog,
  DialogBody,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Progress,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@pikoloo/darwin-ui";
import { Download, FileDown, FileUp, Upload as UploadIcon } from "lucide-react";
import {
  EventsOn,
  ExportTasksSheet,
  ImportTasksCSVText,
} from "../../api";
import { capture } from "../../analytics";
import SearchableSelect from "../SearchableSelect";
import { COLUMNS, TYPES } from "./columns";
import { downloadSampleCSV, downloadSampleGuide } from "./csvSample";
import { BACKLOG_ID } from "./sprints";

const ALL_BOARDS = "__all__";

const PRIORITIES = [
  { id: "low", label: "Low" },
  { id: "medium", label: "Medium" },
  { id: "high", label: "High" },
];

const EXPORT_FORMATS = [
  { id: "xlsx", label: "Excel (.xlsx)", hint: "Spreadsheet — open in Numbers / Excel / Sheets" },
  { id: "pdf", label: "PDF", hint: "Printable landscape sheet or kanban board" },
  { id: "png", label: "Image (.png)", hint: "Shareable board snapshot" },
  { id: "csv", label: "CSV", hint: "Raw data for re-import" },
];

const PDF_LAYOUTS = [
  { id: "board", label: "Board", hint: "Kanban columns — Backlog, To Do, In Progress, Testing, Done" },
  { id: "table", label: "Table", hint: "Tabular spreadsheet-style rows" },
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
// export can ship every task or a filtered subset. The import tab also
// offers a sample CSV + guide download so users can populate offline.
export default function TasksCsvModal({
  projectId,
  projectName,
  tasks,
  sprints,
  initialSprintScope = ALL_BOARDS,
  onImported,
  onClose,
  onError,
}) {
  const [tab, setTab] = useState("import");
  const [busy, setBusy] = useState(false); // "import" | "export" | false
  const [progress, setProgress] = useState(null); // { done, total }
  const [resultMsg, setResultMsg] = useState("");
  const [localErr, setLocalErr] = useState("");
  const [dragOver, setDragOver] = useState(false);
  const [pendingFile, setPendingFile] = useState(null); // { name, text }
  const [importMode, setImportMode] = useState("add"); // "add" | "replace"
  // Which sprint board import/export targets. "__all__" = whole project
  // (CSV sprint column decides on import); "" = backlog; else a sprint id.
  const [boardScope, setBoardScope] = useState(() =>
    initialSprintScope === undefined || initialSprintScope === null
      ? ALL_BOARDS
      : initialSprintScope
  );

  const [exportAll, setExportAll] = useState(true);
  const [exportFormat, setExportFormat] = useState("xlsx");
  // PDF-only: "board" (kanban columns) or "table" (tabular rows).
  const [exportLayout, setExportLayout] = useState("board");
  const [statuses, setStatuses] = useState([]);
  const [sprintIds, setSprintIds] = useState([]);
  const [types, setTypes] = useState([]);
  const [labels, setLabels] = useState([]);
  const [priorities, setPriorities] = useState([]);

  const fileRef = useRef(null);
  const boardScoped = boardScope !== ALL_BOARDS;

  const boardOptions = useMemo(
    () => [
      { id: ALL_BOARDS, label: "All tasks" },
      { id: BACKLOG_ID, label: "Backlog" },
      ...(sprints || []).map((s) => ({
        id: s.id,
        label: s.name || "Untitled sprint",
      })),
    ],
    [sprints]
  );

  const boardLabel =
    boardOptions.find((o) => o.id === boardScope)?.label || "All tasks";

  const scopedTasks = useMemo(() => {
    if (!boardScoped) return tasks || [];
    return (tasks || []).filter((t) => (t.sprintId || "") === boardScope);
  }, [tasks, boardScope, boardScoped]);

  const allLabels = useMemo(() => {
    const set = new Set();
    for (const t of scopedTasks) {
      for (const l of t.labels || []) if (l) set.add(l);
    }
    return [...set].sort((a, b) => a.localeCompare(b));
  }, [scopedTasks]);

  const matchedCount = useMemo(() => {
    if (exportAll) return scopedTasks.length;
    const statusOK = new Set(statuses);
    const sprintOK = new Set(sprintIds);
    const typeOK = new Set(types);
    const labelOK = new Set(labels);
    const prioOK = new Set(priorities);
    return scopedTasks.filter((t) => {
      if (statusOK.size && !statusOK.has(t.status || "todo")) return false;
      if (!boardScoped && sprintOK.size && !sprintOK.has(t.sprintId || "")) return false;
      if (typeOK.size && !typeOK.has(t.type || "task")) return false;
      if (prioOK.size && !prioOK.has(t.priority || "")) return false;
      if (labelOK.size && !(t.labels || []).some((l) => labelOK.has(l))) return false;
      return true;
    }).length;
  }, [exportAll, scopedTasks, boardScoped, statuses, sprintIds, types, labels, priorities]);

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
      const res = await ImportTasksCSVText(
        projectId,
        pendingFile.text,
        importMode,
        boardScope
      );
      if (!res) return;
      onImported && onImported(res.tasks, res.sprints);
      capture("tasks_csv_imported", {
        mode: importMode,
        sprint_scope: boardScope,
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
        ? {
            statuses: [],
            sprintIds: boardScoped ? [boardScope] : [],
            types: [],
            labels: [],
            priorities: [],
          }
        : {
            statuses,
            sprintIds: boardScoped ? [boardScope] : sprintIds,
            types,
            labels,
            priorities,
          };
      const layout = exportFormat === "pdf" ? exportLayout : "table";
      const res = await ExportTasksSheet(projectId, exportFormat, layout, filter);
      if (!res?.path) return; // cancelled folder picker
      capture("tasks_sheet_exported", {
        filename: res.filename,
        count: res.count,
        format: exportFormat,
        layout,
        sprint_scope: boardScope,
      });
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

  const ChipGroup = ({ title, options, selected, onToggle, emptyHint, disabled }) => (
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
              disabled={disabled || busy === "export"}
            >
              {o.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !busy) onClose();
      }}
    >
      <DialogContent size="lg" glass className="csv-modal darwin-csv-modal">
        <DialogHeader>
          <DialogTitle>Import / Export tasks</DialogTitle>
          <DialogDescription>
            Download the {projectName || "board"} sheet as Excel, PDF, or image
            with optional filters, or import a CSV. Pick a sprint board below to
            scope the transfer; with All tasks, use the <code>sprint</code>{" "}
            column on import.
          </DialogDescription>
          <DialogClose />
        </DialogHeader>

        <DialogBody className="csv-modal-body">
          <Tabs
            value={tab}
            onValueChange={(next) => {
              if (busy) return;
              setTab(next);
              setLocalErr("");
              setResultMsg("");
            }}
            glass
            className="csv-tabs-shell"
          >
            <TabsList>
              <TabsTrigger value="import" disabled={!!busy} icon={<FileUp size={14} />}>
                Import
              </TabsTrigger>
              <TabsTrigger value="export" disabled={!!busy} icon={<FileDown size={14} />}>
                Export
              </TabsTrigger>
            </TabsList>

            <div className="csv-board-select">
              <label id="csv-board-scope-label">Sprint board</label>
              <SearchableSelect
                value={boardLabel}
                options={boardOptions.map((o) => o.label)}
                disabled={!!busy}
                placeholder="Select sprint board…"
                searchPlaceholder="Search boards…"
                onChange={(label) => {
                  const next = boardOptions.find((o) => o.label === label);
                  if (!next) return;
                  setBoardScope(next.id);
                  setLocalErr("");
                  setResultMsg("");
                }}
              />
            </div>

            <TabsContent value="export" className="csv-tab-panel">
              <div className="csv-tab-body">
                <div className="csv-format-row">
                  {EXPORT_FORMATS.map((f) => (
                    <button
                      key={f.id}
                      type="button"
                      className={`csv-format-chip ${exportFormat === f.id ? "on" : ""}`}
                      disabled={!!busy}
                      onClick={() => setExportFormat(f.id)}
                      title={f.hint}
                    >
                      {f.label}
                    </button>
                  ))}
                </div>

                {exportFormat === "pdf" && (
                  <div className="csv-layout-row">
                    <div className="csv-filter-label">PDF layout</div>
                    <div className="csv-format-row">
                      {PDF_LAYOUTS.map((l) => (
                        <button
                          key={l.id}
                          type="button"
                          className={`csv-format-chip ${exportLayout === l.id ? "on" : ""}`}
                          disabled={!!busy}
                          onClick={() => setExportLayout(l.id)}
                          title={l.hint}
                        >
                          {l.label}
                        </button>
                      ))}
                    </div>
                  </div>
                )}

                <div className="csv-scope">
                  <label className="csv-radio">
                    <input
                      type="radio"
                      checked={exportAll}
                      disabled={!!busy}
                      onChange={() => setExportAll(true)}
                    />
                    <span>
                      Entire {boardScoped ? boardLabel.toLowerCase() : "board"} —{" "}
                      <strong>{scopedTasks.length}</strong> tasks
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
                      Filtered view{" "}
                      <strong className={!exportAll ? "" : "csv-muted"}>
                        ({matchedCount} matching)
                      </strong>
                    </span>
                  </label>
                </div>

                {/* Always mounted so Entire board ↔ Filtered (and Import ↔ Export)
                    don't collapse the dialog height. */}
                <div
                  className={`csv-filters ${exportAll ? "is-disabled" : ""}`}
                  aria-disabled={exportAll || undefined}
                >
                  <ChipGroup
                    title="Columns (boards / views)"
                    options={COLUMNS.map((c) => ({ id: c.id, label: c.label }))}
                    selected={statuses}
                    onToggle={(id) => setStatuses((s) => toggleIn(s, id))}
                    disabled={exportAll}
                  />
                  {!boardScoped && (
                    <ChipGroup
                      title="Sprints"
                      options={[
                        { id: BACKLOG_ID, label: "Backlog" },
                        ...(sprints || []).map((s) => ({
                          id: s.id,
                          label: s.name || "Untitled sprint",
                        })),
                      ]}
                      selected={sprintIds}
                      onToggle={(id) => setSprintIds((s) => toggleIn(s, id))}
                      emptyHint="No sprints yet — only Backlog is available."
                      disabled={exportAll}
                    />
                  )}
                  <ChipGroup
                    title="Types"
                    options={TYPES.map((t) => ({ id: t.id, label: t.label }))}
                    selected={types}
                    onToggle={(id) => setTypes((s) => toggleIn(s, id))}
                    disabled={exportAll}
                  />
                  <ChipGroup
                    title="Labels / tags"
                    options={allLabels.map((l) => ({ id: l, label: l }))}
                    selected={labels}
                    onToggle={(id) => setLabels((s) => toggleIn(s, id))}
                    emptyHint="No labels on this board yet."
                    disabled={exportAll}
                  />
                  <ChipGroup
                    title="Priority"
                    options={PRIORITIES}
                    selected={priorities}
                    onToggle={(id) => setPriorities((s) => toggleIn(s, id))}
                    disabled={exportAll}
                  />
                  <p className="csv-filter-hint">
                    {exportAll
                      ? boardScoped
                        ? `Exporting from ${boardLabel}. Select Filtered view to narrow further by column, type, label, or priority.`
                        : "Select Filtered view above to narrow by column, sprint, type, label, or priority."
                      : "Leave a group untouched to include every value in that group. Selection uses AND across groups."}
                  </p>
                </div>
              </div>
            </TabsContent>

            <TabsContent value="import" className="csv-tab-panel">
              <div className="csv-tab-body">
                <div className="csv-sample-card">
                  <div className="csv-sample-copy">
                    <strong>New to bulk import?</strong>
                    <p>
                      Download the sample guide, fill in rows offline, then drop the CSV below.
                      Leave <code>id</code> blank for new tasks.
                    </p>
                  </div>
                  <div className="csv-sample-actions">
                    <Button
                      type="button"
                      variant="secondary"
                      size="sm"
                      glass
                      disabled={!!busy}
                      leftIcon={<Download size={14} />}
                      onClick={async () => {
                        setLocalErr("");
                        try {
                          const path = await downloadSampleGuide();
                          if (!path) return; // cancelled save dialog
                          capture("tasks_csv_sample_guide_downloaded");
                          setResultMsg(`Saved sample guide → ${path.split(/[/\\]/).pop()}`);
                        } catch (e) {
                          setLocalErr(String(e));
                        }
                      }}
                    >
                      Sample guide
                    </Button>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      disabled={!!busy}
                      leftIcon={<FileDown size={14} />}
                      onClick={async () => {
                        setLocalErr("");
                        try {
                          const path = await downloadSampleCSV();
                          if (!path) return; // cancelled save dialog
                          capture("tasks_csv_sample_downloaded");
                          setResultMsg(`Saved sample CSV → ${path.split(/[/\\]/).pop()}`);
                        } catch (e) {
                          setLocalErr(String(e));
                        }
                      }}
                    >
                      Sample CSV
                    </Button>
                  </div>
                </div>

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
                        — merge into {boardScoped ? boardLabel : "the board"}. Matching ids update; new
                        rows create tasks
                        {boardScoped ? ` on ${boardLabel}` : ""}. Existing cards not in the file stay
                        put (use this for an incremental CSV).
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
                        — treat the CSV as the full {boardScoped ? boardLabel : "board"}. Matching ids
                        update; rows not in the file are removed
                        {boardScoped ? ` from ${boardLabel} only` : ""}.
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
                  <UploadIcon className="csv-drop-icon" aria-hidden />
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
                        or click to choose a file.
                        {boardScoped
                          ? ` Tasks will be imported into ${boardLabel}.`
                          : (
                            <>
                              {" "}
                              Put sprint names in the <code>sprint</code> column.
                            </>
                          )}
                      </div>
                    </>
                  )}
                </div>
              </div>
            </TabsContent>
          </Tabs>

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
              <Progress
                value={progress?.total ? pct : undefined}
                indeterminate={!progress?.total}
                size="sm"
                showValue={!!progress?.total}
              />
            </div>
          )}

          {resultMsg && !busy && <div className="csv-result ok">{resultMsg}</div>}
          {localErr && <div className="csv-result err">{localErr}</div>}
        </DialogBody>

        <DialogFooter className="csv-modal-footer">
          <Button type="button" variant="ghost" disabled={!!busy} onClick={onClose}>
            Close
          </Button>
          {tab === "export" ? (
            <Button
              type="button"
              variant="primary"
              disabled={!!busy || matchedCount === 0}
              loading={busy === "export"}
              loadingText="Exporting…"
              onClick={runExport}
            >
              Download {matchedCount} as {exportFormat.toUpperCase()}
            </Button>
          ) : (
            <Button
              type="button"
              variant="primary"
              disabled={!!busy || !pendingFile}
              loading={busy === "import"}
              loadingText="Importing…"
              onClick={runImport}
            >
              {importMode === "replace" ? "Replace with CSV" : "Add from CSV"}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
