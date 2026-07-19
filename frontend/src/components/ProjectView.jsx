import { useEffect, useState } from "react";
import { StartAll, StopAll, DockerInfo } from "../api";
import ProcessCard from "./ProcessCard";
import TaskTracker from "./TaskTracker";
import GitPanel from "./GitPanel";
import TestPanel from "./TestPanel";
import ContainersPanel from "./containers/ContainersPanel";
import { isComposeProc } from "../procUtils";

export default function ProjectView({ project, usage, onEdit, onDelete, onError, onInfo, onChanged }) {
  const [tab, setTab] = useState("processes");
  const [hasDocker, setHasDocker] = useState(false);

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
      <div className="main-header">
        <div className="main-header-text">
          <div className="root-path">{project.root}</div>
          {project.description && <p className="project-description">{project.description}</p>}
        </div>
        <div className="header-actions">
          <button className="btn primary" onClick={startAll}>
            Start all
          </button>
          <button className="btn" onClick={() => StopAll(project.id)}>
            Stop all
          </button>
          <button className="btn" onClick={onEdit}>
            Edit
          </button>
          <button className="btn danger" onClick={onDelete}>
            Delete
          </button>
        </div>
      </div>

      <div className="tabs">
        <button
          className={tab === "processes" ? "active" : ""}
          onClick={() => setTab("processes")}
        >
          Processes
        </button>
        {hasDocker && (
          <button
            className={tab === "containers" ? "active" : ""}
            onClick={() => setTab("containers")}
          >
            Containers
          </button>
        )}
        {showTasks && (
          <button
            className={tab === "tasks" ? "active" : ""}
            onClick={() => setTab("tasks")}
          >
            Tasks
          </button>
        )}
        <button
          className={tab === "git" ? "active" : ""}
          onClick={() => setTab("git")}
        >
          Git
        </button>
        <button
          className={tab === "tests" ? "active" : ""}
          onClick={() => setTab("tests")}
        >
          Tests
        </button>
      </div>

      {tab === "containers" && hasDocker && (
        <ContainersPanel projectRoot={project.root} onError={onError} onInfo={onInfo} />
      )}

      {tab === "tasks" && showTasks && (
        <TaskTracker project={project} onChanged={onChanged} onError={onError} />
      )}

      {tab === "git" && (
        <GitPanel projectRoot={project.root} onError={onError} onInfo={onInfo} />
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
    </>
  );
}
