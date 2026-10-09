import { useEffect, useState } from "react";
import { DashboardWidgetData } from "../../api";
import { formatDue } from "../../dueDates";
import { openInApp } from "../../navigate";
import Donut from "./Donut";

// Fallbacks keep colors when a widget renders outside .dash (editor preview).
const PALETTE = [
  "var(--dash-blue, #007aff)", "var(--dash-amber, #ff9500)", "var(--dash-violet, #af52de)", "var(--dash-green, #34c759)",
  "var(--dash-slate, #8e8e93)", "var(--dash-hot, #ff3b30)", "var(--dash-cyan, #32ade6)",
];

// Loads a task widget's data from the backend (the same computation MCP and
// Raycast use) and refreshes when tasks change or the window regains focus.
export function useWidgetData(widget, version) {
  const [data, setData] = useState(null);
  const key = JSON.stringify(widget);
  useEffect(() => {
    let live = true;
    const load = () =>
      DashboardWidgetData(widget)
        .then((d) => live && setData(d))
        .catch((e) => live && setData({ error: String(e), tasks: [], total: 0 }));
    load();
    window.addEventListener("focus", load);
    return () => {
      live = false;
      window.removeEventListener("focus", load);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, version]);
  return data;
}

function TaskRows({ tasks, limit }) {
  return (
    <ul className="dw-tasks">
      {tasks.slice(0, limit).map((t) => (
        <li key={`${t.projectId}/${t.taskId}`}>
          <button type="button" className="dw-task" onClick={() => openInApp(t.projectId, t.taskId)}>
            <span className={`dw-dot type-${t.type}`} aria-hidden />
            <span className="dw-task-main">
              <span className="dw-task-title">{t.title}</span>
              <span className="dw-task-meta">
                {t.projectName}
                {t.statusLabel ? ` · ${t.statusLabel}` : ""}
              </span>
            </span>
            {t.dueDate && <span className={`dw-due ${t.pastDue ? "late" : ""}`}>{formatDue(t.dueDate)}</span>}
          </button>
        </li>
      ))}
    </ul>
  );
}

function Bars({ groups }) {
  const max = Math.max(1, ...groups.map((g) => g.count));
  return (
    <div className="dw-bars">
      {groups.map((g, i) => (
        <div className="dw-bar-row" key={g.key || g.label}>
          <span className="dw-bar-label" title={g.label}>{g.label}</span>
          <span className="dw-bar-track">
            <span className="dw-bar-fill" style={{ width: `${(g.count / max) * 100}%`, background: PALETTE[i % PALETTE.length] }} />
          </span>
          <span className="dw-bar-val">{g.count}</span>
        </div>
      ))}
    </div>
  );
}

// Renders due / saved-filter / custom widgets.
export default function TaskWidget({ widget, version }) {
  const data = useWidgetData(widget, version);
  if (!data) return <div className="dash-empty-inline">Loading…</div>;
  if (data.error)
    return (
      <>
        <div className="panel-head-row">
          <h3>{widget.title || data.title || "Widget"}</h3>
        </div>
        <div className="dw-error">{data.error}</div>
      </>
    );
  const display = widget.display || "list";
  const groups = data.groups || [];
  const limit = widget.size === "s" ? 5 : 8;
  const more = data.total - Math.min(limit, data.tasks.length);

  let body;
  if (data.total === 0) {
    body = <div className="dash-empty-inline">{widget.type === "due" ? "Nothing due." : "No matching tasks."}</div>;
  } else if (display === "count") {
    body = (
      <div className="dw-count">
        <strong>{data.total}</strong>
        <span>task{data.total === 1 ? "" : "s"}</span>
        {groups.length > 0 && <Bars groups={groups.slice(0, 4)} />}
      </div>
    );
  } else if (display === "bar" && groups.length) {
    body = <Bars groups={groups} />;
  } else if (display === "donut" && groups.length) {
    body = (
      <div className="dash-donut-wrap">
        <Donut
          size={112}
          thickness={16}
          segments={groups.map((g, i) => ({ id: g.key || g.label, value: g.count, color: PALETTE[i % PALETTE.length] }))}
          center={<><strong>{data.total}</strong><span>tasks</span></>}
        />
        <ul className="dash-legend">
          {groups.map((g, i) => (
            <li key={g.key || g.label}>
              <i style={{ background: PALETTE[i % PALETTE.length] }} />
              <span>{g.label}</span>
              <b>{g.count}</b>
            </li>
          ))}
        </ul>
      </div>
    );
  } else if (display === "table" && groups.length) {
    body = (
      <table className="dw-table">
        <tbody>
          {groups.map((g) => (
            <tr key={g.key || g.label}>
              <td>{g.label}</td>
              <td>{g.count}</td>
            </tr>
          ))}
        </tbody>
      </table>
    );
  } else {
    body = (
      <>
        <TaskRows tasks={data.tasks} limit={limit} />
        {more > 0 && <div className="dw-more">+{more} more</div>}
      </>
    );
  }
  return (
    <>
      <div className="panel-head-row">
        <h3>{widget.title || data.title}</h3>
        <span className="dash-panel-tag">{data.total}</span>
      </div>
      <div className="dw-body" data-total={data.total}>
        {body}
      </div>
    </>
  );
}
