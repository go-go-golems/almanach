export function getBluetoothSupport() {
  if (typeof window === "undefined") {
    return { ok: false, code: "no-window", message: "Browser APIs are not available." };
  }
  if (!window.isSecureContext) {
    return {
      ok: false,
      code: "insecure-context",
      message: "Web Bluetooth requires HTTPS or localhost.",
      hint: "Run the setup page from http://localhost or a future HTTPS setup URL.",
    };
  }
  if (!navigator.bluetooth) {
    return {
      ok: false,
      code: "unsupported-browser",
      message: "This browser does not expose navigator.bluetooth.",
      hint: "Use Chrome, Edge, or Android Chrome. Safari and Firefox are not supported for this flow.",
    };
  }
  return { ok: true, code: "supported", message: "Web Bluetooth is available." };
}
