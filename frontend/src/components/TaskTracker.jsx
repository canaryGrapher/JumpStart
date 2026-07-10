import { useState } from "react";
import { UpdateTasks } from "../api";
import KanbanBoard from "./kanban/KanbanBoard";
import TaskDetailModal from "./kanban/TaskDetailModal";
import { migrate } from "./kanban/columns";

const uid = () => crypto.randomUUID();

// Per-project kanban tracker: what's built vs. what's still to build.
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

  const add = (title, status) =>
    save([
      ...tasks,
      {
        id: uid(),
        title,
        status,
        done: status === "done",
        description: "",
        subtasks: [],
        labels: [],
        createdAt: Date.now(),
      },
    ]);

  const update = (task) => {
    save(tasks.map((t) => (t.id === task.id ? task : t)));
    setOpenTask(null);
  };

  const remove = (id) => {
    save(tasks.filter((t) => t.id !== id));
    setOpenTask(null);
  };

  const done = tasks.filter((t) => t.status === "done").length;
  const pct = tasks.length ? Math.round((done / tasks.length) * 100) : 0;

  return (
    <div>
      <div className="task-progress">
        <span>
          {done}/{tasks.length} built
        </span>
        <div className="meter">
          <div style={{ width: `${pct}%` }} />
        </div>
        <span>{pct}%</span>
      </div>

      <KanbanBoard
        tasks={tasks}
        onChange={save}
        onOpen={setOpenTask}
        onAdd={add}
      />

      {openTask && (
        <TaskDetailModal
          task={openTask}
          onSave={update}
          onDelete={remove}
          onClose={() => setOpenTask(null)}
        />
      )}
    </div>
  );
}
