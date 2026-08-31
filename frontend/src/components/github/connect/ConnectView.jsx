import { useRef, useState } from "react";
import {
  GitHubGetStatus,
  GitHubStartDeviceAuth,
  GitHubPollDeviceAuth,
  GitHubSaveToken,
  BrowserOpenURL,
  ClipboardSetText,
} from "../../../api";
import RepoPicker from "./RepoPicker";

// No GitHub account is connected yet. Connect first (device flow, or a
// pasted token when this build has no OAuth client id), then pick where
// the project should live from what that account can actually see.
export default function ConnectView({ onConnected, onContinue, onError }) {
  const [device, setDevice] = useState(null);
  const [connected, setConnected] = useState(false);
  const [deviceFlowAvailable, setDeviceFlowAvailable] = useState(true);
  const [token, setToken] = useState("");
  const [busy, setBusy] = useState(false);
  const [copied, setCopied] = useState(false);
  const timer = useRef(null);

  const finishConnecting = () => {
    setDevice(null);
    setConnected(true);
    onConnected && onConnected();
  };

  const startFlow = async () => {
    setBusy(true);
    try {
      const status = await GitHubGetStatus();
      setDeviceFlowAvailable(status.deviceFlow);
      if (!status.deviceFlow) return;
      const d = await GitHubStartDeviceAuth();
      setDevice(d);
      BrowserOpenURL(d.verificationUri);
      clearInterval(timer.current);
      timer.current = setInterval(async () => {
        try {
          const done = await GitHubPollDeviceAuth();
          if (done) {
            clearInterval(timer.current);
            finishConnecting();
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
      await GitHubSaveToken(token.trim());
      setToken("");
      finishConnecting();
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

  if (connected) {
    return (
      <div className="gh-step">
        <div className="gh-step-head">
          <h3>Choose where this lives</h3>
          <p className="gh-muted">Pick the account and repository to sync with.</p>
        </div>
        <RepoPicker busy={busy} onError={onError} onContinue={onContinue} />
      </div>
    );
  }

  return (
    <div className="gh-step">
      <div className="gh-step-head">
        <h3>Connect to GitHub</h3>
        <p className="gh-muted">Sign in once, then choose where you want to sync this project.</p>
      </div>

      {device ? (
        <div className="gh-device">
          <p>
            Enter this code at <strong>{device.verificationUri}</strong>, which should
            have opened in your browser.
          </p>
          <div className="gh-code-row">
            <code className="gh-code">{device.userCode}</code>
            <button className="btn small" onClick={copyCode}>
              {copied ? "Copied" : "Copy"}
            </button>
          </div>
          <span className="gh-muted gh-waiting">
            <span className="gh-check-spinner" /> Waiting for you to authorize…
          </span>
        </div>
      ) : (
        <>
          {deviceFlowAvailable && (
            <button className="btn primary gh-cta" disabled={busy} onClick={startFlow}>
              Connect GitHub
            </button>
          )}
          <div className="gh-pat">
            <label>
              {deviceFlowAvailable
                ? "Or paste a personal access token"
                : "Paste a personal access token with the repo and project scopes"}
            </label>
            <div className="row">
              <input
                type="password"
                value={token}
                placeholder="ghp_…"
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
