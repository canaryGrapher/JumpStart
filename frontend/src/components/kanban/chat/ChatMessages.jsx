import StoryPreview from "./StoryPreview";
import { storiesOf } from "../../../chats";
import { formatElapsed } from "../../../hooks/useAIProgress";

const SUGGESTIONS = [
  "How do I start this project?",
  "Break down a user authentication feature into stories",
  "What packages are installed?",
  "Where is routing configured?",
  "How do I add a new component?",
];

// Transcript for one session: bubbles, cited files, and the selectable
// story previews attached to assistant turns.
export default function ChatMessages({
  messages,
  busy,
  progress,
  indexed,
  bodyRef,
  onSuggest,
  onAddStories,
}) {
  return (
    <div className="chat-body" ref={bodyRef}>
      {messages.length === 0 && (
        <div className="chat-welcome">
          <p>
            Ask about this codebase, or describe what you want to build and
            I'll draft user stories you can pick from.
          </p>
          {!indexed && (
            <p className="chat-welcome-note">
              Index the project first for answers grounded in your actual code.
            </p>
          )}
          <div className="chat-suggestions">
            {SUGGESTIONS.map((s) => (
              <button key={s} onClick={() => onSuggest(s)}>
                {s}
              </button>
            ))}
          </div>
        </div>
      )}

      {messages.map((m, i) => {
        const stories = storiesOf(m);
        return (
          <div className={`chat-msg ${m.role}`} key={m.id || i}>
            <div className="chat-bubble">{m.content}</div>

            {(m.sources || []).length > 0 && (
              <div className="chat-sources">
                <span className="chat-sources-label">Read:</span>
                {m.sources.slice(0, 6).map((s) => (
                  <code key={s}>{s}</code>
                ))}
              </div>
            )}

            {stories.length > 0 && (
              <StoryPreview stories={stories} onAdd={onAddStories} />
            )}
          </div>
        );
      })}

      {busy && (
        <div className="chat-msg assistant">
          <div className="chat-bubble typing ai-progress">
            <span className="ai-progress-status">
              {progress?.chars > 0 ? "Writing the answer" : progress?.thinking ? "Thinking" : "Waiting for the model"}
              {" · "}
              {formatElapsed(progress?.elapsed || 0)}
            </span>
            {progress?.thinking && (
              <details className="ai-thinking">
                <summary>Show reasoning</summary>
                <pre>{progress.thinking.slice(-4000)}</pre>
              </details>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
