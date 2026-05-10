export const ESP_IDF_PROVISIONING_SERVICE_UUID = "021a9004-0382-4aea-bff4-6b3f1c5adfb4";
export const ESP_IDF_PROVISIONING_SERVICE_UUID_FIRMWARE_ORDER = "b4df5a1c-3f6b-f4bf-ea4a-820304901a02";
export const ESP_IDF_PROVISIONING_SERVICE_UUIDS = Object.freeze([
  ESP_IDF_PROVISIONING_SERVICE_UUID,
  ESP_IDF_PROVISIONING_SERVICE_UUID_FIRMWARE_ORDER,
]);
export const ALMANACH_DEVICE_NAME_PREFIX = "ALM_";

function bluetoothUnavailableError() {
  return new Error("Web Bluetooth is not available. Open this page in Chrome or Edge from http://localhost.");
}

function formatBluetoothError(error, context = "bluetooth") {
  if (!error) return `Unknown ${context} error`;
  if (error.name === "NotFoundError" && context === "chooser") return "No printer selected. Choose an ALM_ device from Chrome's Bluetooth picker.";
  if (error.name === "NotFoundError" && context === "service") return `Connected to the printer, but none of the expected ESP-IDF provisioning services were found (${ESP_IDF_PROVISIONING_SERVICE_UUIDS.join(", ")}). Check the firmware service UUID or inspect the device in chrome://bluetooth-internals.`;
  if (error.name === "NotAllowedError") return "Bluetooth permission was denied. Retry and allow Chrome to access the printer.";
  if (error.name === "NetworkError") return "Bluetooth connection failed. Make sure the printer is still in pairing/provisioning mode.";
  return error.message || String(error);
}

async function findProvisioningService(gattServer, log) {
  let lastError = null;
  for (const uuid of ESP_IDF_PROVISIONING_SERVICE_UUIDS) {
    try {
      log(`Looking for ESP-IDF provisioning service ${uuid}`);
      const service = await gattServer.getPrimaryService(uuid);
      return { service, uuid };
    } catch (error) {
      lastError = error;
      log(`Service ${uuid} not found`);
    }
  }
  const error = new Error(formatBluetoothError(lastError, "service"));
  error.cause = lastError;
  throw error;
}

export function createEspIdfProvisioningClient({ log = () => {} } = {}) {
  let bluetoothDevice = null;
  let gattServer = null;
  let provisioningService = null;

  return {
    async chooseDevice() {
      if (!navigator.bluetooth) throw bluetoothUnavailableError();

      log(`Opening Chrome Bluetooth picker for ${ALMANACH_DEVICE_NAME_PREFIX} printers`);
      try {
        bluetoothDevice = await navigator.bluetooth.requestDevice({
          filters: [{ namePrefix: ALMANACH_DEVICE_NAME_PREFIX }],
          optionalServices: ESP_IDF_PROVISIONING_SERVICE_UUIDS,
        });
      } catch (error) {
        throw new Error(formatBluetoothError(error, "chooser"));
      }

      const name = bluetoothDevice.name || "Unnamed Almanach printer";
      log(`Selected ${name}`);
      bluetoothDevice.addEventListener("gattserverdisconnected", () => {
        log(`Disconnected from ${name}`);
      });
      return {
        id: bluetoothDevice.id,
        name,
        mode: "real",
        serviceUuid: ESP_IDF_PROVISIONING_SERVICE_UUID,
        bluetoothDevice,
      };
    },

    async connect(device) {
      if (!bluetoothDevice) {
        bluetoothDevice = device && device.bluetoothDevice;
      }
      if (!bluetoothDevice || !bluetoothDevice.gatt) {
        throw new Error("No Bluetooth device is selected.");
      }

      const name = bluetoothDevice.name || device?.name || "Almanach printer";
      log(`Connecting to ${name} over GATT`);
      try {
        gattServer = await bluetoothDevice.gatt.connect();
        log("GATT connection established");
        const found = await findProvisioningService(gattServer, log);
        provisioningService = found.service;
        log(`Found ESP-IDF provisioning service ${found.uuid}`);

        const characteristics = await provisioningService.getCharacteristics();
        log(`Discovered ${characteristics.length} provisioning characteristic(s)`);
      } catch (error) {
        throw new Error(formatBluetoothError(error, "connect"));
      }
    },

    async establishSession() {
      throw new Error("Real ESP-IDF Security 1 session is not implemented yet. Browser BLE transport is connected; next step is proto-ver and Security 1/protobuf support.");
    },

    async sendCredentials() {
      throw new Error("Real ESP-IDF WiFi credential transfer is not implemented yet.");
    },

    async waitForResult() {
      throw new Error("Real ESP-IDF provisioning result polling is not implemented yet.");
    },

    async disconnect() {
      if (bluetoothDevice?.gatt?.connected) {
        bluetoothDevice.gatt.disconnect();
      }
    },
  };
}
