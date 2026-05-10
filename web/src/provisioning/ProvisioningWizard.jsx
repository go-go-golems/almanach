import React, { useMemo, useState } from "react";
import { Bluetooth, CheckCircle2, AlertTriangle, Wifi, X, Loader2 } from "lucide-react";
import { getBluetoothSupport } from "./bluetooth-support";
import { createMockProvisioningClient } from "./mock-client";
import { createEspIdfProvisioningClient } from "./espidf-client";
import { appendLog, DEFAULT_SETUP_STATE, ProvisioningStep, validateWifiCredentials } from "./types";

const card = {
  background: "#221d18",
  border: "1px solid #3a3128",
  color: "#e8dcc4",
  borderRadius: 8,
  boxShadow: "0 18px 60px rgba(0,0,0,0.45)",
};

function StepBadge({ active, done, children }) {
  return (
    <div style={{
      padding: "6px 9px",
      border: `1px solid ${active ? "#c9a36b" : done ? "#6f8f60" : "#3a3128"}`,
      color: active ? "#d8b67e" : done ? "#a8d08d" : "#8a7c66",
      borderRadius: 999,
      fontSize: 11,
      letterSpacing: "0.08em",
      textTransform: "uppercase",
      whiteSpace: "nowrap",
    }}>{children}</div>
  );
}

function LogPanel({ logs }) {
  return (
    <div style={{ background: "#17130f", border: "1px solid #3a3128", borderRadius: 6, padding: 10, minHeight: 120, maxHeight: 180, overflow: "auto", fontFamily: "monospace", fontSize: 11 }}>
      {logs.length === 0 ? <div style={{ color: "#8a7c66" }}>Progress messages will appear here.</div> : logs.map((entry, i) => (
        <div key={`${entry.at}-${i}`} style={{ marginBottom: 4 }}><span style={{ color: "#8a7c66" }}>{entry.at.slice(11, 19)}</span> {entry.message}</div>
      ))}
    </div>
  );
}

export default function ProvisioningWizard({
  initialState = {},
  supportOverride = null,
  clientFactory = createMockProvisioningClient,
  realClientFactory = createEspIdfProvisioningClient,
  storyMode = false,
}) {
  const [state, setState] = useState(() => ({
    ...DEFAULT_SETUP_STATE,
    ...initialState,
    support: supportOverride || initialState.support || getBluetoothSupport(),
  }));
  const [busy, setBusy] = useState(false);
  const makeLog = () => (message) => setState((s) => ({ ...s, logs: appendLog(s.logs, message) }));
  const mockClient = useMemo(() => clientFactory({ log: makeLog() }), [clientFactory]);
  const realClient = useMemo(() => realClientFactory({ log: makeLog() }), [realClientFactory]);

  const support = state.support || getBluetoothSupport();
  const canUseRealBluetooth = support.ok;
  const credentialError = validateWifiCredentials({ ssid: state.ssid, password: state.password });

  const setField = (key, value) => setState((s) => ({ ...s, [key]: value, error: null }));
  const fail = (error) => setState((s) => ({ ...s, step: ProvisioningStep.ERROR, error: error.message || String(error), logs: appendLog(s.logs, `ERROR: ${error.message || error}`) }));

  async function chooseDevice(mode) {
    const selectedClient = mode === "real" ? realClient : mockClient;
    setBusy(true);
    try {
      setState((s) => ({
        ...s,
        clientMode: mode,
        step: ProvisioningStep.DEVICE,
        logs: appendLog(s.logs, mode === "real" ? "Using real Web Bluetooth ESP-IDF client" : "Using mock provisioning client for UI validation"),
      }));
      const device = await selectedClient.chooseDevice();
      await selectedClient.connect(device);
      setState((s) => ({ ...s, clientMode: mode, device: { ...device, mode }, step: ProvisioningStep.WIFI }));
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  async function runProvisioning() {
    const validation = validateWifiCredentials({ ssid: state.ssid, password: state.password });
    if (validation) {
      fail(new Error(validation));
      return;
    }
    setBusy(true);
    try {
      setState((s) => ({ ...s, step: ProvisioningStep.PROVISIONING, logs: appendLog(s.logs, "Starting mock provisioning flow") }));
      const selectedClient = state.clientMode === "real" ? realClient : mockClient;
      await selectedClient.establishSession({ pop: state.pop });
      await selectedClient.sendCredentials({ ssid: state.ssid, password: state.password });
      const result = await selectedClient.waitForResult();
      setState((s) => ({ ...s, result, step: ProvisioningStep.DONE, logs: appendLog(s.logs, result.message) }));
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div data-story-mode={storyMode ? "true" : "false"} style={{ minHeight: storyMode ? 720 : "100vh", background: "linear-gradient(135deg, #17130f, #272018)", color: "#e8dcc4", fontFamily: "DM Sans, system-ui, sans-serif", padding: 28 }}>
      <div style={{ maxWidth: 980, margin: "0 auto" }}>
        <header style={{ display: "flex", alignItems: "center", gap: 14, marginBottom: 22 }}>
          <div style={{ width: 44, height: 44, borderRadius: 12, background: "#c9a36b", color: "#17130f", display: "grid", placeItems: "center" }}><Bluetooth size={24} /></div>
          <div>
            <div style={{ fontFamily: "serif", fontSize: 31, letterSpacing: "0.16em", fontWeight: 700 }}>ALMANACH SETUP</div>
            <div style={{ color: "#c9b896" }}>Provision an AtomS3R thermal printer over BLE from localhost.</div>
          </div>
        </header>

        <main style={{ ...card, padding: 22 }}>
          <div style={{ display: "flex", gap: 8, flexWrap: "wrap", marginBottom: 22 }}>
            <StepBadge active={state.step === ProvisioningStep.SUPPORT} done={state.step !== ProvisioningStep.SUPPORT}>Browser</StepBadge>
            <StepBadge active={state.step === ProvisioningStep.DEVICE} done={[ProvisioningStep.WIFI, ProvisioningStep.PROVISIONING, ProvisioningStep.DONE].includes(state.step)}>Device</StepBadge>
            <StepBadge active={state.step === ProvisioningStep.WIFI} done={[ProvisioningStep.PROVISIONING, ProvisioningStep.DONE].includes(state.step)}>WiFi</StepBadge>
            <StepBadge active={state.step === ProvisioningStep.PROVISIONING}>Provision</StepBadge>
            <StepBadge active={state.step === ProvisioningStep.DONE} done={state.step === ProvisioningStep.DONE}>Done</StepBadge>
          </div>

          <section style={{ display: "grid", gridTemplateColumns: "minmax(0, 1fr) 340px", gap: 22 }}>
            <div>
              <div style={{ padding: 16, border: "1px solid #3a3128", borderRadius: 8, background: "rgba(0,0,0,0.16)", marginBottom: 16 }}>
                <h2 style={{ marginTop: 0, display: "flex", alignItems: "center", gap: 8 }}><Wifi size={20} /> Browser readiness</h2>
                {canUseRealBluetooth ? (
                  <p style={{ color: "#a8d08d" }}><CheckCircle2 size={16} /> Web Bluetooth is available on this origin.</p>
                ) : (
                  <div style={{ color: "#d8b67e" }}>
                    <p><AlertTriangle size={16} /> {support.message}</p>
                    {support.hint && <p style={{ color: "#c9b896" }}>{support.hint}</p>}
                  </div>
                )}
                <p style={{ color: "#c9b896", lineHeight: 1.5 }}>Chrome can now use the real BLE picker to connect to an Almanach printer and verify the ESP-IDF provisioning service. WiFi credential transfer still waits on the next Security 1/protobuf implementation step.</p>
              </div>

              <div style={{ padding: 16, border: "1px solid #3a3128", borderRadius: 8, background: "rgba(0,0,0,0.16)", marginBottom: 16 }}>
                <h2 style={{ marginTop: 0 }}>Printer and WiFi details</h2>
                <label style={{ display: "block", marginBottom: 10 }}>Proof of possession
                  <input value={state.pop} onChange={(e) => setField("pop", e.target.value)} placeholder="alm-a1b2c3" style={inputStyle} />
                </label>
                <label style={{ display: "block", marginBottom: 10 }}>WiFi SSID
                  <input value={state.ssid} onChange={(e) => setField("ssid", e.target.value)} placeholder="Your 2.4 GHz network" style={inputStyle} />
                </label>
                <label style={{ display: "block", marginBottom: 10 }}>WiFi password
                  <input type="password" value={state.password} onChange={(e) => setField("password", e.target.value)} placeholder="Network password" style={inputStyle} />
                </label>
                {credentialError && <div style={{ color: "#c97766", fontSize: 13 }}>{credentialError}</div>}
              </div>

              {state.device && <div style={{ padding: 12, border: "1px solid #4a3f33", color: "#c9b896", borderRadius: 6, marginBottom: 16 }}><Bluetooth size={14} /> Connected target: <strong>{state.device.name}</strong> ({state.clientMode === "real" ? "real BLE" : "mock"})</div>}
              {state.error && <div style={{ padding: 12, border: "1px solid #c97766", color: "#c97766", borderRadius: 6, marginBottom: 16 }}><X size={14} /> {state.error}</div>}
              {state.result && <div style={{ padding: 12, border: "1px solid #6f8f60", color: "#a8d08d", borderRadius: 6, marginBottom: 16 }}><CheckCircle2 size={14} /> {state.result.message}</div>}

              <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
                <button style={{ ...buttonStyle, background: "#c9a36b", color: "#17130f" }} onClick={() => chooseDevice("real")} disabled={busy || !canUseRealBluetooth}>{busy ? <Loader2 size={14} /> : <Bluetooth size={14} />} Find BLE printer</button>
                <button style={buttonStyle} onClick={() => chooseDevice("mock")} disabled={busy}>{busy ? <Loader2 size={14} /> : <Bluetooth size={14} />} Use mock printer</button>
                <button style={buttonStyle} onClick={runProvisioning} disabled={busy || !state.device}>{busy ? <Loader2 size={14} /> : <Wifi size={14} />} {state.clientMode === "real" ? "Continue provisioning" : "Run mock provisioning"}</button>
                <button style={buttonStyle} onClick={() => setState({ ...DEFAULT_SETUP_STATE, support: supportOverride || getBluetoothSupport() })} disabled={busy}>Reset</button>
              </div>
            </div>

            <aside>
              <h3 style={{ marginTop: 0 }}>Progress</h3>
              <LogPanel logs={state.logs} />
              <div style={{ marginTop: 16, color: "#8a7c66", fontSize: 12, lineHeight: 1.45 }}>
                <strong>Next implementation step:</strong> add ESP-IDF proto-ver, Security 1/protobuf, credential transfer, and status polling on top of the real BLE connection.
              </div>
            </aside>
          </section>
        </main>
      </div>
    </div>
  );
}

const inputStyle = {
  display: "block",
  width: "100%",
  marginTop: 6,
  padding: "10px 11px",
  borderRadius: 5,
  border: "1px solid #4a3f33",
  background: "#17130f",
  color: "#e8dcc4",
};

const buttonStyle = {
  display: "inline-flex",
  alignItems: "center",
  gap: 7,
  padding: "10px 13px",
  borderRadius: 5,
  border: "1px solid #4a3f33",
  background: "#2a241e",
  color: "#e8dcc4",
  cursor: "pointer",
};
