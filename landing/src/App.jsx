import { Switch, Match, createEffect } from "solid-js";
import { route } from "./router";
import Home from "./Home";
import Privacy from "./pages/Privacy";
import Terms from "./pages/Terms";
import Downloads from "./pages/Downloads";
import Docs from "./pages/Docs";
import DocsMcp from "./pages/DocsMcp";
import { trackPageView } from "./analytics";

export default function App() {
  // Hash routing means the browser never navigates, so GA would otherwise
  // record one pageview for the whole visit and attribute every downloads
  // or privacy view to "/". This runs on mount and on every route change.
  createEffect(() => trackPageView(route()));

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
      <Match when={route() === "docs"}>
        <Docs />
      </Match>
      <Match when={route() === "docs-mcp"}>
        <DocsMcp />
      </Match>
    </Switch>
  );
}
