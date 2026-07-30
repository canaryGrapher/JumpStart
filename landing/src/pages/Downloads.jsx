import { createSignal, createMemo, onMount, For, Show } from "solid-js";
import PageShell from "../components/PageShell";
import ReleaseCard from "../components/downloads/ReleaseCard";
import { releases, status, loadReleases, detectPlatform } from "../releases";
import { RELEASES_PAGE } from "../downloads";

const CHANNELS = [
  { id: "stable", label: "Stable" },
  { id: "beta", label: "Beta" },
  { id: "all", label: "All" },
];

export default function Downloads() {
  const [channel, setChannel] = createSignal("stable");
  const [platform, setPlatform] = createSignal(null);

  onMount(() => {
    loadReleases();
    setPlatform(detectPlatform());
  });

  const visible = createMemo(() => {
    const all = releases();
    if (channel() === "stable") return all.filter((r) => !r.beta);
    if (channel() === "beta") return all.filter((r) => r.beta);
    return all;
  });

  // "Latest" marks the newest release within the current channel view, so the
  // badge stays meaningful when the beta tab is selected.
  const latestTag = createMemo(() => (visible()[0] ? visible()[0].tag : null));

  return (
    <PageShell>
      <main class="dl-page container">
        <header class="dl-head">
          <span class="eyebrow-dot">● Downloads</span>
          <h1>Every JumpStart Build</h1>
          <p>
            Grab the newest release, roll back to a previous version, or try a beta ahead of
            everyone else. Builds are published for macOS, Windows, and Linux, straight from
            the GitHub release that produced them.
          </p>
          <Show when={channel() === "beta"}>
            <p class="dl-warn">
              Beta builds ship new features first and can be rough around the edges. The app
              can follow this channel automatically from Settings → Updates.
            </p>
          </Show>
        </header>

        <div class="seg dl-seg">
          <For each={CHANNELS}>
            {(c) => (
              <button classList={{ on: channel() === c.id }} onClick={() => setChannel(c.id)}>
                {c.label}
              </button>
            )}
          </For>
        </div>

        <Show
          when={status() === "ready"}
          fallback={
            <p class="dl-empty">
              <Show
                when={status() === "loading"}
                fallback={
                  <>
                    Couldn't load the release list right now.{" "}
                    <a href={RELEASES_PAGE} target="_blank" rel="noopener noreferrer">
                      Browse releases on GitHub
                    </a>
                    .
                  </>
                }
              >
                Loading releases…
              </Show>
            </p>
          }
        >
          <Show
            when={visible().length > 0}
            fallback={
              <p class="dl-empty">
                No {channel() === "beta" ? "beta" : "stable"} releases published yet.
              </p>
            }
          >
            <div class="dl-list">
              <For each={visible()}>
                {(rel) => (
                  <ReleaseCard
                    release={rel}
                    latest={rel.tag === latestTag()}
                    platform={platform()}
                  />
                )}
              </For>
            </div>
          </Show>
        </Show>

        <p class="dl-foot-note">
          Older builds not listed here live on the{" "}
          <a href={RELEASES_PAGE} target="_blank" rel="noopener noreferrer">
            GitHub releases page
          </a>
          .
        </p>
      </main>
    </PageShell>
  );
}
