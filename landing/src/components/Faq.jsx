import { For } from "solid-js";

const FAQS = [
  {
    q: "What do I need to run JumpStart?",
    a: "macOS or Windows. Download the build for your platform, add your first project, and go. The AI features use a local Ollama model if you have one installed; everything else works out of the box. A Linux build is available on the releases page too.",
    open: true,
  },
  {
    q: "Is it another heavy cross-platform app?",
    a: "No. JumpStart is a lightweight native app. On macOS it uses a native titlebar, translucent sidebar, and system appearance support; on Windows it feels right at home too. It's built to be fast, not bloated.",
  },
  {
    q: "What happens to my processes when I quit?",
    a: "Closing JumpStart cleanly stops all managed processes. Stop asks each process group to shut down gracefully, then force-stops anything still running a few seconds later.",
  },
  {
    q: "Where is my data stored?",
    a: "Locally, in ~/.jumpstart/config.json. Programmatic imports are read from ~/.jumpstart/import.json. Git tokens live in your OS keychain (macOS Keychain / Windows Credential Manager). Nothing syncs to a server.",
  },
  {
    q: "Does my code or data ever leave my machine?",
    a: "No. Projects, boards, and AI drafts all stay local. The only network calls are the ones you ask for: pushing to your git remote, publishing a release, and checking for app updates.",
  },
  {
    q: "How do updates work?",
    a: "JumpStart checks for new releases in the background and shows a small banner when one is available; one click takes you to the latest download. You can also check manually from Preferences → Updates, or dismiss a version to snooze it.",
  },
];

export default function Faq() {
  return (
    <section class="section faq" id="faq">
      <div class="container narrow center">
        <span class="eyebrow-dot">● Our FAQs</span>
        <h2 class="reveal">JumpStart FAQs</h2>
        <p class="faq-sub reveal">
          Everything you need to know before replacing your wall of terminal tabs.
        </p>
        <div class="faq-cta reveal">
          <a href="#" class="btn btn-dark">More Questions <span class="arrow">→</span></a>
          <a href="#" class="btn btn-outline">Contact Us</a>
        </div>
        <div class="faq-list">
          <For each={FAQS}>
            {(f) => (
              <details class="reveal" open={f.open}>
                <summary>{f.q}</summary>
                <p>{f.a}</p>
              </details>
            )}
          </For>
        </div>
      </div>
    </section>
  );
}
