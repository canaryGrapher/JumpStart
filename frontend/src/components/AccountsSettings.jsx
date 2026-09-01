import { useEffect, useState } from "react";
import { GitHubGetStatus, BrowserOpenURL } from "../api";
import GitHubConnect from "./github/GitHubConnect";
import Icon, { ICONS } from "./Icon";

function AccountProfile({ status }) {
  if (!status?.connected) {
    return null;
  }

  const displayName = status.name || status.login;
  const profileUrl = status.profileUrl || (status.login ? `https://github.com/${status.login}` : null);

  return (
    <section className="prefs-account-card">
      <div className="prefs-account-body">
        <div className="prefs-account-head">
          <h4 className="prefs-account-name">{displayName}</h4>
          {status.login && <p className="prefs-account-login">@{status.login}</p>}
        </div>

        {status.bio && <p className="prefs-account-bio">{status.bio}</p>}

        <dl className="prefs-account-facts">
          {status.company && (
            <div className="prefs-account-fact">
              <dt>Company</dt>
              <dd>{status.company}</dd>
            </div>
          )}
          {status.location && (
            <div className="prefs-account-fact">
              <dt>Location</dt>
              <dd>{status.location}</dd>
            </div>
          )}
          {status.websiteUrl && (
            <div className="prefs-account-fact">
              <dt>Website</dt>
              <dd>
                <button type="button" className="prefs-account-link" onClick={() => BrowserOpenURL(status.websiteUrl)}>
                  {status.websiteUrl.replace(/^https?:\/\//, "")}
                </button>
              </dd>
            </div>
          )}
          {status.login && (
            <div className="prefs-account-fact">
              <dt>GitHub</dt>
              <dd>@{status.login}</dd>
            </div>
          )}
        </dl>
      </div>

      <div className="prefs-account-aside">
        {status.avatarUrl && (
          <img className="prefs-account-avatar" src={status.avatarUrl} alt="" />
        )}
        {profileUrl && (
          <button type="button" className="gh-profile-btn" onClick={() => BrowserOpenURL(profileUrl)}>
            <Icon d={ICONS.github} filled />
            <span>View profile</span>
          </button>
        )}
      </div>
    </section>
  );
}

// Settings → Accounts. Profile summary up top; GitHub sign-in and scopes below.
export default function AccountsSettings({ onError }) {
  const [status, setStatus] = useState(null);

  useEffect(() => {
    GitHubGetStatus()
      .then(setStatus)
      .catch((e) => onError && onError(String(e)));
  }, [onError]);

  return (
    <div className="prefs-section prefs-accounts">
      <AccountProfile status={status} />

      {!status?.connected && status !== null && (
        <section className="prefs-account-card prefs-account-empty">
          <p className="prefs-account-lead">No account connected</p>
          <p className="row-hint">
            Sign in with GitHub below to sync project boards, open issues from the app,
            and submit feedback without leaving JumpStart.
          </p>
        </section>
      )}

      <section className="prefs-subsection">
        <h4 className="prefs-subsection-title">GitHub</h4>
        <p className="row-hint prefs-subsection-hint">
          Connect GitHub to sync a project&apos;s tasks with a Projects board.
          JumpStart needs the repo and project scopes: repo to read and open issues,
          project to read and write the board.
        </p>
        <GitHubConnect profileMode="external" onChanged={setStatus} onError={onError} />
      </section>
    </div>
  );
}
