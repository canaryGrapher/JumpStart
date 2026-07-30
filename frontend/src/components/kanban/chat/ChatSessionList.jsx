// Sidebar of saved conversations. Picking one reopens it with its full
// history so the model keeps the earlier turns; "New chat" starts fresh
// with only the code context.
export default function ChatSessionList({
  sessions,
  activeId,
  onSelect,
  onNew,
  onDelete,
}) {
  return (
    <div className="chat-sessions">
      <button className="chat-new-btn" onClick={onNew}>
        + New chat
      </button>

      <div className="chat-session-scroll">
        {sessions.length === 0 && (
          <div className="chat-session-empty">No saved chats yet</div>
        )}

        {sessions.map((s) => (
          <div
            className={`chat-session ${s.id === activeId ? "active" : ""}`}
            key={s.id}
            onClick={() => onSelect(s.id)}
          >
            <div className="chat-session-body">
              <div className="chat-session-title">{s.title || "New chat"}</div>
              <div className="chat-session-sub">
                {s.messages} message{s.messages === 1 ? "" : "s"} ·{" "}
                {relTime(s.updatedAt)}
              </div>
            </div>
            <button
              className="chat-session-del"
              title="Delete chat"
              onClick={(e) => {
                e.stopPropagation();
                onDelete(s.id);
              }}
            >
              ×
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}

function relTime(ms) {
  if (!ms) return "";
  const mins = Math.floor((Date.now() - ms) / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  if (days < 7) return `${days}d ago`;
  return new Date(ms).toLocaleDateString();
}
