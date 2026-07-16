# Changelog

## 2026-07-16

- Initial workspace created


## 2026-07-16

Step 1: Created ticket, design+implementation doc (embedded-bitmap web font approach), and 5 phase tasks. Grounded in DPR=1 render pipeline; fontforge usable via APPIMAGE_EXTRACT_AND_RUN.


## 2026-07-16

Step 2 (Phase 0 spike): Approach A (embedded-bitmap webfont) REJECTED - Chrome ignores strikes, renders outlines. Approach B (disable AA via fontconfig) ADOPTED - hinted vector text renders crisp 1-bit; DM Sans good to ~10px, degrades at 8-9px. Design doc + plan pivoted.


## 2026-07-16

Step 3 (Phase 1): Shipped AA-off in render browser via fontconfig+FONTCONFIG_FILE; render screenshot 2.09%->0.00% gray end-to-end; env override ALMANACH_FONT_ANTIALIAS. lint+tests green (commit 93abdd3).


## 2026-07-16

Step 4 (Phases 3-4): Verified AA-off on real page (0.01% vs 4.58% gray, all strokes complete/legible); printed to K118 via production pipeline. Phase 2 decision: serif roughness at low-res is aesthetic; do not swap theme fonts unilaterally; recommend sans theme / optional future pixel face.


## 2026-07-16

Step 5: Supersampling (3x render + box-average downscale) shipped as default fix for small serif italics; keeps theme fonts; --supersample flag; AA-off kept for scale 1. Printed to K118. lint+tests green (commit f61ec55).

