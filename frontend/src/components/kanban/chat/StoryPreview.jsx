import { useState } from "react";
import { formatDue } from "../../../dueDates";

// Stories the assistant proposed. Nothing is added to the board until the
// user picks: every story starts selected but can be unticked, and the
// button reflects the current selection count.
export default function StoryPreview({ stories, onAdd }) {
  const [selected, setSelected] = useState(() => stories.map((_, i) => i));
  const [added, setAdded] = useState(false);

  if (!stories || stories.length === 0) return null;

  const isOn = (i) => selected.includes(i);
  const toggle = (i) =>
    setSelected((s) => (s.includes(i) ? s.filter((x) => x !== i) : [...s, i]));

  const all = selected.length === stories.length;
  const toggleAll = () =>
    setSelected(all ? [] : stories.map((_, i) => i));

  const add = () => {
    const picked = selected
      .slice()
      .sort((a, b) => a - b)
      .map((i) => stories[i]);
    if (picked.length === 0) return;
    onAdd(picked);
    setAdded(true);
  };

  return (
    <div className="chat-stories">
      <div className="chat-stories-head">
        <label className="chat-select-all">
          <input type="checkbox" checked={all} onChange={toggleAll} />
          <span>Select all</span>
        </label>
        <span className="chat-stories-count">
          {selected.length} of {stories.length} selected
        </span>
      </div>

      {stories.map((s, i) => (
        <label
          className={`chat-story ${isOn(i) ? "selected" : ""}`}
          key={`${s.title}-${i}`}
        >
          <input
            type="checkbox"
            className="chat-story-check"
            checked={isOn(i)}
            onChange={() => toggle(i)}
          />
          <div className="chat-story-main">
            <div className="chat-story-title">{s.title}</div>
            {s.description && (
              <div className="chat-story-desc">{s.description}</div>
            )}
            <div className="chat-story-meta">
              {s.priority && (
                <span className={`kb-pill prio-${s.priority}`}>{s.priority}</span>
              )}
              {s.storyPoints > 0 && (
                <span className="kb-pill">{s.storyPoints} pts</span>
              )}
              {s.dueDate && (
                <span className="kb-pill due">🗓 {formatDue(s.dueDate)}</span>
              )}
              {(s.labels || []).map((l) => (
                <span className="kb-pill label" key={l}>
                  {l}
                </span>
              ))}
            </div>
            {(s.tasks || []).length > 0 && (
              <ul className="chat-story-tasks">
                {s.tasks.map((t, j) => (
                  <li key={j}>{t.title}</li>
                ))}
              </ul>
            )}
          </div>
        </label>
      ))}

      <button
        className="btn primary small"
        onClick={add}
        disabled={selected.length === 0}
      >
        {added
          ? "Add selected again"
          : `+ Add ${selected.length} selected to board`}
      </button>
    </div>
  );
}
