import { useEffect, useMemo, useState } from "react";
import { createPortal } from "react-dom";
import { ExportWidgets, GetDashboard, GetImportPath, ResetDashboard, SaveDashboard, SaveTextFile } from "../api";
import ConfirmDialog from "./ConfirmDialog";
import TaskWidget from "./dashboard/TaskWidget";
import HtmlWidget from "./dashboard/HtmlWidget";
import WidgetEditor, { BUILTINS, DUE_RANGES } from "./dashboard/WidgetEditor";
import ImportWidgetsModal from "./dashboard/ImportWidgetsModal";
import ImportConfigModal from "./ImportConfigModal";
import Icon, { ICONS } from "./Icon";
import ProjectIcon from "./ProjectIcon";
import { PortsTable, usePortMap, portStats } from "./PortsView";
import Donut from "./dashboard/Donut";

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

  // --- Customizable layout ---
  const [layout, setLayout] = useState(null);
  const [draft, setDraft] = useState(null); // layout being edited, or null
  const [editingWidget, setEditingWidget] = useState(null); // { index, widget } | { index: -1 }
  const [importing, setImporting] = useState(false);
  const [confirmReset, setConfirmReset] = useState(false);
  const [dragId, setDragId] = useState(null);
  const [overId, setOverId] = useState(null);

  useEffect(() => {
    GetDashboard()
      .then(setLayout)
      .catch((e) => onError(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const widgets = (draft || layout)?.widgets || [];
  const setWidgets = (fn) => setDraft((d) => ({ ...d, widgets: fn(d.widgets) }));
  const startEdit = () => setDraft(JSON.parse(JSON.stringify(layout)));
  const finishEdit = async () => {
    try {
      const saved = await SaveDashboard(draft);
      setLayout(saved);
      setDraft(null);
    } catch (e) {
      onError(String(e));
    }
  };
  const move = (from, to) =>
    setWidgets((ws) => {
      if (to < 0 || to >= ws.length || from === to) return ws;
      const next = [...ws];
      next.splice(to, 0, next.splice(from, 1)[0]);
      return next;
    });
  const exportWidgets = async (list, name) => {
    try {
      const text = await ExportWidgets(list);
      const slug = (name || "widgets").toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "") || "widgets";
      const path = await SaveTextFile(`${slug}.jumpstart-widget.json`, text, "Export widget");
      if (path) onInfo(`Saved ${path}`);
    } catch (e) {
      onError(String(e));
    }
  };

  const panelTitle = (w, fallback) => w.title || fallback;
  const nameOf = (w) =>
    w.title ||
    BUILTINS.find(([t]) => t === w.type)?.[1] ||
    DUE_RANGES.find(([r]) => r === w.range)?.[1] ||
    (w.type === "filter" ? "Saved filter" : "Custom widget");
  const renderBuiltin = (w) => {
    switch (w.type) {
      case "stats":
        return (
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
        );
      case "flow":
        return (
          <div className="panel dash-chart-panel">
            <div className="panel-head-row">
              <h3>{panelTitle(w, "Task flow")}</h3>
              <span className="dash-panel-tag">{taskTotal} cards</span>
            </div>
            <div className="sub">Kanban pipeline across every project</div>
            {taskTotal === 0 ? (
              <div className="dash-empty-inline">Enable Tasks on a project to see the flow.</div>
            ) : (
              <FlowPipeline counts={flowCounts} />
            )}
          </div>
        );
      case "donut":
        return (
          <div className="panel dash-chart-panel">
            <div className="panel-head-row">
              <h3>{panelTitle(w, "Completion mix")}</h3>
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
        );
      case "recent":
        return (
          <div className="panel">
            <div className="panel-head-row">
              <h3>{panelTitle(w, "Recent projects")}</h3>
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
        );
      case "activity":
        return (
          <div className="panel">
            <h3>{panelTitle(w, "Activity by project")}</h3>
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
        );
      case "ports":
        return (
          <div className="panel ports-panel">
            <div className="panel-head-row">
              <h3>{panelTitle(w, "Live ports")}</h3>
              <div className="dash-port-stats">
                <span>{ports.unique} unique</span>
                <span>{ports.total} bindings</span>
                {ports.conflicts > 0 && <span className="warn">{ports.conflicts} conflicts</span>}
              </div>
            </div>
            <div className="sub">Every listening port from managed subprocesses</div>
            <PortsTable entries={portEntries} compact />
          </div>
        );
      case "import":
        return (
          <div className="panel dash-import">
            <div className="panel-head-row">
              <h3>{panelTitle(w, "Config import")}</h3>
              <button className="btn primary" onClick={() => setShowImport(true)}>
                Import config
              </button>
            </div>
            <div className="sub">Add projects with the block builder, by pasting JSON, or from a file.</div>
            <div className="conf-path">{confPath || "…"}</div>
            <span className="hint">
              The importer lets you build blocks, paste and edit JSON, or load a file. A copy is saved to the path
              above. &quot;Copy prompt&quot; inside gives an AI agent instructions to generate the JSON for you.
            </span>
          </div>
        );
      case "html":
        return (
          <div className="panel dash-widget-panel">
            <div className="panel-head-row">
              <h3>{panelTitle(w, "Custom widget")}</h3>
              <span className="dash-panel-tag">code</span>
            </div>
            <HtmlWidget widget={w} version={projects} />
          </div>
        );
      default:
        return (
          <div className="panel dash-widget-panel">
            <TaskWidget widget={w} version={projects} />
          </div>
        );
    }
  };

  const editing = !!draft;
  return (
    <div className={`dash ${editing ? "dash-editing" : ""}`}>
      <header className="dash-hero">
        <div>
          <p className="dash-eyebrow">Workspace overview</p>
          <h2 className="dash-title">Your JumpStart at a glance</h2>
          <p className="dash-lead">
            Projects, process load, task flow, due dates and live ports — arranged your way.
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

      <div className="dash-toolbar" role="toolbar" aria-label="Dashboard">
        {editing ? (
          <>
            <button type="button" className="btn small primary" onClick={() => setEditingWidget({ index: -1 })}>
              + Add widget
            </button>
            <button type="button" className="btn small" onClick={() => setImporting(true)}>
              Import…
            </button>
            <button type="button" className="btn small" disabled={!widgets.length} onClick={() => exportWidgets(widgets, "dashboard")}>
              Export all…
            </button>
            <button type="button" className="link-btn" onClick={() => setConfirmReset(true)}>
              Reset to default
            </button>
            <span className="dash-toolbar-hint">Drag widgets to reorder.</span>
            <span style={{ flex: 1 }} />
            <button type="button" className="btn small" onClick={() => setDraft(null)}>
              Cancel
            </button>
            <button type="button" className="btn small primary" onClick={finishEdit}>
              Done
            </button>
          </>
        ) : (
          <>
            <span style={{ flex: 1 }} />
            <button type="button" className="btn small" onClick={startEdit} disabled={!layout}>
              Edit dashboard
            </button>
          </>
        )}
      </div>

      {layout && widgets.length === 0 && (
        <div className="dash-empty">
          <p>Your dashboard has no widgets.</p>
          {!editing && (
            <button type="button" className="btn small" onClick={startEdit}>
              Add widgets
            </button>
          )}
        </div>
      )}

      <div className="dash-widgets">
        {widgets.map((w, i) => (
          <section
            key={w.id}
            data-widget={w.id}
            data-type={w.type}
            aria-label={nameOf(w)}
            className={`dash-widget size-${w.size || "m"} ${editing ? "editing" : ""} ${overId === w.id && dragId !== w.id ? "drop-target" : ""} ${dragId === w.id ? "dragging" : ""}`}
            draggable={editing}
            onDragStart={(e) => {
              if (!editing) return;
              setDragId(w.id);
              e.dataTransfer.effectAllowed = "move";
              e.dataTransfer.setData("text/plain", w.id);
            }}
            onDragOver={(e) => {
              if (!editing || !dragId) return;
              e.preventDefault();
              setOverId(w.id);
            }}
            onDrop={(e) => {
              if (!editing || !dragId) return;
              e.preventDefault();
              move(widgets.findIndex((x) => x.id === dragId), i);
              setDragId(null);
              setOverId(null);
            }}
            onDragEnd={() => {
              setDragId(null);
              setOverId(null);
            }}
          >
            {editing && (
              <div className="dw-toolbar">
                <span className="dw-handle" aria-hidden title="Drag to move">⠿</span>
                <div className="seg dw-size" role="radiogroup" aria-label={`Size of ${nameOf(w)}`}>
                  {["s", "m", "l"].map((sz) => (
                    <button
                      type="button"
                      key={sz}
                      role="radio"
                      aria-checked={(w.size || "m") === sz}
                      className={(w.size || "m") === sz ? "on" : ""}
                      onClick={() => setWidgets((ws) => ws.map((x) => (x.id === w.id ? { ...x, size: sz } : x)))}
                    >
                      {sz.toUpperCase()}
                    </button>
                  ))}
                </div>
                <button type="button" className="icon-text" aria-label="Move earlier" disabled={i === 0} onClick={() => move(i, i - 1)}>
                  ←
                </button>
                <button type="button" className="icon-text" aria-label="Move later" disabled={i === widgets.length - 1} onClick={() => move(i, i + 1)}>
                  →
                </button>
                <span style={{ flex: 1 }} />
                <button type="button" className="link-btn" onClick={() => setEditingWidget({ index: i, widget: w })}>
                  Edit
                </button>
                <button type="button" className="link-btn" onClick={() => exportWidgets([w], nameOf(w))}>
                  Export
                </button>
                <button
                  type="button"
                  className="link-btn danger"
                  aria-label={`Remove ${nameOf(w)}`}
                  onClick={() => setWidgets((ws) => ws.filter((x) => x.id !== w.id))}
                >
                  Remove
                </button>
              </div>
            )}
            {renderBuiltin(w)}
          </section>
        ))}
      </div>

      {editingWidget && (
        <WidgetEditor
          initial={editingWidget.index >= 0 ? editingWidget.widget : null}
          projects={projects}
          onClose={() => setEditingWidget(null)}
          onSave={(w) => {
            if (editingWidget.index >= 0) setWidgets((ws) => ws.map((x, j) => (j === editingWidget.index ? w : x)));
            else setWidgets((ws) => [...ws, { ...w, id: "" }]);
            setEditingWidget(null);
          }}
        />
      )}
      {importing && (
        <ImportWidgetsModal
          onClose={() => setImporting(false)}
          onAdd={(ws) => {
            setWidgets((cur) => [...cur, ...ws]);
            setImporting(false);
          }}
        />
      )}
      {confirmReset &&
        createPortal(
        <ConfirmDialog
          title="Reset the dashboard?"
          body="Your widgets are replaced by the default layout. Export them first if you want to keep any."
          confirmLabel="Reset dashboard"
          danger
          onConfirm={async () => {
            try {
              setLayout(await ResetDashboard());
              setDraft(null);
            } catch (e) {
              onError(String(e));
            }
            setConfirmReset(false);
          }}
          onCancel={() => setConfirmReset(false)}
        />,
          document.body
        )}

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
