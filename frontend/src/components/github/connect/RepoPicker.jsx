import { useEffect, useMemo, useState } from "react";
import { GitHubListOwners, GitHubListRepositories } from "../../../api";
import SearchableSelect from "../../SearchableSelect";

const ownerLabel = (o) => `${o.login} ${o.type === "organization" ? "(organisation)" : "(personal account)"}`;

// Account/organization + repository selects, backed entirely by what
// the token can already see — nothing here is typed in by hand. Used
// once GitHub is connected but the local project has no obvious match:
// either right after "Connect GitHub", or as the escape hatch from the
// "create a repository" step for someone who actually wants an existing
// one instead.
export default function RepoPicker({ onContinue, onError, busy, defaultOwner }) {
  const [owners, setOwners] = useState(null);
  const [repos, setRepos] = useState(null);
  const [owner, setOwner] = useState("");
  const [fullName, setFullName] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([GitHubListOwners(), GitHubListRepositories()])
      .then(([o, r]) => {
        setOwners(o || []);
        setRepos(r || []);
        // Prefer the account a configured remote already points at, when
        // it's one this token can actually see, over just picking the
        // first account back.
        const match = defaultOwner && (o || []).find((x) => x.login.toLowerCase() === defaultOwner.toLowerCase());
        setOwner(match?.login || (o || [])[0]?.login || "");
      })
      .catch((e) => onError && onError(String(e)))
      .finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const filtered = useMemo(
    () => (repos || []).filter((r) => !owner || r.owner === owner),
    [repos, owner]
  );

  useEffect(() => {
    if (filtered.length && !filtered.some((r) => r.fullName === fullName)) {
      setFullName(filtered[0].fullName);
    }
  }, [filtered, fullName]);

  if (loading) {
    return <p className="gh-muted">Loading your repositories…</p>;
  }

  const selectedOwner = (owners || []).find((o) => o.login === owner);
  const selectedOwnerLabel = selectedOwner ? ownerLabel(selectedOwner) : "";

  return (
    <div className="gh-picker">
      <div className="field">
        <label>Account / Organisation</label>
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
      </div>

      <div className="field">
        <label>Repository</label>
        {filtered.length === 0 ? (
          <span className="gh-muted">No repositories found for this account.</span>
        ) : (
          <SearchableSelect
            value={fullName}
            options={filtered.map((r) => r.fullName)}
            onChange={setFullName}
            placeholder="Choose a repository…"
            searchPlaceholder="Search repositories…"
          />
        )}
      </div>

      <button
        className="btn primary"
        disabled={busy || !fullName}
        onClick={() => onContinue(fullName, owner)}
      >
        {busy ? "Connecting…" : "Continue"}
      </button>
    </div>
  );
}
