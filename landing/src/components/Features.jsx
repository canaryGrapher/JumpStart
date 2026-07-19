import ProcessCard from "./ProcessCard";

export default function Features() {
  return (
    <section class="section" id="features">
      <div class="container">
        <div class="split-head reveal">
          <div>
            <span class="eyebrow-dot">● Process Control</span>
            <h2>Running Local<br />Projects Is Easier.</h2>
          </div>
          <div class="split-right">
            <p>
              Add a project once and JumpStart finds every runnable part. One click starts it,
              one glance tells you everything.
            </p>
            <a href="#ai" class="btn btn-dark">Learn More <span class="arrow">→</span></a>
          </div>
        </div>

        <div class="duo">
          <ProcessCard
            name="Wails app"
            cmd="wails dev"
            path="/Users/you/Projects/jumpstart"
            scripts={["Build", "Generate", "Tidy", "Vet", "Runs"]}
          />
          <ProcessCard
            name="frontend (Vite)"
            cmd="npm run dev"
            path="/Users/you/Projects/jumpstart/frontend"
            scripts={["Build", "Start Dev", "Runs"]}
          />
        </div>
      </div>
    </section>
  );
}
