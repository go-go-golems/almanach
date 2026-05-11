const WIRE_VARINT = 0;
const WIRE_LEN = 2;

export const SecScheme = Object.freeze({ SEC1: 1 });
export const Sec1Msg = Object.freeze({ COMMAND0: 0, RESPONSE0: 1, COMMAND1: 2, RESPONSE1: 3 });
export const Status = Object.freeze({ SUCCESS: 0 });
export const WifiMsg = Object.freeze({
  CMD_GET_STATUS: 0,
  RESP_GET_STATUS: 1,
  CMD_SET_CONFIG: 2,
  RESP_SET_CONFIG: 3,
  CMD_APPLY_CONFIG: 4,
  RESP_APPLY_CONFIG: 5,
});
export const WifiState = Object.freeze({ CONNECTED: 0, CONNECTING: 1, DISCONNECTED: 2, CONNECTION_FAILED: 3 });
export const WifiFailReason = Object.freeze({ AUTH_ERROR: 0, NETWORK_NOT_FOUND: 1 });
export const WifiCtrlMsg = Object.freeze({
  CMD_RESET: 1,
  RESP_RESET: 2,
  CMD_REPROV: 3,
  RESP_REPROV: 4,
});

export function wifiStateText(state) {
  switch (state) {
    case WifiState.CONNECTED: return "connected";
    case WifiState.CONNECTING: return "connecting";
    case WifiState.DISCONNECTED: return "disconnected";
    case WifiState.CONNECTION_FAILED: return "failed";
    default: return "unknown";
  }
}

export function wifiFailReasonText(reason) {
  switch (reason) {
    case WifiFailReason.AUTH_ERROR: return "AuthError";
    case WifiFailReason.NETWORK_NOT_FOUND: return "NetworkNotFound";
    default: return "";
  }
}

function varint(value) {
  let n = Number(value >>> 0);
  const out = [];
  while (n > 0x7f) {
    out.push((n & 0x7f) | 0x80);
    n >>>= 7;
  }
  out.push(n);
  return new Uint8Array(out);
}

function key(fieldNumber, wireType) {
  return varint((fieldNumber << 3) | wireType);
}

function concat(parts) {
  const length = parts.reduce((n, part) => n + part.length, 0);
  const out = new Uint8Array(length);
  let offset = 0;
  for (const part of parts) {
    out.set(part, offset);
    offset += part.length;
  }
  return out;
}

function fieldVarint(fieldNumber, value) {
  return concat([key(fieldNumber, WIRE_VARINT), varint(value)]);
}

function fieldBytes(fieldNumber, bytes) {
  return concat([key(fieldNumber, WIRE_LEN), varint(bytes.length), bytes]);
}

function emptyMessage() {
  return new Uint8Array();
}

class Reader {
  constructor(bytes) {
    this.bytes = bytes instanceof Uint8Array ? bytes : new Uint8Array(bytes);
    this.offset = 0;
  }
  eof() { return this.offset >= this.bytes.length; }
  varint() {
    let shift = 0;
    let value = 0;
    while (this.offset < this.bytes.length) {
      const b = this.bytes[this.offset++];
      value |= (b & 0x7f) << shift;
      if ((b & 0x80) === 0) return value >>> 0;
      shift += 7;
    }
    throw new Error("truncated protobuf varint");
  }
  bytesField() {
    const length = this.varint();
    const end = this.offset + length;
    if (end > this.bytes.length) throw new Error("truncated protobuf bytes field");
    const out = this.bytes.slice(this.offset, end);
    this.offset = end;
    return out;
  }
  skip(wireType) {
    if (wireType === WIRE_VARINT) { this.varint(); return; }
    if (wireType === WIRE_LEN) { this.bytesField(); return; }
    throw new Error(`unsupported protobuf wire type ${wireType}`);
  }
}

function eachField(bytes, visitor) {
  const r = new Reader(bytes);
  while (!r.eof()) {
    const tag = r.varint();
    const fieldNumber = tag >>> 3;
    const wireType = tag & 0x7;
    visitor(fieldNumber, wireType, r);
  }
}

export function encodeSessionCmd0(clientPubkey) {
  const sc0 = fieldBytes(1, clientPubkey);
  const sec1 = concat([
    fieldVarint(1, Sec1Msg.COMMAND0),
    fieldBytes(20, sc0),
  ]);
  return concat([
    fieldVarint(2, SecScheme.SEC1),
    fieldBytes(11, sec1),
  ]);
}

export function encodeSessionCmd1(clientVerifyData) {
  const sc1 = fieldBytes(2, clientVerifyData);
  const sec1 = concat([
    fieldVarint(1, Sec1Msg.COMMAND1),
    fieldBytes(22, sc1),
  ]);
  return concat([
    fieldVarint(2, SecScheme.SEC1),
    fieldBytes(11, sec1),
  ]);
}

export function decodeSessionResp0(bytes) {
  const session = decodeSessionData(bytes);
  if (session.secVer !== SecScheme.SEC1 || session.sec1?.msg !== Sec1Msg.RESPONSE0 || !session.sec1.sr0) {
    throw new Error(`Unexpected Security 1 response0 message ${session.sec1?.msg}`);
  }
  return session.sec1.sr0;
}

export function decodeSessionResp1(bytes) {
  const session = decodeSessionData(bytes);
  if (session.secVer !== SecScheme.SEC1 || session.sec1?.msg !== Sec1Msg.RESPONSE1 || !session.sec1.sr1) {
    throw new Error(`Unexpected Security 1 response1 message ${session.sec1?.msg}`);
  }
  return session.sec1.sr1;
}

function decodeSessionData(bytes) {
  const out = { secVer: 0, sec1: null };
  eachField(bytes, (field, wire, r) => {
    if (field === 2 && wire === WIRE_VARINT) out.secVer = r.varint();
    else if (field === 11 && wire === WIRE_LEN) out.sec1 = decodeSec1Payload(r.bytesField());
    else r.skip(wire);
  });
  return out;
}

function decodeSec1Payload(bytes) {
  const out = { msg: 0 };
  eachField(bytes, (field, wire, r) => {
    if (field === 1 && wire === WIRE_VARINT) out.msg = r.varint();
    else if (field === 21 && wire === WIRE_LEN) out.sr0 = decodeSessionResp0Payload(r.bytesField());
    else if (field === 23 && wire === WIRE_LEN) out.sr1 = decodeSessionResp1Payload(r.bytesField());
    else r.skip(wire);
  });
  return out;
}

function decodeSessionResp0Payload(bytes) {
  const out = { status: 0, devicePubkey: new Uint8Array(), deviceRandom: new Uint8Array() };
  eachField(bytes, (field, wire, r) => {
    if (field === 1 && wire === WIRE_VARINT) out.status = r.varint();
    else if (field === 2 && wire === WIRE_LEN) out.devicePubkey = r.bytesField();
    else if (field === 3 && wire === WIRE_LEN) out.deviceRandom = r.bytesField();
    else r.skip(wire);
  });
  return out;
}

function decodeSessionResp1Payload(bytes) {
  const out = { status: 0, deviceVerifyData: new Uint8Array() };
  eachField(bytes, (field, wire, r) => {
    if (field === 1 && wire === WIRE_VARINT) out.status = r.varint();
    else if (field === 3 && wire === WIRE_LEN) out.deviceVerifyData = r.bytesField();
    else r.skip(wire);
  });
  return out;
}

export function encodeSetConfig(ssid, password) {
  const enc = new TextEncoder();
  const cmd = concat([
    fieldBytes(1, enc.encode(ssid)),
    fieldBytes(2, enc.encode(password)),
  ]);
  return encodeWifiPayload(WifiMsg.CMD_SET_CONFIG, 12, cmd);
}

export function encodeApplyConfig() {
  return encodeWifiPayload(WifiMsg.CMD_APPLY_CONFIG, 14, emptyMessage());
}

export function encodeGetStatus() {
  return encodeWifiPayload(WifiMsg.CMD_GET_STATUS, 10, emptyMessage());
}

function encodeWifiPayload(msg, payloadField, payloadBytes) {
  return concat([
    fieldVarint(1, msg),
    fieldBytes(payloadField, payloadBytes),
  ]);
}

export function encodeCtrlReset() {
  return fieldVarint(1, WifiCtrlMsg.CMD_RESET);
}

export function encodeCtrlReprov() {
  return fieldVarint(1, WifiCtrlMsg.CMD_REPROV);
}

export function decodeWifiCtrlPayload(bytes) {
  const out = { msg: 0, status: 0 };
  eachField(bytes, (field, wire, r) => {
    if (field === 1 && wire === WIRE_VARINT) out.msg = r.varint();
    else if (field === 2 && wire === WIRE_VARINT) out.status = r.varint();
    else r.skip(wire);
  });
  return out;
}

export function decodeWifiConfigPayload(bytes) {
  const out = { msg: 0 };
  eachField(bytes, (field, wire, r) => {
    if (field === 1 && wire === WIRE_VARINT) out.msg = r.varint();
    else if (field === 11 && wire === WIRE_LEN) out.respGetStatus = decodeRespGetStatus(r.bytesField());
    else if (field === 13 && wire === WIRE_LEN) out.respSetConfig = decodeStatusPayload(r.bytesField());
    else if (field === 15 && wire === WIRE_LEN) out.respApplyConfig = decodeStatusPayload(r.bytesField());
    else r.skip(wire);
  });
  return out;
}

function decodeStatusPayload(bytes) {
  const out = { status: 0 };
  eachField(bytes, (field, wire, r) => {
    if (field === 1 && wire === WIRE_VARINT) out.status = r.varint();
    else r.skip(wire);
  });
  return out;
}

function decodeRespGetStatus(bytes) {
  const out = { status: 0, staState: WifiState.CONNECTED, hasFailReason: false, attemptsRemaining: 0, connected: null };
  eachField(bytes, (field, wire, r) => {
    if (field === 1 && wire === WIRE_VARINT) out.status = r.varint();
    else if (field === 2 && wire === WIRE_VARINT) out.staState = r.varint();
    else if (field === 10 && wire === WIRE_VARINT) { out.failReason = r.varint(); out.hasFailReason = true; }
    else if (field === 11 && wire === WIRE_LEN) out.connected = decodeConnectedState(r.bytesField());
    else if (field === 12 && wire === WIRE_LEN) out.attemptsRemaining = decodeAttemptFailed(r.bytesField()).attemptsRemaining;
    else r.skip(wire);
  });
  return out;
}

function decodeConnectedState(bytes) {
  const dec = new TextDecoder("utf-8");
  const out = { ip4Addr: "", authMode: 0, ssid: "", bssid: new Uint8Array(), channel: 0 };
  eachField(bytes, (field, wire, r) => {
    if (field === 1 && wire === WIRE_LEN) out.ip4Addr = dec.decode(r.bytesField());
    else if (field === 2 && wire === WIRE_VARINT) out.authMode = r.varint();
    else if (field === 3 && wire === WIRE_LEN) out.ssid = dec.decode(r.bytesField());
    else if (field === 4 && wire === WIRE_LEN) out.bssid = r.bytesField();
    else if (field === 5 && wire === WIRE_VARINT) out.channel = r.varint();
    else r.skip(wire);
  });
  return out;
}

function decodeAttemptFailed(bytes) {
  const out = { attemptsRemaining: 0 };
  eachField(bytes, (field, wire, r) => {
    if (field === 1 && wire === WIRE_VARINT) out.attemptsRemaining = r.varint();
    else r.skip(wire);
  });
  return out;
}
