const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

export function createMockProvisioningClient({ log = () => {} } = {}) {
  return {
    async chooseDevice() {
      log("Opening mock device chooser");
      await wait(350);
      const device = { id: "mock-almanach-printer", name: "ALM_MOCK01" };
      log(`Selected ${device.name}`);
      return device;
    },

    async connect(device) {
      log(`Connecting to ${device.name}`);
      await wait(450);
      log("Mock GATT connection established");
    },

    async establishSession({ pop }) {
      log("Starting mock Security 1 session");
      await wait(450);
      if (!String(pop || "").trim()) {
        throw new Error("Proof-of-possession is required.");
      }
      log("Mock proof-of-possession accepted");
    },

    async sendCredentials({ ssid, password }) {
      log(`Sending credentials for ${ssid}`);
      await wait(700);
      if (!String(ssid || "").trim()) throw new Error("SSID is required.");
      if (String(password || "").toLowerCase() === "fail") {
        throw new Error("Mock authentication failure. Use any password except 'fail'.");
      }
      log("Mock firmware accepted WiFi credentials");
    },

    async waitForResult() {
      log("Waiting for printer to join WiFi");
      await wait(800);
      return {
        ok: true,
        deviceName: "ALM_MOCK01",
        message: "Mock printer joined WiFi. Real IP discovery will be added after hardware validation.",
        ip: null,
      };
    },

    async disconnect() {
      log("Disconnected mock BLE session");
    },
  };
}
