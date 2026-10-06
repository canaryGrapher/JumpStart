const MODES = [
  { id: "light", label: "Light" },
  { id: "dark", label: "Dark" },
  { id: "system", label: "Auto" },
];

// Segmented Light / Dark / Auto control for Settings → Appearance.
export default function ThemeToggle({ theme, onChange }) {
  return (
    <div className="prefs-seg" role="group" aria-label="Theme">
      {MODES.map((m) => (
        <button
          key={m.id}
          type="button"
          className={`prefs-seg-btn ${theme === m.id ? "active" : ""}`}
          aria-pressed={theme === m.id}
          onClick={() => onChange(m.id)}
        >
          {m.label}
        </button>
      ))}
    </div>
  );
}
