import { useEffect, useState } from "react";
import { GitHubGetStatus, GitLabGetStatus, BrowserOpenURL } from "../api";
import GitHubConnect from "./github/GitHubConnect";
import GitLabConnect from "./gitlab/GitLabConnect";
import Icon, { ICONS } from "./Icon";

function AccountProfile({ status, provider, profileButtonClass, profileIcon }) {
  if (!status?.connected) {
    return null;
  }

  const displayName = status.name || status.login;
  const profileUrl =
    status.profileUrl ||
    (status.login
      ? provider === "gitlab"
        ? `https://gitlab.com/${status.login}`
        : `https://github.com/${status.login}`
      : null);

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
              <dt>{provider === "gitlab" ? "GitLab" : "GitHub"}</dt>
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
          <button type="button" className={profileButtonClass} onClick={() => BrowserOpenURL(profileUrl)}>
            {profileIcon}
            <span>View profile</span>
          </button>
        )}
      </div>
    </section>
  );
}

// Settings → Accounts. Connected profiles up top; provider sign-in below.
export default function AccountsSettings({ onError }) {
  const [githubStatus, setGitHubStatus] = useState(null);
  const [gitlabStatus, setGitLabStatus] = useState(null);

  useEffect(() => {
    GitHubGetStatus()
      .then(setGitHubStatus)
      .catch((e) => onError && onError(String(e)));
    GitLabGetStatus()
      .then(setGitLabStatus)
      .catch((e) => onError && onError(String(e)));
  }, [onError]);

  const checked = githubStatus !== null && gitlabStatus !== null;
  const anyConnected = githubStatus?.connected || gitlabStatus?.connected;

  return (
    <div className="prefs-section prefs-accounts">
      {checked && !anyConnected && (
        <section className="prefs-account-card prefs-account-empty">
          <p className="prefs-account-lead">No accounts connected</p>
          <p className="row-hint">
            Sign in with GitHub or GitLab below to sync project boards, push and pull
            from private remotes, and submit feedback without leaving JumpStart.
          </p>
        </section>
      )}

      {githubStatus?.connected && (
        <AccountProfile
          status={githubStatus}
          provider="github"
          profileButtonClass="gh-profile-btn"
          profileIcon={<Icon d={ICONS.github} filled />}
        />
      )}

      {gitlabStatus?.connected && (
        <AccountProfile
          status={gitlabStatus}
          provider="gitlab"
          profileButtonClass="gl-profile-btn"
          profileIcon={<Icon d={ICONS.gitlab} filled />}
        />
      )}

      <section className="prefs-subsection">
        <h4 className="prefs-subsection-title">GitHub</h4>
        <p className="row-hint prefs-subsection-hint">
          Connect GitHub to sync a project&apos;s tasks with a Projects board.
          JumpStart needs the repo and project scopes: repo to read and open issues,
          project to read and write the board.
        </p>
        <GitHubConnect profileMode="external" onChanged={setGitHubStatus} onError={onError} />
      </section>

      <section className="prefs-subsection">
        <h4 className="prefs-subsection-title">GitLab</h4>
        <p className="row-hint prefs-subsection-hint">
          Connect GitLab to push and pull from private remotes and publish releases.
          JumpStart needs api, read_repository, and write_repository scopes.
        </p>
        <GitLabConnect profileMode="external" onChanged={setGitLabStatus} onError={onError} />
      </section>
    </div>
  );
}
