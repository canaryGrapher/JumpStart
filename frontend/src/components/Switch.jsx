// macOS-style toggle switch (System Settings look).
export default function Switch({ checked, onChange }) {
  return (
    <label className="switch">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
      />
      <span className="switch-track">
        <span className="switch-knob" />
      </span>
    </label>
  );
}
