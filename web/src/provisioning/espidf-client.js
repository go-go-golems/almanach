export const ESP_IDF_PROVISIONING_SERVICE_UUID = "021a9004-0382-4aea-bff4-6b3f1c5adfb4";
export const ESP_IDF_PROVISIONING_SERVICE_UUID_FIRMWARE_ORDER = "b4df5a1c-3f6b-f4bf-ea4a-820304901a02";
export const ESP_IDF_PROVISIONING_SERVICE_UUIDS = Object.freeze([
  ESP_IDF_PROVISIONING_SERVICE_UUID,
  ESP_IDF_PROVISIONING_SERVICE_UUID_FIRMWARE_ORDER,
]);
export const ALMANACH_DEVICE_NAME_PREFIX = "ALM_";
export const ESP_IDF_ENDPOINTS = Object.freeze({
  PROV_CTRL: "prov-ctrl",
  PROV_SCAN: "prov-scan",
  PROV_SESSION: "prov-session",
  PROV_CONFIG: "prov-config",
  PROTO_VER: "proto-ver",
});
export const FALLBACK_ENDPOINT_UUIDS = Object.freeze({
  [ESP_IDF_ENDPOINTS.PROV_CTRL]: "021aff4f-0382-4aea-bff4-6b3f1c5adfb4",
  [ESP_IDF_ENDPOINTS.PROV_SCAN]: "021aff50-0382-4aea-bff4-6b3f1c5adfb4",
  [ESP_IDF_ENDPOINTS.PROV_SESSION]: "021aff51-0382-4aea-bff4-6b3f1c5adfb4",
  [ESP_IDF_ENDPOINTS.PROV_CONFIG]: "021aff52-0382-4aea-bff4-6b3f1c5adfb4",
  [ESP_IDF_ENDPOINTS.PROTO_VER]: "021aff53-0382-4aea-bff4-6b3f1c5adfb4",
});

const textEncoder = new TextEncoder();
const textDecoder = new TextDecoder("utf-8");

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

function dataViewToText(value) {
  return textDecoder.decode(value.buffer.slice(value.byteOffset, value.byteOffset + value.byteLength));
}

async function writeCharacteristic(characteristic, text) {
  const data = textEncoder.encode(text);
  if (characteristic.writeValueWithResponse) {
    await characteristic.writeValueWithResponse(data);
    return;
  }
  await characteristic.writeValue(data);
}

async function readCharacteristicText(characteristic) {
  const value = await characteristic.readValue();
  return dataViewToText(value);
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

async function discoverEndpointCharacteristics(provisioningService, log) {
  const characteristics = await provisioningService.getCharacteristics();
  const endpoints = new Map();
  log(`Discovered ${characteristics.length} provisioning characteristic(s)`);

  for (const characteristic of characteristics) {
    let endpointName = null;
    try {
      const descriptors = await characteristic.getDescriptors();
      for (const descriptor of descriptors) {
        if (!descriptor.uuid.toLowerCase().includes("2901")) continue;
        const value = await descriptor.readValue();
        endpointName = dataViewToText(value).trim().toLowerCase();
        break;
      }
    } catch (error) {
      log(`Could not read descriptor for ${characteristic.uuid}: ${error.message || error}`);
    }

    if (endpointName) {
      endpoints.set(endpointName, characteristic);
      log(`Mapped endpoint ${endpointName} -> ${characteristic.uuid}`);
    }
  }

  for (const [name, uuid] of Object.entries(FALLBACK_ENDPOINT_UUIDS)) {
    if (endpoints.has(name)) continue;
    const characteristic = characteristics.find((candidate) => candidate.uuid.toLowerCase() === uuid);
    if (characteristic) {
      endpoints.set(name, characteristic);
      log(`Mapped endpoint ${name} by fallback UUID ${uuid}`);
    }
  }

  return endpoints;
}

export function createEspIdfProvisioningClient({ log = () => {} } = {}) {
  let bluetoothDevice = null;
  let gattServer = null;
  let provisioningService = null;
  let endpoints = new Map();

  async function sendEndpointText(endpointName, text) {
    const characteristic = endpoints.get(endpointName);
    if (!characteristic) {
      throw new Error(`ESP-IDF endpoint '${endpointName}' was not discovered.`);
    }
    await writeCharacteristic(characteristic, text);
    return readCharacteristicText(characteristic);
  }

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

        endpoints = await discoverEndpointCharacteristics(provisioningService, log);
        const endpointNames = Array.from(endpoints.keys()).sort();
        log(`Available ESP-IDF endpoints: ${endpointNames.join(", ") || "none"}`);
        const protoVersion = await sendEndpointText(ESP_IDF_ENDPOINTS.PROTO_VER, "v1.1");
        log(`proto-ver response: ${protoVersion}`);
        if (!protoVersion.includes("v1.1")) {
          throw new Error(`Unexpected proto-ver response: ${protoVersion}`);
        }
        log("Verified ESP-IDF provisioning protocol v1.1");
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
