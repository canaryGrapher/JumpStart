/* JumpStart landing — GSAP interactions, run once after mount. */
import gsap from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import { SHOT_PIECES } from "./components/hero/ExplodedShot.jsx";

const FLIGHT_SHADOW = "0 18px 40px rgba(0,0,0,0.18), 0 4px 12px rgba(0,0,0,0.08)";
const REST_SHADOW = "none";

export function initAnimations() {
  const reduced = matchMedia("(prefers-reduced-motion: reduce)").matches;
  if (reduced) {
    document.body.classList.add("reduced");
    document.querySelectorAll(".reveal").forEach((el) => {
      el.style.opacity = 1;
      el.style.transform = "none";
    });
    return;
  }
  gsap.registerPlugin(ScrollTrigger);

  initExplodedWindow();
  gsap.from(".hero-content > *", {
    y: 40, opacity: 0, duration: 1, stagger: 0.12, ease: "power3.out",
    scrollTrigger: { trigger: ".hero-content", start: "top 72%" },
  });

  document.querySelectorAll(".reveal").forEach((el) => {
    gsap.to(el, {
      opacity: 1, y: 0, duration: 0.9, ease: "power3.out",
      scrollTrigger: { trigger: el, start: "top 86%" },
    });
  });

  document.querySelectorAll(".split-head h2, .feature-copy h2").forEach((t) => {
    gsap.from(t, {
      y: 30, ease: "none",
      scrollTrigger: { trigger: t, start: "top bottom", end: "top 55%", scrub: true },
    });
  });

  const fine = matchMedia("(pointer: fine)").matches;
  document.querySelectorAll(".tilt").forEach((card) => {
    card.addEventListener("mousemove", (e) => {
      if (!fine) return;
      const r = card.getBoundingClientRect();
      const rx = ((e.clientY - r.top) / r.height - 0.5) * -5;
      const ry = ((e.clientX - r.left) / r.width - 0.5) * 5;
      gsap.to(card, { rotateX: rx, rotateY: ry, scale: 1.015, duration: 0.4, ease: "power2.out" });
    });
    card.addEventListener("mouseleave", () =>
      gsap.to(card, { rotateX: 0, rotateY: 0, scale: 1, duration: 0.6, ease: "elastic.out(1,0.55)" })
    );
  });

  gsap.from(".watermark", {
    y: 120, ease: "none",
    scrollTrigger: { trigger: "footer", start: "top bottom", end: "bottom bottom", scrub: true },
  });
}

/* Intro fades, then each live UI card flies in from past 100vw/100vh. */
function initExplodedWindow() {
  const stage = document.getElementById("stage");
  if (!stage) return;

  const pieces = SHOT_PIECES.map((p) => ({
    el: document.querySelector(`[data-layer="${p.id}"]`),
    ...p,
  })).filter((p) => p.el);

  if (!pieces.length) return;

  pieces.forEach((p, i) => {
    const f = p.from || {};
    gsap.set(p.el, {
      x: f.x ?? "0vw",
      y: f.y ?? "0vh",
      z: f.z ?? 40,
      rotate: f.rotate ?? 0,
      rotateX: f.rotateX ?? 0,
      rotateY: f.rotateY ?? 0,
      scale: 1,
      opacity: 0,
      boxShadow: FLIGHT_SHADOW,
      transformPerspective: 1400,
      transformOrigin: "50% 50%",
      force3D: true,
      zIndex: 20 + i,
    });
  });

  gsap.set("#win", {
    rotationX: 8,
    rotationY: -3,
    transformPerspective: 1400,
  });
  gsap.set(".shot-plate", { opacity: 0 });

  const tl = gsap.timeline({
    scrollTrigger: {
      trigger: stage,
      start: "top top",
      end: "+=4200",
      scrub: 1.2,
      pin: true,
      pinSpacing: true,
      anticipatePin: 1,
    },
    defaults: { ease: "power2.inOut" },
  });

  tl.to(
    ".intro-text",
    {
      keyframes: [
        { scale: 1.14, duration: 0.28, ease: "none" },
        { scale: 2.8, opacity: 0, duration: 0.38, ease: "power3.in" },
      ],
    },
    0
  );
  tl.to(".intro-hint", { opacity: 0, duration: 0.12 }, 0.04);
  tl.to("#intro", { autoAlpha: 0, duration: 0.32, ease: "power2.in" }, 0.4);

  const t0 = 0.7;
  tl.to(".shot-plate", { opacity: 1, duration: 0.55, ease: "power3.out" }, t0);
  tl.to("#win", { rotationX: 0, rotationY: 0, duration: 1.4, ease: "power3.inOut" }, t0);

  const order = [
    "chrome", "search", "nav", "title",
    "stat1", "stat2", "stat3", "stat4",
    "favorites", "all", "foot",
    "recent", "used", "port",
  ];
  const byId = Object.fromEntries(pieces.map((p) => [p.id, p]));

  order.forEach((id, i) => {
    const p = byId[id];
    if (!p) return;
    const start = t0 + 0.18 + i * 0.09;
    tl.to(
      p.el,
      {
        x: 0,
        y: 0,
        z: 0,
        rotate: 0,
        rotateX: 0,
        rotateY: 0,
        scale: 1,
        opacity: 1,
        boxShadow: FLIGHT_SHADOW,
        duration: 0.85,
        ease: "power3.out",
      },
      start
    );
    tl.to(
      p.el,
      {
        boxShadow: REST_SHADOW,
        duration: 0.35,
        ease: "power2.inOut",
      },
      start + 0.7
    );
  });

  /* Flatten to 2D so the GPU stops re-rasterizing soft layers. */
  const settleAt = t0 + 0.18 + order.length * 0.09 + 0.95;
  tl.set(
    pieces.map((p) => p.el),
    {
      x: 0,
      y: 0,
      z: 0,
      rotate: 0,
      rotateX: 0,
      rotateY: 0,
      scale: 1,
      opacity: 1,
      boxShadow: REST_SHADOW,
      zIndex: 1,
      force3D: false,
      clearProps: "filter,transformPerspective",
    },
    settleAt
  );
  tl.set("#win", {
    rotationX: 0,
    rotationY: 0,
    clearProps: "transformPerspective",
  }, settleAt);
  tl.to({}, { duration: 0.85 }, settleAt + 0.05);

  requestAnimationFrame(() => ScrollTrigger.refresh());
  if (document.readyState !== "complete") {
    window.addEventListener("load", () => ScrollTrigger.refresh(), { once: true });
  }
}
