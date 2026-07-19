import React from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import { initAnalytics } from "./analytics";
import "./styles.scss";

initAnalytics();

createRoot(document.getElementById("root")).render(<App />);
