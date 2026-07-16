// TS-side decode/round-trip test for the Almanach Layout DSL v2 wire contract.
//
// Lives OUTSIDE web/src/pb because that directory is a `buf generate` output
// with `clean: true` — buf wipes it on every regen, so a test kept there would
// be deleted. Reads the SAME golden fixture the Go test reads
// (proto/almanach/layout/v1/testdata/layout_golden.json) and asserts that
// @bufbuild/protobuf fromJson decodes it into the expected values and that
// toJson -> fromJson is stable. Together with the Go test this locks the
// cross-language contract.
//
// No test runner needed — run with `node`:
//   node web/test/layout.roundtrip.test.mjs
// (wired as `pnpm --dir web test:proto`).

import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { fromJson, toJson } from "@bufbuild/protobuf";
import {
  LayoutSchema,
  TextCase,
  RasterMode,
} from "../src/pb/almanach/layout/v1/layout_pb.js";

const goldenUrl = new URL(
  "../../proto/almanach/layout/v1/testdata/layout_golden.json",
  import.meta.url,
);
const golden = JSON.parse(readFileSync(goldenUrl, "utf8"));

// ── decode ───────────────────────────────────────────────────────────
const layout = fromJson(LayoutSchema, golden);

assert.equal(layout.schemaVersion, 1, "schemaVersion");
assert.equal(layout.paperWidth, 384, "paperWidth");
assert.equal(layout.theme, "minimal", "theme");
assert.equal(layout.blocks.length, 3, "blocks length");

// preset with enum-valued textCase + optional scalars
const title = layout.typography.presets["title"];
assert.ok(title, "title preset present");
// tsEnum strips the enum-name prefix: TEXT_CASE_UPPER -> UPPER
assert.equal(title.textCase, TextCase.UPPER, "title.textCase");
assert.equal(title.weight, 700, "title.weight");

// caption preset uses proto3 `optional bool italic`
const caption = layout.typography.presets["caption"];
assert.equal(caption.italic, true, "caption.italic");

// block with per-block render override + Struct content
const quote = layout.blocks[1];
assert.equal(quote.type, "quote", "quote.type");
assert.equal(quote.render.printerDensity, 38, "quote.render.printerDensity");
assert.equal(
  quote.render.rasterMode,
  RasterMode.THRESHOLD,
  "quote.render.rasterMode",
);
// google.protobuf.Struct decodes to a plain JS object
assert.equal(quote.content.author, "Dorothy Parker", "quote.content.author");

const img = layout.blocks[2];
assert.equal(
  img.render.rasterMode,
  RasterMode.ATKINSON,
  "img.render.rasterMode",
);
assert.equal(img.render.gamma, 0.8, "img.render.gamma");

// ── round-trip stability ─────────────────────────────────────────────
const encoded = toJson(LayoutSchema, layout);
const reDecoded = fromJson(LayoutSchema, encoded);
assert.deepEqual(
  toJson(LayoutSchema, reDecoded),
  encoded,
  "toJson -> fromJson -> toJson stable",
);

console.log("✓ layout.roundtrip: %d assertions passed", 13);
