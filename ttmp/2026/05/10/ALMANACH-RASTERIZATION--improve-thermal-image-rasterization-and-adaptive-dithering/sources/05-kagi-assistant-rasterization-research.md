# Kagi Assistant Rasterization Research

Prompt used:

> Research thermal printer monochrome rasterization for preserving photo edges. Compare thresholding, adaptive thresholding, Bayer/ordered dither, blue-noise masks, Floyd-Steinberg, Atkinson, Stucki, Burkes, Sierra, and edge-aware hybrid methods. Focus on implementation guidance for a Go app rendering HTML screenshots to 1-bit ESC/POS-like bitmap data. Include references to authoritative sources and practical recommendations.

Summary of assistant findings:

- Thermal printers are monochrome direct thermal devices; continuous tone image content must be represented as 1-bit raster dots.
- Fixed thresholding is very fast and suitable for text and already-high-contrast line art, but poor for photographs.
- Ordered dithering is deterministic and fast. Bayer matrices produce visible regular patterns; blue-noise masks are better for photographs because the texture is less grid-like.
- Error diffusion methods distribute quantization error to future pixels. Floyd-Steinberg is the classic efficient baseline; Atkinson produces a lighter retro result that can be attractive on thermal paper; Stucki/Burkes/Sierra variants trade smoother tone against speed and softness.
- Edge-aware hybrid methods can preserve outlines by detecting edges with Sobel or Difference-of-Gaussians, dithering a tone layer separately, and OR-ing/merging a restrained edge mask back into the final bitmap.
- Practical Go implementation options include writing a small internal raster package or evaluating existing Go dithering libraries such as `github.com/esimov/dithergo`; any dependency needs license and output-quality review before adoption.
- The immediate recommendation for Almanach is to build a comparison sheet from the stored cat portraits and test threshold, Atkinson, Floyd-Steinberg, Stucki/Sierra, ordered Bayer, blue-noise, and edge-hybrid modes on the real K118 printer.

Key references surfaced by search/assistant:

- Tanner Helland, “Image Dithering: Eleven Algorithms and Source Code” — https://tannerhelland.com/2012/12/28/dithering-eleven-algorithms-source-code.html
- ImageMagick Usage, “Color Quantization and Dithering” — https://usage.imagemagick.org/quantize/
- Sweetcorn, “Dithering algorithms” — https://delucis.github.io/sweetcorn/algorithms/
- `esimov/dithergo` Go library — https://github.com/esimov/dithergo
- Floyd and Steinberg paper reference — https://doi.org/10.1145/360094.360104
