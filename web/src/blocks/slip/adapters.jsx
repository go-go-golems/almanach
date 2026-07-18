import React from "react";
import { defineBlock } from "../registry.js";
import {
  SlipTextBlock,
  BannerBlock,
  RuleBlock,
  SpaceBlock,
  RowBlock,
  KvBlock,
  SlipListBlock,
  ChecksBlock,
  WriteinBlock,
  QrBlock,
  BarsBlock,
  SlipTableBlock,
} from "./components.jsx";

// Adapter list for the work-slip pack. Merged into the studio registry via
// mergeBlockRegistries — the duplicate-type guard there fails loudly if a
// slip type ever collides with a studio type.
const adapt = (type, Component, needsCtx = false) =>
  defineBlock({
    type,
    module: "slip",
    render: (data, ctx) => (
      <Component data={data} theme={ctx.theme} blockStyle={ctx.block?.style} {...(needsCtx ? { ctx } : {})} />
    ),
  });

export const SLIP_ADAPTERS = [
  adapt("text", SlipTextBlock),
  adapt("banner", BannerBlock),
  adapt("rule", RuleBlock),
  adapt("space", SpaceBlock),
  adapt("row", RowBlock, true), // container: renders children via ctx.renderBlock
  adapt("kv", KvBlock),
  adapt("list", SlipListBlock),
  adapt("checks", ChecksBlock),
  adapt("writein", WriteinBlock),
  adapt("qr", QrBlock),
  adapt("bars", BarsBlock),
  adapt("table", SlipTableBlock),
];

// Default content per type — used by the editor palette's "add block" and to
// backfill missing fields on layout import (same contract as studio DEFAULTS).
export const SLIP_DEFAULTS = {
  text: { text: "A line of text", preset: "body", align: "left" },
  banner: { text: "BANNER", right: "", preset: "caption", pad: "s" },
  rule: { weight: "hair" },
  space: { size: "m" },
  row: {
    gap: "s",
    cols: [
      { w: "1fr", text: "Left", preset: "bodyStrong" },
      { w: "1fr", text: "Right", preset: "body", align: "right" },
    ],
  },
  kv: { items: [["KEY", "value"], ["ANOTHER", "value"]] },
  list: { marker: "—", items: ["first item", "second item"] },
  checks: { items: ["option a", "option b"], columns: 2 },
  writein: { label: "NOTES", lines: 2 },
  // A QR must never acquire a fake payload when importing a layout. The editor
  // shows the explicit no-value placeholder until the author supplies value.
  qr: { size: 120, align: "right", caption: "scan me" },
  bars: { height: 14, values: [{ label: "alpha", value: 12 }, { label: "beta", value: 7 }, { label: "gamma", value: 3 }] },
  table: {
    cols: [{ label: "item", w: undefined }, { label: "qty", w: 60, align: "right" }],
    rows: [["Widget", "4"], ["Sprocket", "11"]],
  },
};

// Editor palette metadata (label + group). Icons are assigned in the studio
// (lucide imports live there).
export const SLIP_BLOCK_TYPES = [
  { type: "text", label: "Text" },
  { type: "banner", label: "Banner" },
  { type: "rule", label: "Rule" },
  { type: "space", label: "Space" },
  { type: "row", label: "Row" },
  { type: "kv", label: "Key / Value" },
  { type: "list", label: "List" },
  { type: "checks", label: "Checks" },
  { type: "writein", label: "Write-in" },
  { type: "qr", label: "QR Code" },
  { type: "bars", label: "Bar Chart" },
  { type: "table", label: "Table" },
];
