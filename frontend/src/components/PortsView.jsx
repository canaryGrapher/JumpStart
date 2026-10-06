import { useEffect, useMemo, useState } from "react";
import { GetPortMap, BrowserOpenURL } from "../api";
import { captureOnce } from "../analytics";
import Icon, { ICONS } from "./Icon";

export function conflictCount(entries) {
  const byPort = new Map();
  for (const e of entries) {
    byPort.set(e.port, (byPort.get(e.port) || 0) + 1);
  }
  let n = 0;
  for (const count of byPort.values()) {
    if (count >= 2) n++;
  }
  return n;
}

export function portStats(entries) {
  const list = entries || [];
  const ports = new Set(list.map((e) => e.port));
  const projects = new Set(list.map((e) => e.projectName).filter(Boolean));
  return {
    total: list.length,
    unique: ports.size,
    projects: projects.size,
    conflicts: conflictCount(list),
  };
}

function conflictedPorts(entries) {
  const byPort = new Map();
  for (const e of entries) {
    byPort.set(e.port, (byPort.get(e.port) || 0) + 1);
  }
  const bad = new Set();
  for (const [port, count] of byPort) {
    if (count >= 2) bad.add(port);
  }
  return bad;
}

// Shared table markup so the Dashboard's ports card and the standalone
// Ports view render identically without duplicating markup.
export function PortsTable({ entries, compact = false }) {
  const conflicts = useMemo(() => conflictedPorts(entries || []), [entries]);

  if (!entries || entries.length === 0) {
    return (
      <div className="ports-table-empty">
        <Icon d={ICONS.ports} />
        <p>Start a subprocess and its listening ports will show up here.</p>
      </div>
    );
  }

  return (
    <div className={`ports-table-wrap ${compact ? "compact" : ""}`}>
      <table className="table ports-table">
        <thead>
          <tr>
            <th>Port</th>
            <th>Project</th>
            <th>Subprocess</th>
            <th>PID</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {entries.map((e) => {
            const clash = conflicts.has(e.port);
            return (
              <tr key={`${e.procId}-${e.port}`} className={clash ? "port-conflict" : ""}>
                <td>
                  <button
                    type="button"
                    className={`port-badge ${clash ? "conflict" : ""}`}
                    onClick={() => BrowserOpenURL(`http://localhost:${e.port}`)}
                    title={`Open http://localhost:${e.port}`}
                  >
                    :{e.port}
                  </button>
                </td>
                <td>
                  <span className="ports-proj">{e.projectName}</span>
                </td>
                <td>{e.procName}</td>
                <td className="pid">{e.pid}</td>
                <td>
                  {clash ? (
                    <span className="ports-status conflict">Conflict</span>
                  ) : (
                    <span className="ports-status ok">Listening</span>
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

export function usePortMap(onError, intervalMs = 3000) {
  const [entries, setEntries] = useState([]);

  useEffect(() => {
    const poll = () =>
      GetPortMap()
        .then((e) => setEntries(e || []))
        .catch((e) => onError && onError(String(e)));
    poll();
    const t = setInterval(poll, intervalMs);
    return () => clearInterval(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return entries;
}

function PortsEmpty() {
  return (
    <div className="ports-empty">
      <div className="ports-empty-art" aria-hidden>
        <div className="ports-empty-ring r1" />
        <div className="ports-empty-ring r2" />
        <div className="ports-empty-ring r3" />
        <div className="ports-empty-hub">
          <Icon d={ICONS.ports} />
        </div>
        <span className="ports-empty-orb o1">:3000</span>
        <span className="ports-empty-orb o2">:5173</span>
        <span className="ports-empty-orb o3">:8080</span>
      </div>
      <h2>No ports in use yet</h2>
      <p>
        When a managed subprocess starts listening, JumpStart maps the port here so you can
        open it, spot collisions, and see which project owns it.
      </p>
      <ol className="ports-empty-steps">
        <li>
          <span className="step-num">1</span>
          <div>
            <strong>Open a project</strong>
            <span>Pick one from the sidebar or Dashboard.</span>
          </div>
        </li>
        <li>
          <span className="step-num">2</span>
          <div>
            <strong>Start a subprocess</strong>
            <span>Hit play on a process card — ports are detected automatically.</span>
          </div>
        </li>
        <li>
          <span className="step-num">3</span>
          <div>
            <strong>Watch this board light up</strong>
            <span>Click a badge to open localhost, or fix conflicts when two apps share a port.</span>
          </div>
        </li>
      </ol>
    </div>
  );
}

export default function PortsView({ onError }) {
  const entries = usePortMap(onError);
  const stats = useMemo(() => portStats(entries), [entries]);

  const byProject = useMemo(() => {
    const m = new Map();
    for (const e of entries) {
      const key = e.projectName || "Unknown";
      if (!m.has(key)) m.set(key, []);
      m.get(key).push(e);
    }
    return [...m.entries()].sort((a, b) => b[1].length - a[1].length);
  }, [entries]);

  // Once per session on entering Ports — use the first snapshot, not the poll.
  useEffect(() => {
    let cancelled = false;
    GetPortMap()
      .then((e) => {
        if (cancelled) return;
        const list = e || [];
        captureOnce("ports_viewed", "ports_viewed", {
          port_count: list.length,
          conflict_count: conflictCount(list),
        });
      })
      .catch(() => {
        if (cancelled) return;
        captureOnce("ports_viewed", "ports_viewed", {
          port_count: 0,
          conflict_count: 0,
        });
      });
    return () => {
      cancelled = true;
    };
  }, []);

  if (entries.length === 0) return <PortsEmpty />;

  return (
    <div className="ports-view">
      <header className="ports-hero">
        <div>
          <p className="dash-eyebrow">Network map</p>
          <h2 className="dash-title">Ports in use</h2>
          <p className="dash-lead">
            Live bindings from every managed subprocess — open, inspect, catch collisions.
          </p>
        </div>
      </header>

      <div className="ports-tiles">
        <div className="ports-tile tone-blue">
          <span className="ports-tile-label">Unique ports</span>
          <strong>{stats.unique}</strong>
        </div>
        <div className="ports-tile tone-cyan">
          <span className="ports-tile-label">Bindings</span>
          <strong>{stats.total}</strong>
        </div>
        <div className="ports-tile tone-violet">
          <span className="ports-tile-label">Projects</span>
          <strong>{stats.projects}</strong>
        </div>
        <div className={`ports-tile ${stats.conflicts ? "tone-hot" : "tone-green"}`}>
          <span className="ports-tile-label">Conflicts</span>
          <strong>{stats.conflicts}</strong>
        </div>
      </div>

      <div className="ports-layout">
        <div className="panel ports-main">
          <div className="panel-head-row">
            <h3>All bindings</h3>
            <span className="dash-panel-tag">Live · 3s refresh</span>
          </div>
          <div className="sub">Click a port badge to open it in the browser</div>
          <PortsTable entries={entries} />
        </div>

        <div className="panel ports-side">
          <h3>By project</h3>
          <div className="sub">Where listeners are concentrated</div>
          <ul className="ports-by-project">
            {byProject.map(([name, list]) => {
              const clash = list.some((e) =>
                entries.filter((x) => x.port === e.port).length >= 2
              );
              return (
                <li key={name}>
                  <div className="ports-by-head">
                    <span className="ports-by-name">{name}</span>
                    <span className="ports-by-count">{list.length}</span>
                  </div>
                  <div className="ports-by-badges">
                    {list.map((e) => (
                      <button
                        type="button"
                        key={`${e.procId}-${e.port}`}
                        className={`port-badge sm ${
                          entries.filter((x) => x.port === e.port).length >= 2 ? "conflict" : ""
                        }`}
                        onClick={() => BrowserOpenURL(`http://localhost:${e.port}`)}
                      >
                        :{e.port}
                      </button>
                    ))}
                  </div>
                  {clash && <span className="ports-by-warn">Has a shared port</span>}
                </li>
              );
            })}
          </ul>
        </div>
      </div>
    </div>
  );
}
