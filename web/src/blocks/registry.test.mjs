// Unit test for the block registry (Layout DSL v2, Phase 2). Runner-free —
//   node web/src/blocks/registry.test.mjs
// (wired as `pnpm --dir web test:registry`).

import assert from "node:assert/strict";
import {
  defineBlock,
  createBlockRegistry,
  mergeBlockRegistries,
  resolveBlockAdapter,
} from "./registry.js";

const noop = () => null;

// defineBlock validates shape
assert.throws(() => defineBlock(null), /must be an object/);
assert.throws(() => defineBlock({ render: noop }), /type must be a non-empty string/);
assert.throws(() => defineBlock({ type: "x" }), /render must be a function/);
assert.equal(defineBlock({ type: "x", render: noop }).type, "x");

// createBlockRegistry builds a Map and guards duplicates
const reg = createBlockRegistry([
  { type: "title", render: noop },
  { type: "quote", render: noop },
]);
assert.equal(reg.size, 2, "registry size");
assert.equal(resolveBlockAdapter(reg, "title").type, "title", "lookup hit");
assert.equal(resolveBlockAdapter(reg, "nope"), null, "unknown -> null");
assert.throws(
  () => createBlockRegistry([{ type: "dup", render: noop }, { type: "dup", render: noop }]),
  /duplicate block type "dup"/,
  "duplicate guard",
);

// mergeBlockRegistries flattens and keeps the duplicate guard
const a = createBlockRegistry([{ type: "a", render: noop }]);
const b = createBlockRegistry([{ type: "b", render: noop }]);
const merged = mergeBlockRegistries(a, b);
assert.equal(merged.size, 2, "merged size");
assert.throws(() => mergeBlockRegistries(a, a), /duplicate block type "a"/, "merge duplicate guard");

// render passthrough: adapter.render receives (data, ctx)
let seen;
const rr = createBlockRegistry([
  { type: "echo", render: (data, ctx) => { seen = { data, ctx }; return "ok"; } },
]);
const out = resolveBlockAdapter(rr, "echo").render({ n: 1 }, { theme: "t" });
assert.equal(out, "ok");
assert.deepEqual(seen.data, { n: 1 });
assert.equal(seen.ctx.theme, "t");

console.log("✓ registry: all assertions passed");
