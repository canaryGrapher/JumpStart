import { useEffect, useState } from "react";
import { WikiInfo, WikiPage } from "../../api";
import WikiMarkdown from "./WikiMarkdown";

function PagesList({ pages, current, onSelect }) {
  if (!pages?.length) {
    return <p className="wiki-side-empty">No pages yet.</p>;
  }
  return (
    <ul className="wiki-page-list">
      {pages.map((p) => (
        <li key={p.name}>
          <button
            type="button"
            className={p.name === current ? "active" : ""}
            onClick={() => onSelect(p.name)}
          >
            {p.title || p.name}
          </button>
        </li>
      ))}
    </ul>
  );
}

export default function WikiPanel({ project, onError }) {
  const root = project.root;
  const [info, setInfo] = useState(null);
  const [pageName, setPageName] = useState("Home");
  const [page, setPage] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;
    setLoading(true);
    WikiInfo(root)
      .then((w) => {
        if (!active) return;
        setInfo(w);
        const first =
          (w.pages || []).find((p) => p.name === "Home")?.name ||
          w.pages?.[0]?.name ||
          "Home";
        setPageName(first);
      })
      .catch((e) => {
        if (active) onError(String(e));
      })
      .finally(() => active && setLoading(false));
    return () => {
      active = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [root]);

  useEffect(() => {
    if (!info?.present || !pageName) return;
    let active = true;
    WikiPage(root, pageName)
      .then((p) => active && setPage(p))
      .catch((e) => {
        if (active) {
          setPage(null);
          onError(String(e));
        }
      });
    return () => {
      active = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [root, pageName, info?.present]);

  if (loading) {
    return (
      <div className="wiki-panel">
        <p className="wiki-loading">Loading wiki…</p>
      </div>
    );
  }

  if (!info?.present) {
    return (
      <div className="wiki-panel">
        <div className="empty">
          <p>
            No local wiki found. Add a <code>.wiki</code> folder (or{" "}
            <code>docs/wiki</code>) with GitHub-style markdown pages — start with{" "}
            <code>Home.md</code> and optional <code>_Sidebar.md</code>. Useful for
            private repos that cannot enable GitHub Wiki.
          </p>
        </div>
      </div>
    );
  }

  const navigate = (name) => {
    if (name && name !== pageName) setPageName(name);
  };

  return (
    <div className="wiki-panel">
      <aside className="wiki-sidebar" aria-label="Wiki pages">
        <div className="wiki-sidebar-head">Pages</div>
        {info.hasSidebar && info.sidebarMd ? (
          <WikiMarkdown
            className="wiki-sidebar-md"
            source={info.sidebarMd}
            onNavigate={navigate}
            currentPage={pageName}
          />
        ) : (
          <PagesList pages={info.pages} current={pageName} onSelect={navigate} />
        )}
        {info.hasSidebar && info.pages?.length > 0 && (
          <details className="wiki-all-pages">
            <summary>All pages</summary>
            <PagesList pages={info.pages} current={pageName} onSelect={navigate} />
          </details>
        )}
      </aside>

      <article className="wiki-content">
        <header className="wiki-content-head">
          <h1>{page?.title || pageName}</h1>
          <span className="wiki-content-path mono" title={info.relRoot}>
            {info.relRoot}/{pageName}.md
          </span>
        </header>

        {!page?.hasPage ? (
          <div className="empty">
            <p>
              Page <code>{pageName}</code> was not found in this wiki.
            </p>
          </div>
        ) : (
          <>
            <WikiMarkdown
              source={page.body}
              onNavigate={navigate}
              currentPage={pageName}
            />
            {page.footer ? (
              <footer className="wiki-footer">
                <WikiMarkdown
                  source={page.footer}
                  onNavigate={navigate}
                  currentPage={pageName}
                />
              </footer>
            ) : null}
          </>
        )}
      </article>
    </div>
  );
}
