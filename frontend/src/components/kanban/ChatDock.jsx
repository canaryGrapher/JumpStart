import { useEffect, useRef, useState } from "react";
import { aiConfigured, getAISettings } from "../../ai";
import useChatSessions from "./chat/useChatSessions";
import useCodeContext from "./chat/useCodeContext";
import ChatSessionList from "./chat/ChatSessionList";
import ChatMessages from "./chat/ChatMessages";
import ContextBar from "./chat/ContextBar";
import { track, trackPanel } from "../../analytics";

// Chat bar pinned to the bottom of the board. Expanding it opens the full
// assistant: saved conversations on the left, the transcript on the right,
// and a code-context strip on top. It both answers questions about the
// project and drafts stories the user can select and add to the board.
export default function ChatDock({ projectId, onAddStories, onError }) {
  const [expanded, setExpanded] = useState(false);
  const [input, setInput] = useState("");
  const bodyRef = useRef(null);

  const chat = useChatSessions(projectId, onError);
  const ctx = useCodeContext(projectId, onError);

  useEffect(() => {
    if (bodyRef.current) bodyRef.current.scrollTop = bodyRef.current.scrollHeight;
  }, [chat.messages, expanded]);

  const send = () => {
    const text = input.trim();
    if (!text || chat.busy) return;
    if (!aiConfigured()) {
      onError &&
        onError("No AI model selected. Choose one in Preferences → AI to start chatting.");
      return;
    }
    setInput("");
    chat.send(text);
  };

  // Stories the user actually adds to the board are the payoff signal for
  // the whole chat surface. Generated-but-ignored stories are counted by
  // ai_chat_message_sent's stories_generated on the Go side, so the two
  // together give an acceptance rate.
  const addStories = (stories) => {
    track("ai_suggestion_accepted", {
      surface: "chat_stories",
      story_count: stories.length,
      indexed: !!ctx.status?.indexed,
    });
    onAddStories(stories);
    const n = stories.length;
    onError && onError(`Added ${n} ${n === 1 ? "story" : "stories"} to Backlog.`);
  };

  const model = getAISettings().model;

  if (!expanded) {
    return (
      <div
        className="chat-dock collapsed"
        onClick={() => {
          setExpanded(true);
          trackPanel("chat");
        }}
      >
        <span className="chat-spark">✨</span>
        <span className="chat-hint">Ask about this project, or plan a feature</span>
        {ctx.status?.indexed && <span className="chat-model">indexed</span>}
        <span className="chat-model">{model || "Set up AI"}</span>
      </div>
    );
  }

  return (
    <div className="chat-overlay" onClick={() => !chat.busy && setExpanded(false)}>
      <div className="chat-panel wide" onClick={(e) => e.stopPropagation()}>
        <div className="chat-head">
          <div>
            <strong>Project assistant</strong>
            <span className="chat-model"> · {model || "no model selected"}</span>
          </div>
          <button className="icon-btn" onClick={() => setExpanded(false)}>
            ✕
          </button>
        </div>

        <ContextBar
          status={ctx.status}
          building={ctx.building}
          progress={ctx.progress}
          onBuild={ctx.build}
          onClear={ctx.clear}
        />

        <div className="chat-layout">
          <ChatSessionList
            sessions={chat.sessions}
            activeId={chat.activeId}
            onSelect={chat.open}
            onNew={chat.startNew}
            onDelete={chat.remove}
          />

          <div className="chat-main">
            <ChatMessages
              messages={chat.messages}
              busy={chat.busy}
              indexed={!!ctx.status?.indexed}
              bodyRef={bodyRef}
              onSuggest={setInput}
              onAddStories={addStories}
            />

            <div className="chat-composer">
              <input
                value={input}
                placeholder="Ask about the code, or describe a feature to plan…"
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && send()}
              />
              <button className="btn primary" onClick={send} disabled={chat.busy}>
                {chat.busy ? "…" : "Send"}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
