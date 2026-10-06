import { useEffect, useMemo, useState } from "react";
import { GetImportPath } from "../api";
import ImportConfigModal from "./ImportConfigModal";
import Icon, { ICONS } from "./Icon";
import ProjectIcon from "./ProjectIcon";
import { PortsTable, usePortMap, portStats } from "./PortsView";

const fmtAgo = (ms) => {
  if (!ms) return "never";
  const m = Math.floor((Date.now() - ms) / 60000);
  if (m < 1) return "just now";
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
};

const FLOW = [
  { id: "backlog", label: "Backlog", color: "var(--dash-slate)" },
  { id: "todo", label: "To do", color: "var(--dash-blue)" },
  { id: "inprogress", label: "In progress", color: "var(--dash-amber)" },
  { id: "testing", label: "Testing", color: "var(--dash-violet)" },
  { id: "done", label: "Done", color: "var(--dash-green)" },
];

function taskColumn(t) {
  if (t.done || t.status === "done") return "done";
  if (t.status && FLOW.some((f) => f.id === t.status)) return t.status;
  return "todo";
}

function Donut({ segments, size = 132, thickness = 18, center }) {
  const total = segments.reduce((s, x) => s + x.value, 0) || 1;
  const r = (size - thickness) / 2;
  const c = 2 * Math.PI * r;
  let offset = 0;
  return (
    <svg className="dash-donut" width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
      <circle
        cx={size / 2}
        cy={size / 2}
        r={r}
        fill="none"
        stroke="var(--card-soft)"
        strokeWidth={thickness}
      />
      {segments.map((seg) => {
        const len = (seg.value / total) * c;
        const el = (
          <circle
            key={seg.id}
            cx={size / 2}
            cy={size / 2}
            r={r}
            fill="none"
            stroke={seg.color}
            strokeWidth={thickness}
            strokeDasharray={`${len} ${c - len}`}
            strokeDashoffset={-offset}
            strokeLinecap="butt"
            transform={`rotate(-90 ${size / 2} ${size / 2})`}
          />
        );
        offset += len;
        return el;
      })}
      {center && (
        <foreignObject x={thickness} y={thickness} width={size - thickness * 2} height={size - thickness * 2}>
          <div className="dash-donut-center">{center}</div>
        </foreignObject>
      )}
    </svg>
  );
}

function UsageBars({ projects, onOpen }) {
  const rows = [...projects]
    .filter((p) => p.useCount > 0)
    .sort((a, b) => b.useCount - a.useCount)
    .slice(0, 6);
  const max = Math.max(...rows.map((p) => p.useCount), 1);
  if (rows.length === 0) {
    return <div className="dash-empty-inline">Open a project to start the activity chart.</div>;
  }
  return (
    <div className="dash-bars">
      {rows.map((p, i) => (
        <button
          type="button"
          className="dash-bar-row"
          key={p.id}
          style={{ "--i": i }}
          title={`${p.name} · ${p.useCount} starts`}
          onClick={() => onOpen && onOpen(p.id)}
        >
          <span className="dash-bar-label">{p.name}</span>
          <span className="dash-bar-track">
            <span
              className="dash-bar-fill"
              style={{ width: `${(p.useCount / max) * 100}%`, "--hue": `${(i * 48) % 360}` }}
            />
          </span>
          <span className="dash-bar-val">{p.useCount}</span>
        </button>
      ))}
    </div>
  );
}

function FlowPipeline({ counts }) {
  const max = Math.max(...FLOW.map((f) => counts[f.id] || 0), 1);
  return (
    <div className="dash-flow">
      {FLOW.map((f, i) => {
        const n = counts[f.id] || 0;
        return (
          <div className="dash-flow-step" key={f.id} style={{ "--i": i, "--c": f.color }}>
            <div className="dash-flow-node">
              <div
                className="dash-flow-ring"
                style={{
                  background: `conic-gradient(${f.color} ${(n / max) * 360}deg, var(--card-soft) 0)`,
                }}
              >
                <span>{n}</span>
              </div>
              <span className="dash-flow-label">{f.label}</span>
            </div>
            {i < FLOW.length - 1 && <div className="dash-flow-arrow" aria-hidden />}
          </div>
        );
      })}
    </div>
  );
}

export default function Dashboard({ projects, usage, onOpen, onViewAll, onReload, onError, onInfo }) {
  const [confPath, setConfPath] = useState("");
  const [showImport, setShowImport] = useState(false);
  const portEntries = usePortMap(onError);
  const ports = useMemo(() => portStats(portEntries), [portEntries]);

  useEffect(() => {
    GetImportPath().then(setConfPath).catch(() => {});
  }, []);

  const sys = usage.system || {};
  const runningCount = Object.keys(usage.procs || {}).length;
  const cpu = Math.round(sys.cpu || 0);
  const memPct = sys.totalMemMB ? Math.round((sys.usedMemMB / sys.totalMemMB) * 100) : 0;
  const procCount = projects.reduce((n, p) => n + (p.processes || []).length, 0);
  const linked = projects.filter((p) => p.github?.enabled).length;

  const recent = [...projects]
    .filter((p) => p.lastUsedAt)
    .sort((a, b) => b.lastUsedAt - a.lastUsedAt)
    .slice(0, 5);
  const mostUsed = [...projects]
    .filter((p) => p.useCount)
    .sort((a, b) => b.useCount - a.useCount)
    .slice(0, 5);

  const flowCounts = useMemo(() => {
    const acc = Object.fromEntries(FLOW.map((f) => [f.id, 0]));
    for (const p of projects) {
      for (const t of p.tasks || []) {
        acc[taskColumn(t)] = (acc[taskColumn(t)] || 0) + 1;
      }
    }
    return acc;
  }, [projects]);

  const taskTotal = FLOW.reduce((s, f) => s + (flowCounts[f.id] || 0), 0);
  const taskDone = flowCounts.done || 0;
  const donePct = taskTotal ? Math.round((taskDone / taskTotal) * 100) : 0;

  const donutSegs = FLOW.map((f) => ({
    id: f.id,
    value: flowCounts[f.id] || 0,
    color: f.color,
  })).filter((s) => s.value > 0);

  const tiles = [
    {
      key: "projects",
      label: "Projects",
      value: projects.length,
      hint: `${runningCount} running now`,
      tone: "blue",
      icon: ICONS.folder,
    },
    {
      key: "cpu",
      label: "System CPU",
      value: `${cpu}%`,
      hint: `${sys.numCpu || 0} cores`,
      tone: cpu > 80 ? "hot" : "amber",
      meter: cpu,
      icon: ICONS.bolt,
    },
    {
      key: "mem",
      label: "Memory",
      value: `${memPct}%`,
      hint: `${Math.round((sys.usedMemMB || 0) / 1024)} / ${Math.round((sys.totalMemMB || 0) / 1024)} GB`,
      tone: memPct > 85 ? "hot" : "violet",
      meter: memPct,
      icon: ICONS.layers,
    },
    {
      key: "tasks",
      label: "Tasks shipped",
      value: `${taskDone}/${taskTotal}`,
      hint: taskTotal ? `${donePct}% complete` : "No tasks yet",
      tone: "green",
      meter: donePct,
      icon: ICONS.sparkles,
    },
    {
      key: "procs",
      label: "Subprocesses",
      value: procCount,
      hint: `${runningCount} live`,
      tone: "cyan",
      icon: ICONS.terminal,
    },
    {
      key: "ports",
      label: "Open ports",
      value: ports.unique,
      hint: ports.conflicts ? `${ports.conflicts} conflicts` : `${ports.projects} projects`,
      tone: ports.conflicts ? "hot" : "slate",
      icon: ICONS.ports,
    },
  ];

  return (
    <div className="dash">
      <header className="dash-hero">
        <div>
          <p className="dash-eyebrow">Workspace overview</p>
          <h2 className="dash-title">Your JumpStart at a glance</h2>
          <p className="dash-lead">
            Projects, process load, task flow, and live ports — one colorful board.
          </p>
        </div>
        <div className="dash-hero-chips">
          {linked > 0 && (
            <span className="dash-chip gh">
              <Icon d={ICONS.github} /> {linked} GitHub-linked
            </span>
          )}
          <span className="dash-chip live">
            <span className="dash-live-dot" /> {runningCount} live
          </span>
        </div>
      </header>

      <div className="tiles">
        {tiles.map((t, i) => (
          <div className={`tile tone-${t.tone}`} key={t.key} style={{ "--i": i }}>
            <div className="tile-top">
              <span className="tile-icon">
                <Icon d={t.icon} />
              </span>
              <span className="tile-label">{t.label}</span>
            </div>
            <div className="tile-num">
              <span className="tile-value">{t.value}</span>
              <span className="delta-pill">{t.hint}</span>
            </div>
            {typeof t.meter === "number" && (
              <div className="meter">
                <div className={t.tone === "hot" ? "hot" : ""} style={{ width: `${Math.min(100, t.meter)}%` }} />
              </div>
            )}
          </div>
        ))}
      </div>

      <div className="dash-grid">
        <div className="panel dash-chart-panel">
          <div className="panel-head-row">
            <h3>Task flow</h3>
            <span className="dash-panel-tag">{taskTotal} cards</span>
          </div>
          <div className="sub">Kanban pipeline across every project</div>
          {taskTotal === 0 ? (
            <div className="dash-empty-inline">Enable Tasks on a project to see the flow.</div>
          ) : (
            <FlowPipeline counts={flowCounts} />
          )}
        </div>

        <div className="panel dash-chart-panel">
          <div className="panel-head-row">
            <h3>Completion mix</h3>
            <span className="dash-panel-tag">{donePct}%</span>
          </div>
          <div className="sub">Share of work by column</div>
          <div className="dash-donut-wrap">
            <Donut
              segments={donutSegs.length ? donutSegs : [{ id: "empty", value: 1, color: "var(--card-soft)" }]}
              center={
                <>
                  <strong>{donePct}%</strong>
                  <span>done</span>
                </>
              }
            />
            <ul className="dash-legend">
              {FLOW.map((f) => (
                <li key={f.id}>
                  <i style={{ background: f.color }} />
                  <span>{f.label}</span>
                  <b>{flowCounts[f.id] || 0}</b>
                </li>
              ))}
            </ul>
          </div>
        </div>

        <div className="panel">
          <div className="panel-head-row">
            <h3>Recent projects</h3>
            <button className="link-btn with-icon" onClick={onViewAll}>
              All projects <Icon d={ICONS.chevron} />
            </button>
          </div>
          <div className="sub">Pick up where you left off</div>
          {recent.map((p) => (
            <div className="quick-row" key={p.id} onClick={() => onOpen(p.id)}>
              <ProjectIcon project={p} className="avatar" />
              <span className="q-text">
                <span className="q-name">{p.name}</span>
                <span className="q-sub">{(p.processes || []).length} subprocesses</span>
                {p.description && <span className="q-desc">{p.description}</span>}
              </span>
              <span className="q-meta">{fmtAgo(p.lastUsedAt)}</span>
            </div>
          ))}
          {recent.length === 0 && <div className="dash-empty-inline">Nothing started yet.</div>}
        </div>

        <div className="panel">
          <h3>Activity by project</h3>
          <div className="sub">Starts over time — your go-to workspaces</div>
          <UsageBars projects={projects} onOpen={onOpen} />
          {mostUsed.length > 0 && (
            <div className="dash-mini-list">
              {mostUsed.slice(0, 3).map((p) => (
                <button type="button" className="dash-mini-chip" key={p.id} onClick={() => onOpen(p.id)}>
                  <ProjectIcon project={p} className="avatar sm" />
                  {p.name}
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="panel wide ports-panel">
          <div className="panel-head-row">
            <h3>Live ports</h3>
            <div className="dash-port-stats">
              <span>{ports.unique} unique</span>
              <span>{ports.total} bindings</span>
              {ports.conflicts > 0 && <span className="warn">{ports.conflicts} conflicts</span>}
            </div>
          </div>
          <div className="sub">Every listening port from managed subprocesses</div>
          <PortsTable entries={portEntries} compact />
        </div>

        <div className="panel wide dash-import">
          <div className="panel-head-row">
            <h3>Config import</h3>
            <button className="btn primary" onClick={() => setShowImport(true)}>
              Import config
            </button>
          </div>
          <div className="sub">
            Add projects with the block builder, by pasting JSON, or from a file.
          </div>
          <div className="conf-path">{confPath || "…"}</div>
          <span className="hint">
            The importer lets you build blocks, paste and edit JSON, or load a file. A copy is
            saved to the path above. &quot;Copy prompt&quot; inside gives an AI agent instructions to
            generate the JSON for you.
          </span>
        </div>
      </div>

      {showImport && (
        <ImportConfigModal
          onClose={() => setShowImport(false)}
          onInfo={onInfo}
          onError={onError}
          onReload={onReload}
        />
      )}
    </div>
  );
}
