import { useState } from "react";
import { UpdateTasks, UpdateSprints } from "../api";
import { capture } from "../analytics";
import KanbanBoard from "./kanban/KanbanBoard";
import TaskDetailModal from "./kanban/TaskDetailModal";
import ChatDock from "./kanban/ChatDock";
import RoadmapModal from "./roadmap/RoadmapModal";
import { migrate, blankTask, uid } from "./kanban/columns";
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
    save(tasks.map((t) => (t.id === task.id ? task : t)));
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
      <div className="task-progress">
        <span>
          {done}/{tasks.length} done · {stories} stories
        </span>
        <div className="meter">
          <div style={{ width: `${pct}%` }} />
        </div>
        <span>{pct}%</span>
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

      {openTask && (
        <TaskDetailModal
          task={openTask}
          tasks={tasks}
          sprints={sprints}
          projectId={project.id}
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
