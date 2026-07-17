// QR module-matrix helper for the slip `qr` block (ALMANACH-WORKSLIP).
//
// Wraps the bundled `qrcode-generator` dependency (no CDN — headless renders
// must be self-contained). Returns plain data so the React component stays a
// dumb renderer and this logic is testable in Node.

import qrcode from "qrcode-generator";

/**
 * Build a QR matrix for a value.
 *
 * @param {string} value payload (URL, id, ...)
 * @param {number} targetSize desired square size in px
 * @returns {{ count: number, module: number, size: number, dark: (r,c) => boolean } | null}
 *   null when the value is empty or encoding fails. `module` is an integer px
 *   per module so modules land on pixel boundaries — fractional modules get
 *   eaten or smeared by the 1-bit threshold.
 */
export function buildQrMatrix(value, targetSize = 120) {
  const text = typeof value === "string" ? value.trim() : "";
  if (!text) return null;
  try {
    const qr = qrcode(0 /* auto version */, "M");
    qr.addData(text);
    qr.make();
    const count = qr.getModuleCount();
    const module = Math.max(2, Math.floor(targetSize / count));
    return {
      count,
      module,
      size: count * module,
      dark: (r, c) => qr.isDark(r, c),
    };
  } catch {
    return null;
  }
}
