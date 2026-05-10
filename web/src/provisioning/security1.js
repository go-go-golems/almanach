import { x25519 } from "@noble/curves/ed25519.js";
import {
  decodeSessionResp0,
  decodeSessionResp1,
  encodeSessionCmd0,
  encodeSessionCmd1,
  Status,
} from "./espidf-protobuf";

function xorInPlace(target, mask) {
  for (let i = 0; i < target.length && i < mask.length; i += 1) {
    target[i] ^= mask[i];
  }
}

function equalBytes(a, b) {
  if (!a || !b || a.length !== b.length) return false;
  let diff = 0;
  for (let i = 0; i < a.length; i += 1) diff |= a[i] ^ b[i];
  return diff === 0;
}

function incrementCounter(counter, blocks) {
  let carry = blocks >>> 0;
  for (let i = counter.length - 1; i >= 0 && carry > 0; i -= 1) {
    const sum = counter[i] + (carry & 0xff);
    counter[i] = sum & 0xff;
    carry = (carry >>> 8) + (sum >>> 8);
  }
}

class AESCTRStream {
  constructor(key, iv) {
    this.keyBytes = key;
    this.counter = new Uint8Array(iv);
    this.offset = 0;
    this.cryptoKey = null;
  }

  async importKey() {
    if (!this.cryptoKey) {
      this.cryptoKey = await crypto.subtle.importKey("raw", this.keyBytes, { name: "AES-CTR" }, false, ["encrypt"]);
    }
    return this.cryptoKey;
  }

  async crypt(data) {
    const input = data instanceof Uint8Array ? data : new Uint8Array(data);
    if (input.length === 0) return new Uint8Array();

    const total = this.offset + input.length;
    const zeros = new Uint8Array(total);
    const key = await this.importKey();
    const keystream = new Uint8Array(await crypto.subtle.encrypt({ name: "AES-CTR", counter: this.counter, length: 128 }, key, zeros));
    const out = new Uint8Array(input.length);
    for (let i = 0; i < input.length; i += 1) {
      out[i] = input[i] ^ keystream[this.offset + i];
    }

    const blocks = Math.floor(total / 16);
    this.offset = total % 16;
    incrementCounter(this.counter, blocks);
    return out;
  }
}

export class Security1Session {
  constructor({ pop = "", log = () => {} } = {}) {
    this.pop = pop;
    this.log = log;
    this.privateKey = null;
    this.clientPubkey = null;
    this.devicePubkey = null;
    this.stream = null;
    this.established = false;
  }

  async establish(sendEndpointBytes) {
    const keypair = x25519.keygen();
    this.privateKey = keypair.secretKey;
    this.clientPubkey = keypair.publicKey;
    this.log("Security 1 setup0: sending client X25519 public key");

    const resp0Bytes = await sendEndpointBytes("prov-session", encodeSessionCmd0(this.clientPubkey));
    const resp0 = decodeSessionResp0(resp0Bytes);
    if (resp0.status !== Status.SUCCESS) throw new Error(`Security 1 setup0 failed with status ${resp0.status}`);
    if (resp0.devicePubkey.length !== 32) throw new Error(`Invalid device public key length ${resp0.devicePubkey.length}`);
    if (resp0.deviceRandom.length !== 16) throw new Error(`Invalid device random length ${resp0.deviceRandom.length}`);

    this.devicePubkey = resp0.devicePubkey;
    const shared = x25519.getSharedSecret(this.privateKey, this.devicePubkey);
    const key = new Uint8Array(shared);
    if (this.pop) {
      const popDigest = new Uint8Array(await crypto.subtle.digest("SHA-256", new TextEncoder().encode(this.pop)));
      xorInPlace(key, popDigest);
    }
    this.stream = new AESCTRStream(key, resp0.deviceRandom);

    const clientVerify = await this.stream.crypt(this.devicePubkey);
    this.log("Security 1 setup1: sending encrypted device public-key proof");
    const resp1Bytes = await sendEndpointBytes("prov-session", encodeSessionCmd1(clientVerify));
    const resp1 = decodeSessionResp1(resp1Bytes);
    if (resp1.status !== Status.SUCCESS) throw new Error(`Security 1 setup1 failed with status ${resp1.status}`);

    const deviceVerify = await this.stream.crypt(resp1.deviceVerifyData);
    if (!equalBytes(deviceVerify, this.clientPubkey)) {
      throw new Error("Failed to verify device during Security 1 setup");
    }
    this.established = true;
    this.log("Security 1 session established");
    return this;
  }

  async encrypt(data) {
    if (!this.established || !this.stream) throw new Error("Security 1 session is not established");
    return this.stream.crypt(data);
  }

  async decrypt(data) {
    if (!this.established || !this.stream) throw new Error("Security 1 session is not established");
    return this.stream.crypt(data);
  }
}
