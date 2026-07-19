import { useEffect, useState } from "react";
import { DetectTestConfig, RunTests, SaveProject, EventsOn } from "../api";
import { isComposeProc } from "../procUtils";
import LogPanel from "./LogPanel";

// TestRunner detects and runs tests for a single directory — either the
// project root ("global directory test") or one process's own working
// directory. procId is "" for the global test, or a process ID.
function TestRunner({ dir, projectId, procId, storedCommand, onSaveCommand, onError }) {
  const [config, setConfig] = useState(null);
  const [loading, setLoading] = useState(true);
  const [customCommand, setCustomCommand] = useState(storedCommand || "");
  const [running, setRunning] = useState(false);
  const [logId, setLogId] = useState(null);
  const [exitInfo, setExitInfo] = useState(null); // { code }

  useEffect(() => {
    setLoading(true);
    DetectTestConfig(dir)
      .then(setConfig)
      .catch((e) => onError(String(e)))
      .finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dir]);

  useEffect(() => {
    setCustomCommand(storedCommand || "");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [procId, storedCommand]);

  useEffect(() => {
    if (!logId) return;
    setExitInfo(null);
    const off = EventsOn(`exit:${logId}`, (code) => setExitInfo({ code }));
    return off;
  }, [logId]);

  const saveCustomCommand = async () => {
    try {
      await onSaveCommand(customCommand);
    } catch (e) {
      onError(String(e));
    }
  };

  const runTests = async () => {
    if (running) return;
    setRunning(true);
    setExitInfo(null);
    try {
      const id = await RunTests(projectId, procId, customCommand);
      setLogId(id);
    } catch (e) {
      onError(`Run tests failed: ${String(e)}`);
    } finally {
      setRunning(false);
    }
  };

  const hasCommand = (config && config.detected) || customCommand.trim();

  return (
    <div className="test-runner">
      {loading ? (
        <div className="sub">Detecting test setup…</div>
      ) : config && config.detected ? (
        <div className="sub">
          Detected {config.kind} tests: <code className="cmd inline">{config.command} {(config.args || []).join(" ")}</code>
        </div>
      ) : (
        <div className="sub">No tests detected automatically. Set a custom command to run instead.</div>
      )}

      <div className="field">
        <label htmlFor={`test-cmd-${procId || "global"}`}>Custom test command (optional override)</label>
        <div className="row">
          <input
            id={`test-cmd-${procId || "global"}`}
            placeholder="e.g. npm test"
            value={customCommand}
            onChange={(e) => setCustomCommand(e.target.value)}
            onBlur={saveCustomCommand}
          />
        </div>
      </div>

      <div className="actions">
        <button className="btn primary" disabled={running || !hasCommand} onClick={runTests}>
          {running ? "Starting…" : "Run Tests"}
        </button>
        {exitInfo && (
          <span className={`status-pill ${exitInfo.code === 0 ? "on" : "off"}`}>
            <span className="dot" />
            {exitInfo.code === 0 ? "Passed" : `Exit code ${exitInfo.code}`}
          </span>
        )}
      </div>

      {logId && <LogPanel procId={logId} />}
    </div>
  );
}

export default function TestPanel({ project, onError, onChanged }) {
  const visibleProcs = (project.processes || []).filter((p) => !isComposeProc(p));

  const saveGlobalCommand = async (cmd) => {
    await SaveProject({ ...project, testCommand: cmd });
    onChanged && onChanged();
  };

  const saveProcessCommand = async (procId, cmd) => {
    const processes = (project.processes || []).map((p) =>
      p.id === procId ? { ...p, testCommand: cmd } : p
    );
    await SaveProject({ ...project, processes });
    onChanged && onChanged();
  };

  return (
    <div className="panel test-panel">
      <h3>Tests</h3>

      <div className="test-section">
        <h4>Global directory test</h4>
        <div className="sub">Runs against the project root: <code className="cmd inline">{project.root}</code></div>
        <TestRunner
          key={`global:${project.root}`}
          dir={project.root}
          projectId={project.id}
          procId=""
          storedCommand={project.testCommand}
          onSaveCommand={saveGlobalCommand}
          onError={onError}
        />
      </div>

      {visibleProcs.length > 0 && (
        <div className="test-section">
          <h4>Process tests</h4>
          {visibleProcs.map((proc) => (
            <div className="test-process" key={proc.id}>
              <div className="test-process-name">{proc.name}</div>
              <TestRunner
                key={`proc:${proc.id}`}
                dir={proc.dir}
                projectId={project.id}
                procId={proc.id}
                storedCommand={proc.testCommand}
                onSaveCommand={(cmd) => saveProcessCommand(proc.id, cmd)}
                onError={onError}
              />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
