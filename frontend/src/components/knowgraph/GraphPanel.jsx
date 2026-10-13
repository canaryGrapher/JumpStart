import { useEffect, useState } from "react";
import {
  KnowGraphInfo,
  KnowGraphSearch,
  KnowGraphNodes,
  KnowGraphNode,
  KnowGraphCommunities,
  KnowGraphFlows,
} from "../../api";

const SIDE_VIEWS = [
  { id: "overview", label: "Overview" },
  { id: "communities", label: "Communities" },
  { id: "flows", label: "Flows" },
  { id: "browse", label: "Browse" },
  { id: "search", label: "Search" },
];

const KIND_FILTERS = ["", "File", "Class", "Function", "Test", "Type"];

function formatBytes(n) {
  if (!n || n < 0) return "—";
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

function NodeTable({ rows, onSelect, selected }) {
  if (!rows?.length) {
    return <p className="kg-empty">No nodes.</p>;
  }
  return (
    <div className="kg-table-wrap">
      <table className="kg-table">
        <thead>
          <tr>
            <th>Kind</th>
            <th>Name</th>
            <th>File</th>
            <th>Lang</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((n) => (
            <tr
              key={n.qualifiedName || n.id}
              className={selected === n.qualifiedName ? "active" : ""}
              onClick={() => onSelect(n.qualifiedName)}
              title={n.signature || n.qualifiedName}
            >
              <td>
                <span className="kg-pill">{n.kind}</span>
              </td>
              <td className="mono">{n.name}</td>
              <td className="mono dim" title={n.filePath}>
                {n.filePath}
                {n.lineStart ? `:${n.lineStart}` : ""}
              </td>
              <td>{n.language || "—"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function NodeDetail({ detail, onSelect }) {
  if (!detail) return null;
  const { node, outgoing, incoming } = detail;
  return (
    <div className="kg-detail">
      <header className="kg-detail-head">
        <h3 className="mono">{node.name}</h3>
        <span className="kg-pill">{node.kind}</span>
      </header>
      <dl className="kg-meta">
        <div>
          <dt>Qualified</dt>
          <dd className="mono">{node.qualifiedName}</dd>
        </div>
        <div>
          <dt>File</dt>
          <dd className="mono">
            {node.filePath}
            {node.lineStart ? `:${node.lineStart}` : ""}
            {node.lineEnd && node.lineEnd !== node.lineStart ? `–${node.lineEnd}` : ""}
          </dd>
        </div>
        {node.signature ? (
          <div>
            <dt>Signature</dt>
            <dd className="mono">{node.signature}</dd>
          </div>
        ) : null}
        {node.language ? (
          <div>
            <dt>Language</dt>
            <dd>{node.language}</dd>
          </div>
        ) : null}
      </dl>

      <EdgeList title="Outgoing" edges={outgoing} field="targetQualified" onSelect={onSelect} />
      <EdgeList title="Incoming" edges={incoming} field="sourceQualified" onSelect={onSelect} />
    </div>
  );
}

function EdgeList({ title, edges, field, onSelect }) {
  if (!edges?.length) return null;
  return (
    <div className="kg-edges">
      <h4>{title}</h4>
      <ul>
        {edges.map((e) => (
          <li key={e.id}>
            <span className="kg-pill soft">{e.kind}</span>
            <button type="button" className="link mono" onClick={() => onSelect(e[field])} title={e[field]}>
              {e[field]}
            </button>
            {e.filePath ? (
              <span className="dim mono">
                {e.filePath}
                {e.line ? `:${e.line}` : ""}
              </span>
            ) : null}
          </li>
        ))}
      </ul>
    </div>
  );
}

export default function GraphPanel({ project, onError }) {
  const root = project.root;
  const [info, setInfo] = useState(null);
  const [view, setView] = useState("overview");
  const [loading, setLoading] = useState(true);
  const [kind, setKind] = useState("");
  const [nodes, setNodes] = useState([]);
  const [communities, setCommunities] = useState([]);
  const [flows, setFlows] = useState([]);
  const [query, setQuery] = useState("");
  const [hits, setHits] = useState([]);
  const [selected, setSelected] = useState("");
  const [detail, setDetail] = useState(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let active = true;
    setLoading(true);
    KnowGraphInfo(root)
      .then((i) => active && setInfo(i))
      .catch((e) => active && onError(String(e)))
      .finally(() => active && setLoading(false));
    return () => {
      active = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [root]);

  useEffect(() => {
    if (!info?.browseOk) return;
    let active = true;
    if (view === "communities") {
      KnowGraphCommunities(root)
        .then((c) => active && setCommunities(c || []))
        .catch((e) => active && onError(String(e)));
    } else if (view === "flows") {
      KnowGraphFlows(root, 50)
        .then((f) => active && setFlows(f || []))
        .catch((e) => active && onError(String(e)));
    } else if (view === "browse") {
      setBusy(true);
      KnowGraphNodes(root, kind, 100, 0)
        .then((n) => active && setNodes(n || []))
        .catch((e) => active && onError(String(e)))
        .finally(() => active && setBusy(false));
    }
    return () => {
      active = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [root, view, kind, info?.browseOk]);

  useEffect(() => {
    if (!selected || !info?.browseOk) {
      setDetail(null);
      return;
    }
    let active = true;
    KnowGraphNode(root, selected)
      .then((d) => active && setDetail(d))
      .catch((e) => {
        if (active) {
          setDetail(null);
          onError(String(e));
        }
      });
    return () => {
      active = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [root, selected, info?.browseOk]);

  const runSearch = async (e) => {
    e?.preventDefault?.();
    if (!info?.browseOk) return;
    setBusy(true);
    try {
      const r = await KnowGraphSearch(root, query, 50);
      setHits(r || []);
      setView("search");
    } catch (err) {
      onError(String(err));
    } finally {
      setBusy(false);
    }
  };

  if (loading) {
    return (
      <div className="kg-panel">
        <p className="kg-loading">Loading knowledge graph…</p>
      </div>
    );
  }

  if (!info?.present) {
    return (
      <div className="kg-panel">
        <div className="empty">
          <p>
            No knowledge graph found. Add a <code>.code-review-graph</code> folder (or{" "}
            <code>.knowledge-graph</code>) with a graph database to enable this tab.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="kg-panel">
      <aside className="kg-sidebar" aria-label="Knowledge graph sections">
        <div className="kg-sidebar-head">Graph</div>
        <ul className="kg-nav">
          {SIDE_VIEWS.map((v) => (
            <li key={v.id}>
              <button
                type="button"
                className={view === v.id ? "active" : ""}
                onClick={() => setView(v.id)}
                disabled={v.id !== "overview" && !info.browseOk}
              >
                {v.label}
              </button>
            </li>
          ))}
        </ul>
      </aside>

      <div className="kg-main">
        {view === "overview" && (
          <section className="kg-section">
            <header className="kg-content-head">
              <h1>Knowledge graph</h1>
              <span className="mono dim">{info.relPath}</span>
            </header>
            <div className="kg-stat-grid">
              <Stat label="Kind" value={info.kind} />
              <Stat label="Database" value={info.dbRelPath || "—"} mono />
              <Stat label="Size" value={formatBytes(info.dbSize)} />
              <Stat label="Updated" value={info.dbMTime ? new Date(info.dbMTime).toLocaleString() : "—"} />
              <Stat label="Nodes" value={info.browseOk ? String(info.nodeCount) : "—"} />
              <Stat label="Edges" value={info.browseOk ? String(info.edgeCount) : "—"} />
              <Stat label="Communities" value={info.browseOk ? String(info.communityCount) : "—"} />
              <Stat label="Flows" value={info.browseOk ? String(info.flowCount) : "—"} />
            </div>

            {!info.browseOk && (
              <p className="kg-warn">
                Browse unavailable{info.browseError ? `: ${info.browseError}` : ". Database missing or locked."}
              </p>
            )}

            {info.nodesByKind && Object.keys(info.nodesByKind).length > 0 && (
              <div className="kg-block">
                <h3>Nodes by kind</h3>
                <div className="kg-bars">
                  {Object.entries(info.nodesByKind)
                    .sort((a, b) => b[1] - a[1])
                    .map(([k, n]) => (
                      <Bar key={k} label={k} value={n} max={info.nodeCount || 1} />
                    ))}
                </div>
              </div>
            )}

            {info.edgesByKind && Object.keys(info.edgesByKind).length > 0 && (
              <div className="kg-block">
                <h3>Edges by kind</h3>
                <div className="kg-bars">
                  {Object.entries(info.edgesByKind)
                    .sort((a, b) => b[1] - a[1])
                    .map(([k, n]) => (
                      <Bar key={k} label={k} value={n} max={info.edgeCount || 1} />
                    ))}
                </div>
              </div>
            )}

            {info.agentFiles?.length > 0 && (
              <div className="kg-block">
                <h3>Related agent files</h3>
                <ul className="kg-file-list">
                  {info.agentFiles.map((f) => (
                    <li key={f} className="mono">
                      {f}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {info.metadata && Object.keys(info.metadata).length > 0 && (
              <div className="kg-block">
                <h3>Metadata</h3>
                <dl className="kg-meta">
                  {Object.entries(info.metadata).map(([k, v]) => (
                    <div key={k}>
                      <dt>{k}</dt>
                      <dd className="mono">{v}</dd>
                    </div>
                  ))}
                </dl>
              </div>
            )}
          </section>
        )}

        {view === "communities" && (
          <section className="kg-section">
            <header className="kg-content-head">
              <h1>Communities</h1>
            </header>
            {!communities.length ? (
              <p className="kg-empty">No communities in this graph.</p>
            ) : (
              <div className="kg-table-wrap">
                <table className="kg-table">
                  <thead>
                    <tr>
                      <th>Name</th>
                      <th>Size</th>
                      <th>Cohesion</th>
                      <th>Language</th>
                      <th>Description</th>
                    </tr>
                  </thead>
                  <tbody>
                    {communities.map((c) => (
                      <tr key={c.id} title={c.description || c.name}>
                        <td>{c.name}</td>
                        <td>{c.size}</td>
                        <td>{c.cohesion?.toFixed?.(3) ?? c.cohesion}</td>
                        <td>{c.dominantLanguage || "—"}</td>
                        <td className="dim">{c.description || "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        )}

        {view === "flows" && (
          <section className="kg-section">
            <header className="kg-content-head">
              <h1>Flows</h1>
            </header>
            {!flows.length ? (
              <p className="kg-empty">No flows in this graph.</p>
            ) : (
              <div className="kg-table-wrap">
                <table className="kg-table">
                  <thead>
                    <tr>
                      <th>Name</th>
                      <th>Nodes</th>
                      <th>Files</th>
                      <th>Depth</th>
                      <th>Criticality</th>
                    </tr>
                  </thead>
                  <tbody>
                    {flows.map((f) => (
                      <tr key={f.id} title={f.name}>
                        <td className="mono">{f.name}</td>
                        <td>{f.nodeCount}</td>
                        <td>{f.fileCount}</td>
                        <td>{f.depth}</td>
                        <td>{f.criticality?.toFixed?.(3) ?? f.criticality}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        )}

        {view === "browse" && (
          <section className="kg-section">
            <header className="kg-content-head">
              <h1>Browse nodes</h1>
              <div className="kg-filters">
                {KIND_FILTERS.map((k) => (
                  <button
                    key={k || "all"}
                    type="button"
                    className={`btn small ${kind === k ? "primary" : ""}`}
                    onClick={() => setKind(k)}
                  >
                    {k || "All"}
                  </button>
                ))}
              </div>
            </header>
            {busy ? <p className="kg-loading">Loading…</p> : null}
            <div className={`kg-split ${detail ? "has-detail" : ""}`}>
              <NodeTable rows={nodes} onSelect={setSelected} selected={selected} />
              {detail ? <NodeDetail detail={detail} onSelect={setSelected} /> : null}
            </div>
          </section>
        )}

        {view === "search" && (
          <section className="kg-section">
            <header className="kg-content-head">
              <h1>Search</h1>
            </header>
            <form className="kg-search" onSubmit={runSearch}>
              <input
                type="search"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Name, qualified name, or file…"
                aria-label="Search graph nodes"
              />
              <button className="btn primary" type="submit" disabled={busy || !query.trim()}>
                Search
              </button>
            </form>
            <div className={`kg-split ${detail ? "has-detail" : ""}`}>
              <NodeTable rows={hits} onSelect={setSelected} selected={selected} />
              {detail ? <NodeDetail detail={detail} onSelect={setSelected} /> : null}
            </div>
          </section>
        )}
      </div>
    </div>
  );
}

function Stat({ label, value, mono }) {
  return (
    <div className="kg-stat">
      <span className="kg-stat-label">{label}</span>
      <span className={`kg-stat-value ${mono ? "mono" : ""}`}>{value}</span>
    </div>
  );
}

function Bar({ label, value, max }) {
  const pct = Math.max(2, Math.round((value / max) * 100));
  return (
    <div className="kg-bar" title={`${label}: ${value}`}>
      <span className="kg-bar-label mono">{label}</span>
      <div className="kg-bar-track">
        <div className="kg-bar-fill" style={{ width: `${pct}%` }} />
      </div>
      <span className="kg-bar-val">{value}</span>
    </div>
  );
}
