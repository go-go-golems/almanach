import React from "react";
import { createRoot } from "react-dom/client";
import ProvisioningWizard from "./provisioning/ProvisioningWizard";

const rootElement = document.getElementById("root");
if (rootElement) {
  createRoot(rootElement).render(<ProvisioningWizard />);
} else {
  console.error("Almanach Setup: #root element not found");
}
