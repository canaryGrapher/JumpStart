import { useEffect, useRef, useState } from "react";
import { UpdateSprints, GitStatus, SaveBoardLayout, PatchTasks, GetProjectTasks } from "../api";
import { capture, trackPanel } from "../analytics";
import KanbanBoard from "./kanban/KanbanBoard";
import TaskDetailModal from "./kanban/TaskDetailModal";
import TasksCsvModal from "./kanban/TasksCsvModal";
import AddColumnModal from "./kanban/AddColumnModal";
import ChatDock from "./kanban/ChatDock";
import RoadmapModal from "./roadmap/RoadmapModal";
import { migrate, blankTask, uid, resolveColumns } from "./kanban/columns";
import { isValidDate } from "../dueDates";
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
export default function TaskTracker({ project, reloadToken = 0, onChanged, onError }) {
  const [tasks, setTasks] = useState((project.tasks || []).map(migrate));
  // The tasks as last read from or written to disk. Saves send only what
  // changed relative to this, so edits never overwrite changes made
  // elsewhere (MCP, agents, sync, another window) since the board loaded.
  const baseRef = useRef(new Map((project.tasks || []).map(migrate).map((t) => [t.id, t])));
  const saveChain = useRef(Promise.resolve());
  const setBase = (list) => {
    baseRef.current = new Map(list.map((t) => [t.id, t]));
  };
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

  // An external change (e.g. a JSON import) asks the board to re-read the
  // saved project without resetting the view.
  useEffect(() => {
    if (!reloadToken) return;
    const fresh = (project.tasks || []).map(migrate);
    setTasks(fresh);
    setBase(fresh);
    setSprints(migrateSprints(project.sprints));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [reloadToken]);

  // A sync pass returns the whole reconciled task list, so the board
  // adopts it wholesale. The open modal follows its task to the new copy
  // rather than showing a stale one.
  const adoptSynced = (next) => {
    const migrated = (next || []).map(migrate);
    setTasks(migrated);
    setBase(migrated);
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

  // Saves are queued so each one diffs against the result of the last.
  const save = (next) => {
    setTasks(next);
    const run = async () => {
      const base = baseRef.current;
      const changes = [];
      for (const t of next) {
        const b = base.get(t.id);
        if (!b) changes.push({ base: null, task: t });
        else if (JSON.stringify(t) !== JSON.stringify(b)) changes.push({ base: b, task: t });
      }
      const ids = new Set(next.map((t) => t.id));
      const deletes = [...base.keys()].filter((id) => !ids.has(id));
      if (!changes.length && !deletes.length) return;
      try {
        const saved = ((await PatchTasks(project.id, changes, deletes)) || []).map(migrate);
        setBase(saved);
        // Adopt what changed elsewhere, but keep any newer local edits that
        // are still waiting in the queue.
        setTasks((cur) => (cur === next ? saved : cur));
        onChanged();
      } catch (e) {
        onError(String(e));
      }
    };
    saveChain.current = saveChain.current.then(run, run);
    return saveChain.current;
  };

  // Pick up changes made elsewhere when the window regains focus, unless a
  // task editor is open (its own save merges safely).
  useEffect(() => {
    const refresh = async () => {
      if (openTask) return;
      await saveChain.current;
      try {
        const fresh = ((await GetProjectTasks(project.id)) || []).map(migrate);
        if (JSON.stringify(fresh) === JSON.stringify([...baseRef.current.values()])) return;
        setBase(fresh);
        setTasks(fresh);
      } catch {
        // offline or project removed: keep what is on screen
      }
    };
    window.addEventListener("focus", refresh);
    return () => window.removeEventListener("focus", refresh);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [project.id, openTask]);

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

  // Autosave path: same merge as update(), but the editor stays open.
  const updateQuiet = (task) => {
    const next = tasks.map((t) => {
      if (t.id !== task.id) return t;
      const github = task.github?.itemId ? task.github : t.github?.itemId ? t.github : task.github;
      return { ...task, github };
    });
    return save(next);
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
        dueDate: isValidDate(s.dueDate) ? s.dueDate : "",
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
                        conflictFields: [],
                        pending: !!keepLocal,
                        forcePush: !!keepLocal,
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
                      conflictFields: [],
                      pending: !!keepLocal,
                      forcePush: !!keepLocal,
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
        projectQuarters={project.quarters || []}
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
        statusMap={sync?.statusMap || {}}
        projectId={project.id}
        onAddRow={() =>
          add("New task", {
            type: "task",
            status: columns[0]?.id || "todo",
            sprintId: sprintFilter === "__all__" ? "" : sprintFilter,
          })
        }
        onError={onError}
        onSaveLayout={async (cols, moves) => {
          const result = await SaveBoardLayout(project.id, cols, moves);
          setColumns(resolveColumns(result.columns));
          const fresh = (result.tasks || []).map(migrate);
          setTasks(fresh);
          setBase(fresh);
          if (sync?.statusMap) {
            const keep = new Set((result.columns || []).map((c) => c.id));
            const statusMap = Object.fromEntries(Object.entries(sync.statusMap).filter(([k]) => keep.has(k)));
            setSync({ ...sync, statusMap });
          }
          onChanged();
        }}
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
          onAutoSave={updateQuiet}
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
