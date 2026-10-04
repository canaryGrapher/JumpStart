import React from "react";
import { createRoot } from "react-dom/client";
import {
  OverlayProvider,
  AlertProvider,
  ToastProvider,
} from "@pikoloo/darwin-ui";
import "@pikoloo/darwin-ui/styles.css";
import App from "./App";
import ErrorBoundary from "./components/ErrorBoundary";
import "./styles.scss";

// No analytics bootstrap here: the Go process owns ingestion and has
// already recorded app_launched by the time this bundle runs.

createRoot(document.getElementById("root")).render(
  <ErrorBoundary>
    <OverlayProvider>
      <AlertProvider>
        <ToastProvider>
          <App />
        </ToastProvider>
      </AlertProvider>
    </OverlayProvider>
  </ErrorBoundary>
);
