import { render } from "solid-js/web";
import App from "./App";
import { initAnalytics } from "./analytics";
import "./styles.css";

initAnalytics();

render(() => <App />, document.getElementById("root"));
