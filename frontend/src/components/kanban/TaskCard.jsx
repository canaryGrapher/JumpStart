import { useState } from "react";
import { dueSummary, formatDue, isPastDue } from "../../dueDates";

const TYPE_ICON = { story: "▣", task: "○", bug: "▲" };

// One draggable card on the board. Story cards can expand to show and
// tick off their child tasks. Right-click opens a context menu
// (open / edit / delete) via onContextMenu.
export default function TaskCard({
  task,
  kids = [],
  onOpen,
  onContextMenu,
  onToggleChild,
  onDragStart,
  dragging,
}) {
  const [open, setOpen] = useState(false);
  const subs = task.subtasks || [];
  const subsDone = subs.filter((s) => s.done).length;
  const kidsDone = kids.filter((k) => k.status === "done").length;
  const isStory = task.type === "story";
  const pastDue = isPastDue(task);

  const openMenu = (e, target = task) => {
    if (!onContextMenu) return;
    e.preventDefault();
    e.stopPropagation();
    onContextMenu(e, target);
  };

  return (
    <div
      className={`kb-card type-${task.type || "task"} ${dragging ? "dragging" : ""} ${pastDue ? "overdue" : ""}`}
      draggable
      onContextMenu={openMenu}
      onDragStart={(e) => {
        e.dataTransfer.setData("text/plain", task.id);
        e.dataTransfer.effectAllowed = "move";
        onDragStart(task.id);
      }}
    >
      <div className="kb-card-main" onClick={() => onOpen(task)}>
        <div className="kb-card-title">
          <span className={`kb-type-badge t-${task.type || "task"}`}>
            {TYPE_ICON[task.type] || "○"}
          </span>
          {task.title}
        </div>
        {task.description && (
          <div className="kb-card-desc">{task.description}</div>
        )}
        <div className="kb-card-meta">
          {task.github?.number > 0 && (
            <span className="kb-pill gh-ref" title={task.github.repo || "GitHub issue"}>
              #{task.github.number}
            </span>
          )}
          {task.github?.conflict && (
            <span
              className="kb-pill gh-conflict"
              title={
                (task.github.conflictFields || [])[0]
                  ? `${task.github.conflictFields[0].label || "Field"} differs`
                  : "Local edits differ from GitHub"
              }
            >
              conflict
            </span>
          )}
          {task.github?.pending && (
            <span className="kb-pill gh-pending" title="Waiting for GitHub batch">
              pending
            </span>
          )}
          {pastDue && (
            <span className="kb-pill past-due" title={dueSummary(task)}>
              Past due
            </span>
          )}
          {task.dueDate && (
            <span
              className={`kb-pill due ${pastDue ? "overdue" : ""}`}
              title={dueSummary(task)}
            >
              🗓 {formatDue(task.dueDate)}
            </span>
          )}
          {task.priority && (
            <span className={`kb-pill prio-${task.priority}`}>
              {task.priority}
            </span>
          )}
          {task.assignee &&
            String(task.assignee)
              .split(",")
              .map((a) => a.trim())
              .filter(Boolean)
              .map((a) => (
                <span className="kb-pill assignee" key={a} title={a}>
                  @{a}
                </span>
              ))}
          {isStory && task.storyPoints > 0 && (
            <span className="kb-pill pts">{task.storyPoints} pts</span>
          )}
          {(task.labels || []).map((l) => (
            <span className="kb-pill label" key={l}>
              {l}
            </span>
          ))}
          {(task.links || []).length > 0 && (
            <span
              className="kb-pill links"
              title={(task.links || []).map((l) => l.title || l.url).join("\n")}
            >
              🔗 {task.links.length}
            </span>
          )}
          {(task.attachments || []).length > 0 && (
            <span
              className="kb-pill files"
              title={(task.attachments || []).map((a) => a.name).join("\n")}
            >
              📎 {task.attachments.length}
            </span>
          )}
          {subs.length > 0 && (
            <span className="kb-pill subs">
              ☑ {subsDone}/{subs.length}
            </span>
          )}
        </div>
      </div>

      {isStory && kids.length > 0 && (
        <div className="kb-kids">
          <button
            className="kb-kids-toggle"
            onClick={(e) => {
              e.stopPropagation();
              setOpen((o) => !o);
            }}
          >
            {open ? "▾" : "▸"} {kidsDone}/{kids.length} tasks
          </button>
          {open && (
            <div className="kb-kid-list">
              {kids.map((k) => (
                <div className="kb-kid" key={k.id}>
                  <button
                    className={`kb-kid-check ${k.status === "done" ? "done" : ""}`}
                    onClick={(e) => {
                      e.stopPropagation();
                      onToggleChild(k);
                    }}
                  >
                    {k.status === "done" ? "✓" : ""}
                  </button>
                  <span
                    className={`kb-kid-title ${k.status === "done" ? "done" : ""}`}
                    onClick={(e) => {
                      e.stopPropagation();
                      onOpen(k);
                    }}
                    onContextMenu={(e) => openMenu(e, k)}
                  >
                    {k.title}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
