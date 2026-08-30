import { useEffect, useState } from "react";
import { GitHubGetProject, GitHubUpdateSync } from "../../api";
import { COLUMNS } from "../kanban/columns";
import { OPTION_COLORS } from "./fieldTypes";

// Map each Kanban column to an option on the board's Status field.
// JumpStart guesses these when a board is linked; this is where a user
// whose board says "Shipped" instead of "Done" corrects it.
export default function StatusMapEditor({ projectId, sync, onError }) {
  const [field, setField] = useState(null);
  const [map, setMap] = useState(sync.statusMap || {});
  const [saving, setSaving] = useState(false);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (!open || !sync.projectId) return;
    GitHubGetProject(sync.projectId)
      .then((b) => {
        const f = (b.fields || []).find((x) => x.id === sync.statusFieldId);
        setField(f || null);
      })
      .catch((e) => onError && onError(String(e)));
  }, [open, sync.projectId, sync.statusFieldId, onError]);

  const save = async (next) => {
    setMap(next);
    setSaving(true);
    try {
      await GitHubUpdateSync(projectId, { ...sync, statusMap: next });
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setSaving(false);
    }
  };

  if (!open) {
    return (
      <button className="btn small ghost" onClick={() => setOpen(true)}>
        Edit column mapping
      </button>
    );
  }

  return (
    <div className="gh-statusmap">
      <div className="gh-panel-head">
        <span className="gh-panel-title">Column mapping</span>
        {saving && <span className="gh-muted">Saving…</span>}
        <div className="spacer" />
        <button className="btn small ghost" onClick={() => setOpen(false)}>
          Done
        </button>
      </div>

      {!field && <span className="gh-muted">Loading the Status field…</span>}

      {field &&
        COLUMNS.map((col) => (
          <div className="gh-map-row" key={col.id}>
            <span className="gh-map-col">{col.label}</span>
            <select
              value={map[col.id] || ""}
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
          </div>
        ))}
    </div>
  );
}
