// Analytics for the JumpStart desktop app (separate property from the
// landing site). GA4 (gtag.js) + Microsoft Clarity, loaded from Vite env
// vars so dev builds no-op unless IDs are provided.
//
//   VITE_GA_ID       Google Analytics 4 Measurement ID   (e.g. G-XXXXXXXXXX)
//   VITE_CLARITY_ID  Microsoft Clarity project ID         (e.g. abcdefghij)
//
// Keep these DIFFERENT from the landing site's IDs so app and web traffic
// stay in separate properties. See frontend/.env.example.

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
  // Desktop app: disable automatic page_view; we send events explicitly.
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

// Fire an event to BOTH GA and Microsoft Clarity. Safe no-op when either is
// unconfigured. In Clarity, params become filterable custom tags.
export function track(name, params = {}) {
  if (window.gtag) window.gtag("event", name, params);
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

// Set user-scoped properties (e.g. project_count) for segmentation. Sent to
// GA user_properties and mirrored as Clarity tags.
export function setUserProperties(props) {
  if (window.gtag) window.gtag("set", "user_properties", props);
  if (window.clarity) {
    try {
      for (const [k, v] of Object.entries(props)) {
        window.clarity("set", k, String(v));
      }
    } catch (e) {
      /* clarity not ready */
    }
  }
}

// Record an app launch. Fires app_installed once (first run on this machine)
// and app_open every time. Since page_view is disabled for the app, these
// events are what mark a user as active in GA.
export function trackLaunch(version) {
  try {
    if (!localStorage.getItem("jumpstart_installed")) {
      localStorage.setItem("jumpstart_installed", "1");
      track("app_installed", { app_version: version || "" });
    }
  } catch (e) {
    /* localStorage unavailable */
  }
  track("app_open", { app_version: version || "" });
}

export function initAnalytics() {
  initGA();
  initClarity();
}
