import { useState } from "react";
import { StartAll, StopAll } from "../api";
import ProcessCard from "./ProcessCard";
import TaskTracker from "./TaskTracker";
import GitPanel from "./GitPanel";
import TestPanel from "./TestPanel";

export default function ProjectView({ project, usage, onEdit, onDelete, onError, onInfo, onChanged }) {
  const [tab, setTab] = useState("processes");

  const startAll = async () => {
    const errs = await StartAll(project.id);
    if (errs && errs.length) onError(errs.join("; "));
    onChanged();
  };

  const showTasks = project.tasksEnabled;

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
            {project.processes.map((proc) => (
              <ProcessCard
                key={proc.id}
                projectId={project.id}
                proc={proc}
                usage={(usage.procs || {})[proc.id]}
                onError={onError}
              />
            ))}
          </div>
          {project.processes.length === 0 && (
            <div className="empty">
              <p>No subprocesses yet. Click Edit to add one.</p>
            </div>
          )}
        </>
      )}
    </>
  );
}
