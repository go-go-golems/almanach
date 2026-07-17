// Unit test for the typography preset resolver (Layout DSL v2, Phase 3).
//   node web/src/typography/presets.test.mjs
// (wired as `pnpm --dir web test:presets`).

import assert from "node:assert/strict";
import { resolveStyle, makePresetResolver, mergePresetMaps, DEFAULT_PRESETS } from "./presets.js";

const theme = {
  fontDisplay: "'Display', serif",
  fontBody: "'Body', serif",
  fontMono: "'Mono', monospace",
};

// role picks the theme font when the preset sets none
assert.equal(resolveStyle("body", { theme }).fontFamily, "'Body', serif", "body -> fontBody");
assert.equal(resolveStyle("word", { theme }).fontFamily, "'Display', serif", "display -> fontDisplay");

// size scales by bodyScale and rounds
assert.equal(resolveStyle("body", { theme, bodyScale: 2 }).fontSize, 26, "13 * 2 = 26");

// minSize is an absolute floor applied after scaling
assert.equal(resolveStyle("small", { theme, bodyScale: 0.5 }).fontSize, 11, "small floored at 11 (5.5 -> 11)");

// recipe: emphasis is bold italic
const emph = resolveStyle("emphasis", { theme });
assert.equal(emph.fontStyle, "italic", "emphasis italic");
assert.equal(emph.fontWeight, 600, "emphasis bold");

// textCase -> CSS text-transform, and letterSpacing number -> em
const label = resolveStyle("sectionLabel", { theme });
assert.equal(label.textTransform, "uppercase", "sectionLabel uppercase");
assert.equal(label.letterSpacing, "0.18em", "letterSpacing em");

// merge order: default <- layout preset override <- block/component override
const layout = { body: { weight: 400, size: 20 } };
const merged = resolveStyle("body", { theme, presets: layout, overrides: [{ weight: 900 }] });
assert.equal(merged.fontSize, 20, "layout override size wins over default");
assert.equal(merged.fontWeight, 900, "later override wins over layout");

// explicit font on an override beats the role default
assert.equal(
  resolveStyle("body", { theme, overrides: [{ font: "'X', serif" }] }).fontFamily,
  "'X', serif",
  "explicit font wins",
);

// bound resolver threads presets + bodyScale
const preset = makePresetResolver({ presets: layout, theme, bodyScale: 1 });
assert.equal(preset("body").fontSize, 20, "bound resolver applies layout override");
assert.equal(preset("body", { size: 30 }).fontSize, 30, "bound resolver applies call override");

// an ad-hoc override with no preset/role/font does NOT inject a font
assert.equal(
  resolveStyle("__none", { theme, overrides: [{ weight: 700 }] }).fontFamily,
  undefined,
  "no font injected without role/font",
);

// mergePresetMaps deep-merges per preset name (later wins per field)
const mergedMaps = mergePresetMaps(
  { body: { font: "'A'", size: 12 } },       // theme layer
  { body: { size: 15, weight: 700 } },        // layout layer
);
assert.deepEqual(mergedMaps.body, { font: "'A'", size: 15, weight: 700 }, "per-field merge");
assert.deepEqual(mergePresetMaps(null, undefined, { x: { size: 1 } }).x, { size: 1 }, "skips nullish maps");

// every default preset resolves without throwing and yields a font
for (const name of Object.keys(DEFAULT_PRESETS)) {
  const s = resolveStyle(name, { theme, bodyScale: 1.6 });
  assert.ok(s.fontFamily, `${name} has a fontFamily`);
}

console.log("✓ presets: all assertions passed");
