const MODES = ["light", "dark", "system"];
const LABELS = { light: "Light", dark: "Dark", system: "Auto" };

export default function ThemeToggle({ theme, onChange }) {
  return (
    <div className="theme-toggle">
      {MODES.map((m) => (
        <button
          key={m}
          className={theme === m ? "active" : ""}
          onClick={() => onChange(m)}
        >
          {LABELS[m]}
        </button>
      ))}
    </div>
  );
}
