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

// Fire a GA event from anywhere in the app. Safe no-op when unconfigured.
export function track(name, params = {}) {
  if (window.gtag) window.gtag("event", name, params);
}

export function initAnalytics() {
  initGA();
  initClarity();
}
