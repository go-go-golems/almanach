import React, { useState, useEffect, useRef, useMemo, useCallback } from "react";
import {
  Plus, Trash2, ChevronUp, ChevronDown, Printer, Download,
  Type, Calendar, ListTodo, Newspaper, CloudSun, Quote as QuoteIcon,
  BookOpen, Clock, Sparkles, Brain, Smile, Minus, X,
  GripVertical, Pencil, Copy, Eye, FileText, Star, Layers,
  Sun, Moon, Leaf, Mountain, Rocket, BookMarked, Coffee,
  Image as ImageIcon, Upload
} from "lucide-react";
import { defineBlock, createBlockRegistry, resolveBlockAdapter } from "./blocks/registry.js";
import { makePresetResolver, mergePresetMaps } from "./typography/presets.js";

/* ================================================================
   ALMANACH STUDIO — thermal-printer almanac layout designer
================================================================ */

// ---- THEMES -----------------------------------------------------
const THEMES = {
  classic: {
    name: "Classic",
    icon: "✦",
    paper: "#ffffff",
    ink: "#000000",
    muted: "#000000",
    accent: "#000000",
    rule: "#000000",
    fontDisplay: "'Cormorant Garamond', 'Palatino Linotype', 'Book Antiqua', 'Georgia', serif",
    fontBody: "'EB Garamond', 'Georgia', 'Cambria', 'Times New Roman', serif",
    titleSize: 36,
    titleWeight: 700,
    titleSpacing: "0.1em",
    titleCase: "uppercase",
    ornateFrame: true,
    grain: 0,
  },
  minimal: {
    name: "Minimal",
    icon: "—",
    paper: "#ffffff",
    ink: "#000000",
    muted: "#000000",
    accent: "#000000",
    rule: "#000000",
    fontDisplay: "'EB Garamond', 'Georgia', 'Cambria', 'Times New Roman', serif",
    fontBody: "'EB Garamond', 'Georgia', 'Cambria', 'Times New Roman', serif",
    titleSize: 34,
    titleWeight: 500,
    titleSpacing: "0.18em",
    titleCase: "uppercase",
    ornateFrame: false,
    grain: 0,
  },
  botanical: {
    name: "Botanical",
    icon: "❦",
    paper: "#ffffff",
    ink: "#000000",
    muted: "#000000",
    accent: "#000000",
    rule: "#000000",
    fontDisplay: "'Cormorant Garamond', 'Palatino Linotype', 'Book Antiqua', 'Georgia', serif",
    fontBody: "'EB Garamond', 'Georgia', 'Cambria', 'Times New Roman', serif",
    titleSize: 32,
    titleWeight: 500,
    titleSpacing: "0.06em",
    titleCase: "none",
    ornateFrame: false,
    botanical: true,
    grain: 0,
  },
  notebook: {
    name: "Notebook",
    icon: "✎",
    paper: "#ffffff",
    ink: "#000000",
    muted: "#000000",
    accent: "#000000",
    rule: "#000000",
    fontDisplay: "'Caveat', 'Comic Sans MS', 'Chalkboard SE', cursive",
    fontBody: "'Kalam', 'Comic Sans MS', 'Chalkboard SE', cursive",
    titleSize: 38,
    titleWeight: 500,
    titleSpacing: "0",
    titleCase: "none",
    ornateFrame: false,
    lined: true,
    grain: 0,
  },
  ledger: {
    name: "Vintage Ledger",
    icon: "▣",
    paper: "#ffffff",
    ink: "#000000",
    muted: "#000000",
    accent: "#000000",
    rule: "#000000",
    fontDisplay: "'Cormorant Garamond', 'Palatino Linotype', 'Book Antiqua', 'Georgia', serif",
    fontBody: "'EB Garamond', 'Georgia', 'Cambria', 'Times New Roman', serif",
    titleSize: 30,
    titleWeight: 700,
    titleSpacing: "0.16em",
    titleCase: "uppercase",
    ornateFrame: false,
    boxed: true,
    grain: 0,
  },
  space: {
    name: "Space Age",
    icon: "✦",
    paper: "#ffffff",
    ink: "#000000",
    muted: "#000000",
    accent: "#000000",
    rule: "#000000",
    fontDisplay: "'Cormorant Garamond', 'Palatino Linotype', 'Book Antiqua', 'Georgia', serif",
    fontBody: "'EB Garamond', 'Georgia', 'Cambria', 'Times New Roman', serif",
    titleSize: 32,
    titleWeight: 600,
    titleSpacing: "0.18em",
    titleCase: "uppercase",
    ornateFrame: false,
    space: true,
    grain: 0,
  },
  // Hinted-font themes (ALMANACH-PIXELFONT): DejaVu prints crisp at small sizes
  // where the delicate Garamonds lose strokes at 1-bit. Embedded in
  // fonts-embedded.css.
  crisp: {
    name: "Crisp Serif (DejaVu)",
    icon: "▪",
    paper: "#ffffff",
    ink: "#000000",
    muted: "#000000",
    accent: "#000000",
    rule: "#000000",
    fontDisplay: "'DejaVu Serif', 'Georgia', serif",
    fontBody: "'DejaVu Serif', 'Georgia', serif",
    titleSize: 28,
    titleWeight: 700,
    titleSpacing: "0.08em",
    titleCase: "uppercase",
    ornateFrame: false,
    grain: 0,
  },
  crispsans: {
    name: "Crisp Sans (DejaVu)",
    icon: "▫",
    paper: "#ffffff",
    ink: "#000000",
    muted: "#000000",
    accent: "#000000",
    rule: "#000000",
    fontDisplay: "'DejaVu Sans', 'Helvetica Neue', sans-serif",
    fontBody: "'DejaVu Sans', 'Helvetica Neue', sans-serif",
    titleSize: 26,
    titleWeight: 700,
    titleSpacing: "0.06em",
    titleCase: "uppercase",
    ornateFrame: false,
    grain: 0,
  },
};

// A layout's `theme` may be a built-in name (string) OR an inline theme object
// (the "data-driven theme" case) that patches a base built-in with no code
// change. Returns { themeKey, patch, presetOverrides } where `patch` is merged
// over THEMES[themeKey] and `presetOverrides` is a preset layer under the
// layout's typography.
function resolveThemeSpec(spec) {
  if (typeof spec === "string") {
    return { themeKey: THEMES[spec] ? spec : "classic", patch: null, presetOverrides: {} };
  }
  if (spec && typeof spec === "object") {
    const base = THEMES[spec.base] ? spec.base : (THEMES[spec.name] ? spec.name : "classic");
    const patch = {};
    const c = spec.colors && typeof spec.colors === "object" ? spec.colors : {};
    for (const k of ["paper", "ink", "muted", "accent", "rule"]) {
      if (typeof c[k] === "string") patch[k] = c[k];
    }
    // font_palette: [display, body, mono] in preference order.
    const pal = Array.isArray(spec.fontPalette) ? spec.fontPalette : null;
    if (pal) {
      if (pal[0]) patch.fontDisplay = pal[0];
      if (pal[1] || pal[0]) patch.fontBody = pal[1] || pal[0];
      if (pal[2]) patch.fontMono = pal[2];
    }
    // Explicit font fields win over the palette.
    for (const k of ["fontDisplay", "fontBody", "fontMono"]) {
      if (typeof spec[k] === "string") patch[k] = spec[k];
    }
    if (typeof spec.titleSize === "number") patch.titleSize = spec.titleSize;
    if (typeof spec.titleWeight === "number") patch.titleWeight = spec.titleWeight;
    if (typeof spec.titleSpacing === "string") patch.titleSpacing = spec.titleSpacing;
    if (typeof spec.titleCase === "string") patch.titleCase = spec.titleCase;
    const presetOverrides = spec.presetOverrides && typeof spec.presetOverrides === "object"
      ? spec.presetOverrides
      : {};
    return { themeKey: base, patch, presetOverrides };
  }
  return { themeKey: "classic", patch: null, presetOverrides: {} };
}

// ---- DEFAULT BLOCK CONTENT --------------------------------------
const DEFAULTS = {
  title: { text: "THE ALMANACH", subtitle: "Your daily digest, printed fresh." },
  date: { date: "May 24, 2024", day: "Friday" },
  plan: {
    label: "Today's Plan",
    items: [
      { time: "08:30", text: "Morning routine", done: true },
      { time: "09:30", text: "Deep work: Project Atlas", done: true },
      { time: "12:30", text: "Lunch with Sam", done: false },
      { time: "14:00", text: "Review PRs", done: false },
      { time: "15:30", text: "Workout", done: false },
      { time: "18:30", text: "Call with Alex", done: false },
      { time: "20:00", text: "Read / Unwind", done: false },
    ],
  },
  news: {
    label: "Top News",
    items: [
      { headline: "OpenAI announces new GPT-4o updates with improved reasoning and tools.", source: "The Verge", time: "1h ago" },
      { headline: "Markets rally as inflation cools more than expected in April.", source: "Bloomberg", time: "2h ago" },
      { headline: "SpaceX launches Starship test flight successfully.", source: "Reuters", time: "3h ago" },
    ],
  },
  weather: { temp: "22°C", condition: "Partly cloudy", high: "24°C", low: "15°C", sunrise: "05:47", sunset: "20:32" },
  note: { label: "Daily Note", text: "Make something small, useful, and a little delightful.", author: "Almanach" },
  habits: {
    label: "Habit Tracker",
    range: "May 18 — May 24",
    columns: ["M", "T", "W", "T", "F", "S", "S"],
    items: [
      { name: "Meditate", days: [1, 1, 1, 1, 1, 0, 0] },
      { name: "Exercise", days: [1, 1, 1, 0, 1, 1, 0] },
      { name: "Read", days: [1, 1, 1, 1, 1, 1, 1] },
      { name: "No sugar", days: [1, 0, 1, 1, 1, 0, 0] },
      { name: "Journal", days: [1, 1, 1, 1, 1, 0, 0] },
      { name: "Sleep 7h+", days: [1, 1, 0, 1, 1, 1, 1] },
      { name: "Water 8 cups", days: [1, 1, 1, 1, 1, 1, 0] },
    ],
    reflection: "Felt more consistent and energized. Sleep can improve.",
  },
  quote: { label: "Quote of the Day", text: "The important thing is not to stop questioning. Curiosity has its own reason for existing.", author: "Albert Einstein" },
  word: { label: "Word of the Day", word: "Serendipity", phonetic: "ser·en·dip·i·ty", part: "noun", definition: "The occurrence and development of events by chance in a happy or beneficial way.", example: "A fortunate stroke of serendipity led them to the discovery." },
  history: {
    label: "Today in History",
    items: [
      { year: "1844", event: "Samuel Morse sends the first telegraph message: \"What hath God wrought?\"" },
      { year: "1935", event: "The Golden Gate Bridge opens to the public in San Francisco." },
      { year: "1961", event: "Alan Shepard becomes the first American in space." },
      { year: "2008", event: "The last episode of Mad Men airs on AMC." },
    ],
  },
  did: {
    label: "Did You Know?",
    items: [
      "Honey never spoils. Archaeologists have found pots of honey in ancient Egyptian tombs over 3,000 years old.",
      "Your brain uses about 20% of your body's energy, even though it's only 2% of your weight.",
      "A day on Venus is longer than a year on Venus.",
    ],
  },
  mood: { label: "Mood & Energy", mood: 4, energy: 4, sleep: "7h 15m", notes: "Woke up refreshed. Great focus all day." },
  reading: {
    label: "Reading List",
    current: { title: "Atomic Habits", author: "James Clear", progress: 52 },
    next: ["The Almanack of Naval Ravikant", "A Pattern Language — Christopher Alexander", "The Daily Stoic — Ryan Holiday", "Thinking, Fast and Slow — Daniel Kahneman"],
  },
  reflection: {
    label: "Daily Reflection",
    well: "Focused work in the morning. Good conversations.",
    better: "Less time on social media. Plan tomorrow's tasks earlier.",
    learned: "Consistency beats motivation.",
    quote: "Well done is better than well said. — Benjamin Franklin",
  },
  image: {
    label: "Image Plate",
    src: "",
    alt: "Uploaded photograph",
    caption: "",
    height: 160,
    fit: "cover", // cover | contain
    border: true,
    grayscale: true,
    thermalTone: "normal", // normal | light
  },
  divider: { style: "line" }, // line | dots | wave | leaves
};

// ---- BLOCK TYPE METADATA ----------------------------------------
const BLOCK_TYPES = [
  { type: "title", label: "Title", icon: Type, group: "header" },
  { type: "date", label: "Date Strip", icon: Calendar, group: "header" },
  { type: "divider", label: "Divider", icon: Minus, group: "header" },
  { type: "plan", label: "Today's Plan", icon: ListTodo, group: "daily" },
  { type: "news", label: "Top News", icon: Newspaper, group: "daily" },
  { type: "weather", label: "Weather", icon: CloudSun, group: "daily" },
  { type: "note", label: "Daily Note", icon: FileText, group: "daily" },
  { type: "image", label: "Image Plate", icon: ImageIcon, group: "daily" },
  { type: "habits", label: "Habit Tracker", icon: Layers, group: "tracker" },
  { type: "mood", label: "Mood & Energy", icon: Smile, group: "tracker" },
  { type: "reading", label: "Reading List", icon: BookOpen, group: "tracker" },
  { type: "reflection", label: "Daily Reflection", icon: Pencil, group: "tracker" },
  { type: "quote", label: "Quote of the Day", icon: QuoteIcon, group: "knowledge" },
  { type: "word", label: "Word of the Day", icon: BookMarked, group: "knowledge" },
  { type: "history", label: "Today in History", icon: Clock, group: "knowledge" },
  { type: "did", label: "Did You Know?", icon: Brain, group: "knowledge" },
];

const GROUPS = {
  header: { label: "Header", color: "#c9a36b" },
  daily: { label: "Daily", color: "#9bb088" },
  tracker: { label: "Trackers", color: "#b88869" },
  knowledge: { label: "Knowledge", color: "#9c9bc4" },
};

// ---- HELPERS ----------------------------------------------------
const uid = () => Math.random().toString(36).slice(2, 9);

const newBlock = (type) => ({
  id: uid(),
  type,
  data: JSON.parse(JSON.stringify(DEFAULTS[type])),
});

const STARTER_BLOCKS = [
  newBlock("title"),
  newBlock("date"),
  newBlock("plan"),
  newBlock("news"),
  newBlock("weather"),
  newBlock("note"),
];

// =================================================================
// BLOCK RENDERERS — render each block onto the thermal paper
// =================================================================

const Ornament = ({ theme, kind = "rule" }) => {
  if (kind === "rule") {
    return <div style={{ borderTop: `1px solid ${theme.rule}`, margin: "10px 0" }} />;
  }
  if (kind === "dots") {
    return (
      <div style={{ display: "flex", justifyContent: "center", gap: 6, margin: "10px 0", color: theme.muted }}>
        <span>·</span><span>·</span><span style={{ color: theme.ink }}>✦</span><span>·</span><span>·</span>
      </div>
    );
  }
  if (kind === "wave") {
    return (
      <svg width="100%" height="10" viewBox="0 0 200 10" preserveAspectRatio="none" style={{ margin: "8px 0" }}>
        <path d="M0,5 Q12.5,0 25,5 T50,5 T75,5 T100,5 T125,5 T150,5 T175,5 T200,5" fill="none" stroke={theme.rule} strokeWidth="1" />
      </svg>
    );
  }
  if (kind === "leaves") {
    return (
      <div style={{ display: "flex", justifyContent: "center", alignItems: "center", gap: 8, margin: "10px 0", color: theme.accent, fontSize: theme.fs(14) }}>
        <span style={{ transform: "rotate(-15deg)" }}>❦</span>
        <div style={{ flex: 1, borderTop: `1px solid ${theme.rule}` }} />
        <span>✦</span>
        <div style={{ flex: 1, borderTop: `1px solid ${theme.rule}` }} />
        <span style={{ transform: "rotate(15deg)" }}>❦</span>
      </div>
    );
  }
  return null;
};

const SectionLabel = ({ label, theme, icon }) => (
  <div style={{
    display: "flex", alignItems: "center", gap: 6,
    ...theme.preset("sectionLabel"),
    color: theme.ink,
    margin: "4px 0 8px",
  }}>
    {icon && <span style={{ color: theme.accent, fontSize: theme.fs(13) }}>{icon}</span>}
    <span>{label}</span>
  </div>
);

const TitleBlock = ({ data, theme, blockStyle }) => {
  return (
    <div style={{ textAlign: "center", padding: "4px 0 10px" }}>
      {theme.ornateFrame && (
        <div style={{ fontSize: 18, color: theme.accent, marginBottom: -2, letterSpacing: "0.4em" }}>
          ⌘ ✦ ⌘
        </div>
      )}
      {theme.botanical && <div style={{ fontSize: 18, color: theme.accent, marginBottom: 2 }}>❦</div>}
      <div style={{ ...theme.preset("caption", { italic: true }), color: theme.muted, marginBottom: 2 }}>The</div>
      <h1 style={{
        // Title sizing stays theme-driven (per-theme titleSize/weight/case);
        // an inline block style may still override any of it.
        fontFamily: theme.fontDisplay,
        fontSize: theme.titleSize,
        fontWeight: theme.titleWeight,
        letterSpacing: theme.titleSpacing,
        textTransform: theme.titleCase,
        color: theme.ink,
        margin: 0,
        lineHeight: 1,
        ...theme.preset("__title", blockStyle),
      }}>
        {data.text}
      </h1>
      {data.subtitle && (
        <div style={{
          ...theme.preset("caption", { italic: !theme.lined }),
          color: theme.muted,
          marginTop: 6,
        }}>
          {data.subtitle}
        </div>
      )}
      {theme.ornateFrame && (
        <div style={{ display: "flex", justifyContent: "center", gap: 12, marginTop: 8, color: theme.accent, fontSize: 12 }}>
          <span>❦</span><span>—</span><span>✦</span><span>—</span><span>❦</span>
        </div>
      )}
    </div>
  );
};

const DateBlock = ({ data, theme, blockStyle }) => (
  <div style={{
    ...theme.preset("body", { letterSpacing: 0.05 }, blockStyle),
    color: theme.ink,
    textAlign: "center",
    padding: "4px 0",
    borderTop: `1px solid ${theme.rule}`,
    borderBottom: `1px solid ${theme.rule}`,
  }}>
    {data.date} <span style={{ margin: "0 8px", color: theme.muted }}>|</span> {data.day}
  </div>
);

const PlanBlock = ({ data, theme, blockStyle }) => (
  <div>
    <SectionLabel label={data.label} theme={theme} icon="◷" />
    <div style={{ ...theme.preset("body", blockStyle), color: theme.ink }}>
      {data.items.map((it, i) => (
        <div key={i} style={{
          display: "flex", alignItems: "baseline", gap: 8,
          padding: "2px 0",
          opacity: 1,
        }}>
          <span style={{
            display: "inline-flex", alignItems: "center", justifyContent: "center",
            width: 13, height: 13, border: `1px solid ${theme.ink}`,
            fontSize: theme.fs(10), lineHeight: 1, flexShrink: 0,
            transform: "translateY(2px)",
          }}>
            {it.done ? "✓" : ""}
          </span>
          <span style={{
            ...theme.preset("meta"),
            fontVariantNumeric: "tabular-nums",
            color: theme.muted,
            minWidth: 40,
          }}>
            {it.time}
          </span>
          <span style={{
            flex: 1,
            textDecoration: it.done ? "line-through" : "none",
            textDecorationColor: theme.muted,
          }}>
            {it.text}
          </span>
        </div>
      ))}
    </div>
  </div>
);

const NewsBlock = ({ data, theme, blockStyle }) => (
  <div>
    <SectionLabel label={data.label} theme={theme} icon="❍" />
    <ul style={{ margin: 0, paddingLeft: 14, ...theme.preset("body", blockStyle), color: theme.ink }}>
      {data.items.map((it, i) => (
        <li key={i} style={{ marginBottom: 6, lineHeight: 1.35 }}>
          {it.headline}
          <div style={{ ...theme.preset("small"), color: theme.muted, marginTop: 1 }}>
            <em style={{ fontStyle: "italic" }}>{it.source}</em>
            <span style={{ margin: "0 6px" }}>·</span>
            {it.time}
          </div>
        </li>
      ))}
    </ul>
  </div>
);

const WeatherIcon = ({ theme }) => (
  <svg width="28" height="22" viewBox="0 0 32 24" fill="none" stroke={theme.ink} strokeWidth="1.2">
    <circle cx="11" cy="10" r="4.5" fill="none" />
    <path d="M5 10 L2 10 M11 16 L11 19 M11 1 L11 4 M16.2 4.8 L18.3 2.7 M5.8 4.8 L3.7 2.7" />
    <ellipse cx="20" cy="15" rx="9" ry="5" fill={theme.paper} stroke={theme.ink} />
  </svg>
);

const WeatherBlock = ({ data, theme, blockStyle }) => (
  <div style={{ display: "flex", alignItems: "center", gap: 12, fontFamily: theme.fontBody, color: theme.ink }}>
    <div style={{ flexShrink: 0 }}><WeatherIcon theme={theme} /></div>
    <div style={{ flex: 1 }}>
      <div style={{ ...theme.preset("metric", blockStyle), color: theme.ink }}>
        {data.temp}
      </div>
      <div style={{ ...theme.preset("caption"), color: theme.muted, marginTop: 2 }}>{data.condition}</div>
    </div>
    <div style={{ ...theme.preset("small"), color: theme.muted, textAlign: "right", lineHeight: 1.5 }}>
      <div>H {data.high} &nbsp;/&nbsp; L {data.low}</div>
      <div>{data.sunrise} &nbsp;/&nbsp; {data.sunset}</div>
    </div>
  </div>
);

const NoteBlock = ({ data, theme, blockStyle }) => (
  <div>
    {data.label && <SectionLabel label={data.label} theme={theme} icon="✎" />}
    <div style={{
      ...theme.preset("emphasis", blockStyle),
      color: theme.ink,
      paddingLeft: 8,
      borderLeft: `2px solid ${theme.accent}`,
    }}>
      "{data.text}"
      {data.author && (
        <div style={{ ...theme.preset("caption", { italic: false }), color: theme.muted, marginTop: 4, textAlign: "right" }}>
          — {data.author}
        </div>
      )}
    </div>
  </div>
);

const HabitsBlock = ({ data, theme, blockStyle }) => (
  <div>
    <SectionLabel label={data.label} theme={theme} icon="▦" />
    <div style={{ ...theme.preset("small"), color: theme.muted, textAlign: "center", marginBottom: 6 }}>
      {data.range}
    </div>
    <div style={{ ...theme.preset("body", blockStyle), color: theme.ink }}>
      <div style={{ display: "grid", gridTemplateColumns: "1fr repeat(7, 18px)", gap: 4, alignItems: "center", marginBottom: 4 }}>
        <div />
        {data.columns.map((c, i) => (
          <div key={i} style={{ ...theme.preset("small"), textAlign: "center", color: theme.muted }}>{c}</div>
        ))}
      </div>
      {data.items.map((h, i) => (
        <div key={i} style={{ display: "grid", gridTemplateColumns: "1fr repeat(7, 18px)", gap: 4, alignItems: "center", padding: "1.5px 0" }}>
          <div style={{ ...theme.preset("caption") }}>{h.name}</div>
          {h.days.map((d, j) => (
            <div key={j} style={{ textAlign: "center" }}>
              <span style={{
                display: "inline-block",
                width: 10, height: 10, borderRadius: "50%",
                border: `1px solid ${theme.ink}`,
                background: d ? theme.ink : "transparent",
              }} />
            </div>
          ))}
        </div>
      ))}
    </div>
    {data.reflection && (
      <>
        <Ornament theme={theme} kind="dots" />
        <div style={{ ...theme.preset("overline"), color: theme.muted, textAlign: "center", marginBottom: 4 }}>
          Weekly Reflection
        </div>
        <div style={{ ...theme.preset("emphasis"), color: theme.ink, textAlign: "center" }}>
          {data.reflection}
        </div>
      </>
    )}
  </div>
);

const QuoteBlock = ({ data, theme, blockStyle }) => (
  <div style={{ textAlign: "center" }}>
    {data.label && <SectionLabel label={data.label} theme={theme} icon="❝" />}
    <div style={{ ...theme.preset("emphasis", blockStyle), color: theme.ink, padding: "0 4px" }}>
      "{data.text}"
    </div>
    <div style={{ ...theme.preset("caption", { role: "display", italic: false }), color: theme.muted, marginTop: 8 }}>
      — {data.author}
    </div>
  </div>
);

const WordBlock = ({ data, theme, blockStyle }) => (
  <div style={{ textAlign: "center" }}>
    {data.label && <SectionLabel label={data.label} theme={theme} icon="✦" />}
    <div style={{ ...theme.preset("word", blockStyle), color: theme.ink }}>
      {data.word}
    </div>
    <div style={{ ...theme.preset("caption"), color: theme.muted, marginTop: 2 }}>
      <em>{data.part}</em> &nbsp;|&nbsp; {data.phonetic}
    </div>
    <Ornament theme={theme} kind="dots" />
    <div style={{ ...theme.preset("body"), color: theme.ink, padding: "0 4px" }}>
      {data.definition}
    </div>
    {data.example && (
      <div style={{ ...theme.preset("small", { italic: true }), color: theme.muted, lineHeight: 1.4, marginTop: 8, padding: "0 4px" }}>
        "{data.example}"
      </div>
    )}
  </div>
);

const HistoryBlock = ({ data, theme, blockStyle }) => (
  <div>
    <SectionLabel label={data.label} theme={theme} icon="◷" />
    <div style={{ ...theme.preset("body", blockStyle), color: theme.ink }}>
      {data.items.map((it, i) => (
        <div key={i} style={{ display: "grid", gridTemplateColumns: "auto 1fr", gap: 10, padding: "3px 0", alignItems: "baseline" }}>
          <div style={{ display: "flex", alignItems: "baseline", gap: 6 }}>
            <span style={{
              display: "inline-block", width: 5, height: 5, borderRadius: "50%",
              background: theme.ink, transform: "translateY(-2px)",
            }} />
            <span style={{ ...theme.preset("bodyStrong", { role: "display" }), color: theme.ink, fontVariantNumeric: "tabular-nums" }}>
              {it.year}
            </span>
          </div>
          <div style={{ lineHeight: 1.35 }}>{it.event}</div>
        </div>
      ))}
    </div>
  </div>
);

const DidBlock = ({ data, theme, blockStyle }) => (
  <div>
    <SectionLabel label={data.label} theme={theme} icon="✦" />
    <div style={{ ...theme.preset("body", blockStyle), color: theme.ink }}>
      {data.items.map((t, i) => (
        <div key={i} style={{ display: "flex", gap: 8, padding: "3px 0", lineHeight: 1.4 }}>
          <span style={{ color: theme.accent, fontSize: theme.fs(14), lineHeight: 1, marginTop: 1 }}>❀</span>
          <span style={{ flex: 1 }}>{t}</span>
        </div>
      ))}
    </div>
  </div>
);

const imageThermalFilter = (data, preview = false) => {
  if (data.grayscale === false) return "none";
  if (data.thermalTone === "light") {
    return preview
      ? "grayscale(100%) brightness(1.22) contrast(0.88)"
      : "grayscale(100%) brightness(1.28) contrast(0.82)";
  }
  return preview ? "grayscale(100%) contrast(1.2)" : "grayscale(100%) contrast(1.25)";
};

const ImageBlock = ({ data, theme }) => {
  const height = Math.max(48, Math.min(420, Number(data.height) || 160));
  const hasImage = typeof data.src === "string" && data.src.trim().length > 0;

  return (
    <div>
      {data.label && <SectionLabel label={data.label} theme={theme} icon="▧" />}
      <div style={{
        border: data.border ? `1px solid ${theme.rule}` : "none",
        padding: data.border ? 4 : 0,
        background: theme.paper,
      }}>
        {hasImage ? (
          <img
            src={data.src}
            alt={data.alt || data.caption || "Almanach image"}
            crossOrigin={data.src.startsWith("data:") ? undefined : "anonymous"}
            style={{
              display: "block",
              width: "100%",
              height,
              objectFit: data.fit === "contain" ? "contain" : "cover",
              background: theme.paper,
              filter: imageThermalFilter(data),
            }}
          />
        ) : (
          <div style={{
            height,
            border: `1px dashed ${theme.rule}`,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            ...theme.preset("small", { letterSpacing: 0.08, textCase: "upper" }),
            color: theme.muted,
          }}>
            Add an image URL or upload a file
          </div>
        )}
      </div>
      {data.caption && (
        <div style={{
          ...theme.preset("small", { italic: true }),
          color: theme.muted,
          textAlign: "center",
          marginTop: 5,
        }}>
          {data.caption}
        </div>
      )}
    </div>
  );
};

const MoodFace = ({ active, theme }) => (
  <svg width="14" height="14" viewBox="0 0 16 16" stroke={theme.ink} strokeWidth="1.1" fill={active ? theme.ink : "transparent"}>
    <circle cx="8" cy="8" r="6.5" fill={active ? theme.ink : theme.paper} />
    <circle cx="6" cy="7" r="0.8" fill={active ? theme.paper : theme.ink} stroke="none" />
    <circle cx="10" cy="7" r="0.8" fill={active ? theme.paper : theme.ink} stroke="none" />
    <path d="M5.5 10 Q8 12 10.5 10" fill="none" stroke={active ? theme.paper : theme.ink} strokeLinecap="round" />
  </svg>
);

const Battery = ({ filled, theme }) => (
  <svg width="16" height="10" viewBox="0 0 18 10">
    <rect x="0.5" y="0.5" width="14" height="9" fill="none" stroke={theme.ink} strokeWidth="1" />
    <rect x="15" y="3" width="2" height="4" fill={theme.ink} />
    {filled && <rect x="2" y="2" width="11" height="6" fill={theme.ink} />}
  </svg>
);

const MoodBlock = ({ data, theme, blockStyle }) => {
  const rowLabel = { width: 60, ...theme.preset("small", { letterSpacing: 0.1, textCase: "upper" }), color: theme.muted };
  return (
  <div>
    <SectionLabel label={data.label} theme={theme} icon="☺" />
    <div style={{ ...theme.preset("body", blockStyle), color: theme.ink }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, padding: "4px 0" }}>
        <div style={rowLabel}>Mood</div>
        <div style={{ display: "flex", gap: 6, flex: 1, justifyContent: "center" }}>
          {[1, 2, 3, 4, 5].map((n) => (
            <MoodFace key={n} active={n === data.mood} theme={theme} />
          ))}
        </div>
      </div>
      <div style={{ display: "flex", alignItems: "center", gap: 8, padding: "4px 0" }}>
        <div style={rowLabel}>Energy</div>
        <div style={{ display: "flex", gap: 6, flex: 1, justifyContent: "center", alignItems: "center" }}>
          {[1, 2, 3, 4, 5].map((n) => (
            <Battery key={n} filled={n <= data.energy} theme={theme} />
          ))}
        </div>
      </div>
      <div style={{ display: "flex", alignItems: "center", gap: 8, padding: "4px 0" }}>
        <div style={rowLabel}>Sleep</div>
        <div style={{ flex: 1, textAlign: "center", ...theme.preset("body", { role: "display" }) }}>☾ {data.sleep}</div>
      </div>
      {data.notes && (
        <div style={{ ...theme.preset("small", { italic: true }), color: theme.muted, marginTop: 6, textAlign: "center" }}>
          {data.notes}
        </div>
      )}
    </div>
  </div>
  );
};

const ReadingBlock = ({ data, theme, blockStyle }) => (
  <div>
    <SectionLabel label={data.label} theme={theme} icon="❒" />
    <div style={{ ...theme.preset("body", blockStyle), color: theme.ink }}>
      <div style={{ ...theme.preset("overline"), color: theme.muted, textAlign: "center", marginBottom: 6 }}>
        Currently Reading
      </div>
      <div style={{ border: `1px solid ${theme.rule}`, padding: "8px 10px", marginBottom: 8 }}>
        <div style={{ ...theme.preset("body", { role: "display", size: 15 }), color: theme.ink }}>{data.current.title}</div>
        <div style={{ ...theme.preset("caption", { italic: true }), color: theme.muted, marginBottom: 6 }}>{data.current.author}</div>
        <div style={{ ...theme.preset("small"), color: theme.muted, marginBottom: 3 }}>{data.current.progress}% complete</div>
        <div style={{ height: 6, border: `1px solid ${theme.ink}`, position: "relative" }}>
          <div style={{ position: "absolute", inset: 0, width: `${data.current.progress}%`, background: theme.ink }} />
        </div>
      </div>
      <div style={{ ...theme.preset("overline"), color: theme.muted, textAlign: "center", marginBottom: 4 }}>
        Want to Read
      </div>
      <ul style={{ margin: 0, paddingLeft: 14, ...theme.preset("caption"), lineHeight: 1.5 }}>
        {data.next.map((b, i) => <li key={i}>{b}</li>)}
      </ul>
    </div>
  </div>
);

const ReflectionBlock = ({ data, theme, blockStyle }) => (
  <div>
    <SectionLabel label={data.label} theme={theme} icon="✎" />
    <div style={{ ...theme.preset("body", blockStyle), color: theme.ink }}>
      {[
        { k: "What went well?", v: data.well },
        { k: "What could be better?", v: data.better },
        { k: "What did I learn?", v: data.learned },
      ].map((row, i) => (
        <div key={i} style={{ marginBottom: 8 }}>
          <div style={{ ...theme.preset("overline"), color: theme.muted, marginBottom: 2 }}>
            {row.k}
          </div>
          <div style={{ lineHeight: 1.4 }}>{row.v}</div>
        </div>
      ))}
      {data.quote && (
        <>
          <Ornament theme={theme} kind="dots" />
          <div style={{ ...theme.preset("emphasis"), textAlign: "center", padding: "0 4px" }}>
            "{data.quote}"
          </div>
        </>
      )}
    </div>
  </div>
);

const DividerBlock = ({ data, theme }) => <Ornament theme={theme} kind={data.style} />;

// Visible placeholder for a block whose `type` is not in the registry. Unknown
// types render this instead of being dropped at parse or crashing the render.
const UnknownBlock = ({ type, theme }) => (
  <div style={{
    border: `1px dashed ${theme.rule}`,
    padding: "8px 10px",
    fontFamily: theme.fontBody,
    fontSize: theme.fs(11),
    color: theme.muted,
    textAlign: "center",
    lineHeight: 1.4,
  }}>
    Unknown block type: “{type}” — not registered.
  </div>
);

// Block registry (DSL v2, Phase 2): type -> adapter. Each adapter wraps its
// existing component; `render(data, ctx)` returns the paper node. Building the
// registry from an array (not a bare object) gives us the duplicate-type guard
// and a single place to compose future sub-registries.
const BLOCK_ADAPTERS = [
  defineBlock({ type: "title", module: "studio", render: (data, ctx) => <TitleBlock data={data} theme={ctx.theme} blockStyle={ctx.block?.style} /> }),
  defineBlock({ type: "date", module: "studio", render: (data, ctx) => <DateBlock data={data} theme={ctx.theme} blockStyle={ctx.block?.style} /> }),
  defineBlock({ type: "plan", module: "studio", render: (data, ctx) => <PlanBlock data={data} theme={ctx.theme} blockStyle={ctx.block?.style} /> }),
  defineBlock({ type: "news", module: "studio", render: (data, ctx) => <NewsBlock data={data} theme={ctx.theme} blockStyle={ctx.block?.style} /> }),
  defineBlock({ type: "weather", module: "studio", render: (data, ctx) => <WeatherBlock data={data} theme={ctx.theme} blockStyle={ctx.block?.style} /> }),
  defineBlock({ type: "note", module: "studio", render: (data, ctx) => <NoteBlock data={data} theme={ctx.theme} blockStyle={ctx.block?.style} /> }),
  defineBlock({ type: "image", module: "studio", render: (data, ctx) => <ImageBlock data={data} theme={ctx.theme} blockStyle={ctx.block?.style} /> }),
  defineBlock({ type: "habits", module: "studio", render: (data, ctx) => <HabitsBlock data={data} theme={ctx.theme} blockStyle={ctx.block?.style} /> }),
  defineBlock({ type: "quote", module: "studio", render: (data, ctx) => <QuoteBlock data={data} theme={ctx.theme} blockStyle={ctx.block?.style} /> }),
  defineBlock({ type: "word", module: "studio", render: (data, ctx) => <WordBlock data={data} theme={ctx.theme} blockStyle={ctx.block?.style} /> }),
  defineBlock({ type: "history", module: "studio", render: (data, ctx) => <HistoryBlock data={data} theme={ctx.theme} blockStyle={ctx.block?.style} /> }),
  defineBlock({ type: "did", module: "studio", render: (data, ctx) => <DidBlock data={data} theme={ctx.theme} blockStyle={ctx.block?.style} /> }),
  defineBlock({ type: "mood", module: "studio", render: (data, ctx) => <MoodBlock data={data} theme={ctx.theme} blockStyle={ctx.block?.style} /> }),
  defineBlock({ type: "reading", module: "studio", render: (data, ctx) => <ReadingBlock data={data} theme={ctx.theme} blockStyle={ctx.block?.style} /> }),
  defineBlock({ type: "reflection", module: "studio", render: (data, ctx) => <ReflectionBlock data={data} theme={ctx.theme} blockStyle={ctx.block?.style} /> }),
  defineBlock({ type: "divider", module: "studio", render: (data, ctx) => <DividerBlock data={data} theme={ctx.theme} blockStyle={ctx.block?.style} /> }),
];

const BLOCK_REGISTRY = createBlockRegistry(BLOCK_ADAPTERS);

// Render one block through the registry, falling back to a visible placeholder.
function renderBlock(block, ctx) {
  const adapter = resolveBlockAdapter(ctx.registry, block.type);
  if (!adapter) return <UnknownBlock type={block.type} theme={ctx.theme} />;
  return adapter.render(block.data, { ...ctx, block });
}

// =================================================================
// EDITORS — right-panel forms for each block type
// =================================================================

const Field = ({ label, children }) => (
  <label style={{ display: "block", marginBottom: 12 }}>
    <div style={{ fontSize: 10, letterSpacing: "0.14em", textTransform: "uppercase", color: "var(--ui-muted)", marginBottom: 4, fontWeight: 600 }}>
      {label}
    </div>
    {children}
  </label>
);

const inputStyle = {
  width: "100%",
  background: "rgba(0,0,0,0.25)",
  border: "1px solid var(--ui-border)",
  color: "var(--ui-text)",
  padding: "7px 9px",
  fontSize: 12.5,
  fontFamily: "inherit",
  outline: "none",
  borderRadius: 2,
};

const TextInput = (props) => <input {...props} style={{ ...inputStyle, ...(props.style || {}) }} />;
const TextArea = (props) => <textarea {...props} style={{ ...inputStyle, minHeight: 60, resize: "vertical", ...(props.style || {}) }} />;

const ListEditor = ({ items, onChange, renderItem, makeNew, label }) => (
  <div>
    <div style={{ fontSize: 10, letterSpacing: "0.14em", textTransform: "uppercase", color: "var(--ui-muted)", marginBottom: 6, fontWeight: 600 }}>
      {label}
    </div>
    <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
      {items.map((it, i) => (
        <div key={i} style={{ position: "relative", padding: "10px 10px 8px", background: "rgba(0,0,0,0.18)", border: "1px solid var(--ui-border)", borderRadius: 2 }}>
          {renderItem(it, (next) => {
            const copy = [...items]; copy[i] = next; onChange(copy);
          })}
          <div style={{ display: "flex", justifyContent: "flex-end", gap: 4, marginTop: 6 }}>
            <button onClick={() => {
              if (i === 0) return;
              const copy = [...items]; [copy[i - 1], copy[i]] = [copy[i], copy[i - 1]]; onChange(copy);
            }} className="mini-btn" title="Move up"><ChevronUp size={12} /></button>
            <button onClick={() => {
              if (i === items.length - 1) return;
              const copy = [...items]; [copy[i + 1], copy[i]] = [copy[i], copy[i + 1]]; onChange(copy);
            }} className="mini-btn" title="Move down"><ChevronDown size={12} /></button>
            <button onClick={() => onChange(items.filter((_, j) => j !== i))} className="mini-btn danger" title="Delete"><Trash2 size={12} /></button>
          </div>
        </div>
      ))}
    </div>
    <button
      onClick={() => onChange([...items, makeNew()])}
      style={{
        marginTop: 8, width: "100%", padding: "7px",
        border: "1px dashed var(--ui-border)", background: "transparent",
        color: "var(--ui-muted)", fontFamily: "inherit", fontSize: 11, cursor: "pointer",
        letterSpacing: "0.12em", textTransform: "uppercase",
      }}
    >
      + Add
    </button>
  </div>
);

const TitleEditor = ({ data, set }) => (
  <>
    <Field label="Title"><TextInput value={data.text} onChange={(e) => set({ ...data, text: e.target.value })} /></Field>
    <Field label="Subtitle"><TextInput value={data.subtitle} onChange={(e) => set({ ...data, subtitle: e.target.value })} /></Field>
  </>
);

const DateEditor = ({ data, set }) => (
  <>
    <Field label="Date"><TextInput value={data.date} onChange={(e) => set({ ...data, date: e.target.value })} /></Field>
    <Field label="Day"><TextInput value={data.day} onChange={(e) => set({ ...data, day: e.target.value })} /></Field>
  </>
);

const PlanEditor = ({ data, set }) => (
  <>
    <Field label="Section Label"><TextInput value={data.label} onChange={(e) => set({ ...data, label: e.target.value })} /></Field>
    <ListEditor
      label="Items"
      items={data.items}
      onChange={(items) => set({ ...data, items })}
      makeNew={() => ({ time: "12:00", text: "New item", done: false })}
      renderItem={(it, setIt) => (
        <>
          <div style={{ display: "flex", gap: 6, marginBottom: 6 }}>
            <TextInput style={{ width: 70 }} value={it.time} onChange={(e) => setIt({ ...it, time: e.target.value })} placeholder="Time" />
            <TextInput value={it.text} onChange={(e) => setIt({ ...it, text: e.target.value })} placeholder="Task" />
          </div>
          <label style={{ display: "flex", alignItems: "center", gap: 6, fontSize: 11, color: "var(--ui-muted)" }}>
            <input type="checkbox" checked={it.done} onChange={(e) => setIt({ ...it, done: e.target.checked })} />
            Completed
          </label>
        </>
      )}
    />
  </>
);

const NewsEditor = ({ data, set }) => (
  <>
    <Field label="Section Label"><TextInput value={data.label} onChange={(e) => set({ ...data, label: e.target.value })} /></Field>
    <ListEditor
      label="Items"
      items={data.items}
      onChange={(items) => set({ ...data, items })}
      makeNew={() => ({ headline: "Headline goes here.", source: "Source", time: "1h ago" })}
      renderItem={(it, setIt) => (
        <>
          <TextArea style={{ marginBottom: 6, minHeight: 50 }} value={it.headline} onChange={(e) => setIt({ ...it, headline: e.target.value })} placeholder="Headline" />
          <div style={{ display: "flex", gap: 6 }}>
            <TextInput value={it.source} onChange={(e) => setIt({ ...it, source: e.target.value })} placeholder="Source" />
            <TextInput style={{ width: 80 }} value={it.time} onChange={(e) => setIt({ ...it, time: e.target.value })} placeholder="Time" />
          </div>
        </>
      )}
    />
  </>
);

const WeatherEditor = ({ data, set }) => (
  <>
    <Field label="Temperature"><TextInput value={data.temp} onChange={(e) => set({ ...data, temp: e.target.value })} /></Field>
    <Field label="Condition"><TextInput value={data.condition} onChange={(e) => set({ ...data, condition: e.target.value })} /></Field>
    <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8 }}>
      <Field label="High"><TextInput value={data.high} onChange={(e) => set({ ...data, high: e.target.value })} /></Field>
      <Field label="Low"><TextInput value={data.low} onChange={(e) => set({ ...data, low: e.target.value })} /></Field>
      <Field label="Sunrise"><TextInput value={data.sunrise} onChange={(e) => set({ ...data, sunrise: e.target.value })} /></Field>
      <Field label="Sunset"><TextInput value={data.sunset} onChange={(e) => set({ ...data, sunset: e.target.value })} /></Field>
    </div>
  </>
);

const NoteEditor = ({ data, set }) => (
  <>
    <Field label="Section Label"><TextInput value={data.label || ""} onChange={(e) => set({ ...data, label: e.target.value })} /></Field>
    <Field label="Quote / Note"><TextArea value={data.text} onChange={(e) => set({ ...data, text: e.target.value })} /></Field>
    <Field label="Author"><TextInput value={data.author} onChange={(e) => set({ ...data, author: e.target.value })} /></Field>
  </>
);

const ImageEditor = ({ data, set }) => {
  const fileInputRef = useRef(null);

  const handleFile = async (e) => {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      set({
        ...data,
        src: String(reader.result || ""),
        alt: data.alt || file.name,
        caption: data.caption || file.name.replace(/\.[^.]+$/, ""),
      });
    };
    reader.readAsDataURL(file);
  };

  return (
    <>
      <Field label="Section Label"><TextInput value={data.label || ""} onChange={(e) => set({ ...data, label: e.target.value })} /></Field>
      <Field label="Image URL or data URL"><TextArea value={data.src || ""} onChange={(e) => set({ ...data, src: e.target.value })} placeholder="https://… or data:image/jpeg;base64,…" /></Field>
      <input ref={fileInputRef} type="file" accept="image/*" style={{ display: "none" }} onChange={handleFile} />
      <button
        type="button"
        className="btn"
        onClick={() => fileInputRef.current?.click()}
        style={{ width: "100%", justifyContent: "center", marginBottom: 12 }}
      >
        <Upload size={13} /> Upload Image
      </button>
      {data.src && (
        <div style={{ marginBottom: 12, padding: 8, border: "1px solid var(--ui-border)", background: "rgba(0,0,0,0.18)" }}>
          <img src={data.src} alt={data.alt || "Preview"} style={{ width: "100%", maxHeight: 120, objectFit: "cover", filter: imageThermalFilter(data, true) }} />
        </div>
      )}
      <Field label="Caption"><TextInput value={data.caption || ""} onChange={(e) => set({ ...data, caption: e.target.value })} /></Field>
      <Field label="Alt Text"><TextInput value={data.alt || ""} onChange={(e) => set({ ...data, alt: e.target.value })} /></Field>
      <Field label={`Height (${data.height || 160}px)`}>
        <input type="range" min="48" max="420" step="4" value={data.height || 160} onChange={(e) => set({ ...data, height: +e.target.value })} style={{ width: "100%" }} />
      </Field>
      <Field label="Fit">
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 6 }}>
          {["cover", "contain"].map((fit) => (
            <button key={fit} onClick={() => set({ ...data, fit })} className={data.fit === fit ? "seg active" : "seg"}>{fit}</button>
          ))}
        </div>
      </Field>
      <label style={{ display: "flex", alignItems: "center", gap: 7, fontSize: 11.5, color: "var(--ui-muted)", marginBottom: 8 }}>
        <input type="checkbox" checked={data.border !== false} onChange={(e) => set({ ...data, border: e.target.checked })} />
        Draw border around image
      </label>
      <label style={{ display: "flex", alignItems: "center", gap: 7, fontSize: 11.5, color: "var(--ui-muted)", marginBottom: 8 }}>
        <input type="checkbox" checked={data.grayscale !== false} onChange={(e) => set({ ...data, grayscale: e.target.checked })} />
        Thermal grayscale preview
      </label>
      <Field label="Thermal image tone">
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 6 }}>
          {[
            ["normal", "Normal"],
            ["light", "Light image"],
          ].map(([value, label]) => (
            <button key={value} onClick={() => set({ ...data, thermalTone: value })} className={(data.thermalTone || "normal") === value ? "seg active" : "seg"}>{label}</button>
          ))}
        </div>
      </Field>
      <div style={{ fontSize: 10.5, color: "var(--ui-muted)", lineHeight: 1.4, marginTop: 8 }}>
        Uploaded images are embedded as data URLs in saved layouts, so CLI/headless renders work without fetching external files.
      </div>
    </>
  );
};

const HabitsEditor = ({ data, set }) => (
  <>
    <Field label="Section Label"><TextInput value={data.label} onChange={(e) => set({ ...data, label: e.target.value })} /></Field>
    <Field label="Date Range"><TextInput value={data.range} onChange={(e) => set({ ...data, range: e.target.value })} /></Field>
    <ListEditor
      label="Habits"
      items={data.items}
      onChange={(items) => set({ ...data, items })}
      makeNew={() => ({ name: "New habit", days: [0, 0, 0, 0, 0, 0, 0] })}
      renderItem={(it, setIt) => (
        <>
          <TextInput style={{ marginBottom: 6 }} value={it.name} onChange={(e) => setIt({ ...it, name: e.target.value })} />
          <div style={{ display: "flex", gap: 4, justifyContent: "space-between" }}>
            {it.days.map((d, i) => (
              <button
                key={i}
                onClick={() => {
                  const copy = [...it.days]; copy[i] = d ? 0 : 1; setIt({ ...it, days: copy });
                }}
                style={{
                  width: 26, height: 26, padding: 0,
                  border: "1px solid var(--ui-border)",
                  background: d ? "var(--ui-accent)" : "transparent",
                  color: d ? "#1a1612" : "var(--ui-muted)",
                  cursor: "pointer", fontSize: 10, borderRadius: 2,
                  fontWeight: 600,
                }}
              >
                {data.columns[i]}
              </button>
            ))}
          </div>
        </>
      )}
    />
    <div style={{ height: 12 }} />
    <Field label="Reflection"><TextArea value={data.reflection} onChange={(e) => set({ ...data, reflection: e.target.value })} /></Field>
  </>
);

const QuoteEditor = ({ data, set }) => (
  <>
    <Field label="Section Label"><TextInput value={data.label || ""} onChange={(e) => set({ ...data, label: e.target.value })} /></Field>
    <Field label="Quote"><TextArea value={data.text} onChange={(e) => set({ ...data, text: e.target.value })} /></Field>
    <Field label="Author"><TextInput value={data.author} onChange={(e) => set({ ...data, author: e.target.value })} /></Field>
  </>
);

const WordEditor = ({ data, set }) => (
  <>
    <Field label="Section Label"><TextInput value={data.label || ""} onChange={(e) => set({ ...data, label: e.target.value })} /></Field>
    <Field label="Word"><TextInput value={data.word} onChange={(e) => set({ ...data, word: e.target.value })} /></Field>
    <div style={{ display: "grid", gridTemplateColumns: "1fr 2fr", gap: 8 }}>
      <Field label="Part"><TextInput value={data.part} onChange={(e) => set({ ...data, part: e.target.value })} /></Field>
      <Field label="Phonetic"><TextInput value={data.phonetic} onChange={(e) => set({ ...data, phonetic: e.target.value })} /></Field>
    </div>
    <Field label="Definition"><TextArea value={data.definition} onChange={(e) => set({ ...data, definition: e.target.value })} /></Field>
    <Field label="Example"><TextArea value={data.example} onChange={(e) => set({ ...data, example: e.target.value })} /></Field>
  </>
);

const HistoryEditor = ({ data, set }) => (
  <>
    <Field label="Section Label"><TextInput value={data.label} onChange={(e) => set({ ...data, label: e.target.value })} /></Field>
    <ListEditor
      label="Events"
      items={data.items}
      onChange={(items) => set({ ...data, items })}
      makeNew={() => ({ year: "2024", event: "An event happened." })}
      renderItem={(it, setIt) => (
        <>
          <TextInput style={{ marginBottom: 6 }} value={it.year} onChange={(e) => setIt({ ...it, year: e.target.value })} placeholder="Year" />
          <TextArea value={it.event} onChange={(e) => setIt({ ...it, event: e.target.value })} placeholder="Event" />
        </>
      )}
    />
  </>
);

const DidEditor = ({ data, set }) => (
  <>
    <Field label="Section Label"><TextInput value={data.label} onChange={(e) => set({ ...data, label: e.target.value })} /></Field>
    <ListEditor
      label="Facts"
      items={data.items}
      onChange={(items) => set({ ...data, items })}
      makeNew={() => "A fascinating fact."}
      renderItem={(it, setIt) => (
        <TextArea value={it} onChange={(e) => setIt(e.target.value)} />
      )}
    />
  </>
);

const MoodEditor = ({ data, set }) => (
  <>
    <Field label="Section Label"><TextInput value={data.label} onChange={(e) => set({ ...data, label: e.target.value })} /></Field>
    <Field label={`Mood (${data.mood}/5)`}>
      <input type="range" min="1" max="5" value={data.mood} onChange={(e) => set({ ...data, mood: +e.target.value })} style={{ width: "100%" }} />
    </Field>
    <Field label={`Energy (${data.energy}/5)`}>
      <input type="range" min="1" max="5" value={data.energy} onChange={(e) => set({ ...data, energy: +e.target.value })} style={{ width: "100%" }} />
    </Field>
    <Field label="Sleep"><TextInput value={data.sleep} onChange={(e) => set({ ...data, sleep: e.target.value })} /></Field>
    <Field label="Notes"><TextArea value={data.notes} onChange={(e) => set({ ...data, notes: e.target.value })} /></Field>
  </>
);

const ReadingEditor = ({ data, set }) => (
  <>
    <Field label="Section Label"><TextInput value={data.label} onChange={(e) => set({ ...data, label: e.target.value })} /></Field>
    <div style={{ padding: 10, background: "rgba(0,0,0,0.18)", border: "1px solid var(--ui-border)", borderRadius: 2, marginBottom: 10 }}>
      <div style={{ fontSize: 10, letterSpacing: "0.14em", textTransform: "uppercase", color: "var(--ui-muted)", marginBottom: 6, fontWeight: 600 }}>
        Currently Reading
      </div>
      <TextInput style={{ marginBottom: 6 }} value={data.current.title} onChange={(e) => set({ ...data, current: { ...data.current, title: e.target.value } })} placeholder="Title" />
      <TextInput style={{ marginBottom: 6 }} value={data.current.author} onChange={(e) => set({ ...data, current: { ...data.current, author: e.target.value } })} placeholder="Author" />
      <input type="range" min="0" max="100" value={data.current.progress} onChange={(e) => set({ ...data, current: { ...data.current, progress: +e.target.value } })} style={{ width: "100%" }} />
      <div style={{ fontSize: 10, color: "var(--ui-muted)", textAlign: "right" }}>{data.current.progress}%</div>
    </div>
    <ListEditor
      label="Want to Read"
      items={data.next}
      onChange={(next) => set({ ...data, next })}
      makeNew={() => "New book — Author"}
      renderItem={(it, setIt) => <TextInput value={it} onChange={(e) => setIt(e.target.value)} />}
    />
  </>
);

const ReflectionEditor = ({ data, set }) => (
  <>
    <Field label="Section Label"><TextInput value={data.label} onChange={(e) => set({ ...data, label: e.target.value })} /></Field>
    <Field label="What went well?"><TextArea value={data.well} onChange={(e) => set({ ...data, well: e.target.value })} /></Field>
    <Field label="What could be better?"><TextArea value={data.better} onChange={(e) => set({ ...data, better: e.target.value })} /></Field>
    <Field label="What did I learn?"><TextArea value={data.learned} onChange={(e) => set({ ...data, learned: e.target.value })} /></Field>
    <Field label="Closing Quote"><TextArea value={data.quote} onChange={(e) => set({ ...data, quote: e.target.value })} /></Field>
  </>
);

const DividerEditor = ({ data, set }) => (
  <Field label="Style">
    <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 6 }}>
      {["line", "dots", "wave", "leaves"].map((s) => (
        <button
          key={s}
          onClick={() => set({ ...data, style: s })}
          className={data.style === s ? "seg active" : "seg"}
        >
          {s}
        </button>
      ))}
    </div>
  </Field>
);

const EDITORS = {
  title: TitleEditor,
  date: DateEditor,
  plan: PlanEditor,
  news: NewsEditor,
  weather: WeatherEditor,
  note: NoteEditor,
  image: ImageEditor,
  habits: HabitsEditor,
  quote: QuoteEditor,
  word: WordEditor,
  history: HistoryEditor,
  did: DidEditor,
  mood: MoodEditor,
  reading: ReadingEditor,
  reflection: ReflectionEditor,
  divider: DividerEditor,
};

// =================================================================
// THERMAL PAPER COMPONENT
// =================================================================

const ZigzagEdge = ({ color, flip }) => (
  <svg
    width="100%" height="10" viewBox="0 0 200 10"
    preserveAspectRatio="none"
    style={{ display: "block", transform: flip ? "scaleY(-1)" : "none" }}
  >
    <path
      d="M0,0 L0,5 L5,10 L10,5 L15,10 L20,5 L25,10 L30,5 L35,10 L40,5 L45,10 L50,5 L55,10 L60,5 L65,10 L70,5 L75,10 L80,5 L85,10 L90,5 L95,10 L100,5 L105,10 L110,5 L115,10 L120,5 L125,10 L130,5 L135,10 L140,5 L145,10 L150,5 L155,10 L160,5 L165,10 L170,5 L175,10 L180,5 L185,10 L190,5 L195,10 L200,5 L200,0 Z"
      fill={color}
    />
  </svg>
);

const ThermalPaper = React.forwardRef(({ blocks, theme, selectedId, onSelect, onMove, onDelete, paperWidth }, ref) => {
  const ornateBorder = theme.ornateFrame ? `1.5px double ${theme.accent}` : "none";

  return (
    <div ref={ref} className="paper-shell" style={{ width: paperWidth }}>
      <ZigzagEdge color={theme.paper} flip />
      <div
        className="paper-body"
        style={{
          background: theme.paper,
          color: theme.ink,
          fontFamily: theme.fontBody,
          padding: theme.ornateFrame ? "20px 22px" : (theme.lined ? "16px 24px" : "20px 22px"),
          position: "relative",
          backgroundImage: theme.lined
            ? `repeating-linear-gradient(to bottom, transparent 0, transparent 21px, ${theme.rule} 21px, ${theme.rule} 22px)`
            : "none",
          backgroundSize: "auto",
        }}
      >
        {/* paper grain */}
        <div
          aria-hidden
          style={{
            position: "absolute", inset: 0, pointerEvents: "none",
            opacity: theme.grain,
            mixBlendMode: "multiply",
            backgroundImage: `url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='200' height='200'><filter id='n'><feTurbulence type='fractalNoise' baseFrequency='1.6' numOctaves='3' stitchTiles='stitch'/><feColorMatrix values='0 0 0 0 0  0 0 0 0 0  0 0 0 0 0  0 0 0 0.5 0'/></filter><rect width='100%' height='100%' filter='url(%23n)'/></svg>")`,
          }}
        />

        {/* ornate frame */}
        {theme.ornateFrame && (
          <div
            aria-hidden
            style={{
              position: "absolute", inset: 8, pointerEvents: "none",
              border: `1px solid ${theme.accent}`,
              outline: `1px solid ${theme.accent}`,
              outlineOffset: 2,
            }}
          />
        )}

        {/* botanical corner decorations */}
        {theme.botanical && (
          <>
            <div aria-hidden style={{ position: "absolute", top: 6, left: 8, color: theme.accent, fontSize: 18 }}>❦</div>
            <div aria-hidden style={{ position: "absolute", top: 6, right: 8, color: theme.accent, fontSize: 18, transform: "scaleX(-1)" }}>❦</div>
            <div aria-hidden style={{ position: "absolute", bottom: 14, left: 8, color: theme.accent, fontSize: 18, transform: "scaleY(-1)" }}>❦</div>
            <div aria-hidden style={{ position: "absolute", bottom: 14, right: 8, color: theme.accent, fontSize: 18, transform: "scale(-1, -1)" }}>❦</div>
          </>
        )}

        {/* space age stars */}
        {theme.space && (
          <div aria-hidden style={{ position: "absolute", top: 4, left: 0, right: 0, textAlign: "center", color: theme.accent, fontSize: 8, letterSpacing: "1em" }}>
            ✦ · · ✦ · · ✦
          </div>
        )}

        <div style={{ position: "relative", display: "flex", flexDirection: "column", gap: theme.ornateFrame ? 12 : 14 }}>
          {blocks.map((b, i) => {
            const isSelected = b.id === selectedId;
            const isBoxed = theme.boxed && b.type !== "title" && b.type !== "date" && b.type !== "divider";

            return (
              <div
                key={b.id}
                className={`block-wrap ${isSelected ? "selected" : ""}`}
                onClick={(e) => { e.stopPropagation(); onSelect(b.id); }}
              >
                <div style={{
                  padding: isBoxed ? "10px 12px" : 0,
                  border: isBoxed ? `1px solid ${theme.rule}` : "none",
                }}>
                  {renderBlock(b, { theme, registry: BLOCK_REGISTRY })}
                </div>
                <div className="block-controls" onClick={(e) => e.stopPropagation()}>
                  <button onClick={() => onMove(b.id, -1)} disabled={i === 0} title="Move up"><ChevronUp size={11} /></button>
                  <button onClick={() => onMove(b.id, 1)} disabled={i === blocks.length - 1} title="Move down"><ChevronDown size={11} /></button>
                  <button onClick={() => onDelete(b.id)} title="Delete" className="danger"><X size={11} /></button>
                </div>
              </div>
            );
          })}
          {blocks.length === 0 && (
            <div style={{ padding: 40, textAlign: "center", fontFamily: theme.fontBody, color: theme.muted, fontStyle: "italic" }}>
              Add blocks from the left panel to start your almanac.
            </div>
          )}
        </div>
      </div>
      <ZigzagEdge color={theme.paper} />
    </div>
  );
});

// =================================================================
// EXPORT HELPERS — PNG (via SVG foreignObject) + JSON
// =================================================================

const FONT_CSS_URL =
  "https://fonts.googleapis.com/css2?family=Cormorant+Garamond:wght@400;500;600;700&family=EB+Garamond:ital,wght@0,400;0,500;0,600;1,400;1,500&family=Caveat:wght@400;500;700&family=Kalam:wght@300;400;700&family=DM+Sans:wght@400;500;600;700&display=swap";

let _fontCssCache = null;

async function getInlineFontCss() {
  if (_fontCssCache !== null) return _fontCssCache;
  try {
    const cssRes = await fetch(FONT_CSS_URL);
    let css = await cssRes.text();
    const urls = Array.from(
      css.matchAll(/url\((https:\/\/fonts\.gstatic\.com\/[^)]+)\)/g),
      (m) => m[1]
    );
    const dataUrls = await Promise.all(
      urls.map(async (u) => {
        const r = await fetch(u);
        const buf = await r.arrayBuffer();
        const arr = new Uint8Array(buf);
        let bin = "";
        const CHUNK = 0x8000;
        for (let i = 0; i < arr.length; i += CHUNK) {
          bin += String.fromCharCode.apply(null, arr.subarray(i, i + CHUNK));
        }
        return `data:font/woff2;base64,${btoa(bin)}`;
      })
    );
    urls.forEach((u, i) => {
      css = css.split(u).join(dataUrls[i]);
    });
    _fontCssCache = css;
    return css;
  } catch (e) {
    console.warn("Font inlining failed, falling back to system fonts.", e);
    _fontCssCache = "";
    return "";
  }
}

async function waitForImages(root = document) {
  const images = Array.from(root.querySelectorAll("img"));
  await Promise.all(images.map((img) => {
    if (img.complete) return Promise.resolve();
    return new Promise((resolve) => {
      img.addEventListener("load", resolve, { once: true });
      img.addEventListener("error", resolve, { once: true });
      setTimeout(resolve, 10000);
    });
  }));
}

function triggerDownload(blob, filename) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  setTimeout(() => {
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }, 200);
}

async function exportPaperToPng(paperNode, fileName, scale = 2, themeObj) {
  if (!paperNode) throw new Error("No paper element");
  if (document.fonts && document.fonts.ready) await document.fonts.ready;
  await waitForImages(paperNode);

  const fontCss = await getInlineFontCss();

  const appStyleSheets = Array.from(document.querySelectorAll("style"))
    .map((s) => s.textContent)
    .join("\n");

  const rect = paperNode.getBoundingClientRect();
  const w = Math.ceil(rect.width);
  const h = Math.ceil(rect.height);

  // Deep clone the paper, strip interactive chrome and zigzag edges, ensure SVG namespaces
  const clone = paperNode.cloneNode(true);
  clone.setAttribute("data-export", "true");
  clone.querySelectorAll(".block-controls").forEach((el) => el.remove());
  // Remove zigzag edge SVGs — they create transparent gaps that show as black when printed
  clone.querySelectorAll("svg").forEach((svg) => {
    const parent = svg.parentElement;
    if (parent && parent.style && parent.style.display !== "none") {
      // Keep SVGs inside the paper body (weather icon, mood faces, etc.)
      // but remove the zigzag edge SVGs (they're direct children of paper-shell)
      const path = svg.querySelector("path");
      if (path && path.getAttribute("fill") === (themeObj ? themeObj.paper : "#ffffff")) {
        svg.remove();
        return;
      }
    }
    if (!svg.getAttribute("xmlns")) svg.setAttribute("xmlns", "http://www.w3.org/2000/svg");
  });

  // Serialize as XHTML for safe inclusion inside foreignObject
  const xhtml = new XMLSerializer().serializeToString(clone);

  const exportOverrides = `
    [data-export] { filter: none !important; margin: 0 !important; }
    [data-export] .block-controls { display: none !important; }
    [data-export] .block-wrap::before { display: none !important; }
    [data-export] .block-wrap { padding: 4px 0 !important; cursor: default !important; }
    [data-export] * { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
  `;

  const svgString =
    `<svg xmlns="http://www.w3.org/2000/svg" width="${w}" height="${h}" viewBox="0 0 ${w} ${h}">` +
    `<defs><style type="text/css"><![CDATA[${fontCss}\n${appStyleSheets}\n${exportOverrides}]]></style></defs>` +
    `<foreignObject x="0" y="0" width="${w}" height="${h}">${xhtml}</foreignObject>` +
    `</svg>`;

  const svgBlob = new Blob([svgString], { type: "image/svg+xml;charset=utf-8" });
  const svgUrl = URL.createObjectURL(svgBlob);

  const img = new Image();
  img.crossOrigin = "anonymous";
  await new Promise((resolve, reject) => {
    img.onload = () => resolve();
    img.onerror = (e) => reject(new Error("SVG-to-image rasterization failed."));
    img.src = svgUrl;
  });

  const canvas = document.createElement("canvas");
  canvas.width = w * scale;
  canvas.height = h * scale;
  const ctx = canvas.getContext("2d");
  // Fill with white background first — no transparency for thermal printing
  ctx.fillStyle = "#ffffff";
  ctx.fillRect(0, 0, w * scale, h * scale);
  ctx.imageSmoothingEnabled = true;
  ctx.imageSmoothingQuality = "high";
  ctx.scale(scale, scale);
  ctx.drawImage(img, 0, 0);
  URL.revokeObjectURL(svgUrl);

  const pngBlob = await new Promise((resolve) =>
    canvas.toBlob(resolve, "image/png")
  );
  if (!pngBlob) throw new Error("PNG encoding failed.");
  triggerDownload(pngBlob, fileName);
}

function buildLayoutJson({ themeKey, paperWidth, bodyScale, feedLines, blocks, typography }) {
  const payload = {
    almanach_studio_version: 1,
    exported_at: new Date().toISOString(),
    theme: themeKey,
    paperWidth,
    bodyScale,
    feedLines,
    blocks: blocks.map((b) => (
      b.style ? { id: b.id, type: b.type, data: b.data, style: b.style } : { id: b.id, type: b.type, data: b.data }
    )),
  };
  if (typography && Object.keys(typography).length > 0) {
    payload.typography = { presets: typography };
  }
  return payload;
}

function exportLayoutJson(state, fileName) {
  const payload = buildLayoutJson(state);
  const json = JSON.stringify(payload, null, 2);
  const blob = new Blob([json], { type: "application/json" });
  triggerDownload(blob, fileName);
}

function parseLayoutJson(text) {
  const parsed = JSON.parse(text);
  if (!parsed || !Array.isArray(parsed.blocks)) {
    throw new Error("File is not a valid Almanach Studio layout.");
  }
  // Normalize blocks. Unknown types are kept (not dropped) so the renderer can
  // show a visible placeholder; only their defaults are unavailable.
  const blocks = parsed.blocks
    .filter((b) => b && typeof b.type === "string")
    .map((b) => {
      const defaults = DEFAULTS[b.type]; // undefined for unregistered types
      const data = b.data && typeof b.data === "object"
        ? { ...(defaults || {}), ...b.data }
        : defaults
          ? JSON.parse(JSON.stringify(defaults))
          : {};
      const block = { id: b.id || uid(), type: b.type, data };
      // Per-block typography override (proto Block.style) — applied to the
      // block's primary text preset at render time.
      if (b.style && typeof b.style === "object") block.style = b.style;
      return block;
    });

  // Layout-level typography preset overrides (proto Layout.typography.presets).
  const typography =
    parsed.typography && typeof parsed.typography.presets === "object"
      ? parsed.typography.presets
      : {};

  const { themeKey, patch: themePatch, presetOverrides: themePresets } = resolveThemeSpec(parsed.theme);
  const paperWidth =
    typeof parsed.paperWidth === "number" &&
    parsed.paperWidth >= 280 &&
    parsed.paperWidth <= 600
      ? parsed.paperWidth
      : 384;

  const bodyScale =
    typeof parsed.bodyScale === "number" &&
    parsed.bodyScale >= 1 &&
    parsed.bodyScale <= 2
      ? parsed.bodyScale
      : 1.6;

  const feedLines =
    typeof parsed.feedLines === "number" &&
    parsed.feedLines >= 0 &&
    parsed.feedLines <= 20
      ? parsed.feedLines
      : 3;

  return { blocks, themeKey, themePatch, themePresets, paperWidth, bodyScale, feedLines, typography };
}

// =================================================================
// MAIN APP
// =================================================================

export default function AlmanachStudio() {
  const [blocks, setBlocks] = useState(STARTER_BLOCKS);
  const [selectedId, setSelectedId] = useState(STARTER_BLOCKS[0].id);
  const [paperWidth, setPaperWidth] = useState(() => {
    try { const v = JSON.parse(localStorage.getItem("almanach_paperWidth")); return (typeof v === "number" && v >= 280 && v <= 600) ? v : 384; } catch { return 384; }
  });
  const [bodyScale, setBodyScale] = useState(() => {
    try { const v = JSON.parse(localStorage.getItem("almanach_bodyScale")); return (typeof v === "number" && v >= 1 && v <= 2) ? v : 1.6; } catch { return 1.6; }
  });
  const [feedLines, setFeedLines] = useState(() => {
    try { const v = JSON.parse(localStorage.getItem("almanach_feedLines")); return (typeof v === "number" && v >= 0 && v <= 20) ? v : 3; } catch { return 3; }
  });
  const [themeKey, setThemeKey] = useState(() => {
    try { const v = localStorage.getItem("almanach_themeKey"); return (v && THEMES[v]) ? v : "classic"; } catch { return "classic"; }
  });
  // Layout-level typography preset overrides (name -> partial TextStyle). Empty
  // means "use the built-in defaults". Set by loading a layout that carries a
  // `typography.presets` object.
  const [typography, setTypography] = useState({});
  // Inline theme patch (data-driven theme) merged over the built-in themeKey,
  // and theme-level preset overrides. Both come from a layout whose `theme` is
  // an object rather than a built-in name; null/{} for built-in themes.
  const [inlineTheme, setInlineTheme] = useState(null);
  const [themePresets, setThemePresets] = useState({});
  const [showLeft, setShowLeft] = useState(true);
  const [showRight, setShowRight] = useState(true);
  const [exporting, setExporting] = useState(false);
  const [toast, setToast] = useState(null); // { kind: "ok"|"err", text }
  const paperRef = useRef(null);
  const fileInputRef = useRef(null);

  const flashToast = useCallback((kind, text) => {
    setToast({ kind, text });
    setTimeout(() => setToast(null), 2600);
  }, []);

  // Persist settings to localStorage
  useEffect(() => { localStorage.setItem("almanach_themeKey", themeKey); }, [themeKey]);
  useEffect(() => { localStorage.setItem("almanach_paperWidth", JSON.stringify(paperWidth)); }, [paperWidth]);
  useEffect(() => { localStorage.setItem("almanach_bodyScale", JSON.stringify(bodyScale)); }, [bodyScale]);
  useEffect(() => { localStorage.setItem("almanach_feedLines", JSON.stringify(feedLines)); }, [feedLines]);

  // --- Headless API for programmatic control (Go render service) ---
  const stateRef = useRef({ blocks, setBlocks, setThemeKey, setPaperWidth, setBodyScale, setFeedLines, setSelectedId, setTypography, setInlineTheme, setThemePresets, flashToast });
  stateRef.current = { blocks, setBlocks, setThemeKey, setPaperWidth, setBodyScale, setFeedLines, setSelectedId, setTypography, setInlineTheme, setThemePresets, flashToast };

  useEffect(() => {
    window.almanachReady = true;

    window.almanachLoadLayout = (jsonString) => {
      try {
        const parsed = typeof jsonString === "string" ? JSON.parse(jsonString) : jsonString;
        const result = parseLayoutJson(JSON.stringify(parsed));
        const s = stateRef.current;
        s.setBlocks(result.blocks);
        s.setThemeKey(result.themeKey);
        s.setPaperWidth(result.paperWidth);
        s.setBodyScale(result.bodyScale);
        s.setFeedLines(result.feedLines);
        s.setTypography(result.typography || {});
        s.setInlineTheme(result.themePatch || null);
        s.setThemePresets(result.themePresets || {});
        s.setSelectedId(result.blocks[0]?.id || null);
      } catch (e) {
        console.error("almanachLoadLayout error:", e);
      }
    };

    window.almanachExportBitmap = async () => {
      // Wait for fonts, images, and one render frame
      if (document.fonts && document.fonts.ready) await document.fonts.ready;
      await waitForImages(document);
      await new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)));

      const paperNode = document.querySelector(".paper-shell");
      if (!paperNode) throw new Error("No .paper-shell element found");

      const rect = paperNode.getBoundingClientRect();
      const W = Math.ceil(Math.max(rect.width, paperNode.scrollWidth));
      const H = Math.ceil(Math.max(rect.height, paperNode.scrollHeight) + 8);

      const canvas = document.createElement("canvas");
      canvas.width = W;
      canvas.height = H;
      const ctx = canvas.getContext("2d");

      ctx.fillStyle = "#ffffff";
      ctx.fillRect(0, 0, W, H);

      // Deep clone and strip chrome
      const clone = paperNode.cloneNode(true);
      clone.setAttribute("data-export", "true");
      clone.querySelectorAll(".block-controls").forEach((el) => el.remove());
      clone.querySelectorAll("svg").forEach((svg) => {
        const path = svg.querySelector("path");
        const t = stateRef.current;
        const paperColor = THEMES[t.themeKey || 'classic']?.paper || '#ffffff';
        if (path && path.getAttribute("fill") === paperColor) { svg.remove(); return; }
        if (!svg.getAttribute("xmlns")) svg.setAttribute("xmlns", "http://www.w3.org/2000/svg");
      });

      const fontCss = await getInlineFontCss();
      const appStyleSheets = Array.from(document.querySelectorAll("style")).map((s) => s.textContent).join("\n");
      const exportOverrides = `
        [data-export] { filter: none !important; margin: 0 !important; }
        [data-export] .block-controls { display: none !important; }
        [data-export] .block-wrap::before { display: none !important; }
        [data-export] .block-wrap { padding: 4px 0 !important; cursor: default !important; }
        [data-export] * { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
      `;
      const xhtml = new XMLSerializer().serializeToString(clone);
      const svgString =
        `<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}">` +
        `<defs><style type="text/css"><![CDATA[${fontCss}\n${appStyleSheets}\n${exportOverrides}]]></style></defs>` +
        `<foreignObject x="0" y="0" width="${W}" height="${H}">${xhtml}</foreignObject>` +
        `</svg>`;

      const svgBlob = new Blob([svgString], { type: "image/svg+xml;charset=utf-8" });
      const svgUrl = URL.createObjectURL(svgBlob);
      const img = new Image();
      img.crossOrigin = "anonymous";
      await new Promise((resolve, reject) => {
        img.onload = () => resolve();
        img.onerror = () => reject(new Error("SVG rasterization failed"));
        img.src = svgUrl;
      });
      ctx.drawImage(img, 0, 0);
      URL.revokeObjectURL(svgUrl);

      // Binarize to 1-bit
      const imgData = ctx.getImageData(0, 0, W, H);
      const pixels = imgData.data;
      const Wpadded = Math.ceil(W / 8) * 8;
      const bytesPerRow = Wpadded / 8;
      const bitmap = new Uint8Array(bytesPerRow * H);
      for (let y = 0; y < H; y++) {
        for (let x = 0; x < W; x++) {
          const gray = 0.299 * pixels[(y * W + x) * 4] + 0.587 * pixels[(y * W + x) * 4 + 1] + 0.114 * pixels[(y * W + x) * 4 + 2];
          if (gray < 128) {
            bitmap[y * bytesPerRow + Math.floor(x / 8)] |= (0x80 >> (x % 8));
          }
        }
      }

      return { width: Wpadded, height: H, data: bitmap.buffer };
    };

    console.log("Almanach Studio headless API ready (window.almanachReady = true)");
  }, []);

  const fsRaw = (base) => Math.round(base * bodyScale * 10) / 10;
  // Built-in theme, patched by an inline (data-driven) theme when a layout
  // supplies one.
  const theme = { ...THEMES[themeKey], ...(inlineTheme || {}), bodyScale, fs: fsRaw };
  // Typography preset resolver bound to this theme. Preset overrides layer as
  // default <- theme.presetOverrides <- layout.typography <- block style; the
  // first two are merged here (layout wins per field). Blocks call
  // theme.preset("body"), etc. See web/src/typography/presets.js.
  theme.fontMono = theme.fontMono || "'JetBrains Mono', 'DejaVu Sans Mono', monospace";
  theme.preset = makePresetResolver({ presets: mergePresetMaps(themePresets, typography), theme, bodyScale });
  // Convenience alias used in block renderers
  const fs = fsRaw;
  const selected = blocks.find((b) => b.id === selectedId);

  const addBlock = useCallback((type) => {
    const b = newBlock(type);
    setBlocks((prev) => [...prev, b]);
    setSelectedId(b.id);
  }, []);

  const deleteBlock = useCallback((id) => {
    setBlocks((prev) => {
      const next = prev.filter((b) => b.id !== id);
      if (id === selectedId) setSelectedId(next[0]?.id || null);
      return next;
    });
  }, [selectedId]);

  const moveBlock = useCallback((id, dir) => {
    setBlocks((prev) => {
      const i = prev.findIndex((b) => b.id === id);
      if (i < 0) return prev;
      const j = i + dir;
      if (j < 0 || j >= prev.length) return prev;
      const copy = [...prev];
      [copy[i], copy[j]] = [copy[j], copy[i]];
      return copy;
    });
  }, []);

  const updateBlock = useCallback((id, data) => {
    setBlocks((prev) => prev.map((b) => b.id === id ? { ...b, data } : b));
  }, []);

  const duplicateBlock = useCallback((id) => {
    setBlocks((prev) => {
      const i = prev.findIndex((b) => b.id === id);
      if (i < 0) return prev;
      const copy = [...prev];
      const dup = { ...prev[i], id: uid(), data: JSON.parse(JSON.stringify(prev[i].data)) };
      copy.splice(i + 1, 0, dup);
      return copy;
    });
  }, []);

  const handlePrint = useCallback(async () => {
    if (!paperRef.current) return;
    setExporting(true);
    const prevSelection = selectedId;
    setSelectedId(null);
    await new Promise((r) => requestAnimationFrame(r));
    try {
      await waitForImages(paperRef.current);
      // Render paper to off-screen canvas at 1:1 pixel scale
      const paperNode = paperRef.current;
      const rect = paperNode.getBoundingClientRect();
      const W = Math.ceil(rect.width);
      // Use scrollHeight to capture full content including bottom padding
      const H = Math.ceil(Math.max(rect.height, paperNode.scrollHeight) + 8);

      const canvas = document.createElement("canvas");
      canvas.width = W;
      canvas.height = H;
      const ctx = canvas.getContext("2d");

      // White background
      ctx.fillStyle = "#ffffff";
      ctx.fillRect(0, 0, W, H);

      // Use SVG foreignObject to capture the DOM
      const clone = paperNode.cloneNode(true);
      clone.querySelectorAll(".block-controls").forEach((el) => el.remove());
      // Remove zigzag edge SVGs (same as PNG export)
      clone.querySelectorAll("svg").forEach((svg) => {
        const path = svg.querySelector("path");
        if (path && path.getAttribute("fill") === theme.paper) {
          svg.remove();
          return;
        }
        if (!svg.getAttribute("xmlns")) svg.setAttribute("xmlns", "http://www.w3.org/2000/svg");
      });
      clone.setAttribute("data-export", "true");

      const appStyleSheets = Array.from(document.querySelectorAll("style"))
        .map((s) => s.textContent).join("\n");
      const exportOverrides = `
        [data-export] { filter: none !important; margin: 0 !important; }
        [data-export] .block-controls { display: none !important; }
        [data-export] .block-wrap::before { display: none !important; }
        [data-export] .block-wrap { padding: 4px 0 !important; cursor: default !important; }
        [data-export] * { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
      `;
      const fontCss = await getInlineFontCss();
      const xhtml = new XMLSerializer().serializeToString(clone);
      const svgString =
        `<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}">` +
        `<defs><style type="text/css"><![CDATA[${fontCss}\n${appStyleSheets}\n${exportOverrides}]]></style></defs>` +
        `<foreignObject x="0" y="0" width="${W}" height="${H}">${xhtml}</foreignObject>` +
        `</svg>`;

      const svgBlob = new Blob([svgString], { type: "image/svg+xml;charset=utf-8" });
      const svgUrl = URL.createObjectURL(svgBlob);
      const img = new Image();
      img.crossOrigin = "anonymous";
      await new Promise((resolve, reject) => {
        img.onload = () => resolve();
        img.onerror = () => reject(new Error("SVG rasterization failed"));
        img.src = svgUrl;
      });
      ctx.drawImage(img, 0, 0);
      URL.revokeObjectURL(svgUrl);

      // Read pixels and binarize
      const imgData = ctx.getImageData(0, 0, W, H);
      const pixels = imgData.data;
      const bw = new Uint8Array(W * H); // 1 = black
      for (let i = 0; i < W * H; i++) {
        const gray = 0.299 * pixels[i * 4] + 0.587 * pixels[i * 4 + 1] + 0.114 * pixels[i * 4 + 2];
        bw[i] = gray < 128 ? 1 : 0;
      }

      // Pack into 1-bit bitmap (MSB first)
      // Width must be divisible by 8 for the printer — pad if needed
      const Wpadded = Math.ceil(W / 8) * 8;
      const bytesPerRow = Wpadded / 8;
      const bitmap = new Uint8Array(bytesPerRow * H);
      for (let y = 0; y < H; y++) {
        for (let x = 0; x < W; x++) {
          if (bw[y * W + x]) {
            bitmap[y * bytesPerRow + Math.floor(x / 8)] |= (0x80 >> (x % 8));
          }
        }
      }

      // POST to the printer API
      const r = await fetch("/api/print/bitmap", {
        method: "POST",
        headers: {
          "Content-Type": "application/octet-stream",
          "X-Width": String(Wpadded),
          "X-Height": String(H),
          "X-Feed": String(feedLines),
        },
        body: bitmap,
      });
      const d = await r.json();
      if (!d.ok) {
        flashToast("err", d.error || "Print failed");
      } else {
        flashToast("ok", "Sent to printer");
      }
    } catch (e) {
      console.error(e);
      flashToast("err", e.message || "Print failed");
    } finally {
      setSelectedId(prevSelection);
      setExporting(false);
    }
  }, [selectedId, theme, feedLines, flashToast]);

  const handleExportPng = useCallback(async () => {
    if (!paperRef.current) return;
    setExporting(true);
    // Briefly clear the selection so the export is clean
    const prevSelection = selectedId;
    setSelectedId(null);
    // Wait one frame so React commits the deselect
    await new Promise((r) => requestAnimationFrame(r));
    try {
      const dateStamp = new Date().toISOString().slice(0, 10);
      await exportPaperToPng(
        paperRef.current,
        `almanach-${themeKey}-${dateStamp}.png`,
        2,
        theme
      );
      flashToast("ok", "PNG saved");
    } catch (e) {
      console.error(e);
      flashToast("err", "PNG export failed — try again");
    } finally {
      setSelectedId(prevSelection);
      setExporting(false);
    }
  }, [selectedId, themeKey, flashToast]);

  const handleExportJson = useCallback(() => {
    try {
      const dateStamp = new Date().toISOString().slice(0, 10);
      exportLayoutJson(
        { themeKey, paperWidth, bodyScale, feedLines, blocks, typography },
        `almanach-${dateStamp}.json`
      );
      flashToast("ok", "Layout saved");
    } catch (e) {
      console.error(e);
      flashToast("err", "Save failed");
    }
  }, [themeKey, paperWidth, bodyScale, feedLines, blocks, typography, flashToast]);

  const handleImportClick = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  const handleImportFile = useCallback(
    async (e) => {
      const file = e.target.files?.[0];
      e.target.value = ""; // allow re-selecting the same file later
      if (!file) return;
      try {
        const text = await file.text();
        const parsed = parseLayoutJson(text);
        setBlocks(parsed.blocks);
        setThemeKey(parsed.themeKey);
        setPaperWidth(parsed.paperWidth);
        setBodyScale(parsed.bodyScale);
        setFeedLines(parsed.feedLines);
        setTypography(parsed.typography || {});
        setInlineTheme(parsed.themePatch || null);
        setThemePresets(parsed.themePresets || {});
        setSelectedId(parsed.blocks[0]?.id || null);
        flashToast("ok", `Loaded ${parsed.blocks.length} blocks`);
      } catch (err) {
        console.error(err);
        flashToast("err", err.message || "Could not load file");
      }
    },
    [flashToast]
  );

  const Editor = selected ? EDITORS[selected.type] : null;

  // Group block types
  const groupedTypes = useMemo(() => {
    const out = {};
    BLOCK_TYPES.forEach((t) => {
      if (!out[t.group]) out[t.group] = [];
      out[t.group].push(t);
    });
    return out;
  }, []);

  return (
    <div className="almanach-app">
      <style>{`
        /* Google Fonts — requires internet access.
         * When served from AtomS3R in SoftAP mode (no internet),
         * the system font fallbacks in each theme will be used instead.
         * Uncomment the next line to restore Google Fonts when internet is available:
         *
         * @import url('https://fonts.googleapis.com/css2?family=Cormorant+Garamond:wght@400;500;600;700&family=EB+Garamond:ital,wght@0,400;0,500;0,600;1,400;1,500&family=Caveat:wght@400;500;700&family=Kalam:wght@300;400;700&family=DM+Sans:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap');
         */

        :root {
          --ui-bg: #1a1612;
          --ui-bg-2: #221d18;
          --ui-bg-3: #2a241e;
          --ui-border: #3a3128;
          --ui-border-2: #4a3f33;
          --ui-text: #e8dcc4;
          --ui-text-2: #c9b896;
          --ui-muted: #8a7c66;
          --ui-accent: #c9a36b;
          --ui-accent-2: #d8b67e;
          --ui-danger: #c97766;
        }

        * { box-sizing: border-box; }

        .almanach-app {
          font-family: 'DM Sans', -apple-system, sans-serif;
          background: var(--ui-bg);
          color: var(--ui-text);
          height: 100vh;
          width: 100%;
          display: flex;
          flex-direction: column;
          overflow: hidden;
          position: relative;
        }

        .almanach-app::before {
          content: '';
          position: absolute;
          inset: 0;
          background:
            radial-gradient(ellipse at 20% 0%, rgba(201, 163, 107, 0.08) 0%, transparent 50%),
            radial-gradient(ellipse at 80% 100%, rgba(155, 110, 70, 0.06) 0%, transparent 50%);
          pointer-events: none;
          z-index: 0;
        }

        /* ----- TOPBAR ----- */
        .topbar {
          height: 56px;
          background: var(--ui-bg-2);
          border-bottom: 1px solid var(--ui-border);
          display: flex;
          align-items: center;
          padding: 0 16px;
          gap: 16px;
          flex-shrink: 0;
          position: relative;
          z-index: 10;
        }
        .brand {
          display: flex;
          align-items: baseline;
          gap: 10px;
        }
        .brand-mark {
          font-family: 'Cormorant Garamond', serif;
          font-size: 22px;
          font-weight: 700;
          color: var(--ui-accent);
          letter-spacing: 0.18em;
          line-height: 1;
        }
        .brand-sub {
          font-family: 'EB Garamond', serif;
          font-style: italic;
          font-size: 12px;
          color: var(--ui-muted);
          letter-spacing: 0.08em;
        }
        .topbar-spacer { flex: 1; }
        .topbar-actions {
          display: flex;
          gap: 8px;
          align-items: center;
        }
        .btn {
          display: inline-flex;
          align-items: center;
          gap: 6px;
          padding: 7px 12px;
          background: transparent;
          border: 1px solid var(--ui-border-2);
          color: var(--ui-text);
          font-family: inherit;
          font-size: 11.5px;
          font-weight: 500;
          letter-spacing: 0.08em;
          text-transform: uppercase;
          cursor: pointer;
          border-radius: 2px;
          transition: all 0.15s;
        }
        .btn:hover {
          background: var(--ui-bg-3);
          border-color: var(--ui-accent);
          color: var(--ui-accent-2);
        }
        .btn.primary {
          background: var(--ui-accent);
          border-color: var(--ui-accent);
          color: #1a1612;
        }
        .btn.primary:hover {
          background: var(--ui-accent-2);
          border-color: var(--ui-accent-2);
          color: #1a1612;
        }
        .btn.ghost {
          border-color: transparent;
          color: var(--ui-muted);
        }
        .btn.ghost:hover {
          background: var(--ui-bg-3);
          border-color: var(--ui-border-2);
          color: var(--ui-text);
        }
        .btn:disabled {
          opacity: 0.55;
          cursor: not-allowed;
        }
        .btn:disabled:hover {
          background: transparent;
          border-color: var(--ui-border-2);
          color: var(--ui-text);
        }
        .topbar-divider {
          display: inline-block;
          width: 1px;
          height: 22px;
          background: var(--ui-border);
          margin: 0 4px;
        }
        .spinner {
          width: 12px; height: 12px;
          border: 1.5px solid currentColor;
          border-top-color: transparent;
          border-radius: 50%;
          animation: spin 0.8s linear infinite;
          display: inline-block;
        }
        @keyframes spin { to { transform: rotate(360deg); } }

        /* Toast */
        .toast {
          position: fixed;
          top: 70px;
          left: 50%;
          transform: translateX(-50%);
          padding: 9px 16px;
          background: var(--ui-bg-2);
          border: 1px solid var(--ui-border-2);
          border-radius: 2px;
          font-size: 12px;
          letter-spacing: 0.06em;
          z-index: 100;
          box-shadow: 0 8px 24px rgba(0,0,0,0.4);
          animation: toast-in 0.25s ease-out;
        }
        .toast.ok {
          border-color: var(--ui-accent);
          color: var(--ui-accent-2);
        }
        .toast.err {
          border-color: var(--ui-danger);
          color: var(--ui-danger);
        }
        @keyframes toast-in {
          from { opacity: 0; transform: translate(-50%, -8px); }
          to { opacity: 1; transform: translate(-50%, 0); }
        }

        /* ----- WORKSPACE ----- */
        .workspace {
          flex: 1;
          display: grid;
          overflow: hidden;
          position: relative;
          z-index: 1;
        }

        /* ----- RAILS ----- */
        .rail {
          background: var(--ui-bg-2);
          border: 0 solid var(--ui-border);
          overflow-y: auto;
          overflow-x: hidden;
          transition: width 0.2s;
        }
        .rail.left { border-right-width: 1px; }
        .rail.right { border-left-width: 1px; }
        .rail-section {
          padding: 14px 14px 8px;
          border-bottom: 1px solid var(--ui-border);
        }
        .rail-section:last-child { border-bottom: none; }
        .rail-title {
          font-size: 10px;
          letter-spacing: 0.18em;
          text-transform: uppercase;
          color: var(--ui-muted);
          font-weight: 700;
          margin-bottom: 8px;
          display: flex;
          align-items: center;
          gap: 6px;
        }
        .rail-title-dot {
          width: 5px; height: 5px; border-radius: 50%;
        }

        /* ----- BLOCK LIBRARY ITEM ----- */
        .lib-item {
          display: flex;
          align-items: center;
          gap: 9px;
          width: 100%;
          padding: 8px 10px;
          background: rgba(0,0,0,0.18);
          border: 1px solid var(--ui-border);
          color: var(--ui-text);
          font-family: inherit;
          font-size: 12px;
          cursor: pointer;
          margin-bottom: 5px;
          border-radius: 2px;
          text-align: left;
          transition: all 0.12s;
        }
        .lib-item:hover {
          background: var(--ui-bg-3);
          border-color: var(--ui-accent);
          color: var(--ui-accent-2);
          transform: translateX(2px);
        }
        .lib-item .icon {
          flex-shrink: 0;
          color: var(--ui-muted);
        }
        .lib-item:hover .icon { color: var(--ui-accent); }

        /* ----- THEME GRID ----- */
        .theme-grid {
          display: grid;
          grid-template-columns: 1fr 1fr;
          gap: 6px;
        }
        .theme-card {
          padding: 10px 8px;
          background: rgba(0,0,0,0.2);
          border: 1px solid var(--ui-border);
          color: var(--ui-text);
          font-family: inherit;
          cursor: pointer;
          border-radius: 2px;
          text-align: center;
          transition: all 0.12s;
        }
        .theme-card:hover {
          border-color: var(--ui-accent);
        }
        .theme-card.active {
          background: var(--ui-bg-3);
          border-color: var(--ui-accent);
          box-shadow: inset 0 0 0 1px var(--ui-accent);
        }
        .theme-card .swatch {
          width: 100%;
          height: 22px;
          margin-bottom: 6px;
          border: 1px solid rgba(255,255,255,0.1);
          font-family: 'Cormorant Garamond', serif;
          display: flex;
          align-items: center;
          justify-content: center;
          font-size: 14px;
        }
        .theme-card .name {
          font-size: 10.5px;
          letter-spacing: 0.05em;
          font-weight: 500;
        }
        .theme-card.active .name { color: var(--ui-accent-2); }

        /* ----- CANVAS ----- */
        .canvas {
          background:
            linear-gradient(135deg, #16120e 0%, #1c1814 100%);
          background-image:
            radial-gradient(circle at 1px 1px, rgba(201, 163, 107, 0.04) 1px, transparent 0);
          background-size: 24px 24px;
          overflow: auto;
          display: flex;
          align-items: flex-start;
          justify-content: center;
          padding: 40px 24px 80px;
        }

        /* ----- THERMAL PAPER ----- */
        .paper-shell {
          flex-shrink: 0;
          filter: drop-shadow(0 25px 35px rgba(0,0,0,0.55)) drop-shadow(0 6px 10px rgba(0,0,0,0.35));
          margin: 0 auto;
        }
        .paper-body {
          position: relative;
        }
        .block-wrap {
          position: relative;
          padding: 4px 0;
          cursor: pointer;
          transition: background 0.12s;
        }
        .block-wrap::before {
          content: '';
          position: absolute;
          inset: 0px -8px;
          border: 1px dashed transparent;
          pointer-events: none;
          border-radius: 2px;
          transition: all 0.12s;
        }
        .block-wrap:hover::before {
          border-color: rgba(201, 163, 107, 0.4);
        }
        .block-wrap.selected::before {
          border-color: var(--ui-accent);
          border-style: solid;
          background: rgba(201, 163, 107, 0.05);
        }
        .block-controls {
          position: absolute;
          top: -2px;
          right: -36px;
          display: none;
          flex-direction: column;
          gap: 2px;
          z-index: 5;
        }
        .block-wrap:hover .block-controls,
        .block-wrap.selected .block-controls {
          display: flex;
        }
        .block-controls button {
          width: 24px; height: 22px;
          background: var(--ui-bg-2);
          border: 1px solid var(--ui-border);
          color: var(--ui-muted);
          cursor: pointer;
          padding: 0;
          display: flex;
          align-items: center;
          justify-content: center;
          border-radius: 2px;
        }
        .block-controls button:hover {
          color: var(--ui-accent);
          border-color: var(--ui-accent);
        }
        .block-controls button.danger:hover {
          color: var(--ui-danger);
          border-color: var(--ui-danger);
        }
        .block-controls button:disabled {
          opacity: 0.3;
          cursor: not-allowed;
        }

        /* ----- INSPECTOR ----- */
        .inspector-empty {
          padding: 40px 20px;
          text-align: center;
          color: var(--ui-muted);
          font-size: 12px;
          font-style: italic;
        }
        .inspector-header {
          padding: 12px 14px;
          border-bottom: 1px solid var(--ui-border);
          background: var(--ui-bg-3);
          display: flex;
          align-items: center;
          gap: 8px;
        }
        .inspector-header .type-badge {
          font-size: 9.5px;
          letter-spacing: 0.18em;
          text-transform: uppercase;
          color: var(--ui-accent);
          font-weight: 700;
          padding: 3px 7px;
          background: rgba(201, 163, 107, 0.12);
          border: 1px solid rgba(201, 163, 107, 0.3);
          border-radius: 2px;
        }
        .inspector-header .type-name {
          font-size: 13px;
          color: var(--ui-text);
          font-weight: 600;
        }
        .inspector-body {
          padding: 14px;
        }

        .mini-btn {
          padding: 4px 5px;
          background: rgba(0,0,0,0.2);
          border: 1px solid var(--ui-border);
          color: var(--ui-muted);
          cursor: pointer;
          border-radius: 2px;
          display: inline-flex;
          align-items: center;
          justify-content: center;
        }
        .mini-btn:hover { color: var(--ui-accent); border-color: var(--ui-accent); }
        .mini-btn.danger:hover { color: var(--ui-danger); border-color: var(--ui-danger); }

        .seg {
          padding: 6px 8px;
          background: rgba(0,0,0,0.2);
          border: 1px solid var(--ui-border);
          color: var(--ui-muted);
          cursor: pointer;
          border-radius: 2px;
          font-family: inherit;
          font-size: 11px;
          letter-spacing: 0.08em;
          text-transform: uppercase;
        }
        .seg:hover { color: var(--ui-text); border-color: var(--ui-border-2); }
        .seg.active {
          background: var(--ui-bg-3);
          border-color: var(--ui-accent);
          color: var(--ui-accent-2);
        }

        /* Width slider */
        .width-slider {
          width: 100%;
          accent-color: var(--ui-accent);
        }

        /* Scrollbars */
        .rail::-webkit-scrollbar, .canvas::-webkit-scrollbar { width: 8px; height: 8px; }
        .rail::-webkit-scrollbar-track, .canvas::-webkit-scrollbar-track { background: transparent; }
        .rail::-webkit-scrollbar-thumb, .canvas::-webkit-scrollbar-thumb {
          background: var(--ui-border-2);
          border-radius: 4px;
        }
        .rail::-webkit-scrollbar-thumb:hover, .canvas::-webkit-scrollbar-thumb:hover {
          background: var(--ui-muted);
        }

        /* Print */
        @media print {
          @page { margin: 6mm; }
          body, html { margin: 0; background: white !important; }
          .almanach-app {
            background: white !important;
            height: auto !important;
            overflow: visible !important;
          }
          .almanach-app::before { display: none !important; }
          .topbar, .rail { display: none !important; }
          .workspace {
            display: block !important;
            grid-template-columns: none !important;
            overflow: visible !important;
          }
          .canvas {
            display: block !important;
            padding: 0 !important;
            background: white !important;
            background-image: none !important;
            overflow: visible !important;
          }
          .paper-shell {
            filter: none !important;
            margin: 0 auto !important;
            page-break-inside: avoid;
          }
          .block-controls { display: none !important; }
          .block-wrap::before { display: none !important; }
          .block-wrap { padding: 4px 0 !important; }
        }
      `}</style>

      {/* TOPBAR */}
      <div className="topbar">
        <div className="brand">
          <span className="brand-mark">ALMANACH</span>
          <span className="brand-sub">studio</span>
        </div>

        <div style={{ display: "flex", alignItems: "center", gap: 10, color: "var(--ui-muted)", fontSize: 11.5, letterSpacing: "0.05em" }}>
          <Layers size={13} />
          <span>{blocks.length} blocks</span>
          <span style={{ color: "var(--ui-border-2)" }}>·</span>
          <span>{theme.name}</span>
          <span style={{ color: "var(--ui-border-2)" }}>·</span>
          <span>{paperWidth}px</span>
          <span style={{ color: "var(--ui-border-2)" }}>·</span>
          <span>{bodyScale.toFixed(1)}×</span>
        </div>

        <div className="topbar-spacer" />

        <div className="topbar-actions">
          <input
            ref={fileInputRef}
            type="file"
            accept="application/json,.json"
            style={{ display: "none" }}
            onChange={handleImportFile}
          />

          <button className="btn ghost" onClick={() => setShowLeft((s) => !s)} title="Toggle blocks panel">
            <Eye size={13} /> Blocks
          </button>
          <button className="btn ghost" onClick={() => setShowRight((s) => !s)} title="Toggle inspector">
            <Pencil size={13} /> Inspector
          </button>

          <span className="topbar-divider" />

          <button className="btn" onClick={handleImportClick} title="Open a saved layout (.json)">
            <FileText size={13} /> Open
          </button>
          <button className="btn" onClick={handleExportJson} title="Save layout as JSON">
            <Download size={13} /> Save
          </button>
          <button
            className="btn"
            onClick={handleExportPng}
            disabled={exporting}
            title="Export as PNG image"
          >
            {exporting ? (
              <span className="spinner" />
            ) : (
              <Download size={13} />
            )}
            {exporting ? "Rendering…" : "PNG"}
          </button>
          <button className="btn primary" onClick={handlePrint} disabled={exporting} title="Send directly to thermal printer">
            {exporting ? (
              <span className="spinner" />
            ) : (
              <Printer size={13} />
            )}
            {exporting ? "Printing…" : "Print"}
          </button>
        </div>
      </div>

      {/* TOAST */}
      {toast && (
        <div className={`toast ${toast.kind === "err" ? "err" : "ok"}`}>
          {toast.text}
        </div>
      )}

      {/* WORKSPACE */}
      <div
        className="workspace"
        style={{
          gridTemplateColumns: `${showLeft ? "240px" : "0"} 1fr ${showRight ? "320px" : "0"}`,
        }}
      >
        {/* LEFT RAIL — block library */}
        {showLeft && (
          <div className="rail left">
            {Object.entries(groupedTypes).map(([groupKey, types]) => (
              <div className="rail-section" key={groupKey}>
                <div className="rail-title">
                  <span className="rail-title-dot" style={{ background: GROUPS[groupKey].color }} />
                  {GROUPS[groupKey].label}
                </div>
                {types.map((t) => {
                  const Icon = t.icon;
                  return (
                    <button key={t.type} className="lib-item" onClick={() => addBlock(t.type)}>
                      <Icon size={14} className="icon" />
                      <span style={{ flex: 1 }}>{t.label}</span>
                      <Plus size={12} style={{ color: "var(--ui-muted)" }} />
                    </button>
                  );
                })}
              </div>
            ))}
          </div>
        )}

        {/* CANVAS — thermal paper preview */}
        <div className="canvas" onClick={() => setSelectedId(null)}>
          <ThermalPaper
            ref={paperRef}
            blocks={blocks}
            theme={theme}
            selectedId={selectedId}
            onSelect={setSelectedId}
            onMove={moveBlock}
            onDelete={deleteBlock}
            paperWidth={paperWidth}
          />
        </div>

        {/* RIGHT RAIL — inspector + theme + width */}
        {showRight && (
          <div className="rail right">
            <div className="rail-section">
              <div className="rail-title">
                <span className="rail-title-dot" style={{ background: "var(--ui-accent)" }} />
                Theme
              </div>
              <div className="theme-grid">
                {Object.entries(THEMES).map(([k, t]) => (
                  <button
                    key={k}
                    onClick={() => { setThemeKey(k); setInlineTheme(null); setThemePresets({}); }}
                    className={`theme-card ${themeKey === k ? "active" : ""}`}
                  >
                    <div className="swatch" style={{ background: t.paper, color: t.ink }}>
                      {t.icon}
                    </div>
                    <div className="name">{t.name}</div>
                  </button>
                ))}
              </div>
            </div>

            <div className="rail-section">
              <div className="rail-title">
                <span className="rail-title-dot" style={{ background: "var(--ui-accent)" }} />
                Paper Width
              </div>
              <input
                className="width-slider"
                type="range"
                min="320"
                max="480"
                step="4"
                value={paperWidth}
                onChange={(e) => setPaperWidth(+e.target.value)}
              />
              <div style={{ display: "flex", justifyContent: "space-between", fontSize: 10, color: "var(--ui-muted)", marginTop: 4 }}>
                <span>58mm</span>
                <span style={{ color: "var(--ui-accent-2)", fontVariantNumeric: "tabular-nums" }}>{paperWidth}px</span>
                <span>80mm</span>
              </div>
            </div>

            <div className="rail-section">
              <div className="rail-title">
                <span className="rail-title-dot" style={{ background: "var(--ui-accent)" }} />
                Font Scale
              </div>
              <input
                className="width-slider"
                type="range"
                min="1.0"
                max="2.0"
                step="0.1"
                value={bodyScale}
                onChange={(e) => setBodyScale(+e.target.value)}
              />
              <div style={{ display: "flex", justifyContent: "space-between", fontSize: 10, color: "var(--ui-muted)", marginTop: 4 }}>
                <span>1×</span>
                <span style={{ color: "var(--ui-accent-2)", fontVariantNumeric: "tabular-nums" }}>{bodyScale.toFixed(1)}×</span>
                <span>2×</span>
              </div>
            </div>

            <div className="rail-section">
              <div className="rail-title">
                <span className="rail-title-dot" style={{ background: "var(--ui-accent)" }} />
                Feed After Print
              </div>
              <input
                className="width-slider"
                type="range"
                min="0"
                max="20"
                step="1"
                value={feedLines}
                onChange={(e) => setFeedLines(+e.target.value)}
              />
              <div style={{ display: "flex", justifyContent: "space-between", fontSize: 10, color: "var(--ui-muted)", marginTop: 4 }}>
                <span>0</span>
                <span style={{ color: "var(--ui-accent-2)", fontVariantNumeric: "tabular-nums" }}>{feedLines} lines</span>
                <span>20</span>
              </div>
            </div>

            {selected && Editor ? (
              <>
                <div className="inspector-header">
                  <span className="type-badge">{selected.type}</span>
                  <span className="type-name">
                    {BLOCK_TYPES.find((b) => b.type === selected.type)?.label}
                  </span>
                  <div style={{ flex: 1 }} />
                  <button className="mini-btn" onClick={() => duplicateBlock(selected.id)} title="Duplicate">
                    <Copy size={11} />
                  </button>
                </div>
                <div className="inspector-body">
                  <Editor data={selected.data} set={(d) => updateBlock(selected.id, d)} />
                </div>
              </>
            ) : (
              <div className="inspector-empty">
                Select a block on the paper to edit its content, or click a block from the left to add one.
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
