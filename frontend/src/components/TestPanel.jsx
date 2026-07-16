import { useEffect, useState } from "react";
import { DetectTestConfig, RunTests, SaveProject, EventsOn } from "../api";
import LogPanel from "./LogPanel";

export default function TestPanel({ project, onError, onChanged }) {
  const [config, setConfig] = useState(null);
  const [loading, setLoading] = useState(true);
  const [customCommand, setCustomCommand] = useState(project.testCommand || "");
  const [running, setRunning] = useState(false);
  const [logId, setLogId] = useState(null);
  const [exitInfo, setExitInfo] = useState(null); // { code }

  useEffect(() => {
    setLoading(true);
    DetectTestConfig(project.root)
      .then(setConfig)
      .catch((e) => onError(String(e)))
      .finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [project.root]);

  useEffect(() => {
    if (!logId) return;
    setExitInfo(null);
    const off = EventsOn(`exit:${logId}`, (code) => setExitInfo({ code }));
    return off;
  }, [logId]);

  const saveCustomCommand = async () => {
    try {
      await SaveProject({ ...project, testCommand: customCommand });
      onChanged && onChanged();
    } catch (e) {
      onError(String(e));
    }
  };

  const runTests = async () => {
    if (running) return;
    setRunning(true);
    setExitInfo(null);
    try {
      const id = await RunTests(project.id, project.root, customCommand);
      setLogId(id);
    } catch (e) {
      onError(`Run tests failed: ${String(e)}`);
    } finally {
      setRunning(false);
    }
  };

  const hasCommand = (config && config.detected) || customCommand.trim();

  return (
    <div className="panel test-panel">
      <h3>Tests</h3>

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
        <label htmlFor="test-cmd">Custom test command (optional override)</label>
        <div className="row">
          <input
            id="test-cmd"
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
