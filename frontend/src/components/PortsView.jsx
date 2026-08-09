import { useEffect, useState } from "react";
import { GetPortMap, BrowserOpenURL } from "../api";
import { captureOnce } from "../analytics";

function conflictCount(entries) {
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

// Shared table markup so the Dashboard's ports card and the standalone
// Ports view render identically without duplicating markup.
export function PortsTable({ entries }) {
  if (entries.length === 0)
    return (
      <div className="sub">Start a subprocess and its listening ports will show up here.</div>
    );

  return (
    <table className="table">
      <thead>
        <tr>
          <th>Port</th>
          <th>Project</th>
          <th>Subprocess</th>
          <th>PID</th>
        </tr>
      </thead>
      <tbody>
        {entries.map((e) => (
          <tr key={`${e.procId}-${e.port}`}>
            <td>
              <button
                className="port-badge"
                onClick={() => BrowserOpenURL(`http://localhost:${e.port}`)}
              >
                :{e.port}
              </button>
            </td>
            <td>{e.projectName}</td>
            <td>{e.procName}</td>
            <td className="pid">{e.pid}</td>
          </tr>
        ))}
      </tbody>
    </table>
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

export default function PortsView({ onError }) {
  const entries = usePortMap(onError);

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

  if (entries.length === 0)
    return (
      <div className="empty">
        <h2>No ports in use</h2>
        <p>Start a subprocess and its listening ports will show up here.</p>
      </div>
    );

  return (
    <div className="panel">
      <h3>Port usage & mapping</h3>
      <div className="sub">Live view of every port used by managed subprocesses</div>
      <PortsTable entries={entries} />
    </div>
  );
}
