"""fontforge script: PCF bitmap font -> OpenType with embedded bitmap strike.
Run: APPIMAGE_EXTRACT_AND_RUN=1 fontforge -lang=py -script convert_font.py <in.pcf> <out.otb> <FamilyName> <px>
"""
import sys, fontforge
src, out, family, px = sys.argv[1], sys.argv[2], sys.argv[3], int(sys.argv[4])
f = fontforge.open(src)
f.familyname = family
f.fontname = family.replace(" ", "")
f.fullname = family
# Keep bitmaps; generate an OpenType Bitmap font (sfnt with EBDT/EBLC).
f.generate(out, bitmap_type="otb")
print("generated", out, "strikes:", f.bitmapSizes)
