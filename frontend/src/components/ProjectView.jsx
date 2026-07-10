import { useState } from "react";
import { StartAll, StopAll } from "../api";
import ProcessCard from "./ProcessCard";
import TaskTracker from "./TaskTracker";

export default function ProjectView({ project, usage, onEdit, onDelete, onError, onChanged }) {
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
        <div className="root-path">{project.root}</div>
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

      {showTasks && (
        <div className="tabs">
          <button
            className={tab === "processes" ? "active" : ""}
            onClick={() => setTab("processes")}
          >
            Processes
          </button>
          <button
            className={tab === "tasks" ? "active" : ""}
            onClick={() => setTab("tasks")}
          >
            Tasks
          </button>
        </div>
      )}

      {tab === "tasks" && showTasks ? (
        <TaskTracker project={project} onChanged={onChanged} onError={onError} />
      ) : (
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
