import { Switch, Match } from "solid-js";
import { route } from "./router";
import Home from "./Home";
import Privacy from "./pages/Privacy";
import Terms from "./pages/Terms";
import Downloads from "./pages/Downloads";

export default function App() {
  return (
    <Switch fallback={<Home />}>
      <Match when={route() === "privacy"}>
        <Privacy />
      </Match>
      <Match when={route() === "terms"}>
        <Terms />
      </Match>
      <Match when={route() === "downloads"}>
        <Downloads />
      </Match>
    </Switch>
  );
}
