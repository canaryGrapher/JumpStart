// Analytics for the JumpStart landing site.
//
// GA4 (gtag.js) + Microsoft Clarity + Vercel Analytics. IDs come from Vite
// env vars so the same build works with real IDs in prod and no-ops in dev.
//
//   VITE_GA_ID       Google Analytics 4 Measurement ID   (e.g. G-XXXXXXXXXX)
//   VITE_CLARITY_ID  Microsoft Clarity project ID         (e.g. abcdefghij)
//
// Vercel Analytics needs no ID; it only reports when the site is deployed
// on Vercel. See .env.example. Nothing loads unless the matching ID is set.
//
// This is the SITE's analytics. The desktop app is deliberately separate:
// it reports to PostHog from its Go process and ships no GA or Clarity at
// all, so app and web traffic can never contaminate each other.
import { inject } from "@vercel/analytics";

const GA_ID = import.meta.env.VITE_GA_ID;
const CLARITY_ID = import.meta.env.VITE_CLARITY_ID;

function initGA() {
  if (!GA_ID) return;
  const s = document.createElement("script");
  s.async = true;
  s.src = `https://www.googletagmanager.com/gtag/js?id=${GA_ID}`;
  document.head.appendChild(s);

  window.dataLayer = window.dataLayer || [];
  window.gtag = function () {
    window.dataLayer.push(arguments);
  };
  window.gtag("js", new Date());
  // send_page_view is off because this is a hash-routed SPA: GA's automatic
  // pageview fires once on load and would never see #/downloads or
  // #/privacy. trackPageView sends them explicitly instead.
  window.gtag("config", GA_ID, { send_page_view: false });
}

function initClarity() {
  if (!CLARITY_ID) return;
  (function (c, l, a, r, i, t, y) {
    c[a] =
      c[a] ||
      function () {
        (c[a].q = c[a].q || []).push(arguments);
      };
    t = l.createElement(r);
    t.async = 1;
    t.src = "https://www.clarity.ms/tag/" + i;
    y = l.getElementsByTagName(r)[0];
    y.parentNode.insertBefore(t, y);
  })(window, document, "clarity", "script", CLARITY_ID);
}

function initVercel() {
  try {
    inject();
  } catch (e) {
    /* not deployed on Vercel */
  }
}

// Fire an event to BOTH GA and Microsoft Clarity. Safe no-op when either is
// unconfigured. In Clarity, params become filterable custom tags, which is
// what lets you jump from "people who clicked Download for Linux" to their
// actual session recordings.
export function track(name, params = {}) {
  if (window.gtag) window.gtag("event", name, { ...sessionProps, ...params });
  if (window.clarity) {
    try {
      window.clarity("event", name);
      for (const [k, v] of Object.entries(params)) {
        window.clarity("set", k, String(v));
      }
    } catch (e) {
      /* clarity not ready */
    }
  }
}

// --- Session context ---------------------------------------------------
//
// Attached to every event so any metric can be broken down by where the
// visitor came from and what they are browsing on. Collected once, because
// none of it changes mid-visit.

let sessionProps = {};

/** Read-only view of the session properties, for modules that need to
 *  branch on them (e.g. engagement reporting the visitor's platform). */
export function getSessionProps() {
  return sessionProps;
}

function detectDevice() {
  const w = window.innerWidth;
  if (w < 640) return "mobile";
  if (w < 1024) return "tablet";
  return "desktop";
}

// The visitor's OS decides which download button matters. Getting this
// wrong is the difference between "nobody wants Linux" and "the Linux
// button is below the fold on exactly the machines that would click it".
function detectPlatform() {
  const ua = navigator.userAgent;
  if (/Windows/i.test(ua)) return "windows";
  if (/Android/i.test(ua)) return "android";
  if (/iPhone|iPad|iPod/i.test(ua)) return "ios";
  if (/Mac OS X/i.test(ua)) return "macos";
  if (/Linux/i.test(ua)) return "linux";
  return "other";
}

// Referrer is reduced to its hostname. The full URL can carry a search
// query or a private path from whatever site linked here.
function referrerHost() {
  try {
    if (!document.referrer) return "direct";
    const host = new URL(document.referrer).hostname;
    return host === window.location.hostname ? "internal" : host;
  } catch (e) {
    return "unknown";
  }
}

function campaignProps() {
  const q = new URLSearchParams(window.location.search);
  const out = {};
  for (const key of ["utm_source", "utm_medium", "utm_campaign", "utm_content", "utm_term"]) {
    const v = q.get(key);
    if (v) out[key] = v.slice(0, 64);
  }
  const ref = q.get("ref");
  if (ref) out.ref = ref.slice(0, 64);
  return out;
}

// A returning visitor behaves nothing like a first-timer, and the two
// mixed together hide both. localStorage is enough here: this is a
// marketing site, not an identity system.
function isReturning() {
  try {
    const seen = localStorage.getItem("js_seen");
    localStorage.setItem("js_seen", "1");
    return seen === "1";
  } catch (e) {
    return false;
  }
}

function buildSessionProps() {
  return {
    device: detectDevice(),
    visitor_platform: detectPlatform(),
    referrer_host: referrerHost(),
    is_returning: isReturning(),
    ...campaignProps(),
  };
}

// --- Page views --------------------------------------------------------

/** Send an explicit page_view. Called on load and on every hash route
 *  change, because a hash-routed SPA performs no navigation GA can see. */
export function trackPageView(route) {
  if (window.gtag && GA_ID) {
    window.gtag("event", "page_view", {
      ...sessionProps,
      page_title: document.title,
      page_location: window.location.href,
      page_path: `/${route}`,
      route,
    });
  }
  // Clarity groups recordings by this tag, so "watch someone on the
  // downloads page" becomes a one-click filter.
  if (window.clarity) {
    try {
      window.clarity("set", "route", route);
    } catch (e) {
      /* not ready */
    }
  }
}

// --- Named events ------------------------------------------------------
//
// Wrappers rather than raw track() calls, so every call site spells the
// event name and its properties the same way. A typo in an event name is
// invisible until you go looking for a chart that was never populated.

/** A download button was clicked. location distinguishes hero / footer /
 *  downloads page, which is what shows whether the fold placement works. */
export function trackDownload(platform, location, extra = {}) {
  track("download", {
    platform,
    location,
    matches_visitor_os: platform === sessionProps.visitor_platform,
    ...extra,
  });
}

/** A link off the site was clicked. */
export function trackOutbound(target, location, extra = {}) {
  track("outbound_click", { target, location, ...extra });
}

/** An in-page anchor or nav item was used. Shows which sections people
 *  jump to rather than scroll past. */
export function trackNav(target, location) {
  track("nav_click", { target, location });
}

/** An FAQ item was expanded. The questions people open are the objections
 *  the page has not already answered. */
export function trackFaqOpened(index, question) {
  track("faq_opened", { index, question: String(question).slice(0, 64) });
}

export function initAnalytics() {
  sessionProps = buildSessionProps();
  initGA();
  initClarity();
  initVercel();
}
