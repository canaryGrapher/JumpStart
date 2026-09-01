import { BrowserOpenURL, ClipboardSetText } from "../../api";
import { formatDate } from "./AboutIdentity";

// Build facts table. Rows with no value are dropped rather than shown empty,
// so an unstamped local build simply has no patch-date row.
function rows(info) {
  return [
    ["Version", info.version],
    ["Patch date", formatDate(info.buildDate)],
    ["Platform", info.platform],
    ["Runtime", info.goVersion],
  ].filter(([, v]) => v);
}

export default function AboutDetails({ info }) {
  if (!info) return null;

  // One click gives support a paste-ready environment block, which is the
  // only reason most people open About in the first place.
  const copy = () =>
    ClipboardSetText(rows(info).map(([k, v]) => `${k}: ${v}`).join("\n"));

  return (
    <>
      <dl className="about-facts">
        {rows(info).map(([label, value]) => (
          <div className="about-fact" key={label}>
            <dt>{label}</dt>
            <dd>{value}</dd>
          </div>
        ))}
      </dl>

      <div className="about-links prefs-actions">
        <button className="btn" onClick={() => BrowserOpenURL(info.productUrl)}>
          Website
        </button>
        <button className="btn" onClick={() => BrowserOpenURL(info.repoUrl)}>
          GitHub
        </button>
        <button className="btn" onClick={copy}>
          Copy build info
        </button>
      </div>
    </>
  );
}
