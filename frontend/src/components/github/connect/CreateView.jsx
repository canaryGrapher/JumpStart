import { useEffect, useState } from "react";
import { GitHubListOwners, GitHubCreateRepository } from "../../../api";
import RepoPicker from "./RepoPicker";
import SearchableSelect from "../../SearchableSelect";

const ownerLabel = (o) => `${o.login} ${o.type === "organization" ? "(organisation)" : "(personal account)"}`;

// A local git project exists but nothing on GitHub matches it yet (no
// remote, or a remote that isn't GitHub). Most of the time the intent
// is to connect to a repository that's already sitting on GitHub, so
// that's the default here — a plain picker over what the token can
// see, no typing involved. "Or create a new repository" is the escape
// hatch for a genuinely new project, and even there the name isn't
// something to type: it's the folder name JumpStart already detected,
// shown but locked, with only the account and visibility left to pick.
export default function CreateView({ detection, projectId, isGitRepo, onCreated, onContinue, onError }) {
  const [owners, setOwners] = useState(null);
  const [owner, setOwner] = useState("");
  const [isPrivate, setIsPrivate] = useState(true);
  const [busy, setBusy] = useState(false);
  const [mode, setMode] = useState("link"); // "link" (pick an existing repo) | "create" (brand new)

  useEffect(() => {
    if (mode !== "create") return;
    GitHubListOwners()
      .then((o) => {
        setOwners(o || []);
        // If a remote is already configured (it points somewhere GitHub
        // doesn't recognize yet, or the repo it names doesn't exist),
        // that remote's owner is a much better default than "whichever
        // account came back first" — it's who this project is actually
        // headed for.
        const remoteOwner = detection?.owner;
        const match = remoteOwner && (o || []).find((x) => x.login.toLowerCase() === remoteOwner.toLowerCase());
        setOwner((cur) => cur || match?.login || (o || [])[0]?.login || "");
      })
      .catch((e) => onError && onError(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mode]);

  const selectedOwner = (owners || []).find((o) => o.login === owner);
  const selectedOwnerLabel = selectedOwner ? ownerLabel(selectedOwner) : "";

  const create = async () => {
    const selected = (owners || []).find((o) => o.login === owner);
    setBusy(true);
    try {
      const repo = await GitHubCreateRepository(
        projectId,
        owner,
        selected?.type || "user",
        detection.suggestedName,
        "",
        "",
        isPrivate
      );
      onCreated(repo);
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setBusy(false);
    }
  };

  if (mode === "create") {
    return (
      <div className="gh-step">
        <div className="gh-step-head">
          <span className="gh-badge-icon git" />
          <h3>{isGitRepo ? "Git project detected" : "New project, ready for GitHub"}</h3>
          <p className="gh-muted">JumpStart will create a new repository named “{detection.suggestedName}”.</p>
        </div>

        <div className="field">
          <label>Repository name</label>
          <input value={detection.suggestedName} disabled />
        </div>

        <div className="field">
          <label>Owner</label>
          {owners === null ? (
            <span className="gh-muted">Loading accounts…</span>
          ) : (
            <SearchableSelect
              value={selectedOwnerLabel}
              options={(owners || []).map(ownerLabel)}
              onChange={(label) => {
                const o = (owners || []).find((x) => ownerLabel(x) === label);
                if (o) setOwner(o.login);
              }}
              placeholder="Choose an account…"
              searchPlaceholder="Search accounts…"
            />
          )}
        </div>

        <label className="gh-check">
          <input type="checkbox" checked={isPrivate} onChange={(e) => setIsPrivate(e.target.checked)} />
          Private repository
        </label>

        <button className="btn primary gh-cta" disabled={busy || !owner} onClick={create}>
          {busy ? "Creating…" : "Create Project"}
        </button>

        <button className="gh-link-btn" onClick={() => setMode("link")}>
          Or choose an existing repository
        </button>
      </div>
    );
  }

  return (
    <div className="gh-step">
      <div className="gh-step-head">
        <span className="gh-badge-icon git" />
        <h3>{isGitRepo ? "Git project detected" : "New project, ready for GitHub"}</h3>
        <p className="gh-muted">Choose which GitHub repository this connects to.</p>
      </div>

      <RepoPicker busy={busy} onError={onError} onContinue={onContinue} defaultOwner={detection?.owner} />

      <button className="gh-link-btn" onClick={() => setMode("create")}>
        Or create a new repository
      </button>
    </div>
  );
}
