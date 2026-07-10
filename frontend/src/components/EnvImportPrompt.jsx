import { useState } from "react";
import { ReadEnvFile } from "../api";

// Asks the user whether to import env vars from detected env files.
// If more than one file exists, they pick which one.
export default function EnvImportPrompt({ dir, files, onImport, onDismiss }) {
  const [error, setError] = useState("");

  const importFile = async (file) => {
    try {
      const env = await ReadEnvFile(dir + "/" + file);
      onImport(env, file);
    } catch (e) {
      setError(String(e));
    }
  };

  return (
    <div className="env-import">
      <span>
        Found {files.join(", ")}. Import environment variables
        {files.length > 1 ? " from:" : "?"}
      </span>
      <div className="row">
        {files.map((f) => (
          <button key={f} className="btn small" onClick={() => importFile(f)}>
            {files.length > 1 ? f : "Import " + f}
          </button>
        ))}
        <button className="link-btn" onClick={onDismiss}>
          No thanks
        </button>
      </div>
      {error && <span className="error">{error}</span>}
    </div>
  );
}
