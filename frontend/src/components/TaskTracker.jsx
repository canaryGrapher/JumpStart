import { useState } from "react";
import { UpdateTasks } from "../api";
import KanbanBoard from "./kanban/KanbanBoard";
import TaskDetailModal from "./kanban/TaskDetailModal";
import ChatDock from "./kanban/ChatDock";
import { migrate, blankTask, uid } from "./kanban/columns";

// Per-project board. Tasks are stored flat; a story is a task with
// type "story" and children point to it via parentId.
export default function TaskTracker({ project, onChanged, onError }) {
  const [tasks, setTasks] = useState((project.tasks || []).map(migrate));
  const [openTask, setOpenTask] = useState(null);

  const save = async (next) => {
    setTasks(next);
    try {
      await UpdateTasks(project.id, next);
      onChanged();
    } catch (e) {
      onError(String(e));
    }
  };

  const add = (title, opts) => save([...tasks, blankTask(title, opts)]);

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
    for (const s of stories || []) {
      const storyId = uid();
      extra.push({
        ...blankTask(s.title || "Untitled story", {
          type: "story",
          status: "backlog",
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

      <KanbanBoard tasks={tasks} onChange={save} onOpen={setOpenTask} onAdd={add} />

      {openTask && (
        <TaskDetailModal
          task={openTask}
          tasks={tasks}
          projectName={project.name}
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
        projectName={project.name}
        onAddStories={addStories}
        onError={onError}
      />
    </div>
  );
}
