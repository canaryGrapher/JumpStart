import { useEffect, useState } from "react";
import { UpdateTasks, UpdateSprints, GitStatus } from "../api";
import { capture } from "../analytics";
import KanbanBoard from "./kanban/KanbanBoard";
import TaskDetailModal from "./kanban/TaskDetailModal";
import TasksCsvModal from "./kanban/TasksCsvModal";
import ChatDock from "./kanban/ChatDock";
import RoadmapModal from "./roadmap/RoadmapModal";
import { migrate, blankTask, uid } from "./kanban/columns";
import SyncBar from "./github/SyncBar";
import ActivityPanel from "./github/activity/ActivityPanel";
import useGitHubSync from "../hooks/useGitHubSync";
import {
  migrateSprints,
  blankSprint,
  defaultSprintId,
  resequence,
} from "./kanban/sprints";

// Per-project board. Tasks are stored flat; a story is a task with
// type "story" and children point to it via parentId. Tasks group into
// sprints via sprintId, and sprints sequence into a roadmap by order.
export default function TaskTracker({ project, onChanged, onError }) {
  const [tasks, setTasks] = useState((project.tasks || []).map(migrate));
  const [sprints, setSprints] = useState(migrateSprints(project.sprints));
  const [sprintFilter, setSprintFilter] = useState(() =>
    defaultSprintId(migrateSprints(project.sprints))
  );
  const [openTask, setOpenTask] = useState(null);
  const [roadmapOpen, setRoadmapOpen] = useState(false);
  const [csvOpen, setCsvOpen] = useState(false);
  // null while unknown. The GitHub board is a layer on top of a git
  // remote (syncing needs somewhere on GitHub to sync with), so there is
  // nothing meaningful to show here until one exists — not even the
  // "Connect GitHub" invitation, which would otherwise dead-end into the
  // same "no remote yet" state the Git tab already handles.
  const [hasRemote, setHasRemote] = useState(null);

  // Older builds published --tracker-head-h for a sticky status strip.
  // Clear any leftover value so the board height calc stays accurate.
  useEffect(() => {
    document.documentElement.style.removeProperty("--tracker-head-h");
  }, []);

  useEffect(() => {
    let live = true;
    GitStatus(project.root)
      .then((st) => live && setHasRemote(!!st?.hasRemote))
      .catch(() => live && setHasRemote(false));
    return () => {
      live = false;
    };
  }, [project.root]);

  // A sync pass returns the whole reconciled task list, so the board
  // adopts it wholesale. The open modal follows its task to the new copy
  // rather than showing a stale one.
  const adoptSynced = (next) => {
    const migrated = (next || []).map(migrate);
    setTasks(migrated);
    setOpenTask((cur) => (cur ? migrated.find((t) => t.id === cur.id) || null : null));
    onChanged();
  };

  const adoptImported = (nextTasks, nextSprints) => {
    adoptSynced(nextTasks);
    if (nextSprints) setSprints(migrateSprints(nextSprints));
  };

  const { sync, setSync, state, result, error, progress, syncNow } = useGitHubSync(
    project.id,
    adoptSynced,
    onError
  );

  const save = async (next) => {
    setTasks(next);
    try {
      await UpdateTasks(project.id, next);
      onChanged();
    } catch (e) {
      onError(String(e));
    }
  };

  const saveSprints = async (next) => {
    const seq = resequence(next);
    setSprints(seq);
    setRoadmapOpen(false);
    // Tasks pointing at a deleted sprint fall back to the backlog.
    const ids = new Set(seq.map((s) => s.id));
    const orphaned = tasks.some((t) => t.sprintId && !ids.has(t.sprintId));
    if (orphaned) {
      save(tasks.map((t) => (t.sprintId && !ids.has(t.sprintId) ? { ...t, sprintId: "" } : t)));
    }
    if (sprintFilter && !ids.has(sprintFilter)) setSprintFilter("");
    try {
      await UpdateSprints(project.id, seq);
      onChanged();
    } catch (e) {
      onError(String(e));
    }
  };

  const quickAddSprint = () => {
    const s = blankSprint("", sprints.length);
    saveSprints([...sprints, s]);
    setSprintFilter(s.id);
  };

  // Creating an item is always manual first. The new card is opened
  // straight away so the user can add a description and then, optionally,
  // hit "Fill with AI" to expand it — rather than the AI inventing the
  // item from scratch.
  const add = (title, opts, openAfter = false) => {
    const item = blankTask(title, opts);
    save([...tasks, item]);
    if (openAfter) setOpenTask(item);
    return item;
  };

  const update = (task) => {
    save(
      tasks.map((t) => {
        if (t.id !== task.id) return t;
        // Never drop a synced GitHub link if the modal draft missed it.
        const github =
          task.github?.itemId ? task.github : t.github?.itemId ? t.github : task.github;
        return { ...task, github };
      })
    );
    setOpenTask(null);
  };

  // Removing a story also removes its children.
  const remove = (id) => {
    save(tasks.filter((t) => t.id !== id && t.parentId !== id));
    setOpenTask(null);
  };

  // Insert AI-generated stories (each with its own child tasks).
  const addStories = (stories) => {
    const extra = [];
    const target = sprintFilter === "__all__" ? "" : sprintFilter;
    for (const s of stories || []) {
      const storyId = uid();
      extra.push({
        ...blankTask(s.title || "Untitled story", {
          type: "story",
          status: "backlog",
          sprintId: target,
        }),
        id: storyId,
        description: s.description || "",
        priority: s.priority || "",
        labels: s.labels || [],
        storyPoints: s.storyPoints || 0,
        acceptance: (s.acceptance || []).map((t) => ({
          id: uid(),
          title: t,
          done: false,
        })),
      });
      for (const ct of s.tasks || []) {
        extra.push(
          blankTask(ct.title || "Task", {
            type: "task",
            status: "backlog",
            parentId: storyId,
            sprintId: target,
          })
        );
      }
    }
    if (extra.length) save([...tasks, ...extra]);
  };

  const stories = tasks.filter((t) => t.type === "story").length;
  const done = tasks.filter((t) => t.status === "done").length;
  const pct = tasks.length ? Math.round((done / tasks.length) * 100) : 0;

  return (
    <div className="task-tracker">
      <div className="tracker-head">
        <div className="task-progress">
          <span>
            {done}/{tasks.length} done · {stories} stories
          </span>
          <div className="meter">
            <div style={{ width: `${pct}%` }} />
          </div>
          <span>{pct}%</span>
          <div className="task-csv-actions">
            <button
              type="button"
              className="btn tiny ghost"
              onClick={() => setCsvOpen(true)}
              title="Import or export tasks as CSV"
            >
              Import / Export
            </button>
          </div>
        </div>

        {hasRemote && (
          <SyncBar
            projectId={project.id}
            sync={sync}
            state={state}
            result={result}
            error={error}
            progress={progress}
            onSyncNow={syncNow}
            onLinked={setSync}
            onError={onError}
          />
        )}

        {hasRemote && <ActivityPanel projectId={project.id} sync={sync} onError={onError} />}
      </div>

      <KanbanBoard
        tasks={tasks}
        sprints={sprints}
        sprintFilter={sprintFilter}
        onSprintFilter={setSprintFilter}
        onOpenRoadmap={() => {
          setRoadmapOpen(true);
          capture("roadmap_opened", { item_count: sprints.length });
        }}
        onQuickAddSprint={quickAddSprint}
        onChange={save}
        onOpen={setOpenTask}
        onAdd={(title, opts) => add(title, opts, true)}
      />

      {roadmapOpen && (
        <RoadmapModal
          sprints={sprints}
          tasks={tasks}
          onSave={saveSprints}
          onClose={() => setRoadmapOpen(false)}
        />
      )}

      {csvOpen && (
        <TasksCsvModal
          projectId={project.id}
          projectName={project.name}
          tasks={tasks}
          sprints={sprints}
          onImported={adoptImported}
          onClose={() => setCsvOpen(false)}
          onError={onError}
        />
      )}

      {openTask && (
        <TaskDetailModal
          task={openTask}
          tasks={tasks}
          sprints={sprints}
          projectId={project.id}
          sync={sync}
          onSave={update}
          onDelete={remove}
          onOpen={setOpenTask}
          onAddChild={(storyId, title) =>
            add(title, { type: "task", status: "todo", parentId: storyId })
          }
          onClose={() => setOpenTask(null)}
          onError={onError}
        />
      )}

      <ChatDock
        projectId={project.id}
        onAddStories={addStories}
        onError={onError}
      />
    </div>
  );
}
