import * as esbuild from "esbuild";
import { mkdirSync, writeFileSync, copyFileSync } from "fs";

const isDev = process.argv.includes("--dev");

mkdirSync("dist", { recursive: true });

const common = {
  bundle: true,
  minify: !isDev,
  format: "iife",
  target: ["es2020"],
  define: {
    "process.env.NODE_ENV": isDev ? '"development"' : '"production"',
  },
  logLevel: "info",
  metafile: true,
};

const almanachResult = await esbuild.build({
  ...common,
  entryPoints: ["src/index.jsx"],
  globalName: "AlmanachStudio",
  outfile: "dist/almanach-bundle.js",
});

const setupResult = await esbuild.build({
  ...common,
  entryPoints: ["src/setup.jsx"],
  globalName: "AlmanachSetup",
  outfile: "dist/setup-bundle.js",
});

function html({ title, script, css }) {
  const cssLink = css ? `    <link rel="stylesheet" href="${css}">
` : "";
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>${title}</title>
${cssLink}  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    html, body, #root { width: 100%; min-height: 100%; }
    body { background: #1a1612; }
  </style>
</head>
<body>
  <div id="root"></div>
  <script src="${script}"></script>
</body>
</html>`;
}

// Copy embedded Google Fonts CSS (base64 woff2, no network needed at runtime)
copyFileSync("src/fonts-embedded.css", "dist/fonts.css");

writeFileSync("dist/index.html", html({ title: "Almanach Studio", script: "/almanach/bundle.js", css: "/almanach/fonts.css" }));
writeFileSync("dist/setup.html", html({ title: "Almanach Printer Setup", script: "/setup/bundle.js" }));

function printBundle(result, outfile, label) {
  const bytes = result.metafile.outputs[outfile].bytes;
  const kb = (bytes / 1024).toFixed(1);
  console.log(`✓ Built ${outfile} (${kb} KB, ${label}, ${isDev ? "development" : "minified"})`);
}

printBundle(almanachResult, "dist/almanach-bundle.js", "studio");
printBundle(setupResult, "dist/setup-bundle.js", "setup");
console.log("✓ Built dist/index.html");
console.log("✓ Built dist/setup.html");
