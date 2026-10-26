import React from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App.jsx";
import "./styles.css";
import { MemoryRouter } from "react-router-dom";
import { LocaleProvider } from "../../../web/src/lib/i18n/LocaleProvider";
import { ThemeProvider } from "../../../web/src/lib/theme/ThemeProvider";
import { ToastProvider } from "../../../web/src/components/ui/Toast/ToastProvider";

createRoot(document.getElementById("root")).render(
  <React.StrictMode>
    <LocaleProvider><ThemeProvider><ToastProvider><MemoryRouter><App /></MemoryRouter></ToastProvider></ThemeProvider></LocaleProvider>
  </React.StrictMode>,
);
