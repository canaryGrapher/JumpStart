import { describeIndex, indexAge } from "../../../codeContext";

// Status strip for the project's code index. Until a project is indexed
// the assistant answers from the conversation alone, so this doubles as
// the prompt to build it.
export default function ContextBar({ status, building, progress, onBuild, onClear }) {
  const indexed = !!status?.indexed;

  if (building) {
    const { done = 0, total = 0, path = "" } = progress || {};
    const pct = total > 0 ? Math.round((done / total) * 100) : 0;
    return (
      <div className="chat-context building">
        <div className="chat-context-line">
          <strong>Indexing code…</strong>
          <span className="chat-context-path">{path || `${pct}%`}</span>
        </div>
        <div className="meter">
          <div style={{ width: `${pct}%` }} />
        </div>
      </div>
    );
  }

  if (!indexed) {
    return (
      <div className="chat-context empty">
        <span>
          This project isn't indexed. Build the code context so I can answer
          questions about how it's set up.
        </span>
        <button className="btn small primary" onClick={onBuild}>
          Index project
        </button>
      </div>
    );
  }

  return (
    <div className="chat-context">
      <span className="chat-context-dot" />
      <span className="chat-context-main">
        Code context: {describeIndex(status)}
      </span>
      {(status.preview || []).slice(0, 3).map((p) => (
        <span className="kb-pill" key={p}>
          {p}
        </span>
      ))}
      <span className="chat-context-age">{indexAge(status)}</span>
      <div className="spacer" />
      <button className="link-btn" onClick={onBuild}>
        Rescan
      </button>
      <button className="link-btn" onClick={onClear}>
        Clear
      </button>
    </div>
  );
}
