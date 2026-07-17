// Theme design tokens for the work-slip block pack (ALMANACH-WORKSLIP).
//
// Slip blocks consume vertical spacing, rule thickness, and banner style from
// the theme, never hardcoded px, so a theme can restyle the whole pack (swiss
// hairlines vs. brutalist slabs) as data. Every accessor falls back to these
// defaults, so the pack renders correctly on themes that predate tokens
// (classic, minimal, crisp, ...).
//
// This module is React-free and side-effect-free so it can be unit tested in
// plain Node.

export const DEFAULT_TOKENS = {
  // Named vertical spacing steps in px (block `space`, row gaps).
  space: { xs: 4, s: 8, m: 14, l: 22, xl: 34 },
  // Named rule thicknesses in px (block `rule`, write-in lines).
  rules: { hair: 1, thick: 2, heavy: 4 },
  // How rules draw: "solid" | "dashed".
  ruleStyle: "solid",
  // How banner blocks draw: "invert" (filled bar) | "outline" (border box).
  bannerStyle: "invert",
};

/**
 * Resolve a spacing value: a number passes through as px; a named step is
 * looked up in the theme's tokens, then the defaults.
 */
export function spaceToken(theme, value, fallback = "m") {
  if (typeof value === "number" && isFinite(value)) return value;
  const name = typeof value === "string" ? value : fallback;
  return theme?.tokens?.space?.[name]
    ?? DEFAULT_TOKENS.space[name]
    ?? DEFAULT_TOKENS.space[fallback];
}

/**
 * Resolve a rule thickness: a number passes through as px; a named weight is
 * looked up in the theme's tokens, then the defaults.
 */
export function ruleToken(theme, value, fallback = "hair") {
  if (typeof value === "number" && isFinite(value)) return value;
  const name = typeof value === "string" ? value : fallback;
  return theme?.tokens?.rules?.[name]
    ?? DEFAULT_TOKENS.rules[name]
    ?? DEFAULT_TOKENS.rules[fallback];
}

export function ruleStyleToken(theme) {
  return theme?.tokens?.ruleStyle ?? DEFAULT_TOKENS.ruleStyle;
}

export function bannerStyleToken(theme) {
  return theme?.tokens?.bannerStyle ?? DEFAULT_TOKENS.bannerStyle;
}

/**
 * Column width -> flex CSS. A number is fixed px; "2fr"/"1fr" (any float)
 * shares the remaining space proportionally; missing/invalid defaults to 1fr.
 */
export function colWidthStyle(w) {
  if (typeof w === "number" && isFinite(w)) {
    return { flex: `0 0 ${w}px`, width: w, minWidth: 0 };
  }
  if (typeof w === "string") {
    const m = w.match(/^([0-9]*\.?[0-9]+)fr$/);
    if (m) return { flex: `${parseFloat(m[1])} 1 0%`, minWidth: 0 };
  }
  return { flex: "1 1 0%", minWidth: 0 };
}

// Accept both the compact tuple form and YAML-friendly { k, v } objects.
export function normalizeKVItems(items) {
  return (Array.isArray(items) ? items : []).map((item) =>
    Array.isArray(item) ? item : [item?.k ?? "", item?.v ?? ""]
  );
}
