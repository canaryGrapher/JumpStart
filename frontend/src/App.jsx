import { useEffect, useLayoutEffect, useRef, useState } from "react";
import {
  GetProjects,
  SaveProject,
  DeleteProject,
  SetProjectFavorite,
  GetUsage,
  SetNativeTheme,
  GetAppVersion,
} from "./api";
import { reportUpdateChannel, trackPanel } from "./analytics";
import { isBetaEnabled } from "./updateChannel";
import Icon, { ICONS } from "./components/Icon";
import Sidebar from "./components/Sidebar";
import SidebarResizer from "./components/SidebarResizer";
import useSidebarWidth from "./hooks/useSidebarWidth";
import ProjectView from "./components/ProjectView";
import ProjectModal from "./components/ProjectModal";
import Dashboard from "./components/Dashboard";
import AllProjects from "./components/AllProjects";
import PortsView from "./components/PortsView";
import Preferences from "./components/Preferences";
import UpdateBanner from "./components/UpdateBanner";
import BuildBadge from "./components/BuildBadge";
import AdOverlay from "./components/AdOverlay";
import TerminalDock from "./components/terminal/TerminalDock";
import useUpdateCheck from "./hooks/useUpdateCheck";
import useRemoteBanner from "./hooks/useRemoteBanner";

function useTheme() {
  const [theme, setTheme] = useState(
    () => localStorage.getItem("theme") || "system"
  );
  useEffect(() => {
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const apply = () => {
      const mode = theme === "system" ? (mq.matches ? "dark" : "light") : theme;
      document.documentElement.dataset.theme = mode;
      // Keep AppKit's native window/vibrancy appearance in lockstep with the
      // CSS theme — otherwise the sidebar's native vibrancy tracks the real
      // macOS System Appearance independently of this in-app selection,
      // producing dark-text-on-dark-vibrancy (or the reverse) whenever they
      // disagree.
      SetNativeTheme(mode).catch(() => {});
    };
    apply();
    localStorage.setItem("theme", theme);
    mq.addEventListener("change", apply);
    return () => mq.removeEventListener("change", apply);
  }, [theme]);
  return [theme, setTheme];
}

function useAccent() {
  const [accent, setAccent] = useState(
    () => localStorage.getItem("accent") || "forest"
  );
  useEffect(() => {
    document.documentElement.dataset.accent = accent;
    localStorage.setItem("accent", accent);
  }, [accent]);
  return [accent, setAccent];
}

export default function App() {
  const [projects, setProjects] = useState([]);
  const [view, setView] = useState("dashboard"); // "dashboard" | "ports" | "project" | "all-projects"
  const [selectedId, setSelectedId] = useState(null);
  const [modal, setModal] = useState(null); // null | "new" | project object
  const [toast, setToast] = useState(null); // { msg, ok }
  const [usage, setUsage] = useState({ system: {}, procs: {} });
  const [theme, setTheme] = useTheme();
  const [accent, setAccent] = useAccent();
  const [sidebarOpen, setSidebarOpen] = useState(
    () => localStorage.getItem("sidebarOpen") !== "0"
  );
  const { width: sidebarWidth, resizing, onResizeStart, reset: resetSidebarWidth } = useSidebarWidth();
  const [prefsOpen, setPrefsOpen] = useState(false);
  const { update, dismiss: dismissUpdate } = useUpdateCheck();
  const { banner, dismiss: dismissBanner } = useRemoteBanner();
  const topbarRef = useRef(null);

  // Every sticky bar below (.tabs-bar, .task-progress, .kb-search) and the
  // Kanban board's own height calc stack their offsets on top of this
  // bar's height, and used to assume a hardcoded "68px". That drifts
  // whenever the header's actual content changes height (a longer title
  // wrapping, BuildBadge showing/hiding) — the drift doesn't misplace the
  // sticky bars themselves, but it does throw off how far .main can
  // scroll, letting the page overscroll past the point where .kb-board
  // lines up under them and clipping the column headers behind the
  // search bar. Publishing the real height, same pattern as
  // --tabs-bar-h / --task-progress-h / --kb-search-h, keeps the whole
  // stack self-correcting instead of guessing a constant.
  useLayoutEffect(() => {
    const el = topbarRef.current;
    if (!el) return;
    const publish = () => {
      document.documentElement.style.setProperty("--topbar-h", `${el.offsetHeight}px`);
    };
    publish();
    const ro = new ResizeObserver(publish);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  useEffect(() => {
    localStorage.setItem("sidebarOpen", sidebarOpen ? "1" : "0");
  }, [sidebarOpen]);

  const load = () =>
    GetProjects()
      .then((p) => setProjects(p || []))
      .catch((e) => onError(String(e)));

  const onError = (msg) => setToast({ msg, ok: false });
  const onInfo = (msg) => setToast({ msg, ok: true });

  useEffect(() => {
    load();
  }, []);

  // app_launched is emitted by the Go side during Startup, so there is
  // nothing to record here. The one thing Go cannot know is the update
  // channel, which lives in localStorage — hand it over so the beta cohort
  // is a breakdown on every event.
  useEffect(() => {
    reportUpdateChannel(isBetaEnabled());
  }, []);

  useEffect(() => {
    const poll = () => GetUsage().then(setUsage).catch(() => {});
    poll();
    const t = setInterval(poll, 3000);
    return () => clearInterval(t);
  }, []);

  useEffect(() => {
    if (toast) {
      const t = setTimeout(() => setToast(null), 4000);
      return () => clearTimeout(t);
    }
  }, [toast]);

  const selected = projects.find((p) => p.id === selectedId) || null;

  const openProject = (id) => {
    setSelectedId(id);
    setView("project");
  };

  const handleSave = async (project) => {
    try {
      await SaveProject(project);
      await load();
      setModal(null);
      openProject(project.id);
    } catch (e) {
      onError(String(e));
    }
  };

  const handleDelete = async (id) => {
    try {
      await DeleteProject(id);
      await load();
      if (selectedId === id) {
        setSelectedId(null);
        setView("dashboard");
      }
    } catch (e) {
      onError(String(e));
    }
  };

  // Optimistic star toggle: the sidebar reorders instantly, then we persist
  // and reload. On failure the reload puts the old state back.
  const handleToggleFavorite = async (project) => {
    const next = !project.favorite;
    setProjects((list) =>
      list.map((p) => (p.id === project.id ? { ...p, favorite: next } : p))
    );
    try {
      await SetProjectFavorite(project.id, next);
    } catch (e) {
      onError(String(e));
    }
    load();
  };

  const titles = { dashboard: "Dashboard", ports: "Ports", "all-projects": "All Projects" };

  return (
    <div
      className={`layout ${sidebarOpen ? "" : "sidebar-hidden"} ${resizing ? "resizing" : ""}`}
      style={{ "--sidebar-w": `${sidebarWidth}px` }}
    >
      <Sidebar
        projects={projects}
        view={view}
        selectedId={view === "project" ? selectedId : null}
        onNavigate={(v) => {
          setView(v);
          setSelectedId(null);
          trackPanel(v);
        }}
        onSelect={openProject}
        onAdd={() => setModal("new")}
        onOpenPrefs={() => {
          setPrefsOpen(true);
          trackPanel("preferences");
        }}
        onToggleFavorite={handleToggleFavorite}
      />
      {sidebarOpen && (
        <SidebarResizer onResizeStart={onResizeStart} onReset={resetSidebarWidth} />
      )}
      <main className="main">
        <div className="topbar" ref={topbarRef}>
          <button
            className="icon-btn"
            title="Toggle Sidebar"
            onClick={() => setSidebarOpen((o) => !o)}
          >
            <Icon d={ICONS.sidebar} />
          </button>
          <h1>{view === "project" && selected ? selected.name : titles[view] || "Dashboard"}</h1>
          <BuildBadge />
        </div>

        {view === "dashboard" && (
          <Dashboard
            projects={projects}
            usage={usage}
            onOpen={openProject}
            onViewAll={() => {
              setView("all-projects");
              trackPanel("all-projects");
            }}
            onReload={load}
            onError={onError}
            onInfo={onInfo}
          />
        )}
        {view === "all-projects" && (
          <AllProjects
            projects={projects}
            onOpen={openProject}
            onToggleFavorite={handleToggleFavorite}
          />
        )}
        {view === "ports" && <PortsView onError={onError} />}
        {view === "project" &&
          (selected ? (
            // key={selected.id} forces a clean remount on every project
            // switch. ProjectView otherwise keeps living (same component,
            // just a new `project` prop), so its tab selection and
            // fetched-but-not-yet-refetched state (hasDocker, githubUrl,
            // etc.) briefly showed the previous project's data until each
            // effect caught up — a visible flash of stale content.
            <ProjectView
              key={selected.id}
              project={selected}
              usage={usage}
              onEdit={() => setModal(selected)}
              onDelete={() => handleDelete(selected.id)}
              onError={onError}
              onInfo={onInfo}
              onChanged={load}
            />
          ) : (
            <div className="empty">
              <h2>No project selected</h2>
              <p>Select a project on the left, or add one to get started.</p>
            </div>
          ))}
      </main>
      {modal && (
        <ProjectModal
          initial={modal === "new" ? null : modal}
          onSave={handleSave}
          onClose={() => setModal(null)}
        />
      )}
      {prefsOpen && (
        <Preferences
          theme={theme}
          onThemeChange={setTheme}
          accent={accent}
          onAccentChange={setAccent}
          onError={onError}
          onClose={() => setPrefsOpen(false)}
        />
      )}
      <AdOverlay banner={banner} onDismiss={dismissBanner} />
      {toast && <div className={`toast ${toast.ok ? "ok" : ""}`}>{toast.msg}</div>}
      <UpdateBanner update={update} onDismiss={dismissUpdate} />
      <TerminalDock />
    </div>
  );
}
