import { useEffect, useMemo, useRef, useState } from "react";
import { ArchitectureInfo, ArchitectureDoc, ClipboardSetText } from "../../api";
import { buildArchPrompt } from "../../archPrompt";
import ArchGraph from "./ArchGraph";
import ArchFlow from "./ArchFlow";
import ArchDB from "./ArchDB";
import { ArchCharts, ArchTables } from "./ArchCharts";

function OverviewBody({ project, overview }) {
  return (
    <section className="arch-section-body">
      <header className="arch-content-head">
        <h1>{project?.name || "Overview"}</h1>
        {project?.summary ? <p className="arch-lead">{project.summary}</p> : null}
      </header>
      {overview?.description ? <p className="arch-desc">{overview.description}</p> : null}
      {overview?.stack?.length ? (
        <div className="arch-tags">
          <h3>Stack</h3>
          <ul>
            {overview.stack.map((s) => (
              <li key={s}>{s}</li>
            ))}
          </ul>
        </div>
      ) : null}
      {overview?.principles?.length ? (
        <div className="arch-tags">
          <h3>Principles</h3>
          <ul>
            {overview.principles.map((p) => (
              <li key={p}>{p}</li>
            ))}
          </ul>
        </div>
      ) : null}
    </section>
  );
}

function SectionBody({ section }) {
  if (!section) return null;
  return (
    <section className="arch-section-body">
      <header className="arch-content-head">
        <h1>{section.title}</h1>
        <span className="arch-kind-pill">{section.kind}</span>
      </header>
      {section.summary ? <p className="arch-desc">{section.summary}</p> : null}

      {section.steps?.length > 0 && <ArchFlow steps={section.steps} />}

      {(section.kind === "db" || section.entities?.length > 0) && (
        <ArchDB entities={section.entities} relations={section.relations} />
      )}

      {section.nodes?.length > 0 && (
        <ArchGraph nodes={section.nodes} edges={section.edges || []} />
      )}

      <ArchTables tables={section.tables} />
      <ArchCharts charts={section.charts} />
    </section>
  );
}

export default function ArchitecturePanel({ project, onError, onInfo }) {
  const root = project.root;
  const [info, setInfo] = useState(null);
  const [doc, setDoc] = useState(null);
  const [sectionId, setSectionId] = useState("overview");
  const [loading, setLoading] = useState(true);
  const [copied, setCopied] = useState(false);
  // Parent re-creates onError/onInfo every render (usage poll). Keep stable
  // refs so loads only re-run when the project root changes — otherwise the
  // section selection snaps back to Overview every few seconds.
  const onErrorRef = useRef(onError);
  const onInfoRef = useRef(onInfo);
  onErrorRef.current = onError;
  onInfoRef.current = onInfo;

  const loadDoc = (opts = {}) => {
    const { resetSection = false, showLoading = true } = opts;
    let cancelled = false;
    if (showLoading) setLoading(true);
    (async () => {
      try {
        const i = await ArchitectureInfo(root);
        if (cancelled) return;
        setInfo(i);
        if (i?.present) {
          const d = await ArchitectureDoc(root);
          if (cancelled) return;
          setDoc(d);
          const ids = (i.sections || []).map((s) => s.id);
          const fallback = ids[0] || "overview";
          setSectionId((prev) => {
            if (!resetSection && ids.includes(prev)) return prev;
            return fallback;
          });
        } else {
          setDoc(null);
        }
      } catch (e) {
        if (!cancelled) {
          onErrorRef.current?.(String(e));
          setDoc(null);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  };

  useEffect(() => loadDoc({ resetSection: true }), [root]);

  const reload = () => loadDoc({ resetSection: false, showLoading: true });

  const nav = useMemo(() => {
    if (info?.sections?.length) return info.sections;
    return [];
  }, [info]);

  const activeSection = useMemo(() => {
    if (!doc) return null;
    if (sectionId === "overview") return null;
    return (doc.sections || []).find((s) => s.id === sectionId) || null;
  }, [doc, sectionId]);

  const copyPrompt = async () => {
    try {
      await ClipboardSetText(buildArchPrompt(root));
      setCopied(true);
      onInfoRef.current?.(
        "Prompt copied. Paste it into your AI agent — it will write .jumpstart/architecture.xml, then click Refresh here."
      );
      setTimeout(() => setCopied(false), 2500);
    } catch (e) {
      onErrorRef.current?.(String(e));
    }
  };

  if (loading) {
    return (
      <div className="arch-panel">
        <p className="arch-loading">Loading architecture…</p>
      </div>
    );
  }

  if (!info?.present || !doc) {
    return (
      <div className="arch-panel arch-empty-panel">
        <div className="arch-empty-card">
          <h2>Architecture</h2>
          <p>
            Generate a detailed visual map of this project — code structure, data flows, database
            schemas, features, and APIs — stored at{" "}
            <code>.jumpstart/architecture.xml</code>.
          </p>
          <ol className="arch-steps">
            <li>Copy the prompt and paste it into your AI coding agent for this repo.</li>
            <li>The agent writes <code>.jumpstart/architecture.xml</code> in the project root.</li>
            <li>Come back here and refresh to explore graphs, tables, and diagrams.</li>
          </ol>
          <div className="arch-empty-actions">
            <button className="btn primary" onClick={copyPrompt}>
              {copied ? "Copied" : "Copy prompt"}
            </button>
            <button className="btn" onClick={reload}>
              Refresh
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="arch-panel">
      <aside className="arch-sidebar" aria-label="Architecture sections">
        <div className="arch-sidebar-head">Sections</div>
        <ul className="arch-nav">
          {nav.map((s) => (
            <li key={s.id}>
              <button
                type="button"
                className={sectionId === s.id ? "active" : ""}
                onClick={() => setSectionId(s.id)}
              >
                {s.title}
              </button>
            </li>
          ))}
        </ul>
        <div className="arch-sidebar-actions">
          <button className="btn small" onClick={copyPrompt} title="Regenerate via AI prompt">
            {copied ? "Copied" : "Copy prompt"}
          </button>
          <button className="btn small" onClick={reload}>
            Refresh
          </button>
        </div>
        <p className="arch-path mono" title={info.relPath}>
          {info.relPath}
          {info.generatedAt ? ` · ${new Date(info.generatedAt).toLocaleString()}` : ""}
        </p>
      </aside>

      <div className="arch-main">
        {sectionId === "overview" || !activeSection ? (
          <OverviewBody project={doc.project} overview={doc.overview} />
        ) : (
          <SectionBody section={activeSection} />
        )}
      </div>
    </div>
  );
}
