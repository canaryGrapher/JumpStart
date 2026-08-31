import { useEffect, useState } from "react";
import {
  StartProcess,
  StopProcess,
  GetStatus,
  EventsOn,
  BrowserOpenURL,
} from "../api";
import LogPanel from "./LogPanel";
import OpenActions from "./OpenActions";
import DepsPanel from "./DepsPanel";
import IconToggleButton from "./IconToggleButton";
import { ICONS } from "./Icon";
import ScriptBar from "./scripts/ScriptBar";
import ScriptRunsPanel from "./scripts/ScriptRunsPanel";
import useScriptRuns from "../hooks/useScriptRuns";
import { trackPanel } from "../analytics";

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
      // Read the live status rather than trusting possibly-stale local state,
      // so a fast reclick can't fire a duplicate StartProcess.
      const cur = await GetStatus(proc.id).catch(() => status);
      setStatus(cur);
      if (cur.running) {
        await StopProcess(proc.id);
      } else {
        await StartProcess(projectId, proc.id);
        setStatus((s) => ({ ...s, running: true }));
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
      {status.running && (
        <div className="card-meta-row">
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
            {(!status.ports || !status.ports.length) && (
              <span className="pid">detecting port…</span>
            )}
          </div>
          <span className="pid">PID {status.pid}</span>
        </div>
      )}
      <div className="card-actions-row">
        <div className="card-actions-utility">
          <OpenActions dir={proc.dir} onError={onError} iconOnly />
          <IconToggleButton
            icon={ICONS.layers}
            active={showDeps}
            label="Deps"
            onClick={(e) => {
              e.stopPropagation();
              setShowDeps((v) => {
                if (!v) trackPanel("deps");
                return !v;
              });
            }}
          />
          <IconToggleButton
            icon={ICONS.fileText}
            active={showLogs}
            label="Logs"
            onClick={(e) => {
              e.stopPropagation();
              setShowLogs((v) => {
                if (!v) trackPanel("logs");
                return !v;
              });
            }}
          />
        </div>
        <button
          className={`btn small ${status.running ? "danger" : "primary"}`}
          onClick={toggle}
          disabled={busy}
        >
          {status.running ? "Stop" : "Start"}
        </button>
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
