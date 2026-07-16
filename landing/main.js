/* JumpStart landing — animations & interactions */
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

  /* ── Particles background ── */
  const canvas = document.getElementById("particles");
  const ctx = canvas.getContext("2d");
  let pts = [], W, H;
  const resize = () => {
    W = canvas.width = canvas.offsetWidth * devicePixelRatio;
    H = canvas.height = canvas.offsetHeight * devicePixelRatio;
  };
  resize();
  addEventListener("resize", resize);
  const N = Math.min(70, innerWidth / 16 | 0);
  for (let i = 0; i < N; i++) pts.push({
    x: Math.random(), y: Math.random(),
    vx: (Math.random() - 0.5) * 0.0004, vy: (Math.random() - 0.5) * 0.0004,
    r: Math.random() * 1.6 + 0.4
  });
  let heroVisible = true;
  new IntersectionObserver(([e]) => heroVisible = e.isIntersecting)
    .observe(document.querySelector(".hero"));
  const LINK = 120; // px link distance
  function draw() {
    if (heroVisible && !reduced) {
      ctx.clearRect(0, 0, W, H);
      const s = devicePixelRatio;
      for (const p of pts) {
        p.x += p.vx; p.y += p.vy;
        if (p.x < 0 || p.x > 1) p.vx *= -1;
        if (p.y < 0 || p.y > 1) p.vy *= -1;
        ctx.beginPath();
        ctx.arc(p.x * W, p.y * H, p.r * s, 0, 7);
        ctx.fillStyle = "rgba(46,224,110,0.35)";
        ctx.fill();
      }
      const max = (LINK * s) ** 2;
      for (let i = 0; i < pts.length; i++)
        for (let j = i + 1; j < pts.length; j++) {
          const dx = (pts[i].x - pts[j].x) * W, dy = (pts[i].y - pts[j].y) * H;
          const d = dx * dx + dy * dy;
          if (d < max) {
            ctx.strokeStyle = `rgba(46,224,110,${0.10 * (1 - d / max)})`;
            ctx.lineWidth = s * 0.5;
            ctx.beginPath();
            ctx.moveTo(pts[i].x * W, pts[i].y * H);
            ctx.lineTo(pts[j].x * W, pts[j].y * H);
            ctx.stroke();
          }
        }
    }
    requestAnimationFrame(draw);
  }
  draw();

  /* ── GSAP animations ── */
  if (typeof gsap === "undefined" || reduced) {
    document.querySelectorAll(".reveal").forEach(el => {
      el.style.opacity = 1; el.style.transform = "none";
    });
    return;
  }
  gsap.registerPlugin(ScrollTrigger);

  // Hero: staggered line reveal (HackerRank-style)
  gsap.from(".hero-title .line", {
    y: 60, opacity: 0, filter: "blur(8px)",
    duration: 1.1, stagger: 0.18, ease: "power3.out", delay: 0.15
  });
  gsap.from(".hero-sub, .hero-actions, .hero-meta", {
    y: 30, opacity: 0, duration: 0.9, stagger: 0.12, ease: "power2.out", delay: 0.75
  });

  // Scroll reveals
  document.querySelectorAll(".reveal").forEach(el => {
    gsap.to(el, {
      opacity: 1, y: 0, duration: 0.9, ease: "power3.out",
      scrollTrigger: { trigger: el, start: "top 85%" }
    });
  });

  // White sheet slides up over hero (parallax)
  gsap.from(".sheet", {
    yPercent: 6, ease: "none",
    scrollTrigger: { trigger: ".sheet", start: "top bottom", end: "top 40%", scrub: true }
  });

  // Mock window: subtle 3D settle + parallax
  gsap.fromTo("#mockWindow",
    { rotateX: 14, y: 60 },
    { rotateX: 0, y: 0, ease: "none",
      scrollTrigger: { trigger: "#mockWindow", start: "top 95%", end: "top 45%", scrub: true } }
  );

  // AI chat bubbles pop in sequence
  gsap.from("#aiMock > *", {
    y: 26, opacity: 0, duration: 0.7, stagger: 0.2, ease: "back.out(1.4)",
    scrollTrigger: { trigger: "#aiMock", start: "top 80%" }
  });

  // Section titles: slight parallax drift
  document.querySelectorAll(".dark-section .section-title").forEach(t => {
    gsap.from(t, {
      y: 40, ease: "none",
      scrollTrigger: { trigger: t, start: "top bottom", end: "top 50%", scrub: true }
    });
  });

  /* ── Card tilt + cursor glow ── */
  const fine = matchMedia("(pointer: fine)").matches;
  document.querySelectorAll(".card.tilt").forEach(card => {
    card.addEventListener("mousemove", e => {
      const r = card.getBoundingClientRect();
      const x = e.clientX - r.left, y = e.clientY - r.top;
      card.style.setProperty("--mx", x + "px");
      card.style.setProperty("--my", y + "px");
      if (!fine) return;
      const rx = ((y / r.height) - 0.5) * -7;
      const ry = ((x / r.width) - 0.5) * 7;
      gsap.to(card, { rotateX: rx, rotateY: ry, scale: 1.02, duration: 0.4, ease: "power2.out" });
    });
    card.addEventListener("mouseleave", () =>
      gsap.to(card, { rotateX: 0, rotateY: 0, scale: 1, duration: 0.6, ease: "elastic.out(1,0.5)" })
    );
  });
})();
