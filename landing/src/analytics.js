// Analytics for the JumpStart landing site.
// GA4 (gtag.js) + Microsoft Clarity. IDs come from Vite env vars so the
// same build works with real IDs in prod and no-ops in dev.
//
//   VITE_GA_ID       Google Analytics 4 Measurement ID   (e.g. G-XXXXXXXXXX)
//   VITE_CLARITY_ID  Microsoft Clarity project ID         (e.g. abcdefghij)
//
// See .env.example. Nothing loads unless the matching ID is set.

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
  window.gtag("config", GA_ID);
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

// Fire a GA event. Safe no-op when GA is not configured.
export function track(name, params = {}) {
  if (window.gtag) window.gtag("event", name, params);
}

export function initAnalytics() {
  initGA();
  initClarity();
}
