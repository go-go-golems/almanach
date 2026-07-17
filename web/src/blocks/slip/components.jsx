import React from "react";
import { spaceToken, ruleToken, ruleStyleToken, bannerStyleToken, colWidthStyle } from "./tokens.js";
import { buildQrMatrix } from "./qr.js";

/* ================================================================
   WORK-SLIP BLOCK PACK (ALMANACH-WORKSLIP)

   Generic layout primitives ported from the slip-studio prototype:
   text, banner, rule, space, row, kv, list, checks, writein, qr,
   bars, table. All text goes through the typography preset resolver
   (theme.preset), all spacing/rule/banner styling through the theme
   token accessors, so themes restyle the pack as data.

   Every component receives ({ data, theme, blockStyle }) like the
   studio blocks; `row` additionally receives ctx to render nested
   blocks through the registry (ctx.renderBlock).
================================================================ */

// Multi-line clamp with ellipsis. Chrome-only CSS is fine: Chrome is the
// production renderer (headless screenshot) and the only supported preview.
const clampStyle = (lines) =>
  lines > 0
    ? {
        display: "-webkit-box",
        WebkitBoxOrient: "vertical",
        WebkitLineClamp: lines,
        overflow: "hidden",
      }
    : {};

// One text run. data: { text, preset?, align?, lines? }; the block's `style`
// (TextStyle) is the last override layer, so weight/case/size tweaks ride the
// standard DSL v2 mechanism rather than pack-private fields.
export const SlipTextBlock = ({ data, theme, blockStyle }) => (
  <div
    style={{
      ...theme.preset(data.preset || "body", blockStyle),
      color: theme.ink,
      textAlign: data.align || "left",
      overflowWrap: "break-word",
      ...clampStyle(Number(data.lines) || 0),
    }}
  >
    {data.text}
  </div>
);

// Full-width emphasis bar. data: { text, right?, preset?, pad? }. The theme's
// bannerStyle token picks filled ("invert") or bordered ("outline").
export const BannerBlock = ({ data, theme, blockStyle }) => {
  const padMap = { s: 5, m: 8, l: 13 };
  const pad = padMap[data.pad] ?? padMap.s;
  const invert = bannerStyleToken(theme) !== "outline";
  const st = theme.preset(data.preset || "caption", { weight: 700, letterSpacing: 0.12, textCase: "upper" }, blockStyle);
  return (
    <div
      style={{
        ...st,
        display: "flex",
        alignItems: "baseline",
        justifyContent: "space-between",
        gap: 10,
        padding: `${pad}px 8px`,
        background: invert ? theme.ink : "transparent",
        color: invert ? theme.paper : theme.ink,
        border: invert ? "none" : `2px solid ${theme.ink}`,
      }}
    >
      <span>{data.text}</span>
      {data.right != null && data.right !== "" && <span>{data.right}</span>}
    </div>
  );
};

// Horizontal rule. data: { weight?: "hair"|"thick"|"heavy"|number }.
export const RuleBlock = ({ data, theme }) => {
  const h = ruleToken(theme, data.weight);
  const dashed = ruleStyleToken(theme) === "dashed";
  return (
    <div
      style={{
        height: h,
        background: dashed
          ? `repeating-linear-gradient(90deg, ${theme.ink} 0, ${theme.ink} 8px, transparent 8px, transparent 13px)`
          : theme.ink,
      }}
    />
  );
};

// Vertical gap. data: { size?: name|number }.
export const SpaceBlock = ({ data, theme }) => (
  <div style={{ height: spaceToken(theme, data.size) }} />
);

// Horizontal columns. data: { gap?, cols: [{ w?, blocks?: [...] } | { w?, ...textData }] }.
// A column without `blocks` is shorthand for a single text block. Nested
// blocks render through the registry via ctx.renderBlock; note they do NOT
// get data-block-id wrappers, so per-block raster/heat overrides only apply
// to top-level blocks.
export const RowBlock = ({ data, theme, ctx }) => {
  const gap = spaceToken(theme, data.gap, "s");
  const cols = Array.isArray(data.cols) ? data.cols : [];
  return (
    <div style={{ display: "flex", alignItems: "flex-start", gap }}>
      {cols.map((col, i) => {
        const children = Array.isArray(col.blocks)
          ? col.blocks
          : [{ id: `col-${i}`, type: "text", data: { ...col, w: undefined } }];
        return (
          <div key={i} style={colWidthStyle(col.w)}>
            {children.map((child, j) => (
              <div key={child.id || j}>{ctx.renderBlock(child)}</div>
            ))}
          </div>
        );
      })}
    </div>
  );
};

// Aligned key/value rows. data: { items: [[key, value], ...] }.
export const KvBlock = ({ data, theme, blockStyle }) => {
  const kSt = { ...theme.preset("overline"), color: theme.muted };
  const vSt = { ...theme.preset("bodyStrong", blockStyle), color: theme.ink };
  return (
    <div style={{ display: "grid", gridTemplateColumns: "auto 1fr", columnGap: 12, rowGap: 5 }}>
      {(data.items || []).map(([k, v], i) => (
        <React.Fragment key={i}>
          <div style={{ ...kSt, alignSelf: "baseline", paddingTop: 2 }}>{k}</div>
          <div style={{ ...vSt, overflowWrap: "break-word", minWidth: 0 }}>{v}</div>
        </React.Fragment>
      ))}
    </div>
  );
};

// Marker list. data: { items: [...], marker?, preset?, lines? }.
export const SlipListBlock = ({ data, theme, blockStyle }) => {
  const st = { ...theme.preset(data.preset || "body", blockStyle), color: theme.ink };
  const marker = data.marker ?? "—";
  return (
    <div style={st}>
      {(data.items || []).map((item, i) => (
        <div key={i} style={{ display: "flex", gap: 8, padding: "2px 0" }}>
          <span style={{ flexShrink: 0 }}>{marker}</span>
          <span style={{ flex: 1, minWidth: 0, overflowWrap: "break-word", ...clampStyle(Number(data.lines) || 0) }}>
            {item}
          </span>
        </div>
      ))}
    </div>
  );
};

const CheckSquare = ({ theme, box }) => (
  <span
    style={{
      display: "inline-block",
      width: box,
      height: box,
      border: `2px solid ${theme.ink}`,
      flexShrink: 0,
    }}
  />
);

// Checkbox options to tick with a pen. data: { items, inline?, columns?, preset? }.
export const ChecksBlock = ({ data, theme, blockStyle }) => {
  const st = { ...theme.preset(data.preset || "body", blockStyle), color: theme.ink };
  const box = Math.round((parseFloat(st.fontSize) || 13) * 0.9);
  const items = data.items || [];
  const one = (label, i) => (
    <span key={i} style={{ display: "inline-flex", alignItems: "center", gap: 7 }}>
      <CheckSquare theme={theme} box={box} />
      <span>{label}</span>
    </span>
  );
  if (data.inline) {
    return <div style={{ ...st, display: "flex", flexWrap: "wrap", gap: "6px 18px" }}>{items.map(one)}</div>;
  }
  const columns = Math.max(1, Number(data.columns) || 1);
  return (
    <div style={{ ...st, display: "grid", gridTemplateColumns: `repeat(${columns}, 1fr)`, gap: "7px 12px" }}>
      {items.map(one)}
    </div>
  );
};

// Blank ruled lines to write on. data: { label?, lines? }.
export const WriteinBlock = ({ data, theme, blockStyle }) => {
  const labelSt = { ...theme.preset("overline", blockStyle), color: theme.muted };
  const n = Math.max(1, Number(data.lines) || 2);
  const h = ruleToken(theme, "hair");
  return (
    <div>
      {data.label && <div style={{ ...labelSt, marginBottom: 4 }}>{data.label}</div>}
      {Array.from({ length: n }, (_, i) => (
        <div key={i} style={{ height: 22, borderBottom: `${h}px solid ${theme.ink}` }} />
      ))}
    </div>
  );
};

// QR code. data: { value, size?, align?, caption? }. Rendered as crisp-edge
// SVG rects with an integer px module size so the 1-bit threshold keeps every
// module intact; falls back to a crossed box when the value is empty/invalid.
export const QrBlock = ({ data, theme, blockStyle }) => {
  const target = Math.max(48, Math.min(320, Number(data.size) || 120));
  const m = buildQrMatrix(data.value, target);
  const justify = data.align === "right" ? "flex-end" : data.align === "center" ? "center" : "flex-start";
  return (
    <div style={{ display: "flex", flexDirection: "column", alignItems: justify }}>
      {m ? (
        <svg
          width={m.size}
          height={m.size}
          viewBox={`0 0 ${m.size} ${m.size}`}
          shapeRendering="crispEdges"
          role="img"
          aria-label={data.caption || "QR code"}
        >
          <rect width={m.size} height={m.size} fill={theme.paper} />
          {Array.from({ length: m.count }, (_, r) =>
            Array.from({ length: m.count }, (_, c) =>
              m.dark(r, c) ? (
                <rect key={`${r}-${c}`} x={c * m.module} y={r * m.module} width={m.module} height={m.module} fill={theme.ink} />
              ) : null,
            ),
          )}
        </svg>
      ) : (
        <div
          style={{
            width: target,
            height: target,
            border: `2px solid ${theme.ink}`,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            ...theme.preset("small"),
            color: theme.muted,
          }}
        >
          no QR value
        </div>
      )}
      {data.caption && (
        <div style={{ ...theme.preset("small", { role: "mono" }, blockStyle), color: theme.muted, marginTop: 4 }}>
          {data.caption}
        </div>
      )}
    </div>
  );
};

// Horizontal bar chart. data: { values: [{label, value}], height? }.
export const BarsBlock = ({ data, theme, blockStyle }) => {
  const rows = (data.values || []).map((v) => ({ label: String(v.label ?? ""), value: Number(v.value) || 0 }));
  if (rows.length === 0) return null;
  const max = Math.max(...rows.map((r) => r.value), 1);
  const barH = Math.max(8, Number(data.height) || 14);
  const labelSt = { ...theme.preset("overline", blockStyle), color: theme.ink };
  const valueSt = { ...theme.preset("caption"), color: theme.ink };
  return (
    <div style={{ display: "grid", gridTemplateColumns: "auto 1fr auto", columnGap: 8, rowGap: 6, alignItems: "center" }}>
      {rows.map((r, i) => (
        <React.Fragment key={i}>
          <div style={labelSt}>{r.label}</div>
          <div style={{ minWidth: 0 }}>
            <div style={{ height: barH, width: `${Math.max(2, Math.round((r.value / max) * 100))}%`, background: theme.ink }} />
          </div>
          <div style={valueSt}>{r.value}</div>
        </React.Fragment>
      ))}
    </div>
  );
};

// Simple table. data: { cols: [{label, w?, align?}], rows: [[cell, ...]], preset? }.
export const SlipTableBlock = ({ data, theme, blockStyle }) => {
  const cols = data.cols || [];
  const headSt = { ...theme.preset("overline"), color: theme.muted };
  const cellSt = { ...theme.preset(data.preset || "body", blockStyle), color: theme.ink };
  const align = (i) => (cols[i]?.align === "right" ? "right" : cols[i]?.align === "center" ? "center" : "left");
  const widthStyle = (c) => (typeof c.w === "number" ? { width: c.w } : {});
  return (
    <table style={{ width: "100%", borderCollapse: "collapse" }}>
      <thead>
        <tr>
          {cols.map((c, i) => (
            <th
              key={i}
              style={{
                ...headSt,
                ...widthStyle(c),
                textAlign: align(i),
                padding: "0 4px 3px 0",
                borderBottom: `${ruleToken(theme, "hair")}px solid ${theme.ink}`,
              }}
            >
              {c.label || ""}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {(data.rows || []).map((row, r) => (
          <tr key={r}>
            {cols.map((_, i) => (
              <td key={i} style={{ ...cellSt, textAlign: align(i), padding: "3px 4px 0 0", verticalAlign: "baseline" }}>
                {row[i] != null ? String(row[i]) : ""}
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  );
};
