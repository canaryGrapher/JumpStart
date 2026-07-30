import { useEffect, useState } from "react";
import { GetContributeInfo, BrowserOpenURL } from "../../api";
import IssueForm from "./IssueForm";
import IssueList from "./IssueList";

// Settings → Contribute. Shows the repository, whether GitHub is connected,
// and either the in-app issue composer or a browser fallback.
export default function ContributeSettings({ onError, onConnectGitHub }) {
  const [info, setInfo] = useState(null);
  const [composing, setComposing] = useState(false);

  // Re-checked on mount so returning from the Git tab reflects a token that
  // was just saved.
  useEffect(() => {
    GetContributeInfo()
      .then(setInfo)
      .catch((e) => onError && onError(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (!info) {
    return (
      <div className="prefs-section">
        <span className="row-hint">Loading…</span>
      </div>
    );
  }

  return (
    <div className="prefs-section">
      <div className="prefs-row col">
        <label>{info.repo}</label>
        <span className="row-hint">
          JumpStart is open source. Bug reports, feature requests, and pull requests are all
          welcome, and small fixes are a good place to start.
        </span>
        <div className="row">
          <button className="btn small" onClick={() => BrowserOpenURL(info.repoUrl)}>
            View Repository
          </button>
          <span className={`ai-status ${info.connected ? "ok" : ""}`}>
            {info.connected ? "GitHub connected" : "GitHub not connected"}
          </span>
        </div>
      </div>

      {info.connected ? (
        composing ? (
          <IssueForm onCancel={() => setComposing(false)} onError={onError} />
        ) : (
          <div className="prefs-row col">
            <label>Submit an issue</label>
            <span className="row-hint">
              File a bug report or feature request straight from the app. Your app version
              ({info.environment.appVersion}), OS ({info.environment.os}/{info.environment.arch})
              are attached automatically.
            </span>
            <div className="row">
              <button className="btn small primary" onClick={() => setComposing(true)}>
                Submit Issue
              </button>
            </div>
          </div>
        )
      ) : (
        <div className="prefs-row col">
          <label>Submit an issue</label>
          <span className="row-hint">
            Connect your GitHub account (or add a GitHub Personal Access Token) to submit
            issues directly from the app.
          </span>
          <div className="row">
            <button className="btn small primary" onClick={onConnectGitHub}>
              Connect GitHub
            </button>
            <button className="btn small" onClick={() => BrowserOpenURL(info.newIssueUrl)}>
              Open GitHub Issues
            </button>
          </div>
        </div>
      )}

      <IssueList />
    </div>
  );
}
