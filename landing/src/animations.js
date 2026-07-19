/* JumpStart landing — GSAP interactions, run once after mount. */
import gsap from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";

/* Exploded start state for each window layer; scrolling composes them. */
const EXPLODED = {
  chrome: { y: -110, z: 190, rotationX: 24, opacity: 0.35 },
  side: { x: -190, z: 130, rotationY: 28, opacity: 0.35 },
  stats: { y: -70, z: 260, rotationX: 26, opacity: 0.4 },
  panels: { y: 90, z: 150, rotationX: -16, opacity: 0.4 },
  port: { y: 170, z: 70, rotationX: -24, opacity: 0.35 },
};

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

  /* Black opening screen first, then the window assembly; the hero text
     only reveals after (it sits below the pinned stage). The intro text's
     load fade lives in CSS so the scrubbed timeline owns its GSAP values
     and scrolling back up restores it correctly. */
  initExplodedWindow();
  gsap.from(".hero-content > *", {
    y: 40, opacity: 0, duration: 1, stagger: 0.12, ease: "power3.out",
    scrollTrigger: { trigger: ".hero-content", start: "top 72%" },
  });

  /* Scroll reveals */
  document.querySelectorAll(".reveal").forEach((el) => {
    gsap.to(el, {
      opacity: 1, y: 0, duration: 0.9, ease: "power3.out",
      scrollTrigger: { trigger: el, start: "top 86%" },
    });
  });

  /* Section headline drift */
  document.querySelectorAll(".split-head h2, .feature-copy h2").forEach((t) => {
    gsap.from(t, {
      y: 30, ease: "none",
      scrollTrigger: { trigger: t, start: "top bottom", end: "top 55%", scrub: true },
    });
  });

  /* Tilt + hover micro-interaction */
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

  /* Watermark parallax */
  gsap.from(".watermark", {
    y: 120, ease: "none",
    scrollTrigger: { trigger: "footer", start: "top bottom", end: "bottom bottom", scrub: true },
  });
}

/* Scroll story: the black opening screen scales away, revealing the
   graphified dashboard exploded in 3D; continued scrolling assembles it
   into the composed window. One pinned, scrubbed timeline drives both. */
function initExplodedWindow() {
  const stage = document.getElementById("stage");
  if (!stage) return;

  /* Place every layer at its exploded position. */
  Object.entries(EXPLODED).forEach(([name, pose]) => {
    gsap.set(`[data-layer="${name}"]`, { ...pose, transformPerspective: 1600 });
  });
  gsap.set("#win", { rotationX: 10, transformPerspective: 1600 });

  const tl = gsap.timeline({
    scrollTrigger: {
      trigger: stage,
      start: "top top",
      end: "+=1800",
      scrub: 1,
      pin: true,
      pinSpacing: true,
    },
    defaults: { ease: "power2.inOut" },
  });

  /* Phase 1 — the black screen resists, then gives way. The first stretch
     of scroll only nudges the text; past the threshold it rips open.
     One tween per element (keyframes for the text) so scrub reversal
     cleanly restores the intro when scrolling back to the top. */
  tl.to(
    ".intro-text",
    {
      keyframes: [
        { scale: 1.14, duration: 0.3, ease: "none" },
        { scale: 2.6, opacity: 0, duration: 0.35, ease: "power3.in" },
      ],
    },
    0
  );
  tl.to(".intro-hint", { opacity: 0, duration: 0.15 }, 0.05);
  tl.to("#intro", { autoAlpha: 0, duration: 0.35, ease: "power2.in" }, 0.42);

  /* This module can finish loading after the window "load" event, so
     ScrollTrigger misses its automatic refresh and never tracks scroll.
     Refresh explicitly once everything is mounted and laid out. */
  requestAnimationFrame(() => ScrollTrigger.refresh());
  if (document.readyState !== "complete") {
    window.addEventListener("load", () => ScrollTrigger.refresh(), { once: true });
  }

  /* Phase 2 — the exploded window assembles. */
  const t0 = 0.8;
  tl.to("#win", { rotationX: 0, duration: 1 }, t0);
  Object.keys(EXPLODED).forEach((name, i) => {
    tl.to(
      `[data-layer="${name}"]`,
      { x: 0, y: 0, z: 0, rotationX: 0, rotationY: 0, opacity: 1, duration: 1 },
      t0 + i * 0.12
    );
  });
}
