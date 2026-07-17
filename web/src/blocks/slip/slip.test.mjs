// Unit test for the work-slip pack's React-free logic (ALMANACH-WORKSLIP
// Phase 2). Runner-free —
//   node web/src/blocks/slip/slip.test.mjs
// (wired as `pnpm --dir web test:slip`).

import assert from "node:assert/strict";
import {
  DEFAULT_TOKENS,
  spaceToken,
  ruleToken,
  ruleStyleToken,
  bannerStyleToken,
  colWidthStyle,
} from "./tokens.js";
import { buildQrMatrix } from "./qr.js";

// ---- spacing tokens ----
// Numbers pass through; names resolve from theme tokens, then defaults.
assert.equal(spaceToken({}, 17), 17);
assert.equal(spaceToken({}, "s"), DEFAULT_TOKENS.space.s);
assert.equal(spaceToken({}, undefined), DEFAULT_TOKENS.space.m);
assert.equal(spaceToken({ tokens: { space: { s: 99 } } }, "s"), 99);
// A theme that overrides one step doesn't lose the others.
assert.equal(spaceToken({ tokens: { space: { s: 99 } } }, "xl"), DEFAULT_TOKENS.space.xl);
// Unknown name falls back to the fallback step.
assert.equal(spaceToken({}, "nope"), DEFAULT_TOKENS.space.m);

// ---- rule tokens ----
assert.equal(ruleToken({}, 3), 3);
assert.equal(ruleToken({}, "heavy"), DEFAULT_TOKENS.rules.heavy);
assert.equal(ruleToken({}, undefined), DEFAULT_TOKENS.rules.hair);
assert.equal(ruleToken({ tokens: { rules: { heavy: 12 } } }, "heavy"), 12);

// ---- style tokens ----
assert.equal(ruleStyleToken({}), "solid");
assert.equal(ruleStyleToken({ tokens: { ruleStyle: "dashed" } }), "dashed");
assert.equal(bannerStyleToken({}), "invert");
assert.equal(bannerStyleToken({ tokens: { bannerStyle: "outline" } }), "outline");

// ---- column widths ----
assert.deepEqual(colWidthStyle(90), { flex: "0 0 90px", width: 90, minWidth: 0 });
assert.deepEqual(colWidthStyle("1fr"), { flex: "1 1 0%", minWidth: 0 });
assert.deepEqual(colWidthStyle("2.5fr"), { flex: "2.5 1 0%", minWidth: 0 });
assert.deepEqual(colWidthStyle(undefined), { flex: "1 1 0%", minWidth: 0 });
assert.deepEqual(colWidthStyle("garbage"), { flex: "1 1 0%", minWidth: 0 });

// ---- QR matrix ----
const m = buildQrMatrix("https://www.upwork.com/jobs/~022075297946215353790/", 120);
assert.ok(m, "matrix for a real URL");
assert.ok(m.count >= 21, "at least version-1 module count");
// Integer module size, no fractional px (they die at the 1-bit threshold).
assert.equal(m.module, Math.floor(m.module));
assert.ok(m.module >= 2);
assert.equal(m.size, m.count * m.module);
// Finder pattern geometry: dark outer ring (row 0, col 0), light ring inside
// it (1,1), dark 3x3 core (2,2).
assert.equal(m.dark(0, 0), true);
assert.equal(m.dark(1, 0), true);
assert.equal(m.dark(1, 1), false);
assert.equal(m.dark(2, 2), true);

// Empty / whitespace values produce no matrix (component draws a placeholder).
assert.equal(buildQrMatrix(""), null);
assert.equal(buildQrMatrix("   "), null);
assert.equal(buildQrMatrix(undefined), null);

console.log("slip pack tests passed");
