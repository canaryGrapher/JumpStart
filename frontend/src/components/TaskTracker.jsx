import { useEffect, useState } from "react";
import { UpdateTasks, UpdateSprints, GitStatus } from "../api";
import { capture, trackPanel } from "../analytics";
import KanbanBoard from "./kanban/KanbanBoard";
import TaskDetailModal from "./kanban/TaskDetailModal";
import TasksCsvModal from "./kanban/TasksCsvModal";
import AddColumnModal from "./kanban/AddColumnModal";
import ChatDock from "./kanban/ChatDock";
import RoadmapModal from "./roadmap/RoadmapModal";
import { migrate, blankTask, uid, resolveColumns } from "./kanban/columns";
import CollapsibleSection from "./CollapsibleSection";
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
  const [columns, setColumns] = useState(() => resolveColumns(project));
  const [sprints, setSprints] = useState(migrateSprints(project.sprints));
  const [sprintFilter, setSprintFilter] = useState(() =>
    defaultSprintId(migrateSprints(project.sprints))
  );
  const [openTask, setOpenTask] = useState(null);
  const [roadmapOpen, setRoadmapOpen] = useState(false);
  const [csvOpen, setCsvOpen] = useState(false);
  const [chatOpen, setChatOpen] = useState(false);
  const [addColumnOpen, setAddColumnOpen] = useState(false);
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

  useEffect(() => {
    setColumns(resolveColumns(project));
  }, [project.id, project.columns]);

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
        const childTitle =
          (typeof ct === "string" ? ct : ct?.title || ct?.description || "") ||
          "Task";
        extra.push(
          blankTask(childTitle, {
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

  const syncSummary = (() => {
    if (!hasRemote) return `${done}/${tasks.length} done · ${pct}%`;
    if (!sync?.enabled) return `${done}/${tasks.length} done · ${pct}% · GitHub not linked`;
    if (state === "syncing") return `${done}/${tasks.length} done · ${pct}% · Syncing…`;
    if (state === "error") return `${done}/${tasks.length} done · ${pct}% · Sync error`;
    return `${done}/${tasks.length} done · ${pct}%`;
  })();

  return (
    <div className="task-tracker">
      <CollapsibleSection
        id={`tracker-chrome:${project.id}`}
        className="tracker-chrome"
        defaultOpen={false}
        title="Progress & sync"
        summary={syncSummary}
      >
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
                title="Import CSV or download the board as Excel, PDF, or image"
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
              tasks={tasks}
              columns={columns}
              onSyncNow={syncNow}
              onLinked={setSync}
              onColumnsChange={(next) => setColumns(resolveColumns(next))}
              onError={onError}
              onResolvedConflicts={(ids, keepLocal) => {
                const idSet = new Set(ids);
                const now = Date.now();
                setTasks((list) =>
                  list.map((t) => {
                    if (!idSet.has(t.id) || !t.github) return t;
                    return {
                      ...t,
                      updatedAt: keepLocal ? now : t.updatedAt,
                      github: {
                        ...t.github,
                        conflict: false,
                        pending: keepLocal ? true : false,
                        syncedAt: now,
                      },
                    };
                  })
                );
                setOpenTask((cur) => {
                  if (!cur || !idSet.has(cur.id) || !cur.github) return cur;
                  return {
                    ...cur,
                    updatedAt: keepLocal ? now : cur.updatedAt,
                    github: {
                      ...cur.github,
                      conflict: false,
                      pending: keepLocal ? true : false,
                      syncedAt: now,
                    },
                  };
                });
                onChanged();
              }}
            />
          )}

          {hasRemote && <ActivityPanel projectId={project.id} sync={sync} onError={onError} />}
        </div>
      </CollapsibleSection>

      <KanbanBoard
        tasks={tasks}
        columns={columns}
        sprints={sprints}
        sprintFilter={sprintFilter}
        onSprintFilter={setSprintFilter}
        onOpenRoadmap={() => {
          setRoadmapOpen(true);
          capture("roadmap_opened", { item_count: sprints.length });
        }}
        onQuickAddSprint={quickAddSprint}
        onOpenAI={() => {
          setChatOpen(true);
          trackPanel("chat");
        }}
        onChange={save}
        onOpen={setOpenTask}
        onDelete={remove}
        onAdd={(title, opts) => add(title, opts, true)}
        onAddColumn={() => setAddColumnOpen(true)}
      />

      {addColumnOpen && (
        <AddColumnModal
          projectId={project.id}
          sync={sync}
          onClose={() => setAddColumnOpen(false)}
          onError={onError}
          onCreated={(result) => {
            if (result?.columns) setColumns(resolveColumns(result.columns));
            if (result?.sync) setSync(result.sync);
            onChanged();
          }}
        />
      )}

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
          columns={columns}
          sprints={sprints}
          initialSprintScope={sprintFilter}
          onImported={adoptImported}
          onClose={() => setCsvOpen(false)}
          onError={onError}
        />
      )}

      {openTask && (
        <TaskDetailModal
          task={openTask}
          tasks={tasks}
          columns={columns}
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
          onGitHubChange={(taskId, gh) => {
            setTasks((list) =>
              list.map((t) => (t.id === taskId ? { ...t, github: gh } : t))
            );
            setOpenTask((cur) =>
              cur && cur.id === taskId ? { ...cur, github: gh } : cur
            );
          }}
        />
      )}

      <ChatDock
        projectId={project.id}
        open={chatOpen}
        onClose={() => setChatOpen(false)}
        onAddStories={addStories}
        onError={onError}
      />
    </div>
  );
}
