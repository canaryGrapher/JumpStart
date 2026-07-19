import EnvEditor from "./EnvEditor";
import EnvImportPrompt from "./EnvImportPrompt";
import ScriptsEditor from "./scripts/ScriptsEditor";

// One subprocess fieldset inside the project sheet.
export default function ProcForm({
  proc,
  index,
  rootHint,
  prompt,
  onChange,
  onRemove,
  onPickDir,
  onImportEnv,
  onDismissPrompt,
}) {
  return (
    <div className="proc-block">
      <div className="proc-block-top">
        <strong>{proc.name.trim() || `Subprocess ${index + 1}`}</strong>
        <button className="link-btn" onClick={onRemove}>
          Remove
        </button>
      </div>

      <div className="field">
        <label>Name</label>
        <input
          value={proc.name}
          placeholder="frontend (Next.js)"
          onChange={(e) => onChange({ name: e.target.value })}
        />
      </div>

      <div className="field">
        <label>Working Directory</label>
        <div className="row">
          <input
            value={proc.dir}
            placeholder={rootHint ? rootHint + "/frontend" : "/path/to/frontend"}
            onChange={(e) => onChange({ dir: e.target.value })}
          />
          <button className="btn" onClick={onPickDir}>
            Choose…
          </button>
        </div>
      </div>

      <div className="field">
        <label>Start Command</label>
        <input
          value={proc.command}
          placeholder="npm run dev"
          onChange={(e) => onChange({ command: e.target.value })}
        />
      </div>

      <div className="field">
        <label>Environment Variables</label>
        {prompt && (
          <EnvImportPrompt
            dir={prompt.dir}
            files={prompt.files}
            onImport={onImportEnv}
            onDismiss={onDismissPrompt}
          />
        )}
        <EnvEditor env={proc.env || {}} onChange={(env) => onChange({ env })} />
      </div>

      <ScriptsEditor
        scripts={proc.scripts || []}
        dir={proc.dir}
        onChange={(scripts) => onChange({ scripts })}
      />
    </div>
  );
}
