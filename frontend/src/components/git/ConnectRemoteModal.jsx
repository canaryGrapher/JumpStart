import { useEffect, useState } from "react";
import {
  GitHubListOwners,
  GitHubDetectRepo,
  GitHubCreateRepository,
  GenerateProjectDescription,
  SaveProject,
} from "../../api";
import { getAISettings } from "../../ai";
import SearchableSelect from "../SearchableSelect";

const ownerLabel = (o) => `${o.login} ${o.type === "organization" ? "(organisation)" : "(personal account)"}`;

// GitHub's own curated list is much longer; these are the ones people
// actually reach for. "None" (empty string) skips the gitignore step
// entirely, which is also the only option once the project has local
// commits — see the note on CreateRepository in internal/github/repos.go
// for why mixing the two isn't safe.
const GITIGNORE_TEMPLATES = [
  "", "Node", "Python", "Go", "Rust", "Java", "C++", "C", "Swift",
  "Ruby", "PHP", "Elixir", "Dart", "Terraform", "macOS", "VisualStudioCode",
];
const gitignoreLabel = (t) => (t === "" ? "None" : t);

// A local project not yet wired to any GitHub repository is one click
// (folder name, whichever account owns it) away from being one — this
// gathers just enough to make that click safe: which account, what
// visibility, and a couple of optional touches (description, a starter
// .gitignore) that GitHub's own "New repository" page also offers.
export default function ConnectRemoteModal({ project, status, onClose, onConnected, onError, onChanged }) {
  const [owners, setOwners] = useState(null);
  const [owner, setOwner] = useState("");
  const [name, setName] = useState(project.name || "");
  const [isPrivate, setIsPrivate] = useState(true);
  const [description, setDescription] = useState(project.description || "");
  const [gitignore, setGitignore] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [generatingDesc, setGeneratingDesc] = useState(false);
  const [descMsg, setDescMsg] = useState(null); // { text, kind }

  // A local repo with commit history already has a first commit of its
  // own; GitHub can only add a .gitignore as part of an initial commit
  // it makes itself, which would give the two histories no common
  // ancestor and break the push that wires this project up.
  const hasLocalCommits = !!status?.lastCommit;

  useEffect(() => {
    GitHubListOwners()
      .then((o) => {
        setOwners(o || []);
        setOwner((o || [])[0]?.login || "");
      })
      .catch((e) => onError && onError(String(e)));
    GitHubDetectRepo(project.id)
      .then((d) => {
        if (d?.suggestedName) setName(d.suggestedName);
      })
      .catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const selectedOwner = (owners || []).find((o) => o.login === owner);
  const selectedOwnerLabel = selectedOwner ? ownerLabel(selectedOwner) : "";

  const generateDescription = async () => {
    setGeneratingDesc(true);
    setDescMsg(null);
    try {
      const { host, model } = getAISettings();
      const text = await GenerateProjectDescription(host, model, project.root, project.name);
      if (text) {
        setDescription(text);
        // Written back immediately, not just held for this form — the
        // project didn't have a description before, and now it does,
        // independent of whether repository creation goes on to succeed.
        try {
          await SaveProject({ ...project, description: text });
          onChanged && onChanged();
        } catch (e) {
          setDescMsg({ text: `Generated, but couldn't save it to the project: ${e}`, kind: "err" });
        }
      } else {
        setDescMsg({ text: "The model returned nothing.", kind: "err" });
      }
    } catch (e) {
      setDescMsg({ text: String(e), kind: "err" });
    } finally {
      setGeneratingDesc(false);
    }
  };

  const create = async () => {
    if (!name.trim() || !owner || busy) return;
    setBusy(true);
    setError("");
    try {
      const repo = await GitHubCreateRepository(
        project.id,
        owner,
        selectedOwner?.type || "user",
        name.trim(),
        description.trim(),
        hasLocalCommits ? "" : gitignore,
        isPrivate
      );
      onConnected && onConnected(repo);
    } catch (e) {
      setError(String(e));
      onError && onError(String(e));
    } finally {
      setBusy(false);
    }
  };

  const onKeyDown = (e) => {
    if (e.key === "Escape") onClose();
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal git-connect-remote-modal" onClick={(e) => e.stopPropagation()} onKeyDown={onKeyDown}>
        <h2>Connect Remote</h2>
        <p className="sub">Create a new GitHub repository and wire it up as this project's remote.</p>

        <div className="field">
          <label htmlFor="cr-name">Repository name</label>
          <input id="cr-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="my-project" autoFocus />
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

        <div className="field">
          <label htmlFor="cr-desc">Description</label>
          <textarea
            id="cr-desc"
            rows={2}
            placeholder="What this project does, in a sentence or two…"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>
        <div className="actions">
          <button type="button" className="btn small ai" disabled={generatingDesc} onClick={generateDescription}>
            {generatingDesc ? "Generating…" : "✨ Generate with AI"}
          </button>
        </div>
        {descMsg && <span className={descMsg.kind === "err" ? "error" : "sub"}>{descMsg.text}</span>}

        <div className="field">
          <label>.gitignore template</label>
          <SearchableSelect
            value={gitignoreLabel(gitignore)}
            options={GITIGNORE_TEMPLATES.map(gitignoreLabel)}
            onChange={(label) => setGitignore(label === "None" ? "" : label)}
            placeholder="None"
            searchPlaceholder="Search templates…"
            disabled={hasLocalCommits}
          />
          {hasLocalCommits && (
            <span className="row-hint">
              Not available — this project already has commit history, so an auto-generated .gitignore would
              conflict with pushing it. Add one manually once connected instead.
            </span>
          )}
        </div>

        <label className="check-row">
          <input type="checkbox" checked={isPrivate} onChange={(e) => setIsPrivate(e.target.checked)} />
          Private repository
        </label>

        {error && <span className="error">{error}</span>}

        <div className="modal-actions">
          <button className="btn" disabled={busy} onClick={onClose}>
            Cancel
          </button>
          <button className="btn primary" disabled={busy || !name.trim() || !owner} onClick={create}>
            {busy ? "Creating…" : "Create & Connect"}
          </button>
        </div>
      </div>
    </div>
  );
}
