# Changelog

## 2026-07-16

- Initial workspace created


## 2026-07-16

Step 1: Created ticket, design+implementation doc (embedded-bitmap web font approach), and 5 phase tasks. Grounded in DPR=1 render pipeline; fontforge usable via APPIMAGE_EXTRACT_AND_RUN.


## 2026-07-16

Step 2 (Phase 0 spike): Approach A (embedded-bitmap webfont) REJECTED - Chrome ignores strikes, renders outlines. Approach B (disable AA via fontconfig) ADOPTED - hinted vector text renders crisp 1-bit; DM Sans good to ~10px, degrades at 8-9px. Design doc + plan pivoted.

