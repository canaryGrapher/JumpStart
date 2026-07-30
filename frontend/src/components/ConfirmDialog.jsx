// Generic confirmation dialog. Buttons are labelled with the action they
// perform rather than OK/Cancel, so the choice reads unambiguously even if
// the body text is skimmed.
export default function ConfirmDialog({
  title,
  body,
  confirmLabel = "Confirm",
  cancelLabel = "Cancel",
  danger = false,
  onConfirm,
  onCancel,
}) {
  return (
    <div className="modal-overlay" onClick={onCancel}>
      <div
        className="modal confirm"
        onClick={(e) => e.stopPropagation()}
        role="alertdialog"
        aria-modal="true"
      >
        <h2>{title}</h2>
        {body && <p className="confirm-body">{body}</p>}
        <div className="confirm-actions">
          <button className="btn" onClick={onCancel}>
            {cancelLabel}
          </button>
          <button
            className={`btn ${danger ? "danger" : "primary"}`}
            autoFocus
            onClick={onConfirm}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
