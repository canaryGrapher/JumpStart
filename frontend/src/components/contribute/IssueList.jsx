import { useEffect, useState } from "react";
import { ListRepoIssues, BrowserOpenURL } from "../../api";

// Read-only list of open issues from the upstream repo, so contributors can
// find something to pick up without leaving the app.
export default function IssueList() {
  const [issues, setIssues] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");
    ListRepoIssues("", 10)
      .then((list) => !cancelled && setIssues(list || []))
      .catch((e) => !cancelled && setError(String(e)))
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div className="prefs-row col">
      <label>Open issues</label>

      {loading ? (
        <span className="row-hint">Loading…</span>
      ) : error ? (
        <span className="row-hint">{error}</span>
      ) : issues.length === 0 ? (
        <span className="row-hint">No open issues.</span>
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
