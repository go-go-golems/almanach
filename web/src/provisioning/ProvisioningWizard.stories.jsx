import React from "react";
import ProvisioningWizard from "./ProvisioningWizard";
import { ProvisioningStep } from "./types";

const supported = { ok: true, code: "supported", message: "Web Bluetooth is available." };
const insecure = {
  ok: false,
  code: "insecure-context",
  message: "Web Bluetooth requires HTTPS or localhost.",
  hint: "Open this setup page from http://localhost or a future HTTPS setup URL.",
};
const logs = [
  { at: "2026-05-10T17:00:01Z", message: "Selected ALM_MOCK01" },
  { at: "2026-05-10T17:00:02Z", message: "Mock GATT connection established" },
  { at: "2026-05-10T17:00:03Z", message: "Mock proof-of-possession accepted" },
];

const meta = {
  title: "Provisioning/Setup Page",
  component: ProvisioningWizard,
  parameters: {
    viewport: { defaultViewport: "desktop" },
  },
  args: {
    storyMode: true,
    supportOverride: supported,
  },
};

export default meta;

export const Ready = {};

export const UnsupportedInsecureOrigin = {
  args: {
    supportOverride: insecure,
  },
};

export const WifiDetailsEntered = {
  args: {
    initialState: {
      step: ProvisioningStep.WIFI,
      support: supported,
      device: { id: "mock", name: "ALM_MOCK01" },
      pop: "alm-mock01",
      ssid: "Workshop WiFi",
      password: "correct horse battery staple",
      logs: logs.slice(0, 2),
    },
  },
};

export const ProvisioningProgress = {
  args: {
    initialState: {
      step: ProvisioningStep.PROVISIONING,
      support: supported,
      device: { id: "mock", name: "ALM_MOCK01" },
      pop: "alm-mock01",
      ssid: "Workshop WiFi",
      password: "correct horse battery staple",
      logs,
    },
  },
};

export const Success = {
  args: {
    initialState: {
      step: ProvisioningStep.DONE,
      support: supported,
      device: { id: "mock", name: "ALM_MOCK01" },
      pop: "alm-mock01",
      ssid: "Workshop WiFi",
      password: "correct horse battery staple",
      logs: [...logs, { at: "2026-05-10T17:00:04Z", message: "Mock printer joined WiFi" }],
      result: {
        ok: true,
        deviceName: "ALM_MOCK01",
        message: "Mock printer joined WiFi. Open the printer page once you know its IP address.",
      },
    },
  },
};

export const ErrorState = {
  args: {
    initialState: {
      step: ProvisioningStep.ERROR,
      support: supported,
      device: { id: "mock", name: "ALM_MOCK01" },
      pop: "wrong-pop",
      ssid: "Workshop WiFi",
      password: "fail",
      logs: [...logs, { at: "2026-05-10T17:00:04Z", message: "ERROR: Proof-of-possession did not match" }],
      error: "Proof-of-possession did not match this printer.",
    },
  },
};
