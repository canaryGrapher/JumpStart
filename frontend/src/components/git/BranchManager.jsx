import { useEffect, useState } from "react";
import { GitListBranches, GitCheckout, GitCreateBranch, GitDeleteBranch } from "../../api";
import SearchableSelect from "../SearchableSelect";
import ConfirmInline from "./ConfirmInline";

export default function BranchManager({ projectRoot, current, refreshKey, busy, onError, onInfo, onChanged }) {
  const [branches, setBranches] = useState([]);
  const [switchTo, setSwitchTo] = useState("");
  const [newName, setNewName] = useState("");
  const [deleteTarget, setDeleteTarget] = useState("");
  const [confirm, setConfirm] = useState(null); // "switch" | "create" | "delete"
  const [working, setWorking] = useState(false);

  const locals = branches.filter((b) => !b.remote);
  const switchable = locals.filter((b) => !b.current).map((b) => b.name);
  const deletable = switchable;

  useEffect(() => {
    GitListBranches(projectRoot)
      .then((b) => setBranches(b || []))
      .catch((e) => onError && onError(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectRoot, refreshKey]);

  const act = async (fn, msg) => {
    setWorking(true);
    try {
      await fn();
      onInfo && onInfo(msg);
      setConfirm(null);
      onChanged && onChanged();
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setWorking(false);
    }
  };

  const doSwitch = () =>
    act(() => GitCheckout(projectRoot, switchTo), `Switched to ${switchTo}`).then(() => setSwitchTo(""));
  const doCreate = () =>
    act(() => GitCreateBranch(projectRoot, newName.trim(), true), `Created ${newName.trim()}`).then(() =>
      setNewName("")
    );
  const doDelete = () =>
    act(() => GitDeleteBranch(projectRoot, deleteTarget, false), `Deleted ${deleteTarget}`).then(() =>
      setDeleteTarget("")
    );

  const disabled = busy || working;

  return (
    <div className="git-card git-branch-grid">
      {/* Switch */}
      <div className="git-branch-cell">
        <label className="git-cell-label">Switch</label>
        <div className="git-cell-row">
          <SearchableSelect
            value={switchTo}
            options={switchable}
            onChange={setSwitchTo}
            placeholder="Select branch…"
            searchPlaceholder="Search branches…"
            disabled={disabled}
          />
          <button className="btn small" disabled={disabled || !switchTo} onClick={() => setConfirm("switch")}>
            Switch
          </button>
        </div>
        {confirm === "switch" && (
          <ConfirmInline
            message={`Switch to "${switchTo}"?`}
            confirmLabel="Switch"
            busy={working}
            onConfirm={doSwitch}
            onCancel={() => setConfirm(null)}
          />
        )}
      </div>

      {/* Create */}
      <div className="git-branch-cell">
        <label className="git-cell-label">New branch</label>
        <div className="git-cell-row">
          <input
            className="git-cell-input"
            placeholder="feature/my-branch"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            disabled={disabled}
          />
          <button className="btn small" disabled={disabled || !newName.trim()} onClick={() => setConfirm("create")}>
            Create
          </button>
        </div>
        {confirm === "create" && (
          <ConfirmInline
            message={`Create "${newName.trim()}" from ${current || "HEAD"} and switch to it?`}
            confirmLabel="Create"
            busy={working}
            onConfirm={doCreate}
            onCancel={() => setConfirm(null)}
          />
        )}
      </div>

      {/* Delete */}
      <div className="git-branch-cell">
        <label className="git-cell-label">Delete</label>
        <div className="git-cell-row">
          <SearchableSelect
            value={deleteTarget}
            options={deletable}
            onChange={setDeleteTarget}
            placeholder="Select branch…"
            searchPlaceholder="Search branches…"
            disabled={disabled}
          />
          <button
            className="btn small danger"
            disabled={disabled || !deleteTarget}
            onClick={() => setConfirm("delete")}
          >
            Delete
          </button>
        </div>
        {confirm === "delete" && (
          <ConfirmInline
            message={`Delete "${deleteTarget}"? This cannot be undone.`}
            confirmLabel="Delete"
            danger
            busy={working}
            onConfirm={doDelete}
            onCancel={() => setConfirm(null)}
          />
        )}
      </div>
    </div>
  );
}
