import { useEffect, useState } from "react";
import { GitGraphLog } from "../../api";
import { layoutGraph, laneColor } from "./graphLayout";

const ROW_H = 30;
const COL_W = 16;
const PAD_X = 10;
const DOT_R = 4;

const fmtShort = (iso) => {
  if (!iso) return "";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "";
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
};

export default function BranchTimeline({ projectRoot, refreshKey, onError }) {
  const [commits, setCommits] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    GitGraphLog(projectRoot, 60)
      .then((c) => alive && setCommits(c || []))
      .catch((e) => alive && onError && onError(String(e)))
      .finally(() => alive && setLoading(false));
    return () => {
      alive = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectRoot, refreshKey]);

  if (loading && !commits) return <div className="git-timeline-empty sub">Loading history…</div>;
  if (!commits || commits.length === 0)
    return <div className="git-timeline-empty sub">No commits yet.</div>;

  const { colOf, edges, maxCol } = layoutGraph(commits);
  const graphW = (maxCol + 1) * COL_W + PAD_X * 2;
  const height = commits.length * ROW_H;
  const cx = (col) => PAD_X + col * COL_W;
  const cy = (row) => row * ROW_H + ROW_H / 2;

  return (
    <div className="git-timeline">
      <svg
        className="git-timeline-graph"
        width={graphW}
        height={height}
        style={{ minWidth: graphW }}
      >
        {edges.map((e, i) => {
          const x1 = cx(colOf[e.fromRow]);
          const y1 = cy(e.fromRow);
          const x2 = cx(colOf[e.toRow]);
          const y2 = cy(e.toRow);
          const d =
            x1 === x2
              ? `M ${x1} ${y1} L ${x2} ${y2}`
              : `M ${x1} ${y1} C ${x1} ${(y1 + y2) / 2}, ${x2} ${(y1 + y2) / 2}, ${x2} ${y2}`;
          return (
            <path
              key={i}
              d={d}
              fill="none"
              stroke={laneColor(colOf[e.toRow])}
              strokeWidth="1.6"
              opacity="0.8"
            />
          );
        })}
        {commits.map((c, row) => (
          <circle
            key={c.hash}
            cx={cx(colOf[row])}
            cy={cy(row)}
            r={c.merge ? DOT_R - 1 : DOT_R}
            fill={c.merge ? "var(--card-solid)" : laneColor(colOf[row])}
            stroke={laneColor(colOf[row])}
            strokeWidth="1.8"
          />
        ))}
      </svg>

      <div className="git-timeline-rows">
        {commits.map((c) => (
          <div className="git-timeline-row" key={c.hash} style={{ height: ROW_H }}>
            {c.refs && c.refs.length > 0 && (
              <span className="git-timeline-refs">
                {c.refs.map((r) => (
                  <span
                    key={r}
                    className={`git-ref-badge ${r.startsWith("origin/") ? "remote" : ""}`}
                  >
                    {r}
                  </span>
                ))}
              </span>
            )}
            <span className="git-timeline-subject" title={c.subject}>
              {c.subject}
            </span>
            <span className="git-timeline-meta">
              {c.short} · {c.author} · {fmtShort(c.iso)}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
