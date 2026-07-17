// Typography presets for the Almanach studio (Layout DSL v2, Phase 3).
//
// Typography is modeled as named presets, not raw per-element sizes. A block's
// text element references a preset ("body", "caption", ...); the resolver merges
// three layers into a concrete CSS style object:
//
//   built-in default  <-  layout preset override  <-  block style
//
// The built-in defaults bake in the ALMANACH-PIXELFONT paper findings so a page
// prints legibly with no configuration:
//   - bigger small text with an absolute `minSize` floor (sub-pixel strokes get
//     thresholded away at 1-bit, so nothing should render tiny),
//   - heavier weight for body/small text (weight survives the threshold),
//   - bold italic for quotes/notes (normal italic loses strokes; bold italic
//     stays legible).
// Font family is deliberately left to the theme here (Phase 4 embeds the hinted
// DejaVu families and lets presets/themes pick them); a preset or override may
// still set `font` explicitly.
//
// The style objects use the same field names as the protobuf TextStyle message
// (font, size, weight, lineHeight, letterSpacing, textCase, italic, minSize) so
// a decoded layout's presets/overrides plug in directly. `size` is a BASE px
// value (before bodyScale); `role` picks the theme font when `font` is unset.

// text_case -> CSS text-transform. Accepts both the proto enum names and plain
// CSS values (so hand-written layouts can use either).
const CASE_TO_TRANSFORM = {
  TEXT_CASE_UPPER: "uppercase",
  TEXT_CASE_LOWER: "lowercase",
  TEXT_CASE_CAPITALIZE: "capitalize",
  TEXT_CASE_NONE: "none",
  upper: "uppercase",
  lower: "lowercase",
  capitalize: "capitalize",
  none: "none",
  uppercase: "uppercase",
  lowercase: "lowercase",
};

// role -> which theme font family to inherit when a preset sets no explicit font.
function fontForRole(role, theme) {
  if (role === "display") return theme.fontDisplay;
  if (role === "mono") return theme.fontMono || theme.fontBody;
  return theme.fontBody;
}

export const DEFAULT_PRESETS = {
  // Uppercase section headers ("WORD OF THE DAY").
  sectionLabel: { role: "display", size: 12, weight: 700, letterSpacing: 0.18, textCase: "upper", lineHeight: 1, minSize: 11 },
  // Uppercase sub-headers inside a block ("Currently Reading").
  overline: { role: "display", size: 11, weight: 600, letterSpacing: 0.18, textCase: "upper", lineHeight: 1.2, minSize: 11 },
  // The big display word (Word of the Day).
  word: { role: "display", size: 26, weight: 600, letterSpacing: 0.02, lineHeight: 1.05 },
  // A large numeric/display metric (weather temperature).
  metric: { role: "display", size: 22, weight: 600, lineHeight: 1 },
  // Default running text.
  body: { role: "body", size: 13, weight: 500, lineHeight: 1.4, minSize: 12 },
  // Emphasized inline text (names, years, book titles) — heavier so it survives.
  bodyStrong: { role: "body", size: 13, weight: 700, lineHeight: 1.4, minSize: 12 },
  // Quote / note prose: bold italic is the paper-verified legible italic.
  emphasis: { role: "body", size: 14, weight: 600, italic: true, lineHeight: 1.4, minSize: 12 },
  // Secondary text (authors, sources, conditions, captions).
  caption: { role: "body", size: 12, weight: 600, lineHeight: 1.35, minSize: 11 },
  // Smallest muted text — floored and bolded so it stays readable.
  small: { role: "body", size: 11, weight: 600, letterSpacing: 0.02, lineHeight: 1.3, minSize: 11 },
  // Tabular metadata (plan times).
  meta: { role: "body", size: 12, weight: 600, lineHeight: 1.2, minSize: 11 },
};

/**
 * Resolve a preset (plus layout/block/component overrides) into a CSS style
 * object. Later layers win; `overrides` are applied in array order after the
 * layout preset override, so a per-block style (passed last by callers) wins.
 *
 * @param {string} name preset name
 * @param {object} opts
 * @param {object} [opts.presets] layout-level preset overrides (name -> TextStyle)
 * @param {object} opts.theme the studio theme (for fontDisplay/fontBody/fontMono)
 * @param {number} [opts.bodyScale] global size multiplier
 * @param {Array<object|null|undefined>} [opts.overrides] extra style layers
 * @returns {object} CSS style props (fontFamily, fontSize, fontWeight, ...)
 */
export function resolveStyle(name, { presets = {}, theme = {}, bodyScale = 1, overrides = [] } = {}) {
  const layers = [DEFAULT_PRESETS[name] || {}, presets[name] || {}, ...overrides];
  const merged = Object.assign({}, ...layers.filter(Boolean));

  // Only inject a font when the resolved style actually names one — either an
  // explicit family or a role. Ad-hoc overrides (e.g. a per-block style with no
  // font/role) must not clobber a font the caller set outside the preset.
  const font = merged.font || (merged.role ? fontForRole(merged.role, theme) : undefined);

  const css = {};
  if (font) css.fontFamily = font;

  if (typeof merged.size === "number") {
    let size = merged.size * bodyScale;
    if (typeof merged.minSize === "number") size = Math.max(size, merged.minSize);
    css.fontSize = Math.round(size * 10) / 10;
  }
  if (merged.weight != null) css.fontWeight = merged.weight;
  if (merged.lineHeight != null) css.lineHeight = merged.lineHeight;
  if (merged.letterSpacing != null) {
    css.letterSpacing = typeof merged.letterSpacing === "number"
      ? `${merged.letterSpacing}em`
      : merged.letterSpacing;
  }
  const transform = CASE_TO_TRANSFORM[merged.textCase];
  if (transform) css.textTransform = transform;
  if (merged.italic != null) css.fontStyle = merged.italic ? "italic" : "normal";

  return css;
}

/**
 * Deep-merge several preset-override maps (name -> partial TextStyle) into one,
 * shallow-merging the TextStyle objects per preset name. Later maps win per
 * field, so a theme's `body.size` and a layout's `body.weight` both survive.
 * Used to combine theme `presetOverrides` (lower) with layout `typography`
 * (higher) before resolution.
 */
export function mergePresetMaps(...maps) {
  const out = {};
  for (const map of maps) {
    if (!map) continue;
    for (const [name, style] of Object.entries(map)) {
      out[name] = { ...(out[name] || {}), ...(style || {}) };
    }
  }
  return out;
}

/**
 * Bind a resolver to a theme + layout presets + bodyScale. Returns
 * `preset(name, ...overrides)` for components to spread into style objects.
 */
export function makePresetResolver({ presets = {}, theme, bodyScale = 1 }) {
  return (name, ...overrides) => resolveStyle(name, { presets, theme, bodyScale, overrides });
}
