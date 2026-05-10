export const ProvisioningStep = Object.freeze({
  SUPPORT: "support",
  DEVICE: "device",
  WIFI: "wifi",
  PROVISIONING: "provisioning",
  DONE: "done",
  ERROR: "error",
});

export const DEFAULT_SETUP_STATE = Object.freeze({
  step: ProvisioningStep.SUPPORT,
  support: null,
  device: null,
  ssid: "",
  password: "",
  pop: "",
  logs: [],
  error: null,
  result: null,
});

export function appendLog(logs, message) {
  return [...logs, { at: new Date().toISOString(), message }];
}

export function validateWifiCredentials({ ssid, password }) {
  const cleanSSID = String(ssid || "").trim();
  if (!cleanSSID) return "SSID is required.";
  if (cleanSSID.length > 32) return "SSID must be 32 characters or fewer.";
  if (String(password || "").length > 64) return "Password must be 64 characters or fewer.";
  return "";
}
