import { useEffect, useState } from "react";
import { GetDependencies, InstallDeps, EventsOn } from "../api";
import LogPanel from "./LogPanel";

const ICONS = { ok: "✓", missing: "✗", unknown: "?" };

// Lists a process folder's dependencies with installed state,
// and can run the package manager's install command with live logs.
export default function DepsPanel({ projectId, proc, onError }) {
  const [info, setInfo] = useState(null);
  const [installing, setInstalling] = useState(false);
  const [depsProcId, setDepsProcId] = useState(null);

  const refresh = () =>
    GetDependencies(proc.dir)
      .then(setInfo)
      .catch((e) => onError(String(e)));

  useEffect(() => {
    refresh();
  }, [proc.dir]);

  useEffect(() => {
    if (!depsProcId) return;
    return EventsOn(`exit:${depsProcId}`, (code) => {
      setInstalling(false);
      if (code !== 0) onError(`Install failed (exit ${code})`);
      refresh();
    });
  }, [depsProcId]);

  const install = async (e) => {
    e.stopPropagation();
    try {
      setInstalling(true);
      setDepsProcId(await InstallDeps(projectId, proc.id));
    } catch (err) {
      setInstalling(false);
      onError(String(err));
    }
  };

  if (!info) return <div className="deps-panel">Loading dependencies...</div>;
  if (info.manager === "none")
    return <div className="deps-panel">No package manager detected in this folder.</div>;

  return (
    <div className="deps-panel" onClick={(e) => e.stopPropagation()}>
      <div className="deps-head">
        <span>
          <strong>{info.manager}</strong> · {info.manifestFile} ·{" "}
          {info.installed}/{info.dependencies.length} installed
          {info.missing > 0 && <span className="dep-missing"> · {info.missing} missing</span>}
          {info.unknown > 0 && ` · ${info.unknown} unknown`}
        </span>
        <button className="btn small primary" onClick={install} disabled={installing}>
          {installing ? "Installing..." : `Install (${info.installCommand})`}
        </button>
      </div>
      <div className="deps-list">
        {info.dependencies.map((d) => (
          <div className="dep-row" key={d.kind + d.name}>
            <span className={`dep-status ${d.status}`}>{ICONS[d.status] || "?"}</span>
            <span className="dep-name">{d.name}</span>
            <span className="dep-version">{d.version}</span>
            {d.kind !== "dep" && <span className="dep-kind">{d.kind}</span>}
          </div>
        ))}
        {info.dependencies.length === 0 && <div className="dep-row">No dependencies declared.</div>}
      </div>
      {depsProcId && <LogPanel procId={depsProcId} />}
    </div>
  );
}
