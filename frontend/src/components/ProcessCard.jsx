import { useEffect, useState } from "react";
import {
  StartProcess,
  StopProcess,
  GetStatus,
  EventsOn,
  BrowserOpenURL,
} from "../api";
import LogPanel from "./LogPanel";
import DepsPanel from "./DepsPanel";
import ScriptBar from "./scripts/ScriptBar";
import ScriptRunsPanel from "./scripts/ScriptRunsPanel";
import useScriptRuns from "../hooks/useScriptRuns";

export default function ProcessCard({ projectId, proc, usage, onError }) {
  const [status, setStatus] = useState({ running: false, ports: [], pid: 0 });
  const [showLogs, setShowLogs] = useState(false);
  const [showDeps, setShowDeps] = useState(false);
  const [showRuns, setShowRuns] = useState(false);
  const [busy, setBusy] = useState(false);
  const scriptRuns = useScriptRuns(projectId, proc.id, onError);

  // Running a script opens the runs panel so its log is visible right away.
  const runScript = async (script) => {
    setShowRuns(true);
    await scriptRuns.run(script);
  };

  const refresh = () => GetStatus(proc.id).then(setStatus).catch(() => {});

  useEffect(() => {
    refresh();
    const offExit = EventsOn(`exit:${proc.id}`, refresh);
    const offPorts = EventsOn(`ports:${proc.id}`, (ports) =>
      setStatus((s) => ({ ...s, ports: ports || [] }))
    );
    const poll = setInterval(refresh, 5000);
    return () => {
      offExit();
      offPorts();
      clearInterval(poll);
    };
  }, [proc.id]);

  const toggle = async (e) => {
    e.stopPropagation();
    if (busy) return;
    setBusy(true);
    try {
      if (status.running) {
        await StopProcess(proc.id);
      } else {
        await StartProcess(projectId, proc.id);
      }
      await refresh();
    } catch (err) {
      onError(String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div
      className={`card ${status.running ? "running" : "stopped"}`}
      onClick={toggle}
      title={status.running ? "Click to stop" : "Click to start"}
    >
      <div className="card-top">
        <h3>{proc.name}</h3>
        <span className={`status-pill ${status.running ? "on" : "off"}`}>
          <span className="dot" />
          {status.running ? "Running" : "Stopped"}
        </span>
      </div>
      <div className="cmd">{proc.command}</div>
      <div className="dir">{proc.dir}</div>
      <div className="card-bottom">
        <div className="ports">
          {(status.ports || []).map((p) => (
            <span
              key={p}
              className="port-badge"
              onClick={(e) => {
                e.stopPropagation();
                BrowserOpenURL(`http://localhost:${p}`);
              }}
            >
              :{p}
            </span>
          ))}
          {status.running && (!status.ports || !status.ports.length) && (
            <span className="pid">detecting port...</span>
          )}
        </div>
        <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
          {status.running && <span className="pid">PID {status.pid}</span>}
          <button
            className="btn small"
            onClick={(e) => {
              e.stopPropagation();
              setShowDeps((v) => !v);
            }}
          >
            {showDeps ? "Hide deps" : "Deps"}
          </button>
          <button
            className="btn small"
            onClick={(e) => {
              e.stopPropagation();
              setShowLogs((v) => !v);
            }}
          >
            {showLogs ? "Hide logs" : "Logs"}
          </button>
          <button
            className={`btn small ${status.running ? "danger" : "primary"}`}
            onClick={toggle}
            disabled={busy}
          >
            {status.running ? "Stop" : "Start"}
          </button>
        </div>
      </div>
      {status.running && usage && (
        <div className="usage-badges">
          <span className={`usage-badge ${usage.cpu > 100 ? "hot" : ""}`}>
            CPU {usage.cpu.toFixed(1)}%
          </span>
          <span className="usage-badge">RAM {Math.round(usage.memMB)} MB</span>
        </div>
      )}
      <ScriptBar
        scripts={proc.scripts}
        busyScriptId={scriptRuns.busyScriptId}
        runsOpen={showRuns}
        onRun={runScript}
        onToggleRuns={() => setShowRuns((v) => !v)}
      />
      {showRuns && (
        <ScriptRunsPanel
          runs={scriptRuns.runs}
          activeRunId={scriptRuns.activeRunId}
          onSelect={scriptRuns.setActiveRunId}
          onStop={scriptRuns.stop}
          onFinished={scriptRuns.markFinished}
        />
      )}
      {showDeps && <DepsPanel projectId={projectId} proc={proc} onError={onError} />}
      {showLogs && <LogPanel procId={proc.id} />}
    </div>
  );
}
