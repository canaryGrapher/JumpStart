import { StartContainer, StopContainer, RemoveContainer } from "../../api";

// isRunning treats compose "running" / docker "Up …" states as active.
const isRunning = (c) => {
  const s = (c.state || "").toLowerCase();
  if (s) return s === "running" || s.startsWith("up");
  return (c.status || "").toLowerCase().startsWith("up");
};

export default function ContainerList({ containers, busy, onAction }) {
  if (!containers.length) {
    return <div className="sub">No containers. Bring the stack up to create them.</div>;
  }

  return (
    <div className="dk-table">
      <div className="dk-row dk-head">
        <span>Service</span>
        <span>Image</span>
        <span>State</span>
        <span>Ports</span>
        <span className="dk-actions-col">Actions</span>
      </div>
      {containers.map((c) => {
        const running = isRunning(c);
        return (
          <div className="dk-row" key={c.id || c.name}>
            <span className="dk-name">
              <span className={`dot ${running ? "on" : "off"}`} />
              {c.service || c.name}
            </span>
            <span className="dk-muted" title={c.image}>{c.image}</span>
            <span>
              <span className={`status-pill ${running ? "on" : "off"}`}>
                {c.status || c.state || "—"}
              </span>
            </span>
            <span className="dk-muted">{c.ports || "—"}</span>
            <span className="dk-actions-col">
              {running ? (
                <button className="btn small" disabled={busy} onClick={() => onAction("Stop", () => StopContainer(c.id))}>
                  Stop
                </button>
              ) : (
                <button className="btn small primary" disabled={busy} onClick={() => onAction("Start", () => StartContainer(c.id))}>
                  Start
                </button>
              )}
              <button className="btn small danger" disabled={busy} onClick={() => onAction("Remove", () => RemoveContainer(c.id))}>
                Remove
              </button>
            </span>
          </div>
        );
      })}
    </div>
  );
}
