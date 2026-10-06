import { useEffect, useState } from "react";
import {
  CreateBoardColumn,
  GitHubAddStatusOption,
  GitHubGetProject,
  GitHubUpdateSync,
} from "../../api";
import { DEFAULT_COLUMNS } from "../kanban/columns";
import { OPTION_COLORS } from "./fieldTypes";

const DEFAULT_CREATE = {
  backlog: { name: "Backlog", color: "GRAY" },
  todo: { name: "To Do", color: "BLUE" },
  inprogress: { name: "In Progress", color: "YELLOW" },
  testing: { name: "Testing", color: "PURPLE" },
  done: { name: "Done", color: "GREEN" },
};

// Map each Kanban column to an option on the board's Status field.
// JumpStart guesses these when a board is linked; this is where a user
// whose board says "Shipped" instead of "Done" corrects it — or creates
// a missing Status option (Testing is the common gap) without leaving
// JumpStart. New columns can be added here too when a Status has no
// local column to land in.
export default function StatusMapEditor({
  projectId,
  sync,
  columns = DEFAULT_COLUMNS,
  onUpdated,
  onColumnsChange,
  onError,
}) {
  const [field, setField] = useState(null);
  const [map, setMap] = useState(sync.statusMap || {});
  const [cols, setCols] = useState(columns);
  const [saving, setSaving] = useState(false);
  const [creating, setCreating] = useState(null); // column id being created, or "custom"
  const [open, setOpen] = useState(false);
  const [customName, setCustomName] = useState("");
  const [customColumn, setCustomColumn] = useState("");
  const [newColumnLabel, setNewColumnLabel] = useState("");

  const loadField = () => {
    if (!sync.projectId) return Promise.resolve();
    return GitHubGetProject(sync.projectId)
      .then((b) => {
        const f = (b.fields || []).find((x) => x.id === sync.statusFieldId)
          || (b.fields || []).find(
            (x) => x.dataType === "SINGLE_SELECT" && /^status$/i.test(x.name || "")
          );
        setField(f || null);
      })
      .catch((e) => onError && onError(String(e)));
  };

  useEffect(() => {
    setMap(sync.statusMap || {});
  }, [sync.statusMap]);

  useEffect(() => {
    setCols(columns);
  }, [columns]);

  useEffect(() => {
    if (!open) return;
    loadField();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, sync.projectId, sync.statusFieldId]);

  const save = async (next) => {
    setMap(next);
    setSaving(true);
    try {
      const cfg = { ...sync, statusMap: next };
      await GitHubUpdateSync(projectId, cfg);
      onUpdated && onUpdated(cfg);
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setSaving(false);
    }
  };

  const createForColumn = async (colId) => {
    const col = cols.find((c) => c.id === colId);
    const preset = DEFAULT_CREATE[colId] || { name: col?.label || colId, color: "GRAY" };
    setCreating(colId);
    try {
      const cfg = await GitHubAddStatusOption(projectId, preset.name, preset.color, colId);
      setMap(cfg.statusMap || {});
      onUpdated && onUpdated(cfg);
      await loadField();
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setCreating(null);
    }
  };

  const createCustom = async () => {
    const name = customName.trim();
    if (!name) return;
    setCreating("custom");
    try {
      if (customColumn === "__new__") {
        const label = newColumnLabel.trim() || name;
        const result = await CreateBoardColumn(projectId, label, "", name, "");
        if (result?.columns) {
          setCols(result.columns.map((c, i) => ({
            id: c.id,
            label: c.label,
            order: c.order ?? i,
          })));
          onColumnsChange && onColumnsChange(result.columns);
        }
        if (result?.sync) {
          setMap(result.sync.statusMap || {});
          onUpdated && onUpdated(result.sync);
        }
      } else {
        const cfg = await GitHubAddStatusOption(projectId, name, "", customColumn || "");
        setMap(cfg.statusMap || {});
        onUpdated && onUpdated(cfg);
      }
      setCustomName("");
      setCustomColumn("");
      setNewColumnLabel("");
      await loadField();
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setCreating(null);
    }
  };

  if (!open) {
    return (
      <button className="btn small ghost" onClick={() => setOpen(true)}>
        Edit column mapping
      </button>
    );
  }

  const busy = saving || !!creating;

  return (
    <div className="gh-statusmap">
      <div className="gh-panel-head">
        <span className="gh-panel-title">Column mapping</span>
        {busy && <span className="gh-muted">{creating ? "Creating…" : "Saving…"}</span>}
        <div className="spacer" />
        <button className="btn small ghost" onClick={() => setOpen(false)}>
          Done
        </button>
      </div>

      {!field && <span className="gh-muted">Loading the Status field…</span>}

      {field &&
        cols.map((col) => (
          <div className="gh-map-row" key={col.id}>
            <span className="gh-map-col">{col.label}</span>
            <select
              value={map[col.id] || ""}
              disabled={busy}
              onChange={(e) => save({ ...map, [col.id]: e.target.value })}
            >
              <option value="">Leave unset</option>
              {(field.options || []).map((o) => (
                <option key={o.id} value={o.id}>
                  {o.name}
                </option>
              ))}
            </select>
            <span
              className="gh-dot"
              style={{
                "--gh-dot":
                  OPTION_COLORS[
                    (field.options || []).find((o) => o.id === map[col.id])?.color
                  ] || "transparent",
              }}
            />
            {!map[col.id] && (
              <button
                type="button"
                className="btn small ghost gh-map-create"
                disabled={busy}
                title={`Create "${DEFAULT_CREATE[col.id]?.name || col.label}" on the GitHub board`}
                onClick={() => createForColumn(col.id)}
              >
                {creating === col.id ? "…" : "Create"}
              </button>
            )}
          </div>
        ))}

      {field && (
        <div className="gh-map-add">
          <span className="gh-map-add-label">New status on board</span>
          <div className="gh-map-add-row">
            <input
              value={customName}
              disabled={busy}
              placeholder="Status name…"
              onChange={(e) => setCustomName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  createCustom();
                }
              }}
            />
            <select
              value={customColumn}
              disabled={busy}
              onChange={(e) => setCustomColumn(e.target.value)}
              title="Map to JumpStart column"
            >
              <option value="">No column</option>
              {cols.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.label}
                </option>
              ))}
              <option value="__new__">New column…</option>
            </select>
            <button
              type="button"
              className="btn small"
              disabled={
                busy ||
                !customName.trim() ||
                (customColumn === "__new__" && !newColumnLabel.trim() && !customName.trim())
              }
              onClick={createCustom}
            >
              {creating === "custom" ? "…" : "Add"}
            </button>
          </div>
          {customColumn === "__new__" && (
            <input
              className="gh-map-newcol"
              value={newColumnLabel}
              disabled={busy}
              placeholder="New column name (defaults to status name)"
              onChange={(e) => setNewColumnLabel(e.target.value)}
            />
          )}
        </div>
      )}
    </div>
  );
}
