import { useState } from "react";
import { DetectScripts } from "../../api";

const uid = () => crypto.randomUUID();

const emptyScript = () => ({ id: uid(), name: "", command: "" });

// ScriptsEditor edits the one-off commands attached to a subprocess.
// "Auto-detect" scans the working directory for npm scripts, Makefile
// targets, Go flags, Cargo/Poetry/Composer tasks and similar.
export default function ScriptsEditor({ scripts = [], dir, onChange }) {
  const [detecting, setDetecting] = useState(false);
  const [msg, setMsg] = useState(null); // { text, kind }

  const set = (i, patch) =>
    onChange(scripts.map((s, idx) => (idx === i ? { ...s, ...patch } : s)));

  const remove = (i) => onChange(scripts.filter((_, idx) => idx !== i));

  const add = () => onChange([...scripts, emptyScript()]);

  const autoDetect = async () => {
    if (!dir?.trim()) {
      setMsg({ text: "Set the working directory first.", kind: "err" });
      return;
    }
    setDetecting(true);
    setMsg(null);
    try {
      const found = await DetectScripts(dir);
      if (!found || found.length === 0) {
        setMsg({ text: "No scripts found in this folder.", kind: "" });
        return;
      }
      // Keep whatever is already configured; only add new commands.
      const have = new Set(scripts.map((s) => s.command.trim()));
      const added = found
        .filter((f) => !have.has(f.command))
        .map((f) => ({ id: uid(), name: f.name, command: f.command, source: f.source }));
      if (added.length === 0) {
        setMsg({ text: "Everything found is already added.", kind: "" });
        return;
      }
      onChange([...scripts.filter((s) => s.name.trim() || s.command.trim()), ...added]);
      setMsg({
        text: `Added ${added.length} script${added.length > 1 ? "s" : ""}.`,
        kind: "ok",
      });
    } catch (e) {
      setMsg({ text: String(e), kind: "err" });
    } finally {
      setDetecting(false);
    }
  };

  return (
    <div className="field">
      <div className="scripts-head">
        <label>Scripts</label>
        <div className="actions">
          <button className="btn small" disabled={detecting} onClick={autoDetect}>
            {detecting ? "Scanning…" : "Auto-detect"}
          </button>
          <button className="btn small" onClick={add}>
            Add
          </button>
        </div>
      </div>
      <p className="row-hint">
        One-off commands shown as buttons on the process card, e.g. Migrate →
        <code> go run . --migrate</code>
      </p>

      {msg && <p className={`detect-note ${msg.kind}`}>{msg.text}</p>}

      {scripts.map((s, i) => (
        <div className="script-row" key={s.id}>
          <input
            className="script-name"
            value={s.name}
            placeholder="Migrate"
            onChange={(e) => set(i, { name: e.target.value })}
          />
          <input
            className="script-command"
            value={s.command}
            placeholder="go run . --migrate"
            onChange={(e) => set(i, { command: e.target.value })}
          />
          <button className="link-btn" onClick={() => remove(i)}>
            Remove
          </button>
        </div>
      ))}
    </div>
  );
}
