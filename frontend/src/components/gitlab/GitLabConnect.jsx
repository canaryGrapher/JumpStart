import { useEffect, useRef, useState } from "react";
import {
  GitLabGetStatus,
  GitLabStartDeviceAuth,
  GitLabPollDeviceAuth,
  GitLabSaveToken,
  GitLabDisconnect,
  BrowserOpenURL,
  ClipboardSetText,
} from "../../api";

// Connect JumpStart to GitLab. The device flow is the front door; builds
// without an OAuth client id fall back to pasting a personal access token.
export default function GitLabConnect({ onChanged, onError, profileMode = "inline" }) {
  const [status, setStatus] = useState(null);
  const [device, setDevice] = useState(null);
  const [token, setToken] = useState("");
  const [busy, setBusy] = useState(false);
  const [copied, setCopied] = useState(false);
  const timer = useRef(null);

  const refresh = () =>
    GitLabGetStatus()
      .then((s) => {
        setStatus(s);
        onChanged && onChanged(s);
      })
      .catch((e) => onError && onError(String(e)));

  useEffect(() => {
    refresh();
    return () => clearInterval(timer.current);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const startFlow = async () => {
    setBusy(true);
    try {
      const d = await GitLabStartDeviceAuth();
      setDevice(d);
      BrowserOpenURL(d.verificationUri);
      clearInterval(timer.current);
      timer.current = setInterval(async () => {
        try {
          const done = await GitLabPollDeviceAuth();
          if (done) {
            clearInterval(timer.current);
            setDevice(null);
            refresh();
          }
        } catch (e) {
          clearInterval(timer.current);
          setDevice(null);
          onError && onError(String(e));
        }
      }, Math.max(d.interval, 5) * 1000);
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setBusy(false);
    }
  };

  const savePat = async () => {
    setBusy(true);
    try {
      await GitLabSaveToken(token.trim());
      setToken("");
      refresh();
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setBusy(false);
    }
  };

  const disconnect = async () => {
    setBusy(true);
    try {
      await GitLabDisconnect();
      refresh();
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setBusy(false);
    }
  };

  const copyCode = () => {
    ClipboardSetText(device.userCode);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  if (!status) return <div className="gh-muted">Checking GitLab…</div>;

  if (status.connected) {
    if (profileMode === "external") {
      return (
        <div className="prefs-row gh-connection-row">
          <label>Status</label>
          <span className="ai-status ok">Connected as @{status.login}</span>
          <button className="btn" disabled={busy} onClick={disconnect}>
            Disconnect
          </button>
        </div>
      );
    }

    return (
      <div className="gh-connected">
        {status.avatarUrl && <img src={status.avatarUrl} alt="" className="gh-avatar" />}
        <div className="gh-connected-who">
          <strong>{status.name || status.login}</strong>
          <span className="gh-muted">@{status.login}</span>
        </div>
        <div className="spacer" />
        <button className="btn" disabled={busy} onClick={disconnect}>
          Disconnect
        </button>
      </div>
    );
  }

  return (
    <div className="gh-connect">
      {status.error && <div className="gh-warn">{status.error}</div>}

      {device ? (
        <div className="gh-device">
          <p>
            Enter this code at <strong>{device.verificationUri}</strong>, which
            should have opened in your browser.
          </p>
          <div className="gh-code-row">
            <code className="gh-code">{device.userCode}</code>
            <button className="btn small" onClick={copyCode}>
              {copied ? "Copied" : "Copy"}
            </button>
          </div>
          <span className="gh-muted">Waiting for you to authorize…</span>
        </div>
      ) : (
        <>
          {status.deviceFlow && (
            <button className="btn primary" disabled={busy} onClick={startFlow}>
              Connect GitLab
            </button>
          )}
          <div className="gh-pat">
            <label>
              {status.deviceFlow
                ? "Or paste a personal access token"
                : "Paste a personal access token with api and write_repository scopes"}
            </label>
            <div className="row">
              <input
                type="password"
                value={token}
                placeholder="glpat-…"
                onChange={(e) => setToken(e.target.value)}
              />
              <button className="btn" disabled={busy || !token.trim()} onClick={savePat}>
                Save
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
