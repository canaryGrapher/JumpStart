// Small inline confirmation that renders directly beneath the button
// that triggered it (rather than a separate modal). Used for branch
// switch / create / delete so destructive-ish actions get a second tap.
export default function ConfirmInline({ message, confirmLabel = "Confirm", danger, busy, onConfirm, onCancel }) {
  return (
    <div className={`git-confirm ${danger ? "danger" : ""}`}>
      <span className="git-confirm-msg">{message}</span>
      <div className="git-confirm-actions">
        <button className="btn tiny" disabled={busy} onClick={onCancel}>
          Cancel
        </button>
        <button
          className={`btn tiny ${danger ? "danger" : "primary"}`}
          disabled={busy}
          onClick={onConfirm}
        >
          {confirmLabel}
        </button>
      </div>
    </div>
  );
}
