import { useEffect } from "react";
import { BrowserOpenURL } from "../api";
import { track, trackOnce } from "../analytics";

// Overlay banner driven by a remote JSON config (announcements/promos).
// Renders as a dismissible card floating over the app's bottom-right.
export default function AdOverlay({ banner, onDismiss }) {
  const bannerId = banner ? banner.id || "unknown" : null;

  // shown / clicked / dismissed is the whole funnel for a banner. Without
  // banner_shown as the denominator, a click count says nothing about
  // whether an announcement actually landed.
  useEffect(() => {
    if (!bannerId) return;
    trackOnce(`banner:${bannerId}`, "banner_shown", { banner_id: bannerId });
  }, [bannerId]);

  if (!banner) return null;
  const style = ["info", "promo", "warning"].includes(banner.style)
    ? banner.style
    : "info";

  const dismiss = () => {
    track("banner_dismissed", { banner_id: bannerId, style });
    onDismiss();
  };

  const openLink = () => {
    track("banner_clicked", { banner_id: bannerId, style });
    BrowserOpenURL(banner.linkUrl);
  };

  return (
    <div className={`ad-overlay ad-${style}`} role="dialog" aria-label="Announcement">
      <button className="ad-close" title="Dismiss" onClick={dismiss}>
        ✕
      </button>
      {banner.imageUrl && (
        <img className="ad-image" src={banner.imageUrl} alt="" />
      )}
      {banner.title && <strong className="ad-title">{banner.title}</strong>}
      {banner.message && <p className="ad-message">{banner.message}</p>}
      {banner.linkUrl && (
        <button className="btn small ad-cta" onClick={openLink}>
          {banner.linkText || "Learn more"}
        </button>
      )}
    </div>
  );
}
