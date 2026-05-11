Sweetcorn supports two main types of dithering algorithms: [threshold maps](https://delucis.github.io/sweetcorn/algorithms/#threshold-maps) and [error diffusion](https://delucis.github.io/sweetcorn/algorithms/#error-diffusion).

This page includes examples of all the built-in dithering algorithms accompanied by a demonstration image dithered by the algorithm:

![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_2b2svJ.webp)

## Threshold maps

Threshold maps apply a fixed pattern to the image to determine which pixels should be turned on or off. These algorithms are faster to compute than error diffusion algorithms. They can produce noticeable repeating patterns in the output image.

Sweetcorn provides the following threshold map algorithms:

- `  threshold  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_Z1xyE1u.webp)
- `  bayer-2  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_Pc8nQ.webp)
- `  bayer-4  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_Zf32zh.webp)
- `  bayer-8  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_Z2pxovx.webp)
- `  bayer-16  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_Z1gp4BU.webp)
- `  dot-4  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_Z2bvtIl.webp)
- `  dot-6  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_1Nqt7s.webp)
- `  dot-diagonal-6  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_Zh5xfq.webp)
- `  dot-8  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_Ibi9k.webp)
- `  dot-diagonal-8  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_Z1mkIdy.webp)
- `  dot-diagonal-10  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_ZXByiz.webp)
- `  dot-diagonal-16  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_PP2AX.webp)
- `  dot-horizontal-6  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_1JVxbM.webp)
- `  dot-vertical-6  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_ZWH4Rp.webp)
- `  dot-horizontal-3x5  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_Knwcs.webp)
- `  dot-vertical-5x3  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_ZiN2Jz.webp)
- `  horizontal-checkers-6  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_1j3s38.webp)
- `  vertical-checkers-6  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_Zdgleo.webp)
- `  blue-noise  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_BIqHr.webp)
- `  white-noise  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_1VwS5j.webp)

Error diffusion algorithms work by spreading the quantization error of a pixel to its neighboring pixels. This can result in a softer, less structured dithering effect, but it is more expensive, computing multiple values for each pixel of an image. Because of how errors are propagated, diffusion algorithms can also create artifacts in some images.

Sweetcorn provides the following error diffusion algorithms:

- `  simple-diffusion  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_28c4mo.webp)
- `  row-diffusion  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_Z1FsXPU.webp)
- `  column-diffusion  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_Z2f2JD1.webp)
- `  floyd-steinberg  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_Z1Jl3YQ.webp)
- `  false-floyd-steinberg  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_1KVUOK.webp)
- `  jarvis-judice-ninke  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_Z2qPH1F.webp)
- `  stucki  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_Z27Sfb0.webp)
- `  burkes  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_9fDiX.webp)
- `  atkinson  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_2jz7bp.webp)
- `  pigeon  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_Z2oY6U.webp)
- `  sierra  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_27PNDI.webp)
- `  sierra-two-row  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_2aFAte.webp)
- `  sierra-lite  ` ![](https://delucis.github.io/sweetcorn/_astro/demo.C7N_HB47_N9W8j.webp)

Choose an image file below and explore how it looks with different configurations:

![Greyscale moon](https://delucis.github.io/sweetcorn/_astro/9237597241_7bb0b5ff7b_o.10DoAG0-_Z2cQeoC.webp)

The Moon. August 25th 1890 © Tyne & Wear Archives & Museums

## Which algorithm should I use?

It’s up to you!

Choose an algorithm you like visually. Often the best choice depends on your specific image and taste, but here are a few rules of thumb:

- For speed, choose a threshold map algorithm: they are both faster to compute and result in smaller file sizes.
- `blue-noise` dithering is a good general-purpose choice, especially for photographic images.
- The `bayer-8` and `bayer-16` algorithms dither using a recognisable retro pattern.
- The `floyd-steinberg` algorithm is one of the most widely used error diffusion algorithms and is quite efficient.
- The `atkinson` algorithm was first used on Apple computers in the 1980s, so also has a retro feel.
- There are no wrong answers — have fun and choose one you like!