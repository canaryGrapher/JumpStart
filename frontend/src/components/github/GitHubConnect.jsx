import { useEffect, useRef, useState } from "react";
import {
  GitHubGetStatus,
  GitHubStartDeviceAuth,
  GitHubPollDeviceAuth,
  GitHubSaveToken,
  GitHubDisconnect,
  BrowserOpenURL,
  ClipboardSetText,
} from "../../api";

// Connect JumpStart to GitHub. The device flow is the front door: the
// user gets a short code, types it on github.com, and the token lands in
// the keychain. Builds without an OAuth client id fall back to pasting a
// personal access token, which needs the repo and project scopes.
// profileMode="external" — caller renders the connected profile elsewhere
// (Settings → Accounts) and this panel only handles connect/disconnect.
export default function GitHubConnect({ onChanged, onError, profileMode = "inline" }) {
  const [status, setStatus] = useState(null);
  const [device, setDevice] = useState(null);
  const [token, setToken] = useState("");
  const [busy, setBusy] = useState(false);
  const [copied, setCopied] = useState(false);
  // Stays set after a rejected token until the user finishes reconnecting,
  // so a failed reconnect attempt does not hide the Reconnect CTA.
  const [reauthNeeded, setReauthNeeded] = useState(false);
  const [reauthError, setReauthError] = useState("");
  const timer = useRef(null);

  const refresh = () =>
    GitHubGetStatus()
      .then((s) => {
        setStatus(s);
        if (s?.error) {
          setReauthNeeded(true);
          setReauthError(s.error);
        }
        if (s?.connected) {
          setReauthNeeded(false);
          setReauthError("");
        }
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
      const d = await GitHubStartDeviceAuth();
      setDevice(d);
      BrowserOpenURL(d.verificationUri);
      // GitHub tells us how often it is willing to be asked; going faster
      // gets the flow throttled, so honour the interval it sends back.
      clearInterval(timer.current);
      timer.current = setInterval(async () => {
        try {
          const done = await GitHubPollDeviceAuth();
          if (done) {
            clearInterval(timer.current);
            setDevice(null);
            refresh();
          }
        } catch (e) {
          clearInterval(timer.current);
          setDevice(null);
          setReauthNeeded(true);
          setReauthError(String(e));
          onError && onError(String(e));
        }
      }, Math.max(d.interval, 5) * 1000);
    } catch (e) {
      setReauthNeeded(true);
      setReauthError(String(e));
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
      refresh();
    } catch (e) {
      setReauthNeeded(true);
      setReauthError(String(e));
      onError && onError(String(e));
    } finally {
      setBusy(false);
    }
  };

  const disconnect = async () => {
    setBusy(true);
    try {
      await GitHubDisconnect();
      setReauthNeeded(false);
      setReauthError("");
      refresh();
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setBusy(false);
    }
  };

  // Drop the rejected token, then restart device auth when available;
  // otherwise open GitHub's PAT page so the user can paste a fresh one.
  // Keep the Reconnect banner visible until a new connection succeeds.
  const reconnect = async () => {
    const fallback =
      status?.error ||
      reauthError ||
      "GitHub rejected the token. Paste a new token or try again.";
    setBusy(true);
    setReauthNeeded(true);
    setReauthError(fallback);
    try {
      await GitHubDisconnect();
      setStatus((s) => (s ? { ...s, error: "", connected: false } : s));
      onChanged && onChanged({ ...(status || {}), error: "", connected: false });
    } catch (e) {
      setReauthError(String(e));
      onError && onError(String(e));
      setBusy(false);
      return;
    }
    setBusy(false);
    if (status?.deviceFlow) {
      await startFlow();
      return;
    }
    BrowserOpenURL("https://github.com/settings/tokens");
    refresh();
  };

  const copyCode = () => {
    ClipboardSetText(device.userCode);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  if (!status) return <div className="gh-muted">Checking GitHub…</div>;

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

  const bannerError = status.error || reauthError;
  const needsReconnect = Boolean(bannerError) || reauthNeeded;

  return (
    <div className="gh-connect">
      {needsReconnect && (
        <div className="gh-warn gh-warn-action">
          <span>
            {(bannerError || "GitHub sign-in needs to be renewed.").replace(
              /,?\s*reconnect in Settings\.?$/i,
              ""
            )}
          </span>
          <button type="button" className="btn small primary" disabled={busy} onClick={reconnect}>
            Reconnect
          </button>
        </div>
      )}

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
          {status.deviceFlow && !needsReconnect && (
            <button className="btn primary" disabled={busy} onClick={startFlow}>
              Connect GitHub
            </button>
          )}
          <div className="gh-pat">
            <label>
              {status.deviceFlow
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
