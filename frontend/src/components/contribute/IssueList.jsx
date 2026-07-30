import { useEffect, useState } from "react";
import { ListRepoIssues, BrowserOpenURL } from "../../api";

const FILTERS = [
  { id: "good first issue", label: "Good first issues" },
  { id: "", label: "All open" },
];

// Read-only list of open issues from the upstream repo, so contributors can
// find something to pick up without leaving the app.
export default function IssueList() {
  const [filter, setFilter] = useState("good first issue");
  const [issues, setIssues] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");
    ListRepoIssues(filter, 10)
      .then((list) => !cancelled && setIssues(list || []))
      .catch((e) => !cancelled && setError(String(e)))
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [filter]);

  return (
    <div className="prefs-row col">
      <label>Open issues</label>
      <div className="seg">
        {FILTERS.map((f) => (
          <button
            key={f.label}
            className={filter === f.id ? "on" : ""}
            onClick={() => setFilter(f.id)}
          >
            {f.label}
          </button>
        ))}
      </div>

      {loading ? (
        <span className="row-hint">Loading…</span>
      ) : error ? (
        <span className="row-hint">{error}</span>
      ) : issues.length === 0 ? (
        <span className="row-hint">
          {filter ? "No good first issues right now." : "No open issues."}
        </span>
      ) : (
        <ul className="contrib-issues">
          {issues.map((i) => (
            <li key={i.number}>
              <button className="contrib-issue" onClick={() => BrowserOpenURL(i.url)}>
                <span className="ci-num">#{i.number}</span>
                <span className="ci-title">{i.title}</span>
                {i.labels.slice(0, 2).map((l) => (
                  <span key={l} className="ci-label">
                    {l}
                  </span>
                ))}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
