// A round icon button that reflects an on/off panel state (used for the
// Deps/Logs toggles on a process card). Kept separate from ProcessCard so
// that card stays focused on process state rather than button markup.
import Icon from "./Icon";

export default function IconToggleButton({ icon, active, label, onClick }) {
  return (
    <button
      className={`icon-btn outline ${active ? "active" : ""}`.trim()}
      title={active ? `Hide ${label.toLowerCase()}` : label}
      onClick={onClick}
    >
      <Icon d={icon} />
    </button>
  );
}
