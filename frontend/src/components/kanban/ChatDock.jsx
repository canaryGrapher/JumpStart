import { useEffect, useRef, useState } from "react";
import { chat, aiConfigured, getAISettings } from "../../ai";

// Preview of stories the assistant proposed, with an add-to-board action.
function StoryPreview({ stories, onAdd }) {
  if (!stories || stories.length === 0) return null;
  return (
    <div className="chat-stories">
      {stories.map((s, i) => (
        <div className="chat-story" key={i}>
          <div className="chat-story-title">▣ {s.title}</div>
          {s.description && <div className="chat-story-desc">{s.description}</div>}
          {(s.tasks || []).length > 0 && (
            <ul className="chat-story-tasks">
              {s.tasks.map((t, j) => (
                <li key={j}>{t.title}</li>
              ))}
            </ul>
          )}
        </div>
      ))}
      <button className="btn primary small" onClick={() => onAdd(stories)}>
        + Add {stories.length} to board
      </button>
    </div>
  );
}

// Chat bar pinned to the bottom of the board; expands to a fullscreen
// overlay for generating stories with the local model.
export default function ChatDock({ projectName, onAddStories, onError }) {
  const [expanded, setExpanded] = useState(false);
  const [messages, setMessages] = useState([]); // {role, content, stories?}
  const [input, setInput] = useState("");
  const [busy, setBusy] = useState(false);
  const bodyRef = useRef(null);

  useEffect(() => {
    if (bodyRef.current) bodyRef.current.scrollTop = bodyRef.current.scrollHeight;
  }, [messages, expanded]);

  const send = async () => {
    const text = input.trim();
    if (!text || busy) return;
    if (!aiConfigured()) {
      onError && onError("Pick an Ollama model in Preferences → AI first.");
      return;
    }
    setInput("");
    const history = [...messages, { role: "user", content: text }];
    setMessages(history);
    setBusy(true);
    try {
      const r = await chat(
        history.map((m) => ({ role: m.role, content: m.content })),
        projectName
      );
      setMessages([
        ...history,
        {
          role: "assistant",
          content: r.reply || "Here are some stories.",
          stories: r.stories || [],
        },
      ]);
    } catch (e) {
      onError && onError(String(e));
      setMessages([
        ...history,
        { role: "assistant", content: "Sorry, I couldn't reach the model." },
      ]);
    } finally {
      setBusy(false);
    }
  };

  const addStories = (stories) => {
    onAddStories(stories);
    onError && onError(`Added ${stories.length} story(ies) to the board.`);
  };

  const model = getAISettings().model || "no model";

  const composer = (
    <div className="chat-composer">
      <input
        value={input}
        placeholder="Describe a feature and I'll draft user stories..."
        onChange={(e) => setInput(e.target.value)}
        onFocus={() => setExpanded(true)}
        onKeyDown={(e) => e.key === "Enter" && send()}
      />
      <button className="btn primary" onClick={send} disabled={busy}>
        {busy ? "…" : "Send"}
      </button>
    </div>
  );

  if (!expanded) {
    return (
      <div className="chat-dock collapsed" onClick={() => setExpanded(true)}>
        <span className="chat-spark">✨</span>
        <span className="chat-hint">Ask AI to generate user stories…</span>
        <span className="chat-model">{model}</span>
      </div>
    );
  }

  return (
    <div className="chat-overlay" onClick={() => !busy && setExpanded(false)}>
      <div className="chat-panel" onClick={(e) => e.stopPropagation()}>
        <div className="chat-head">
          <div>
            <strong>Story assistant</strong>
            <span className="chat-model"> · {model}</span>
          </div>
          <button className="icon-btn" onClick={() => setExpanded(false)}>
            ✕
          </button>
        </div>

        <div className="chat-body" ref={bodyRef}>
          {messages.length === 0 && (
            <div className="chat-welcome">
              <p>
                Describe what you want to build. I'll draft user stories with
                acceptance criteria and child tasks, then you can add them to the
                board.
              </p>
              <div className="chat-suggestions">
                {[
                  "Break down a user authentication feature into stories",
                  "Draft stories for a settings page",
                  "Plan an onboarding flow",
                ].map((s) => (
                  <button key={s} onClick={() => setInput(s)}>
                    {s}
                  </button>
                ))}
              </div>
            </div>
          )}
          {messages.map((m, i) => (
            <div className={`chat-msg ${m.role}`} key={i}>
              <div className="chat-bubble">{m.content}</div>
              {m.stories && (
                <StoryPreview stories={m.stories} onAdd={addStories} />
              )}
            </div>
          ))}
          {busy && <div className="chat-msg assistant"><div className="chat-bubble typing">Thinking…</div></div>}
        </div>

        {composer}
      </div>
    </div>
  );
}
