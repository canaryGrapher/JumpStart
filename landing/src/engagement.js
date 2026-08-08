// Passive engagement tracking for the landing page.
//
// Click events tell you what people did. These tell you what they saw and
// how far they got, which is the half of the funnel a marketing site
// usually flies blind on: a download button nobody clicks means one thing
// if 80% of visitors scrolled past it and something completely different if
// 5% ever reached it.
//
// Everything here is throttled or fires once. Scroll handlers that report
// on every frame are how an analytics property ends up unusable.
import { track } from "./analytics";

const SCROLL_MILESTONES = [25, 50, 75, 90];

let started = false;
let startedAt = 0;
let maxDepth = 0;
const firedDepths = new Set();
const seenSections = new Set();

/** Set up scroll-depth, section-visibility, and time-on-page reporting.
 *  Safe to call more than once; only the first call takes effect. */
export function initEngagement() {
  if (started) return;
  started = true;
  startedAt = Date.now();

  watchScrollDepth();
  watchSections();
  watchExit();
}

// --- Scroll depth ------------------------------------------------------

function scrollPercent() {
  const doc = document.documentElement;
  const scrollable = doc.scrollHeight - window.innerHeight;
  if (scrollable <= 0) return 100;
  return Math.min(100, Math.round((window.scrollY / scrollable) * 100));
}

function watchScrollDepth() {
  let queued = false;
  const onScroll = () => {
    if (queued) return;
    queued = true;
    // rAF-coalesced: the handler runs at most once per frame no matter how
    // fast the wheel spins.
    requestAnimationFrame(() => {
      queued = false;
      const pct = scrollPercent();
      if (pct > maxDepth) maxDepth = pct;
      for (const milestone of SCROLL_MILESTONES) {
        if (pct >= milestone && !firedDepths.has(milestone)) {
          firedDepths.add(milestone);
          track("scroll_depth", { depth: milestone });
        }
      }
    });
  };
  addEventListener("scroll", onScroll, { passive: true });
}

// --- Section visibility ------------------------------------------------

// Sections are identified by their existing DOM ids, so adding a section to
// the page automatically adds it to the funnel with no extra wiring.
function watchSections() {
  if (typeof IntersectionObserver === "undefined") return;

  const sections = document.querySelectorAll("section[id], header[id]");
  if (sections.length === 0) return;

  const observer = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        const id = entry.target.id;
        if (!entry.isIntersecting || seenSections.has(id)) continue;
        seenSections.add(id);
        track("section_viewed", {
          section: id,
          seconds_to_reach: Math.round((Date.now() - startedAt) / 1000),
        });
        observer.unobserve(entry.target);
      }
    },
    // Half the section has to be on screen before it counts as seen; a
    // one-pixel sliver scrolling past is not a view.
    { threshold: 0.5 }
  );

  sections.forEach((s) => observer.observe(s));
}

// --- Exit summary ------------------------------------------------------

// One event per visit summarising how long they stayed, how far they got,
// and how much of the page they saw. It is the row you sort by when asking
// "which traffic source sends people who actually read this?".
//
// visibilitychange rather than beforeunload: mobile browsers frequently
// never fire beforeunload, which is exactly where bounce data matters most.
function watchExit() {
  let sent = false;
  const report = () => {
    if (sent) return;
    sent = true;
    track("page_exit", {
      seconds_on_page: Math.round((Date.now() - startedAt) / 1000),
      max_scroll_depth: maxDepth,
      sections_seen: seenSections.size,
    });
  };

  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "hidden") report();
  });
  addEventListener("pagehide", report);
}
