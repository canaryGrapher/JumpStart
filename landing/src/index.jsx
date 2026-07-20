import { render } from "solid-js/web";
import App from "./App";
import { initAnalytics } from "./analytics";
import "./styles.css";

initAnalytics();

const root = document.getElementById("root");
root.querySelector("[data-seo-fallback]")?.remove();
render(() => <App />, root);
