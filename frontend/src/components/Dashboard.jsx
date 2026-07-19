import { useEffect, useState } from "react";
import { GetImportPath } from "../api";
import ImportConfigModal from "./ImportConfigModal";
import { PortsTable, usePortMap } from "./PortsView";

const fmtAgo = (ms) => {
  if (!ms) return "never";
  const m = Math.floor((Date.now() - ms) / 60000);
  if (m < 1) return "just now";
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
};

export default function Dashboard({ projects, usage, onOpen, onReload, onError, onInfo }) {
  const [confPath, setConfPath] = useState("");
  const [showImport, setShowImport] = useState(false);
  const portEntries = usePortMap(onError);

  useEffect(() => {
    GetImportPath().then(setConfPath).catch(() => {});
  }, []);

  const sys = usage.system || {};
  const runningCount = Object.keys(usage.procs || {}).length;
  const cpu = Math.round(sys.cpu || 0);
  const memPct = sys.totalMemMB ? Math.round((sys.usedMemMB / sys.totalMemMB) * 100) : 0;

  const recent = [...projects]
    .filter((p) => p.lastUsedAt)
    .sort((a, b) => b.lastUsedAt - a.lastUsedAt)
    .slice(0, 5);
  const mostUsed = [...projects]
    .filter((p) => p.useCount)
    .sort((a, b) => b.useCount - a.useCount)
    .slice(0, 5);

  const taskStats = projects.reduce(
    (acc, p) => {
      const t = p.tasks || [];
      acc.done += t.filter((x) => x.done).length;
      acc.total += t.length;
      return acc;
    },
    { done: 0, total: 0 }
  );

  return (
    <>
      <div className="tiles">
        <div className="tile">
          <div className="tile-label">Projects</div>
          <div className="tile-num">
            <span className="tile-value">{projects.length}</span>
            <span className="delta-pill">{runningCount} running</span>
          </div>
          <div className="tile-sub">subprocesses currently managed</div>
        </div>
        <div className="tile">
          <div className="tile-label">System CPU</div>
          <div className="tile-num">
            <span className="tile-value">{cpu}%</span>
            <span className="delta-pill neutral">{sys.numCpu || 0} cores</span>
          </div>
          <div className="meter">
            <div className={cpu > 80 ? "hot" : ""} style={{ width: `${cpu}%` }} />
          </div>
        </div>
        <div className="tile">
          <div className="tile-label">System memory</div>
          <div className="tile-num">
            <span className="tile-value">{memPct}%</span>
            <span className="delta-pill neutral">
              {Math.round((sys.usedMemMB || 0) / 1024)}/{Math.round((sys.totalMemMB || 0) / 1024)} GB
            </span>
          </div>
          <div className="meter">
            <div className={memPct > 85 ? "hot" : ""} style={{ width: `${memPct}%` }} />
          </div>
        </div>
        <div className="tile">
          <div className="tile-label">Features built</div>
          <div className="tile-num">
            <span className="tile-value">
              {taskStats.done}/{taskStats.total}
            </span>
            {taskStats.total > 0 && (
              <span className="delta-pill">
                {Math.round((taskStats.done / taskStats.total) * 100)}%
              </span>
            )}
          </div>
          <div className="tile-sub">across all task trackers</div>
          {taskStats.total > 0 && (
            <div className="meter">
              <div style={{ width: `${(taskStats.done / taskStats.total) * 100}%` }} />
            </div>
          )}
        </div>
      </div>

      <div className="dash-grid">
        <div className="panel">
          <h3>Recent projects</h3>
          <div className="sub">Pick up where you left off</div>
          {recent.map((p) => (
            <div className="quick-row" key={p.id} onClick={() => onOpen(p.id)}>
              <span className="avatar">{(p.name || "?").charAt(0).toUpperCase()}</span>
              <span className="q-text">
                <span className="q-name">{p.name}</span>
                <span className="q-sub">{(p.processes || []).length} subprocesses</span>
                {p.description && <span className="q-desc">{p.description}</span>}
              </span>
              <span className="q-meta">{fmtAgo(p.lastUsedAt)}</span>
            </div>
          ))}
          {recent.length === 0 && <div className="sub">Nothing started yet.</div>}
        </div>

        <div className="panel">
          <h3>Most used</h3>
          <div className="sub">Your go-to projects</div>
          {mostUsed.map((p) => (
            <div className="quick-row" key={p.id} onClick={() => onOpen(p.id)}>
              <span className="avatar">{(p.name || "?").charAt(0).toUpperCase()}</span>
              <span className="q-text">
                <span className="q-name">{p.name}</span>
                <span className="q-sub">{(p.processes || []).length} subprocesses</span>
                {p.description && <span className="q-desc">{p.description}</span>}
              </span>
              <span className="q-meta">{p.useCount} starts</span>
            </div>
          ))}
          {mostUsed.length === 0 && <div className="sub">No usage recorded yet.</div>}
        </div>

        <div className="panel wide">
          <h3>Port usage & mapping</h3>
          <div className="sub">Live view of every port used by managed subprocesses</div>
          <PortsTable entries={portEntries} />
        </div>

        <div className="panel wide">
          <div className="panel-head-row">
            <h3>Config import</h3>
            <button className="btn primary" onClick={() => setShowImport(true)}>
              Import config
            </button>
          </div>
          <div className="sub">
            Add projects with the block builder, by pasting JSON, or from a file.
          </div>
          <div className="conf-path">{confPath || "..."}</div>
          <span className="hint">
            The importer lets you build blocks, paste and edit JSON, or load a file. A copy is
            saved to the path above. "Copy prompt" inside gives an AI agent instructions to
            generate the JSON for you.
          </span>
        </div>
      </div>

      {showImport && (
        <ImportConfigModal
          onClose={() => setShowImport(false)}
          onInfo={onInfo}
          onError={onError}
          onReload={onReload}
        />
      )}
    </>
  );
}
