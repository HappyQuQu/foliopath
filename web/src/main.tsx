import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import { App } from "./app/App";
import { AppErrorBoundary } from "./app/AppErrorBoundary";
import { AppProviders } from "./app/AppProviders";
import { applyInitialTheme } from "./lib/theme/ThemeProvider";
import "./styles/global.css";

applyInitialTheme();

const root = document.getElementById("root");
if (root === null) throw new Error("FolioPath application root is missing");

createRoot(root).render(
  <StrictMode>
    <AppErrorBoundary>
      <AppProviders>
        <App />
      </AppProviders>
    </AppErrorBoundary>
  </StrictMode>,
);
