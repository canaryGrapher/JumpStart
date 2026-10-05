import { useEffect, useState } from "react";
import {
  GetMCPSettings,
  SetMCPSettings,
  RotateMCPToken,
  ClipboardSetText,
  BrowserOpenURL,
} from "../api";
import Switch from "./Switch";

function mcpClientSnippet(url, token) {
  return JSON.stringify(
    {
      mcpServers: {
        jumpstart: {
          url,
          headers: {
            Authorization: `Bearer ${token}`,
          },
        },
      },
    },
    null,
    2
  );
}

// Settings → Agents. Toggle the localhost MCP server so Cursor / Claude /
// other agents can drive projects, processes, tasks, and files.
export default function AgentsSettings({ onError }) {
  const [enabled, setEnabled] = useState(false);
  const [port, setPort] = useState(8787);
  const [token, setToken] = useState("");
  const [url, setUrl] = useState("");
  const [running, setRunning] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [copied, setCopied] = useState("");

  const apply = (s) => {
    if (!s) return;
    setEnabled(!!s.enabled);
    setPort(s.port || 8787);
    setToken(s.token || "");
    setUrl(s.url || "");
    setRunning(!!s.running);
    setError(s.error || "");
  };

  useEffect(() => {
    GetMCPSettings()
      .then(apply)
      .catch((e) => onError && onError(String(e)));
  }, [onError]);

  const persist = async (nextEnabled, nextPort) => {
    setBusy(true);
    try {
      const s = await SetMCPSettings(nextEnabled, nextPort);
      apply(s);
    } catch (e) {
      onError && onError(String(e));
      try {
        apply(await GetMCPSettings());
      } catch (_) {
        /* ignore */
      }
    } finally {
      setBusy(false);
    }
  };

  const toggle = (next) => {
    setEnabled(next);
    persist(next, port);
  };

  const savePort = () => {
    const n = Number(port);
    if (!Number.isInteger(n) || n < 1 || n > 65535) {
      onError && onError("Port must be between 1 and 65535.");
      return;
    }
    persist(enabled, n);
  };

  const rotate = async () => {
    setBusy(true);
    try {
      apply(await RotateMCPToken());
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setBusy(false);
    }
  };

  const copy = async (label, text) => {
    try {
      await ClipboardSetText(text);
      setCopied(label);
      setTimeout(() => setCopied(""), 1500);
    } catch (e) {
      onError && onError(String(e));
    }
  };

  return (
    <div className="prefs-section prefs-agents">
      <div className="prefs-row">
        <label>Enable MCP server</label>
        <Switch checked={enabled} onChange={toggle} disabled={busy} />
      </div>

      <div className="prefs-row col">
        <span className="row-hint">
          Binds to 127.0.0.1 only. Agents must send the bearer token. When
          enabled, agents can list projects, start and edit processes, read and
          write files inside project roots, inspect git changes, and work with
          kanban tasks.
        </span>
      </div>

      <div className="prefs-row col">
        <label>Port</label>
        <div className="prefs-actions">
          <input
            className="ai-host"
            type="number"
            min={1}
            max={65535}
            value={port}
            disabled={busy}
            onChange={(e) => setPort(e.target.value)}
            onBlur={savePort}
            onKeyDown={(e) => e.key === "Enter" && savePort()}
          />
          <button className="btn" onClick={savePort} disabled={busy}>
            Apply
          </button>
        </div>
      </div>

      <div className="prefs-row col">
        <label>Status</label>
        <span className="ai-status">
          {running ? "Running" : enabled ? "Starting…" : "Stopped"}
          {error ? ` — ${error}` : ""}
        </span>
      </div>

      {enabled && (
        <>
          <div className="prefs-row col">
            <label>Server URL</label>
            <div className="prefs-actions">
              <code className="prefs-mono">{url}</code>
              <button className="btn" onClick={() => copy("url", url)}>
                {copied === "url" ? "Copied" : "Copy"}
              </button>
            </div>
          </div>

          <div className="prefs-row col">
            <label>Bearer token</label>
            <div className="prefs-actions">
              <code className="prefs-mono prefs-token">{token}</code>
              <button className="btn" onClick={() => copy("token", token)}>
                {copied === "token" ? "Copied" : "Copy"}
              </button>
              <button className="btn" onClick={rotate} disabled={busy}>
                Rotate
              </button>
            </div>
          </div>

          <div className="prefs-row col">
            <label>Client config</label>
            <span className="row-hint">
              Paste into Cursor, or open the{" "}
              <button
                type="button"
                className="link-btn"
                onClick={() =>
                  BrowserOpenURL("https://jumpstart.workvar.com/#/docs/mcp")
                }
              >
                Agents MCP guide
              </button>{" "}
              for Claude, Codex, ChatGPT, Hermes, Paperclip, and others.
            </span>
            <pre className="prefs-code">{mcpClientSnippet(url, token)}</pre>
            <div className="prefs-actions">
              <button
                className="btn"
                onClick={() => copy("cfg", mcpClientSnippet(url, token))}
              >
                {copied === "cfg" ? "Copied" : "Copy config"}
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
