import React from "react";
import { createRoot } from "react-dom/client";
import { PrototypeApp } from "./PrototypeApp.jsx";
import "./styles.css";
import "./prototype-pages.css";

createRoot(document.getElementById("root")).render(
  <React.StrictMode>
    <PrototypeApp />
  </React.StrictMode>,
);
