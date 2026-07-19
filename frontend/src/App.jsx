import { useEffect, useState } from "react";
import { GetProjects, SaveProject, DeleteProject, GetUsage, SetNativeTheme } from "./api";
import Icon, { ICONS } from "./components/Icon";
import Sidebar from "./components/Sidebar";
import SidebarResizer from "./components/SidebarResizer";
import useSidebarWidth from "./hooks/useSidebarWidth";
import ProjectView from "./components/ProjectView";
import ProjectModal from "./components/ProjectModal";
import Dashboard from "./components/Dashboard";
import PortsView from "./components/PortsView";
import Preferences from "./components/Preferences";
import UpdateBanner from "./components/UpdateBanner";
import AdOverlay from "./components/AdOverlay";
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
  const [view, setView] = useState("dashboard"); // "dashboard" | "ports" | "project"
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

  const titles = { dashboard: "Dashboard", ports: "Ports" };

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
        }}
        onSelect={openProject}
        onAdd={() => setModal("new")}
        onOpenPrefs={() => setPrefsOpen(true)}
      />
      {sidebarOpen && (
        <SidebarResizer onResizeStart={onResizeStart} onReset={resetSidebarWidth} />
      )}
      <main className="main">
        <UpdateBanner update={update} onDismiss={dismissUpdate} />
        <div className="topbar">
          <button
            className="icon-btn"
            title="Toggle Sidebar"
            onClick={() => setSidebarOpen((o) => !o)}
          >
            <Icon d={ICONS.sidebar} />
          </button>
          <h1>{view === "project" && selected ? selected.name : titles[view] || "Dashboard"}</h1>
        </div>

        {view === "dashboard" && (
          <Dashboard
            projects={projects}
            usage={usage}
            onOpen={openProject}
            onReload={load}
            onError={onError}
            onInfo={onInfo}
          />
        )}
        {view === "ports" && <PortsView onError={onError} />}
        {view === "project" &&
          (selected ? (
            <ProjectView
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
    </div>
  );
}
