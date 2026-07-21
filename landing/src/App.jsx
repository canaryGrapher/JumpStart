import { Show } from "solid-js";
import { route } from "./router";
import Home from "./Home";
import Privacy from "./pages/Privacy";
import Terms from "./pages/Terms";

export default function App() {
  return (
    <Show when={route() === "home"} fallback={route() === "privacy" ? <Privacy /> : <Terms />}>
      <Home />
    </Show>
  );
}
