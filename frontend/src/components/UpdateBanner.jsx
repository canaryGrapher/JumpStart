import { BrowserOpenURL } from "../api";

// Slim top-of-window bar shown when a newer GitHub release exists.
export default function UpdateBanner({ update, onDismiss }) {
  if (!update) return null;
  return (
    <div className="update-banner" role="status">
      <span className="update-banner-text">
        <strong>Update available</strong> — JumpStart {update.latestVersion} is
        out (you have {update.currentVersion}).
      </span>
      <button
        className="btn small"
        onClick={() => BrowserOpenURL(update.releaseUrl)}
      >
        Download
      </button>
      <button
        className="update-banner-close"
        title="Dismiss"
        onClick={onDismiss}
      >
        ✕
      </button>
    </div>
  );
}
