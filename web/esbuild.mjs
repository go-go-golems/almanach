import * as esbuild from "esbuild";
import { writeFileSync } from "fs";

const isDev = process.argv.includes("--dev");

const result = await esbuild.build({
  entryPoints: ["src/index.jsx"],
  bundle: true,
  minify: !isDev,
  format: "iife",
  globalName: "AlmanachStudio",
  outfile: "dist/almanach-bundle.js",
  target: ["es2020"],
  define: {
    "process.env.NODE_ENV": isDev ? '"development"' : '"production"',
  },
  logLevel: "info",
  metafile: true,
});

// Generate the host HTML page
const html = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Almanach Studio</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    html, body, #root { width: 100%; height: 100%; }
    body { background: #1a1612; }
  </style>
</head>
<body>
  <div id="root"></div>
  <script src="/almanach/bundle.js"></script>
</body>
</html>`;

writeFileSync("dist/index.html", html);

// Print bundle analysis
const bytes = result.metafile.outputs["dist/almanach-bundle.js"].bytes;
const kb = (bytes / 1024).toFixed(1);
console.log(`\n✓ Built dist/almanach-bundle.js (${kb} KB, ${isDev ? "development" : "minified"})`);
console.log("✓ Built dist/index.html");
