import { BrowserOpenURL } from "../../api";
import { LABELS, classify, PRODUCTION } from "../../buildInfo";
import jumpstartLogo from "../../assets/jumpstart.svg";

// formatDate renders the ldflags-stamped YYYY-MM-DD as "4 August 2026".
// Anything unparseable falls through unchanged rather than showing "Invalid Date".
export function formatDate(iso) {
  if (!iso) return "";
  const d = new Date(`${iso}T00:00:00Z`);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, {
    day: "numeric",
    month: "long",
    year: "numeric",
    timeZone: "UTC",
  });
}

// The masthead of the About pane: vendor logo, product name, version line.
export default function AboutIdentity({ info }) {
  const version = info?.version || "…";
  const kind = info ? classify(info.version) : null;
  const badge = kind && kind !== PRODUCTION ? LABELS[kind] : "";

  return (
    <div className="about-identity">
      <img className="about-logo" src={jumpstartLogo} alt={`${info?.appName || "JumpStart"} logo`} />

      <div className="about-titles">
        <h3 className="about-name">{info?.appName || "JumpStart"}</h3>

        <div className="about-version-line">
          <span className="about-version">Version {version}</span>
          {badge && <span className="beta-tag">{badge}</span>}
        </div>

        <button
          className="about-vendor"
          onClick={() => info?.vendorUrl && BrowserOpenURL(info.vendorUrl)}
        >
          by {info?.vendor || "Workvar"}
        </button>
      </div>
    </div>
  );
}
