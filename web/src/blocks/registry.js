// Block renderer registry for the Almanach studio (Layout DSL v2, Phase 2).
//
// A block's `type` is a dispatch key resolved against a registry of adapters,
// instead of a hardcoded switch or a bare object map. This mirrors the widget-IR
// renderer pattern (rag-evaluation-system) but stripped for a non-interactive
// print DSL: adapter objects, a Map registry built from arrays, a merge helper,
// and a graceful fallback for unknown types.
//
// This module is intentionally React-free and side-effect-free so it can be unit
// tested in plain Node. The React pieces (the <UnknownBlock> placeholder and the
// per-block dispatch) live with the studio components; each adapter's `render`
// returns a ReactNode.
//
// An adapter:
//   { type: string, module?: string, render: (data, ctx) => ReactNode }
// where ctx carries at least { theme, block, registry }.

/**
 * Validate and return an adapter (identity, for typing/consistency at call
 * sites — like rag-eval's defineWidget).
 */
export function defineBlock(adapter) {
  if (!adapter || typeof adapter !== "object") {
    throw new Error("defineBlock: adapter must be an object");
  }
  if (typeof adapter.type !== "string" || adapter.type === "") {
    throw new Error("defineBlock: adapter.type must be a non-empty string");
  }
  if (typeof adapter.render !== "function") {
    throw new Error(
      `defineBlock: adapter.render must be a function (type "${adapter.type}")`,
    );
  }
  return adapter;
}

/**
 * Build a registry (a Map from type -> adapter) from an array of adapters.
 * Throws on a duplicate type so a registration mistake fails loudly.
 */
export function createBlockRegistry(adapters) {
  const map = new Map();
  for (const adapter of adapters) {
    defineBlock(adapter);
    if (map.has(adapter.type)) {
      throw new Error(`createBlockRegistry: duplicate block type "${adapter.type}"`);
    }
    map.set(adapter.type, adapter);
  }
  return map;
}

/**
 * Flatten several registries into one, preserving the duplicate-type guard so
 * two sub-registries can't silently claim the same type.
 */
export function mergeBlockRegistries(...registries) {
  const all = [];
  for (const registry of registries) {
    for (const adapter of registry.values()) all.push(adapter);
  }
  return createBlockRegistry(all);
}

/**
 * Resolve a block's adapter. Returns the adapter, or null when the type is not
 * registered — callers render a visible placeholder rather than dropping the
 * block or crashing.
 */
export function resolveBlockAdapter(registry, type) {
  return registry.get(type) ?? null;
}
