import { useCallback, useEffect, useState } from "react";
import { ListSavedFilters, SaveFilter, DeleteSavedFilter, RestoreDefaultFilters } from "../../api";
import { filtersFromQuery, queryFromFilters, sameFilters, activeFilterCount } from "../../dueDates";

// Loads app-wide and project saved filters; shared by the quick picker and
// the manager so both stay in step.
export function useSavedFilters(projectId, onError) {
  const [lists, setLists] = useState({ appWide: [], project: [] });
  const reload = useCallback(async () => {
    try {
      const l = await ListSavedFilters(projectId || "");
      setLists({ appWide: l?.appWide || [], project: l?.project || [] });
    } catch (e) {
      onError && onError(String(e));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);
  useEffect(() => {
    reload();
  }, [reload]);
  return { ...lists, reload };
}

const all = (lists) => [
  ...lists.project.map((f) => ({ ...f, scope: "project" })),
  ...lists.appWide.map((f) => ({ ...f, scope: "app" })),
];

// Compact dropdown next to the Filters button: apply a saved filter in one click.
export function SavedFilterPicker({ saved, filters, onApply }) {
  const list = all(saved);
  const active = list.find((f) => activeFilterCount(filters) > 0 && sameFilters(filters, filtersFromQuery(f.query)));
  return (
    <select
      className="saved-filter-picker"
      aria-label="Saved filter"
      value={active ? `${active.scope}:${active.id}` : ""}
      onChange={(e) => {
        const f = list.find((x) => `${x.scope}:${x.id}` === e.target.value);
        onApply(f ? filtersFromQuery(f.query) : null);
      }}
    >
      <option value="">{active ? "Clear saved filter" : "Saved filters…"}</option>
      {saved.project.length > 0 && (
        <optgroup label="This project">
          {saved.project.map((f) => (
            <option key={f.id} value={`project:${f.id}`}>
              {f.name}
            </option>
          ))}
        </optgroup>
      )}
      <optgroup label="All projects">
        {saved.appWide.map((f) => (
          <option key={f.id} value={`app:${f.id}`}>
            {f.name}
          </option>
        ))}
      </optgroup>
    </select>
  );
}

// Inside the filter panel: save the current filters, rename, delete, restore defaults.
export function SavedFilterManager({ projectId, saved, filters, onApply, onError }) {
  const [name, setName] = useState("");
  const [scope, setScope] = useState("app");
  const [editing, setEditing] = useState(null); // `${scope}:${id}` being renamed
  const [rename, setRename] = useState("");
  const list = all(saved);
  const fail = (e) => onError && onError(String(e));

  const save = async () => {
    try {
      await SaveFilter(scope, projectId, { id: "", name: name.trim(), query: queryFromFilters(filters) });
      setName("");
      await saved.reload();
    } catch (e) {
      fail(e);
    }
  };
  const doRename = async (f) => {
    try {
      await SaveFilter(f.scope, projectId, { id: f.id, name: rename.trim(), query: f.query, builtin: f.builtin });
      setEditing(null);
      await saved.reload();
    } catch (e) {
      fail(e);
    }
  };
  const remove = async (f) => {
    try {
      await DeleteSavedFilter(f.scope, projectId, f.id);
      await saved.reload();
    } catch (e) {
      fail(e);
    }
  };
  const restore = async () => {
    try {
      await RestoreDefaultFilters();
      await saved.reload();
    } catch (e) {
      fail(e);
    }
  };

  return (
    <div className="kb-filter-group saved-filters">
      <span className="kb-filter-label">Saved filters</span>
      <div className="saved-filter-list">
        {list.map((f) => {
          const key = `${f.scope}:${f.id}`;
          const on = activeFilterCount(filters) > 0 && sameFilters(filters, filtersFromQuery(f.query));
          return editing === key ? (
            <span className="saved-filter-edit" key={key}>
              <input
                aria-label={`Rename ${f.name}`}
                value={rename}
                autoFocus
                onChange={(e) => setRename(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") doRename(f);
                  if (e.key === "Escape") setEditing(null);
                }}
              />
              <button type="button" className="btn tiny" onClick={() => doRename(f)} disabled={!rename.trim()}>
                Save
              </button>
            </span>
          ) : (
            <span className={`saved-filter ${on ? "on" : ""}`} key={key}>
              <button type="button" className="kb-chip" onClick={() => onApply(filtersFromQuery(f.query))} title={f.scope === "project" ? "This project only" : "All projects"}>
                {f.name}
                {f.scope === "project" && <span className="saved-filter-scope">project</span>}
              </button>
              <button type="button" className="link-btn" aria-label={`Rename ${f.name}`} onClick={() => { setEditing(key); setRename(f.name); }}>
                ✎
              </button>
              <button type="button" className="link-btn danger" aria-label={`Delete ${f.name}`} onClick={() => remove(f)}>
                ✕
              </button>
            </span>
          );
        })}
      </div>
      <div className="saved-filter-save">
        <input
          placeholder="Name these filters…"
          aria-label="Saved filter name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && name.trim() && activeFilterCount(filters) && save()}
        />
        <select aria-label="Save filter for" value={scope} onChange={(e) => setScope(e.target.value)}>
          <option value="app">All projects</option>
          <option value="project">This project</option>
        </select>
        <button type="button" className="btn small" onClick={save} disabled={!name.trim() || !activeFilterCount(filters)}>
          Save filter
        </button>
        <button type="button" className="link-btn" onClick={restore}>
          Restore defaults
        </button>
      </div>
    </div>
  );
}
