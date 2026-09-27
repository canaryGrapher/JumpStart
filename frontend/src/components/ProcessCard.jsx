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
import Icon, { ICONS } from "./Icon";
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

  const running = status.running;
  const ports = status.ports || [];
  const toggleLogs = (e) => {
    e.stopPropagation();
    setShowLogs((v) => {
      if (!v) trackPanel("logs");
      return !v;
    });
  };
  const toggleDeps = (e) => {
    e.stopPropagation();
    setShowDeps((v) => {
      if (!v) trackPanel("deps");
      return !v;
    });
  };

  // Layout, top to bottom: identity + primary action, what it runs, live
  // state (ports, CPU/RAM), the process's own scripts, then a footer
  // toolbar of secondary actions. Only the Run/Stop button starts or stops
  // the process — clicking elsewhere on the card does nothing surprising.
  return (
    <div className={`card proc-card ${running ? "running" : "stopped"}`}>
      <div className="proc-head">
        <span className={`proc-glyph ${running ? "on" : ""}`} aria-hidden="true">
          <Icon d={ICONS.terminal} />
        </span>
        <div className="proc-title">
          <h3 title={proc.name}>{proc.name}</h3>
          <span className={`proc-state ${running ? "on" : ""}`}>
            <span className="dot" />
            {busy ? (running ? "Stopping…" : "Starting…") : running ? `Running · PID ${status.pid}` : "Stopped"}
          </span>
        </div>
        <button
          className={`btn proc-run ${running ? "danger" : "primary"}`}
          onClick={toggle}
          disabled={busy}
          title={running ? "Stop this process" : "Start this process"}
        >
          {busy ? <span className="spinner" /> : <Icon d={running ? ICONS.stop : ICONS.play} filled />}
          {running ? "Stop" : "Run"}
        </button>
      </div>

      <div className="cmd" title={proc.command}>
        <span className="cmd-prompt">$</span>
        {proc.command}
      </div>
      {proc.dir && (
        <div className="dir" title={proc.dir}>
          <Icon d={ICONS.folder} />
          <span>{proc.dir}</span>
        </div>
      )}

      {running && (
        <div className="proc-live">
          <div className="ports">
            {ports.map((p) => (
              <button
                key={p}
                className="port-badge"
                title={`Open http://localhost:${p}`}
                onClick={(e) => {
                  e.stopPropagation();
                  BrowserOpenURL(`http://localhost:${p}`);
                }}
              >
                localhost:{p}
                <Icon d={ICONS.chevron} />
              </button>
            ))}
            {!ports.length && <span className="pid">Detecting port…</span>}
          </div>
          {usage && (
            <div className="usage-badges">
              <span className={`usage-badge ${usage.cpu > 100 ? "hot" : ""}`}>
                CPU <b>{usage.cpu.toFixed(1)}%</b>
              </span>
              <span className="usage-badge">
                RAM <b>{Math.round(usage.memMB)} MB</b>
              </span>
            </div>
          )}
        </div>
      )}

      <ScriptBar
        scripts={proc.scripts}
        busyScriptId={scriptRuns.busyScriptId}
        runCount={scriptRuns.runs.length}
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

      <div className="proc-footer">
        <div className="card-actions-utility">
          <OpenActions dir={proc.dir} onError={onError} iconOnly />
        </div>
        <div className="proc-footer-toggles">
          <button
            className={`chip-toggle ${showDeps ? "active" : ""}`}
            aria-pressed={showDeps}
            onClick={toggleDeps}
          >
            <Icon d={ICONS.layers} />
            Dependencies
          </button>
          <button
            className={`chip-toggle ${showLogs ? "active" : ""}`}
            aria-pressed={showLogs}
            onClick={toggleLogs}
          >
            <Icon d={ICONS.fileText} />
            Logs
          </button>
        </div>
      </div>
      {showDeps && <DepsPanel projectId={projectId} proc={proc} onError={onError} />}
      {showLogs && <LogPanel procId={proc.id} />}
    </div>
  );
}
