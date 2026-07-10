import ThemeToggle from "./ThemeToggle";

export const ACCENTS = [
  "blue",
  "purple",
  "pink",
  "red",
  "orange",
  "yellow",
  "green",
  "graphite",
];

export default function Preferences({
  theme,
  onThemeChange,
  accent,
  onAccentChange,
  onClose,
}) {
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal prefs" onClick={(e) => e.stopPropagation()}>
        <h2>Preferences</h2>

        <div className="prefs-row">
          <label>Appearance</label>
          <ThemeToggle theme={theme} onChange={onThemeChange} />
        </div>

        <div className="prefs-row">
          <label>Accent color</label>
          <div className="swatches">
            {ACCENTS.map((a) => (
              <button
                key={a}
                className={`swatch ${accent === a ? "active" : ""}`}
                data-swatch={a}
                title={a}
                aria-label={a}
                aria-pressed={accent === a}
                onClick={() => onAccentChange(a)}
              />
            ))}
          </div>
        </div>

        <div className="modal-actions">
          <button className="btn primary" onClick={onClose}>
            Done
          </button>
        </div>
      </div>
    </div>
  );
}
