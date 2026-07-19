import { BrowserOpenURL } from "../api";

// Overlay banner driven by a remote JSON config (announcements/promos).
// Renders as a dismissible card floating over the app's bottom-right.
export default function AdOverlay({ banner, onDismiss }) {
  if (!banner) return null;
  const style = ["info", "promo", "warning"].includes(banner.style)
    ? banner.style
    : "info";
  return (
    <div className={`ad-overlay ad-${style}`} role="dialog" aria-label="Announcement">
      <button className="ad-close" title="Dismiss" onClick={onDismiss}>
        ✕
      </button>
      {banner.imageUrl && (
        <img className="ad-image" src={banner.imageUrl} alt="" />
      )}
      {banner.title && <strong className="ad-title">{banner.title}</strong>}
      {banner.message && <p className="ad-message">{banner.message}</p>}
      {banner.linkUrl && (
        <button
          className="btn small ad-cta"
          onClick={() => BrowserOpenURL(banner.linkUrl)}
        >
          {banner.linkText || "Learn more"}
        </button>
      )}
    </div>
  );
}
