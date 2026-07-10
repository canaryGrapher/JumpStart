import { useEffect, useState } from "react";
import { GetPortMap, BrowserOpenURL } from "../api";

export default function PortsView({ onError }) {
  const [entries, setEntries] = useState([]);

  useEffect(() => {
    const poll = () =>
      GetPortMap()
        .then((e) => setEntries(e || []))
        .catch((e) => onError(String(e)));
    poll();
    const t = setInterval(poll, 3000);
    return () => clearInterval(t);
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
    </div>
  );
}
