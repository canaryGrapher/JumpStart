import { useEffect, useState } from "react";
import { GetDependencies, InstallDeps, EventsOn } from "../api";

const ICONS = { ok: "✓", missing: "✗", unknown: "?" };

// Lists a process folder's dependencies with installed state,
// and can run the package manager's install command. While that
// command runs we show a live progress bar (polled from the
// filesystem) instead of a raw log terminal — then swap to a
// short-lived success status once it exits cleanly.
export default function DepsPanel({ projectId, proc, onError }) {
  const [info, setInfo] = useState(null);
  const [installing, setInstalling] = useState(false);
  const [depsProcId, setDepsProcId] = useState(null);
  const [installResult, setInstallResult] = useState(null); // "ok" | null

  const refresh = () =>
    GetDependencies(proc.dir)
      .then(setInfo)
      .catch((e) => onError(String(e)));

  useEffect(() => {
    refresh();
  }, [proc.dir]);

  // Poll install state while the command is running so the bar can
  // advance as packages land in the cache / node_modules.
  useEffect(() => {
    if (!installing) return;
    refresh();
    const tick = setInterval(refresh, 1000);
    return () => clearInterval(tick);
  }, [installing, proc.dir]);

  useEffect(() => {
    if (!depsProcId) return;
    return EventsOn(`exit:${depsProcId}`, (code) => {
      setInstalling(false);
      setDepsProcId(null);
      if (code !== 0) {
        setInstallResult(null);
        onError(`Install failed (exit ${code})`);
      } else {
        setInstallResult("ok");
      }
      refresh();
    });
  }, [depsProcId]);

  // Success strip is temporary — clear it once the user has had a
  // chance to notice, or as soon as they kick off another install.
  useEffect(() => {
    if (installResult !== "ok") return;
    const t = setTimeout(() => setInstallResult(null), 4000);
    return () => clearTimeout(t);
  }, [installResult]);

  const install = async (e) => {
    e.stopPropagation();
    try {
      setInstallResult(null);
      setInstalling(true);
      setDepsProcId(await InstallDeps(projectId, proc.id));
    } catch (err) {
      setInstalling(false);
      onError(String(err));
    }
  };

  if (!info) return <div className="deps-panel">Loading dependencies…</div>;
  if (info.manager === "none")
    return <div className="deps-panel">No package manager detected in this folder.</div>;

  const total = info.dependencies.length;
  // Prefer a counted bar when we can tell installed vs missing. If
  // everything is already ok (re-install) or still unknown, fall back
  // to an indeterminate pulse so the bar still communicates activity.
  const measurable = total > 0 && info.unknown < total && info.installed < total;
  const pct = measurable ? Math.round((info.installed / total) * 100) : 0;

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
          {installing ? "Installing…" : `Install (${info.installCommand})`}
        </button>
      </div>

      {installing && (
        <div className="deps-install-progress">
          <span className="deps-install-label">
            {measurable
              ? `Installing ${info.installed}/${total}…`
              : "Installing dependencies…"}
          </span>
          <div className={`meter ${measurable ? "" : "indeterminate"}`}>
            <div style={measurable ? { width: `${pct}%` } : undefined} />
          </div>
          {measurable && <span className="deps-install-pct">{pct}%</span>}
        </div>
      )}

      {!installing && installResult === "ok" && (
        <div className="deps-install-success" role="status">
          <span className="deps-install-success-icon">✓</span>
          Dependencies installed
        </div>
      )}

      {!installing && (
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
      )}
    </div>
  );
}
