/* JumpStart landing — light redesign interactions */
(() => {
  const reduced = matchMedia("(prefers-reduced-motion: reduce)").matches;
  if (reduced) document.body.classList.add("reduced");

  /* ── Sticky navbar shrink ── */
  const nav = document.getElementById("navbar");
  addEventListener("scroll", () => {
    nav.classList.toggle("shrunk", scrollY > 40);
  }, { passive: true });

  /* ── Mobile menu ── */
  document.getElementById("burger").addEventListener("click", () =>
    nav.classList.toggle("open")
  );
  nav.querySelectorAll(".nav-mobile a").forEach(a =>
    a.addEventListener("click", () => nav.classList.remove("open"))
  );

  /* ── Active pill link on scroll ── */
  const pillLinks = [...document.querySelectorAll(".nav-pill a")];
  const sections = pillLinks
    .map(a => document.querySelector(a.getAttribute("href")))
    .filter(Boolean);
  addEventListener("scroll", () => {
    let current = 0;
    sections.forEach((s, i) => { if (scrollY + 200 >= s.offsetTop) current = i; });
    pillLinks.forEach((a, i) => a.classList.toggle("active", i === current));
  }, { passive: true });

  if (typeof gsap === "undefined" || reduced) {
    document.querySelectorAll(".reveal").forEach(el => {
      el.style.opacity = 1; el.style.transform = "none";
    });
    return;
  }
  gsap.registerPlugin(ScrollTrigger);

  /* ── Hero intro ── */
  gsap.from(".eyebrow-dot.reveal-instant, .hero-title, .hero-sub, .hero-content .btn", {
    y: 40, opacity: 0, duration: 1, stagger: 0.14, ease: "power3.out", delay: 0.1
  });

  /* Dashboard settles in with 3D tilt + parallax */
  gsap.from(".dash", {
    y: 90, opacity: 0, rotateX: 10, duration: 1.3, ease: "power3.out", delay: 0.5
  });
  gsap.to(".dash", {
    y: -30, ease: "none",
    scrollTrigger: { trigger: ".dash-wrap", start: "top 80%", end: "bottom top", scrub: true }
  });

  /* Dash cards stagger */
  gsap.from(".dash-card", {
    y: 30, opacity: 0, duration: 0.8, stagger: 0.1, ease: "power2.out", delay: 0.9
  });

  /* ── Scroll reveals ── */
  document.querySelectorAll(".reveal").forEach(el => {
    gsap.to(el, {
      opacity: 1, y: 0, duration: 0.9, ease: "power3.out",
      scrollTrigger: { trigger: el, start: "top 86%" }
    });
  });

  /* ── Bars grow when visible ── */
  document.querySelectorAll(".bars").forEach(group => {
    gsap.from(group.querySelectorAll(".bar"), {
      scaleY: 0, duration: 0.9, stagger: 0.08, ease: "power3.out",
      scrollTrigger: { trigger: group, start: "top 88%" }
    });
  });

  /* ── Section headline drift ── */
  document.querySelectorAll(".split-head h2, .feature-copy h2").forEach(t => {
    gsap.from(t, {
      y: 30, ease: "none",
      scrollTrigger: { trigger: t, start: "top bottom", end: "top 55%", scrub: true }
    });
  });

  /* ── Logo strip fade ── */
  gsap.from(".logo-row span", {
    y: 20, opacity: 0, duration: 0.7, stagger: 0.08, ease: "power2.out",
    scrollTrigger: { trigger: ".logos", start: "top 90%" }
  });

  /* ── Tilt + hover micro-interaction ── */
  const fine = matchMedia("(pointer: fine)").matches;
  document.querySelectorAll(".tilt").forEach(card => {
    card.addEventListener("mousemove", e => {
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

  /* ── Watermark parallax ── */
  gsap.from(".watermark", {
    y: 120, ease: "none",
    scrollTrigger: { trigger: "footer", start: "top bottom", end: "bottom bottom", scrub: true }
  });
})();
