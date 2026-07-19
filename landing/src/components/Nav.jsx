import { createSignal, onMount, onCleanup, For } from "solid-js";
import RocketLogo from "./RocketLogo";
import { track } from "../analytics";

export const NAV_LINKS = [
  { href: "#top", label: "Home" },
  { href: "#features", label: "Features" },
  { href: "#ship", label: "Ship" },
  { href: "#ai", label: "AI Board" },
  { href: "#faq", label: "FAQ" },
];

export default function Nav() {
  const [shrunk, setShrunk] = createSignal(false);
  const [open, setOpen] = createSignal(false);
  const [active, setActive] = createSignal(0);

  onMount(() => {
    const onScroll = () => {
      setShrunk(scrollY > 40);
      let current = 0;
      NAV_LINKS.forEach((l, i) => {
        const s = document.querySelector(l.href);
        if (s && scrollY + 200 >= s.offsetTop) current = i;
      });
      setActive(current);
    };
    addEventListener("scroll", onScroll, { passive: true });
    onCleanup(() => removeEventListener("scroll", onScroll));
  });

  return (
    <nav id="navbar" classList={{ shrunk: shrunk(), open: open() }}>
      <div class="nav-inner">
        <a class="logo" href="#top">
          <RocketLogo /> JumpStart<span class="logo-end">.</span>
        </a>
        <div class="nav-pill">
          <For each={NAV_LINKS}>
            {(l, i) => (
              <a href={l.href} classList={{ active: active() === i() }}>
                {l.label}
              </a>
            )}
          </For>
        </div>
        <div class="nav-cta">
          <a
            href="https://github.com/canaryGrapher/JumpStart"
            class="nav-link"
            onClick={() => track("outbound_github", { target: "source", location: "nav" })}
          >
            Source
          </a>
          <a href="#download" class="btn btn-outline">Download</a>
        </div>
        <button class="nav-burger" aria-label="Menu" onClick={() => setOpen(!open())}>
          <span></span>
          <span></span>
        </button>
      </div>
      <div class="nav-mobile" onClick={() => setOpen(false)}>
        <For each={NAV_LINKS}>{(l) => <a href={l.href}>{l.label}</a>}</For>
        <a href="#download" class="btn btn-dark">Download</a>
      </div>
    </nav>
  );
}
