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
            running
            pid={4821}
            ports={["34115"]}
            cpu="4.1"
            ram="312"
            scripts={["Build", "Generate", "Tidy", "Vet"]}
          />
          <ProcessCard
            name="frontend (Vite)"
            cmd="pnpm run dev"
            path="/Users/you/Projects/jumpstart/frontend"
            running
            pid={5102}
            ports={["5173"]}
            cpu="1.8"
            ram="148"
            scripts={["Build", "Start Dev"]}
          />
        </div>
      </div>
    </section>
  );
}
