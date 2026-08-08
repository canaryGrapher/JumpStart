// The commit message field plus its "Generate with AI" action. Split out
// of GitPanel so the AI drafting state (busy flag, hint line, edited
// tracking) stays local instead of adding four more useState calls to a
// panel that already manages branches, history and remotes.
import { useState } from "react";
import { generateCommitMessage, aiConfigured } from "../../ai";
import { track } from "../../analytics";

export default function CommitBox({ projectRoot, busy, onCommit, onError }) {
  const [msg, setMsg] = useState("");
  const [generating, setGenerating] = useState(false);
  const [hint, setHint] = useState(null);
  // Whether the current text came from the model, so the commit can be
  // reported as keeping the suggestion.
  const [suggested, setSuggested] = useState(null);

  const edit = (text) => {
    setMsg(text);
    setHint(null);
  };

  const generate = async () => {
    if (generating || busy) return;
    if (!aiConfigured()) {
      onError("Pick an Ollama model in Preferences → AI first.");
      return;
    }
    setGenerating(true);
    setHint(null);
    try {
      const r = await generateCommitMessage(projectRoot);
      setMsg(r.message || "");
      setSuggested(r.message || "");
      setHint({
        kind: "ok",
        text:
          `Summarised ${r.fileCount} file${r.fileCount === 1 ? "" : "s"} ` +
          `(${r.staged ? "staged changes" : "all uncommitted changes"})` +
          (r.truncated ? " · diff was large, so only part of it was read" : ""),
      });
    } catch (e) {
      setHint({ kind: "err", text: String(e) });
    } finally {
      setGenerating(false);
    }
  };

  const commit = async () => {
    const text = msg.trim();
    if (!text) return;
    // Committing is the explicit "keep". Edited-then-committed still
    // counts as accepted, but is flagged so the two can be told apart.
    if (suggested) {
      track("ai_suggestion_accepted", {
        surface: "commit_message",
        edited: text !== suggested.trim(),
        message_length: text.length,
      });
    }
    const hash = await onCommit(text);
    if (hash) {
      setMsg("");
      setSuggested(null);
      setHint(null);
    }
  };

  return (
    <div className="git-commit">
      <textarea
        className="git-commit-msg"
        rows={msg.includes("\n") ? 5 : 2}
        placeholder="Commit message"
        value={msg}
        onChange={(e) => edit(e.target.value)}
        // Enter inserts a newline (commit bodies are multi-line);
        // Cmd/Ctrl+Enter commits, matching every other commit editor.
        onKeyDown={(e) => {
          if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
            e.preventDefault();
            commit();
          }
        }}
      />
      <div className="git-commit-actions">
        <button
          className="btn small ai"
          disabled={busy || generating}
          onClick={generate}
          title="Draft a commit message from your pending changes"
        >
          {generating ? "Generating…" : "✨ Generate with AI"}
        </button>
        <button
          className="btn small primary"
          disabled={busy || generating || !msg.trim()}
          onClick={commit}
        >
          Commit
        </button>
      </div>
      {hint && <p className={`detect-note ${hint.kind}`}>{hint.text}</p>}
    </div>
  );
}
