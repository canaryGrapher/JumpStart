import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { StartAll, StopAll, DockerInfo, GitHubRepoURL, BrowserOpenURL } from "../api";
import ProcessCard from "./ProcessCard";
import TaskTracker from "./TaskTracker";
import GitPanel from "./GitPanel";
import TestPanel from "./TestPanel";
import ContainersPanel from "./containers/ContainersPanel";
import ConfirmDialog from "./ConfirmDialog";
import CollapsibleSection from "./CollapsibleSection";
import OpenActions from "./OpenActions";
import Icon, { ICONS } from "./Icon";
import ProjectIcon from "./ProjectIcon";
import { isComposeProc } from "../procUtils";
import { trackPanel } from "../analytics";

export default function ProjectView({ project, usage, onEdit, onDelete, onError, onInfo, onChanged }) {
  const [tab, setTab] = useState("processes");
  const [hasDocker, setHasDocker] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [githubUrl, setGithubUrl] = useState("");
  const tabsBarRef = useRef(null);

  // .task-progress (rendered further down, inside TaskTracker) sticks
  // directly beneath this bar. Its exact height isn't a safe constant to
  // hardcode: it shifts a few px once the Inter webfont finishes loading,
  // which would otherwise leave a gap that the Tasks tab's GitHub sync row
  // scrolls through. Publish the real height as a CSS var instead, so the
  // two sticky bars always stack flush regardless of font metrics.
  useLayoutEffect(() => {
    const el = tabsBarRef.current;
    if (!el) return;
    const publish = () => {
      document.documentElement.style.setProperty("--tabs-bar-h", `${el.offsetHeight}px`);
    };
    publish();
    const ro = new ResizeObserver(publish);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  // Panel reach is the adoption matrix: anything under a few percent of
  // monthly actives after a month is a deletion candidate. Deduplicated per
  // session by trackPanel, so tab-flipping does not inflate it.
  const openTab = (next) => {
    setTab(next);
    trackPanel(next);
  };

  useEffect(() => {
    let active = true;
    DockerInfo(project.root)
      .then((info) => {
        if (active) setHasDocker(!!(info && (info.hasCompose || info.hasDockerfile)));
      })
      .catch(() => active && setHasDocker(false));
    return () => {
      active = false;
    };
  }, [project.root]);

  // Only shown once we know the project has a GitHub remote; a plain git
  // remote (or no remote at all) just leaves this button off the header.
  useEffect(() => {
    let active = true;
    GitHubRepoURL(project.root)
      .then((url) => active && setGithubUrl(url || ""))
      .catch(() => active && setGithubUrl(""));
    return () => {
      active = false;
    };
  }, [project.root]);

  // If the active tab disappears (e.g. Docker removed), fall back to Processes.
  useEffect(() => {
    if (tab === "containers" && !hasDocker) setTab("processes");
  }, [hasDocker, tab]);

  const startAll = async () => {
    const errs = await StartAll(project.id);
    if (errs && errs.length) onError(errs.join("; "));
    onChanged();
  };

  const showTasks = project.tasksEnabled;
  const visibleProcs = (project.processes || []).filter((p) => !isComposeProc(p));

  return (
    <>
      <CollapsibleSection
        id={`project-chrome:${project.id}`}
        className="project-chrome"
        defaultOpen
        lead={<ProjectIcon project={project} className="main-header-icon" />}
        title={project.name || "Project"}
        summary={project.root}
        summaryClassName="mono"
      >
        {project.description && (
          <div className="main-header">
            <p className="project-description" title={project.description}>
              {project.description}
            </p>
          </div>
        )}

        {/* Toolbar: labeled process controls first, then the "open in" group,
            then project management — with Delete set apart at the far end so
            it can't be hit by accident next to Start. */}
        <div className="header-toolbar">
          <div className="toolbar-group">
            <button className="btn primary" onClick={startAll}>
              <Icon d={ICONS.play} filled />
              Start All
            </button>
            <button className="btn" onClick={() => StopAll(project.id)}>
              <Icon d={ICONS.stop} filled />
              Stop All
            </button>
          </div>
          <div className="toolbar-capsule" role="group" aria-label="Open project in">
            <OpenActions dir={project.root} onError={onError} iconOnly showCode colored />
            {githubUrl && (
              <button
                className="icon-btn outline mono"
                title="Open on GitHub"
                aria-label="Open on GitHub"
                onClick={() => BrowserOpenURL(githubUrl)}
              >
                <Icon d={ICONS.github} filled />
              </button>
            )}
          </div>
          <div className="toolbar-group toolbar-end">
            <button className="btn" onClick={onEdit}>
              <Icon d={ICONS.pencil} />
              Edit
            </button>
            <button
              className="icon-btn outline danger"
              title="Delete project…"
              aria-label="Delete project"
              onClick={() => setConfirmDelete(true)}
            >
              <Icon d={ICONS.trash} />
            </button>
          </div>
        </div>
      </CollapsibleSection>

      {/* Full-width wrapper gives the sticky strip a solid background that
          spans the row; .tabs itself is only pill-width (inline-flex), so
          it can't carry that background alone without leaving the rest of
          the row transparent to whatever scrolls up behind it. */}
      <div className="tabs-bar" ref={tabsBarRef}>
        <div className="tabs">
          <button
            className={tab === "processes" ? "active" : ""}
            onClick={() => openTab("processes")}
          >
            Processes
          </button>
          {hasDocker && (
            <button
              className={tab === "containers" ? "active" : ""}
              onClick={() => openTab("containers")}
            >
              Containers
            </button>
          )}
          {showTasks && (
            <button
              className={tab === "tasks" ? "active" : ""}
              onClick={() => openTab("tasks")}
            >
              Tasks
            </button>
          )}
          <button
            className={tab === "git" ? "active" : ""}
            onClick={() => openTab("git")}
          >
            Git
          </button>
          <button
            className={tab === "tests" ? "active" : ""}
            onClick={() => openTab("tests")}
          >
            Tests
          </button>
        </div>
      </div>

      {tab === "containers" && hasDocker && (
        <ContainersPanel projectRoot={project.root} onError={onError} onInfo={onInfo} />
      )}

      {tab === "tasks" && showTasks && (
        <TaskTracker project={project} onChanged={onChanged} onError={onError} />
      )}

      {tab === "git" && (
        <GitPanel project={project} onError={onError} onInfo={onInfo} onChanged={onChanged} />
      )}

      {tab === "tests" && (
        <TestPanel project={project} onError={onError} onChanged={onChanged} />
      )}

      {tab === "processes" && (
        <>
          <div className="cards">
            {visibleProcs.map((proc) => (
              <ProcessCard
                key={proc.id}
                projectId={project.id}
                proc={proc}
                usage={(usage.procs || {})[proc.id]}
                onError={onError}
              />
            ))}
          </div>
          {visibleProcs.length === 0 && (
            <div className="empty">
              <p>
                {hasDocker
                  ? "No subprocesses. This project's containers live in the Containers tab."
                  : "No subprocesses yet. Click Edit to add one."}
              </p>
            </div>
          )}
        </>
      )}

      {confirmDelete && (
        <ConfirmDialog
          title={`Delete ${project.name}?`}
          body="JumpStart will forget this project's processes, tasks, and sprints. Files on disk are not touched."
          confirmLabel="Delete project"
          cancelLabel="Keep it"
          danger
          onConfirm={() => {
            setConfirmDelete(false);
            onDelete();
          }}
          onCancel={() => setConfirmDelete(false)}
        />
      )}
    </>
  );
}
