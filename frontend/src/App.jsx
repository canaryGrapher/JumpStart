import { useEffect, useState } from "react";
import { GetProjects, SaveProject, DeleteProject, GetUsage } from "./api";
import IconRail from "./components/IconRail";
import Sidebar from "./components/Sidebar";
import ProjectView from "./components/ProjectView";
import ProjectModal from "./components/ProjectModal";
import Dashboard from "./components/Dashboard";
import PortsView from "./components/PortsView";
import ThemeToggle from "./components/ThemeToggle";

function useTheme() {
  const [theme, setTheme] = useState(
    () => localStorage.getItem("theme") || "system"
  );
  useEffect(() => {
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const apply = () => {
      const mode = theme === "system" ? (mq.matches ? "dark" : "light") : theme;
      document.documentElement.dataset.theme = mode;
    };
    apply();
    localStorage.setItem("theme", theme);
    mq.addEventListener("change", apply);
    return () => mq.removeEventListener("change", apply);
  }, [theme]);
  return [theme, setTheme];
}

export default function App() {
  const [projects, setProjects] = useState([]);
  const [view, setView] = useState("dashboard"); // "dashboard" | "ports" | "project"
  const [selectedId, setSelectedId] = useState(null);
  const [modal, setModal] = useState(null); // null | "new" | project object
  const [toast, setToast] = useState(null); // { msg, ok }
  const [usage, setUsage] = useState({ system: {}, procs: {} });
  const [theme, setTheme] = useTheme();

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
    <div className="layout">
      <IconRail
        view={view}
        onNavigate={(v) => {
          setView(v);
          setSelectedId(null);
        }}
      />
      <Sidebar
        projects={projects}
        selectedId={view === "project" ? selectedId : null}
        onSelect={openProject}
        onAdd={() => setModal("new")}
      />
      <main className="main">
        <div className="topbar">
          <h1>{view === "project" && selected ? selected.name : titles[view] || "Dashboard"}</h1>
          <div className="right">
            <ThemeToggle theme={theme} onChange={setTheme} />
          </div>
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
      {toast && <div className={`toast ${toast.ok ? "ok" : ""}`}>{toast.msg}</div>}
    </div>
  );
}
