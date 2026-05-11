import React from "react";
import { createRoot } from "react-dom/client";
import AlmanachStudio from "./almanach-studio";

const rootElement = document.getElementById("root");
if (rootElement) {
  const root = createRoot(rootElement);
  root.render(React.createElement(AlmanachStudio));
} else {
  console.error("Almanach Studio: #root element not found");
}
