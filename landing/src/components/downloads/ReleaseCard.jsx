import { For, Show } from "solid-js";
import { PLATFORMS, formatDate } from "../../releases";
import { track, trackDownload, trackOutbound } from "../../analytics";
import { AppleLogo, WindowsLogo, LinuxLogo } from "../OSIcon";

const ICONS = { macos: AppleLogo, windows: WindowsLogo, linux: LinuxLogo };

// One release: version, channel badge, date, per-platform download buttons,
// and the raw release notes behind a disclosure.
export default function ReleaseCard(props) {
  const rel = () => props.release;

  return (
    <article class="dl-card" classList={{ latest: props.latest }}>
      <div class="dl-card-head">
        <div class="dl-card-title">
          <h3>{rel().tag}</h3>
          <Show when={props.latest}>
            <span class="dl-badge latest">Latest</span>
          </Show>
          <span class="dl-badge" classList={{ beta: rel().beta }}>
            {rel().beta ? "Beta" : "Stable"}
          </span>
        </div>
        <span class="dl-date">{formatDate(rel().publishedAt)}</span>
      </div>

      <div class="dl-assets">
        <For each={PLATFORMS}>
          {(p) => {
            const asset = () => rel().assets[p.id];
            const Icon = ICONS[p.id];
            return (
              <Show
                when={asset()}
                fallback={<span class="dl-asset missing">{p.label} — not published</span>}
              >
                <a
                  class="dl-asset"
                  classList={{ mine: props.platform === p.id }}
                  href={asset().url}
                  onClick={() =>
                    trackDownload(p.id, "downloads_page", {
                      version: rel().tag,
                      channel: rel().beta ? "beta" : "stable",
                      is_latest: !!props.latest,
                    })
                  }
                >
                  <Icon />
                  <span class="dl-asset-label">{p.label}</span>
                  <span class="dl-asset-size">{asset().size}</span>
                </a>
              </Show>
            );
          }}
        </For>
      </div>

      <div class="dl-card-foot">
        <Show when={rel().body.trim()}>
          <details class="dl-notes">
            {/* Whether people read release notes before downloading an
                older build is the question this answers. onToggle fires in
                both directions, so only the opening one is reported. */}
            <summary
              onClick={(e) => {
                if (!e.currentTarget.parentElement.open) {
                  track("release_notes_opened", {
                    version: rel().tag,
                    channel: rel().beta ? "beta" : "stable",
                  });
                }
              }}
            >
              Release notes
            </summary>
            <pre>{rel().body.trim()}</pre>
          </details>
        </Show>
        <a
          class="dl-gh-link"
          href={rel().url}
          target="_blank"
          rel="noopener noreferrer"
          onClick={() =>
            trackOutbound("github_release", "downloads_page", { version: rel().tag })
          }
        >
          View on GitHub <span class="arrow">→</span>
        </a>
      </div>
    </article>
  );
}
