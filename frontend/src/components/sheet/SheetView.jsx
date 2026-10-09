import { useEffect, useMemo, useRef, useState } from "react";
import {
  flexRender,
  getCoreRowModel,
  getExpandedRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";
import { formatDue, isPastDue, isValidDate } from "../../dueDates";
import ConfirmDialog from "../ConfirmDialog";

const PRIORITIES = ["", "low", "medium", "high"];
const TYPES = ["story", "task", "bug"];
const fmtTs = (ms) => (ms ? new Date(ms).toISOString().slice(0, 10) : "");
const prefsKey = (projectId) => `jumpstart.sheet.${projectId}`;

const loadPrefs = (projectId) => {
  try {
    return JSON.parse(localStorage.getItem(prefsKey(projectId)) || "{}");
  } catch {
    return {};
  }
};

// --- column filters (Excel-style) ---
const filterFns = {
  values: (row, id, allowed) => !allowed || allowed.includes(String(row.getValue(id) ?? "")),
  text: (row, id, needle) => !needle || String(row.getValue(id) ?? "").toLowerCase().includes(needle.toLowerCase()),
  dateRange: (row, id, f) => {
    if (!f) return true;
    const v = row.getValue(id) || "";
    if (f.empty === "empty") return !v;
    if (f.empty === "set" && !v) return false;
    if ((f.from || f.to) && !v) return false;
    if (f.from && v < f.from) return false;
    if (f.to && v > f.to) return false;
    return true;
  },
  numberRange: (row, id, f) => {
    if (!f) return true;
    const v = Number(row.getValue(id) || 0);
    if (f.min !== "" && f.min !== undefined && v < Number(f.min)) return false;
    if (f.max !== "" && f.max !== undefined && v > Number(f.max)) return false;
    return true;
  },
};

function HeaderMenu({ column, rows, onClose }) {
  const kind = column.columnDef.meta?.filter || "text";
  const current = column.getFilterValue();
  const [search, setSearch] = useState("");
  const ref = useRef(null);

  useEffect(() => {
    const onDoc = (e) => ref.current && !ref.current.contains(e.target) && onClose();
    const onKey = (e) => e.key === "Escape" && onClose();
    setTimeout(() => document.addEventListener("mousedown", onDoc), 0);
    window.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      window.removeEventListener("keydown", onKey);
    };
  }, [onClose]);

  const values = useMemo(() => {
    if (kind !== "values") return [];
    const set = new Set();
    rows.forEach((r) => set.add(String(r.getValue(column.id) ?? "")));
    return [...set].sort((a, b) => a.localeCompare(b));
  }, [kind, rows, column.id]);

  const allowed = current || values;
  const toggleValue = (v) => {
    const next = allowed.includes(v) ? allowed.filter((x) => x !== v) : [...allowed, v];
    column.setFilterValue(next.length === values.length ? undefined : next);
  };

  return (
    <div className="sheet-menu" ref={ref} role="dialog" aria-label={`${column.columnDef.header} column menu`} onClick={(e) => e.stopPropagation()}>
      <div className="sheet-menu-row">
        <button type="button" className="btn tiny" onClick={() => column.toggleSorting(false)}>Sort A → Z</button>
        <button type="button" className="btn tiny" onClick={() => column.toggleSorting(true)}>Sort Z → A</button>
        {column.getIsSorted() && (
          <button type="button" className="link-btn" onClick={() => column.clearSorting()}>Clear sort</button>
        )}
      </div>

      {kind === "values" && (
        <div className="sheet-menu-values">
          <input placeholder="Search values…" aria-label="Search values" value={search} onChange={(e) => setSearch(e.target.value)} />
          <label className="sheet-menu-check">
            <input type="checkbox" checked={!current} onChange={() => column.setFilterValue(current ? undefined : [])} />
            (Select all)
          </label>
          <div className="sheet-menu-list">
            {values
              .filter((v) => !search || v.toLowerCase().includes(search.toLowerCase()))
              .map((v) => (
                <label className="sheet-menu-check" key={v || "(blank)"}>
                  <input type="checkbox" checked={allowed.includes(v)} onChange={() => toggleValue(v)} />
                  {v || "(blank)"}
                </label>
              ))}
          </div>
        </div>
      )}

      {kind === "text" && (
        <input
          aria-label={`Filter ${column.columnDef.header} containing`}
          placeholder="Contains…"
          value={current || ""}
          autoFocus
          onChange={(e) => column.setFilterValue(e.target.value || undefined)}
        />
      )}

      {kind === "dateRange" && (
        <div className="sheet-menu-range">
          <select
            aria-label="Date filter type"
            value={current?.empty || ""}
            onChange={(e) => column.setFilterValue({ ...(current || {}), empty: e.target.value || undefined })}
          >
            <option value="">Any</option>
            <option value="set">Has a date</option>
            <option value="empty">No date</option>
          </select>
          <input type="date" aria-label="From date" value={current?.from || ""} onChange={(e) => column.setFilterValue({ ...(current || {}), from: e.target.value })} />
          <span>to</span>
          <input type="date" aria-label="To date" value={current?.to || ""} onChange={(e) => column.setFilterValue({ ...(current || {}), to: e.target.value })} />
        </div>
      )}

      {kind === "numberRange" && (
        <div className="sheet-menu-range">
          <input type="number" aria-label="Minimum" placeholder="Min" value={current?.min ?? ""} onChange={(e) => column.setFilterValue({ ...(current || {}), min: e.target.value })} />
          <span>to</span>
          <input type="number" aria-label="Maximum" placeholder="Max" value={current?.max ?? ""} onChange={(e) => column.setFilterValue({ ...(current || {}), max: e.target.value })} />
        </div>
      )}

      <div className="sheet-menu-row">
        {column.getFilterValue() !== undefined && (
          <button type="button" className="link-btn" onClick={() => column.setFilterValue(undefined)}>Clear filter</button>
        )}
        {column.getCanHide() && (
          <button type="button" className="link-btn" onClick={() => { column.toggleVisibility(false); onClose(); }}>Hide column</button>
        )}
      </div>
    </div>
  );
}

// --- cell editors ---
function CellEditor({ meta, value, onCommit, onCancel, onTab }) {
  const [v, setV] = useState(value ?? "");
  const ref = useRef(null);
  useEffect(() => {
    ref.current?.focus();
    if (ref.current?.select && meta.editor !== "date") ref.current.select();
  }, [meta.editor]);
  const commit = (next = v) => onCommit(next);
  const keys = (e) => {
    if (e.key === "Enter") { e.preventDefault(); commit(); }
    if (e.key === "Escape") { e.preventDefault(); onCancel(); }
    if (e.key === "Tab") { e.preventDefault(); commit(); onTab(e.shiftKey ? -1 : 1); }
  };
  if (meta.editor === "select") {
    return (
      <select ref={ref} className="sheet-editor" aria-label="Edit cell" value={v} onKeyDown={keys}
        onChange={(e) => { setV(e.target.value); onCommit(e.target.value); }} onBlur={() => onCancel()}>
        {meta.options.map((o) => (
          <option key={o.value} value={o.value}>{o.label}</option>
        ))}
      </select>
    );
  }
  return (
    <input
      ref={ref}
      className="sheet-editor"
      aria-label="Edit cell"
      type={meta.editor === "date" ? "date" : meta.editor === "number" ? "number" : "text"}
      value={v}
      onChange={(e) => setV(e.target.value)}
      onKeyDown={keys}
      onBlur={() => commit()}
    />
  );
}

// Spreadsheet view of a project's tasks: Excel-style column filters and
// sorting, inline editing, add row, bulk edit. Rows are the tasks that pass
// the board's search, sprint and filter panel; stories show their child
// tasks indented beneath them.
export default function SheetView({ projectId, rows: topLevel, childrenOf, columns: boardColumns, sprints, onUpdate, onBulkUpdate, onBulkDelete, onAddRow, onOpen }) {
  const prefs = useMemo(() => loadPrefs(projectId), [projectId]);
  const [sorting, setSorting] = useState(prefs.sorting || []);
  const [columnFilters, setColumnFilters] = useState([]);
  const [visibility, setVisibility] = useState(prefs.visibility || { created: false, updated: false, milestone: false });
  const [order, setOrder] = useState(prefs.order || []);
  const [sizing, setSizing] = useState(prefs.sizing || {});
  const [rowSelection, setRowSelection] = useState({});
  const [menu, setMenu] = useState(null); // column id
  const [colsOpen, setColsOpen] = useState(false);
  const [active, setActive] = useState(null); // { rowId, colId }
  const [editing, setEditing] = useState(null); // { rowId, colId }
  const [dragCol, setDragCol] = useState(null);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const tableRef = useRef(null);

  useEffect(() => {
    try {
      localStorage.setItem(prefsKey(projectId), JSON.stringify({ sorting, visibility, order, sizing }));
    } catch {
      // storage unavailable: preferences just won't persist
    }
  }, [projectId, sorting, visibility, order, sizing]);

  const data = useMemo(
    () => topLevel.map((t) => ({ ...t, subRows: (childrenOf[t.id] || []).map((c) => ({ ...c })) })),
    [topLevel, childrenOf]
  );

  const sprintName = (id) => (id ? sprints.find((s) => s.id === id)?.name || id : "Backlog");
  const statusLabel = (id) => boardColumns.find((c) => c.id === id)?.label || id;
  const customFields = useMemo(() => {
    const names = new Set();
    const walk = (t) => Object.values(t.fields || {}).forEach((f) => f?.name && names.add(f.name));
    data.forEach((t) => { walk(t); t.subRows.forEach(walk); });
    return [...names].sort();
  }, [data]);

  const columns = useMemo(() => {
    // `extra` may carry its own meta; merge it rather than letting it replace ours.
    const text = (id, header, accessor, { meta = {}, ...extra } = {}) => ({
      id, header, accessorFn: accessor, filterFn: "text", ...extra,
      meta: { filter: "text", editor: "text", ...meta },
    });
    const sel = (id, header, accessor, options, { meta = {}, ...extra } = {}) => ({
      id, header, accessorFn: accessor, filterFn: "values", ...extra,
      meta: { filter: "values", editor: "select", options, ...meta },
    });
    const progress = (list) => (list?.length ? `${list.filter((x) => x.done).length}/${list.length}` : "");
    const cols = [
      {
        id: "select", header: "", size: 34, enableSorting: false, enableHiding: false, enableColumnFilter: false,
        meta: { readonly: true },
        cell: ({ row }) => (
          <input type="checkbox" aria-label={`Select ${row.original.title}`} checked={row.getIsSelected()} onChange={row.getToggleSelectedHandler()} onClick={(e) => e.stopPropagation()} />
        ),
      },
      text("title", "Title", (t) => t.title, { size: 280, enableHiding: false }),
      sel("type", "Type", (t) => t.type || "task", TYPES.map((v) => ({ value: v, label: v }))),
      sel("status", "Status", (t) => statusLabel(t.status), boardColumns.map((c) => ({ value: c.id, label: c.label })), { meta: { valueOf: (t) => t.status } }),
      sel("priority", "Priority", (t) => t.priority || "", PRIORITIES.map((v) => ({ value: v, label: v || "none" }))),
      { id: "due", header: "Due", // undefined (not "") so blank dates sort last in both directions, as in Excel
      accessorFn: (t) => t.dueDate || undefined, filterFn: "dateRange", sortUndefined: "last", meta: { filter: "dateRange", editor: "date" }, size: 130 },
      sel("sprint", "Sprint", (t) => sprintName(t.sprintId), [{ value: "", label: "Backlog" }, ...sprints.map((s) => ({ value: s.id, label: s.name }))], { meta: { valueOf: (t) => t.sprintId || "" } }),
      text("assignee", "Assignees", (t) => t.assignee || ""),
      text("labels", "Labels", (t) => (t.labels || []).join(", "), { meta: { list: true } }),
      { id: "points", header: "Points", accessorFn: (t) => t.storyPoints || 0, filterFn: "numberRange", meta: { filter: "numberRange", editor: "number" }, size: 80 },
      text("milestone", "Milestone", (t) => t.milestone || ""),
      { id: "criteria", header: "Criteria", accessorFn: (t) => progress(t.acceptance), filterFn: "values", meta: { filter: "values", readonly: true }, size: 90 },
      { id: "subtasks", header: "Subtasks", accessorFn: (t) => progress(t.subtasks), filterFn: "values", meta: { filter: "values", readonly: true }, size: 90 },
      { id: "links", header: "Links", accessorFn: (t) => (t.links || []).length, filterFn: "numberRange", meta: { filter: "numberRange", readonly: true }, size: 70 },
      { id: "files", header: "Files", accessorFn: (t) => (t.attachments || []).length, filterFn: "numberRange", meta: { filter: "numberRange", readonly: true }, size: 70 },
      { id: "created", header: "Created", accessorFn: (t) => fmtTs(t.createdAt), filterFn: "dateRange", meta: { filter: "dateRange", readonly: true }, size: 110 },
      { id: "updated", header: "Updated", accessorFn: (t) => fmtTs(t.updatedAt), filterFn: "dateRange", meta: { filter: "dateRange", readonly: true }, size: 110 },
      ...customFields.map((name) => ({
        id: `field:${name}`, header: name, filterFn: "values", meta: { filter: "values", readonly: true },
        accessorFn: (t) => {
          const f = Object.values(t.fields || {}).find((x) => x?.name === name);
          return f ? f.display || f.text || f.date || f.optionKey || (f.number ?? "") || (f.users || []).join(", ") : "";
        },
      })),
    ];
    return cols;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [boardColumns, sprints, customFields]);

  const table = useReactTable({
    data,
    columns,
    state: { sorting, columnFilters, columnVisibility: visibility, columnOrder: order, columnSizing: sizing, rowSelection, expanded: true },
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setVisibility,
    onColumnOrderChange: setOrder,
    onColumnSizingChange: setSizing,
    onRowSelectionChange: setRowSelection,
    getSubRows: (r) => r.subRows,
    getRowId: (r) => r.id,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getExpandedRowModel: getExpandedRowModel(),
    filterFns,
    filterFromLeafRows: true,
    columnResizeMode: "onChange",
    enableRowSelection: true,
    enableSubRowSelection: false,
    defaultColumn: { size: 140, minSize: 60 },
  });

  const rows = table.getRowModel().rows;
  const visibleCols = table.getVisibleLeafColumns();
  const selected = table.getSelectedRowModel().flatRows.map((r) => r.original.id);

  // Turn an edited display value into a task patch.
  const commit = (task, colId, value) => {
    const patch = {};
    switch (colId) {
      case "title": if (!String(value).trim()) return; patch.title = String(value).trim(); break;
      case "type": patch.type = value; break;
      case "status": patch.status = value; break;
      case "priority": patch.priority = value; break;
      case "due": if (value && !isValidDate(value)) return; patch.dueDate = value; break;
      case "sprint": patch.sprintId = value; break;
      case "assignee": patch.assignee = String(value).split(",").map((s) => s.trim()).filter(Boolean).join(", "); break;
      case "labels": patch.labels = String(value).split(",").map((s) => s.trim()).filter(Boolean); break;
      case "points": patch.storyPoints = Math.max(0, parseInt(value, 10) || 0); break;
      case "milestone": patch.milestone = String(value).trim(); break;
      default: return;
    }
    onUpdate(task.id, patch);
  };

  const editable = (col) => !col.columnDef.meta?.readonly;
  const moveActive = (dr, dc) => {
    if (!active) return;
    const ri = rows.findIndex((r) => r.id === active.rowId);
    const ci = visibleCols.findIndex((c) => c.id === active.colId);
    const r = rows[Math.min(rows.length - 1, Math.max(0, ri + dr))];
    const c = visibleCols[Math.min(visibleCols.length - 1, Math.max(0, ci + dc))];
    if (r && c) setActive({ rowId: r.id, colId: c.id });
  };

  const onKeyDown = (e) => {
    if (editing || !active || menu) return;
    const moves = { ArrowUp: [-1, 0], ArrowDown: [1, 0], ArrowLeft: [0, -1], ArrowRight: [0, 1] };
    if (moves[e.key]) {
      e.preventDefault();
      moveActive(...moves[e.key]);
    } else if (e.key === "Tab") {
      e.preventDefault();
      moveActive(0, e.shiftKey ? -1 : 1);
    } else if (e.key === "Enter" || e.key === "F2") {
      const col = visibleCols.find((c) => c.id === active.colId);
      if (col && editable(col)) {
        e.preventDefault();
        setEditing(active);
      }
    } else if (e.key === "Escape") {
      setActive(null);
    }
  };

  const filterCount = columnFilters.length;
  const reorder = (from, to) => {
    const ids = order.length ? [...order] : table.getAllLeafColumns().map((c) => c.id);
    const a = ids.indexOf(from), b = ids.indexOf(to);
    if (a < 0 || b < 0 || a === b) return;
    ids.splice(b, 0, ids.splice(a, 1)[0]);
    setOrder(ids);
  };

  return (
    <div className="sheet" onKeyDown={onKeyDown}>
      <div className="sheet-toolbar">
        <button type="button" className="btn small" onClick={onAddRow}>+ Add task</button>
        <div className="sheet-cols-wrap">
          <button type="button" className="btn small" aria-expanded={colsOpen} onClick={() => setColsOpen((o) => !o)}>Columns</button>
          {colsOpen && (
            <div className="sheet-menu sheet-cols" role="dialog" aria-label="Show columns">
              {table.getAllLeafColumns().filter((c) => c.getCanHide()).map((c) => (
                <label className="sheet-menu-check" key={c.id}>
                  <input type="checkbox" checked={c.getIsVisible()} onChange={c.getToggleVisibilityHandler()} />
                  {c.columnDef.header}
                </label>
              ))}
              <button type="button" className="link-btn" onClick={() => { setOrder([]); setSizing({}); setVisibility({ created: false, updated: false, milestone: false }); }}>
                Reset layout
              </button>
            </div>
          )}
        </div>
        {filterCount > 0 && (
          <button type="button" className="link-btn" onClick={() => setColumnFilters([])}>
            Clear {filterCount} column filter{filterCount === 1 ? "" : "s"}
          </button>
        )}
        <span className="sheet-count">{rows.length} row{rows.length === 1 ? "" : "s"}</span>
        {selected.length > 0 && (
          <div className="sheet-bulk" role="group" aria-label="Bulk edit">
            <strong>{selected.length} selected</strong>
            <select aria-label="Set status for selected" value="" onChange={(e) => e.target.value && onBulkUpdate(selected, { status: e.target.value })}>
              <option value="">Status…</option>
              {boardColumns.map((c) => <option key={c.id} value={c.id}>{c.label}</option>)}
            </select>
            <select aria-label="Set priority for selected" value="" onChange={(e) => e.target.value !== "" && onBulkUpdate(selected, { priority: e.target.value === "none" ? "" : e.target.value })}>
              <option value="">Priority…</option>
              {["high", "medium", "low", "none"].map((p) => <option key={p} value={p}>{p}</option>)}
            </select>
            <select aria-label="Set sprint for selected" value="" onChange={(e) => e.target.value && onBulkUpdate(selected, { sprintId: e.target.value === "__backlog__" ? "" : e.target.value })}>
              <option value="">Sprint…</option>
              <option value="__backlog__">Backlog</option>
              {sprints.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
            </select>
            <button type="button" className="btn tiny danger" onClick={() => setConfirmDelete(true)}>Delete</button>
            <button type="button" className="link-btn" onClick={() => setRowSelection({})}>Clear</button>
          </div>
        )}
      </div>

      <div className="sheet-scroll" ref={tableRef}>
        <table className="sheet-table" style={{ width: table.getTotalSize() }} role="grid">
          <thead>
            {table.getHeaderGroups().map((hg) => (
              <tr key={hg.id}>
                {hg.headers.map((h) => {
                  const col = h.column;
                  const sorted = col.getIsSorted();
                  const filtered = col.getFilterValue() !== undefined;
                  return (
                    <th
                      key={h.id}
                      style={{ width: h.getSize() }}
                      className={`${filtered ? "filtered" : ""} ${dragCol === col.id ? "dragging" : ""}`}
                      draggable={col.id !== "select"}
                      onDragStart={() => setDragCol(col.id)}
                      onDragOver={(e) => e.preventDefault()}
                      onDrop={(e) => { e.preventDefault(); if (dragCol) reorder(dragCol, col.id); setDragCol(null); }}
                      onDragEnd={() => setDragCol(null)}
                      data-col={col.id}
                    >
                      {col.id === "select" ? (
                        <input type="checkbox" aria-label="Select all rows" checked={table.getIsAllRowsSelected()} onChange={table.getToggleAllRowsSelectedHandler()} />
                      ) : (
                        <>
                          <button type="button" className="sheet-th-label" onClick={col.getToggleSortingHandler()} title="Sort">
                            {flexRender(col.columnDef.header, h.getContext())}
                            {sorted === "asc" ? " ▲" : sorted === "desc" ? " ▼" : ""}
                          </button>
                          <button type="button" className="sheet-th-menu" aria-label={`${col.columnDef.header} menu`} onClick={() => setMenu(menu === col.id ? null : col.id)}>
                            {filtered ? "⏷" : "▾"}
                          </button>
                          {menu === col.id && (
                            <HeaderMenu column={col} rows={table.getPreFilteredRowModel().flatRows} onClose={() => setMenu(null)} />
                          )}
                          <span
                            className="sheet-resize"
                            onMouseDown={h.getResizeHandler()}
                            onClick={(e) => e.stopPropagation()}
                            role="separator"
                            aria-label={`Resize ${col.columnDef.header}`}
                          />
                        </>
                      )}
                    </th>
                  );
                })}
              </tr>
            ))}
          </thead>
          <tbody>
            {rows.map((row) => {
              const t = row.original;
              return (
                <tr key={row.id} className={`${row.depth ? "child" : ""} ${isPastDue(t) ? "overdue" : ""} ${row.getIsSelected() ? "selected" : ""}`} data-task={t.id}>
                  {row.getVisibleCells().map((cell) => {
                    const col = cell.column;
                    const meta = col.columnDef.meta || {};
                    const isActive = active && active.rowId === row.id && active.colId === col.id;
                    const isEditing = editing && editing.rowId === row.id && editing.colId === col.id;
                    const raw = meta.valueOf ? meta.valueOf(t) : cell.getValue();
                    let content;
                    if (col.id === "select") content = flexRender(col.columnDef.cell, cell.getContext());
                    else if (isEditing)
                      content = (
                        <CellEditor
                          meta={meta}
                          value={raw}
                          onCommit={(v) => { commit(t, col.id, v); setEditing(null); }}
                          onCancel={() => setEditing(null)}
                          onTab={(d) => moveActive(0, d)}
                        />
                      );
                    else if (col.id === "title")
                      content = (
                        <span className="sheet-title" style={{ paddingLeft: row.depth * 18 }}>
                          {row.depth > 0 && <span className="sheet-child-mark" aria-hidden>↳</span>}
                          <span className="sheet-title-text">{t.title}</span>
                          <button type="button" className="sheet-open" aria-label={`Open ${t.title}`} title="Open details" onClick={(e) => { e.stopPropagation(); onOpen(t); }}>
                            ↗
                          </button>
                        </span>
                      );
                    else if (col.id === "due")
                      content = t.dueDate ? <span className={isPastDue(t) ? "sheet-overdue" : ""}>{formatDue(t.dueDate)}</span> : "";
                    else content = String(cell.getValue() ?? "");
                    return (
                      <td
                        key={cell.id}
                        style={{ width: col.getSize() }}
                        className={`${isActive ? "active" : ""} ${meta.readonly ? "readonly" : ""}`}
                        data-col={col.id}
                        tabIndex={isActive ? 0 : -1}
                        onClick={() => col.id !== "select" && setActive({ rowId: row.id, colId: col.id })}
                        onDoubleClick={() => editable(col) && col.id !== "select" && setEditing({ rowId: row.id, colId: col.id })}
                      >
                        {content}
                      </td>
                    );
                  })}
                </tr>
              );
            })}
            {rows.length === 0 && (
              <tr>
                <td colSpan={visibleCols.length} className="sheet-empty">No tasks match.</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {confirmDelete && (
        <ConfirmDialog
          title={`Delete ${selected.length} task${selected.length === 1 ? "" : "s"}?`}
          body="Stories are deleted with their child tasks. This cannot be undone."
          confirmLabel="Delete"
          danger
          onConfirm={() => { onBulkDelete(selected); setRowSelection({}); setConfirmDelete(false); }}
          onCancel={() => setConfirmDelete(false)}
        />
      )}
    </div>
  );
}
