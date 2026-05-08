import fs from 'node:fs/promises';
import path from 'node:path';
import http from 'node:http';
import { fileURLToPath } from 'node:url';
import { spawn } from 'node:child_process';
import puppeteer from 'puppeteer-core';
import { AxePuppeteer } from '@axe-core/puppeteer';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const repo = path.resolve(scriptDir, '../..');
const fixture = path.join(repo, '.tmp', 'a11y-rendered');
const outputDir = path.join(fixture, 'output');
const port = Number(process.env.MARKATA_A11Y_PORT || 4177);
const chromeBin = process.env.CHROME_BIN || process.env.PUPPETEER_EXECUTABLE_PATH || '/home/u_walkews/.local/bin/chromium';

async function run(command, args, cwd) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, { cwd, stdio: 'inherit' });
    child.on('exit', (code) => code === 0 ? resolve() : reject(new Error(`${command} exited ${code}`)));
  });
}

async function ensureFixture() {
  await fs.rm(fixture, { recursive: true, force: true });
  await fs.mkdir(path.join(fixture, 'posts'), { recursive: true });
  await fs.writeFile(path.join(fixture, 'markata-go.toml'), `[markata-go]
title = "Rendered Accessibility Fixture"
description = "Rendered contrast fixture"
url = "http://127.0.0.1:${port}"
output_dir = "output"
templates_dir = "templates"
assets_dir = "static"
license = false

[markata-go.glob]
patterns = ["posts/**/*.md"]
use_gitignore = false

[markata-go.theme]
palette = "default-light"

[markata-go.search]
enabled = false

[markata-go.one_line_link]
enabled = true
`);
  await fs.writeFile(path.join(fixture, 'posts', 'contrast-components.md'), `---
title: "Contrast Components"
description: "Exercises bundled default theme colors"
date: 2026-05-07
published: true
tags:
  - accessibility
---

# Contrast Components

This paragraph includes an inline [example link](https://example.com) surrounded by normal body text.

This fixture also includes ==highlighted text==, ++Ctrl+C++, and a standalone URL.

https://example.com/standalone-card

::: note
This note exercises generated container colors.
:::

| Name | Value |
| ---- | ----- |
| Alpha | 1 |
| Beta | 2 |
`);
}

async function paletteNames() {
  const dir = path.join(repo, 'pkg', 'palettes', 'palettes');
  const files = await fs.readdir(dir);
  return files.filter((file) => file.endsWith('.toml')).map((file) => file.replace(/\.toml$/, '')).sort();
}

function updatePaletteConfig(config, paletteName) {
  return config.replace(/palette = "[^"]+"/, `palette = "${paletteName}"`);
}

function serveStatic(dir) {
  const server = http.createServer(async (req, res) => {
    const urlPath = decodeURIComponent(new URL(req.url, `http://127.0.0.1:${port}`).pathname);
    let filePath = path.join(dir, urlPath);
    if (urlPath.endsWith('/')) filePath = path.join(filePath, 'index.html');
    try {
      const data = await fs.readFile(filePath);
      const ext = path.extname(filePath);
      res.setHeader('content-type', ext === '.css' ? 'text/css' : ext === '.js' ? 'application/javascript' : 'text/html');
      res.end(data);
    } catch {
      res.statusCode = 404;
      res.end('not found');
    }
  });
  return new Promise((resolve) => server.listen(port, '127.0.0.1', () => resolve(server)));
}

async function inspectPage(page) {
  return page.evaluate(() => {
    const link = document.querySelector('.post-content p a[href="https://example.com"]');
    const paragraph = link?.closest('p');
    const linkStyle = link ? getComputedStyle(link) : null;
    const textStyle = paragraph ? getComputedStyle(paragraph) : null;
    return {
      inlineLink: {
        found: Boolean(link),
        linkColor: linkStyle?.color || '',
        textColor: textStyle?.color || '',
        textDecorationLine: linkStyle?.textDecorationLine || '',
      },
    };
  });
}

function summarizeAxe(violations) {
  return violations.map((violation) => ({
    id: violation.id,
    impact: violation.impact,
    help: violation.help,
    nodes: violation.nodes.map((node) => ({
      target: node.target,
      html: node.html,
      failureSummary: node.failureSummary,
    })),
  }));
}

await ensureFixture();
await run('go', ['build', '-o', path.join(fixture, 'markata-go'), './cmd/markata-go'], repo);

const configPath = path.join(fixture, 'markata-go.toml');
const baseConfig = await fs.readFile(configPath, 'utf8');
const names = await paletteNames();
const browser = await puppeteer.launch({
  executablePath: chromeBin,
  args: ['--headless=new', '--no-sandbox', '--disable-dev-shm-usage'],
});

const results = [];
for (const palette of names) {
  await fs.writeFile(configPath, updatePaletteConfig(baseConfig, palette));
  await fs.rm(outputDir, { recursive: true, force: true });
  await run(path.join(fixture, 'markata-go'), ['build'], fixture);

  const server = await serveStatic(outputDir);
  const page = await browser.newPage();
  await page.setViewport({ width: 1280, height: 900, deviceScaleFactor: 1 });
  await page.goto(`http://127.0.0.1:${port}/contrast-components/`, { waitUntil: 'networkidle0' });
  const axe = await new AxePuppeteer(page).withRules(['color-contrast']).analyze();
  const custom = await inspectPage(page);
  await page.close();
  await new Promise((resolve) => server.close(resolve));

  results.push({ palette, axeViolations: summarizeAxe(axe.violations), ...custom });
}

await browser.close();
await fs.writeFile(path.join(fixture, 'a11y-results.json'), JSON.stringify(results, null, 2));

const contrastFailures = results.filter((result) => result.axeViolations.length > 0);
const linkAffordanceFailures = results.filter((result) => result.inlineLink.found && result.inlineLink.linkColor === result.inlineLink.textColor && result.inlineLink.textDecorationLine === 'none');

console.log(JSON.stringify({
  paletteCount: results.length,
  palettesWithColorContrastViolations: contrastFailures.length,
  palettesWithInlineLinksSameColorAndNoDecoration: linkAffordanceFailures.length,
  resultsPath: path.join(fixture, 'a11y-results.json'),
  contrastExamples: contrastFailures.slice(0, 10).map((result) => ({ palette: result.palette, violations: result.axeViolations })),
  linkAffordanceExamples: linkAffordanceFailures.slice(0, 10).map((result) => ({ palette: result.palette, inlineLink: result.inlineLink })),
}, null, 2));

if (contrastFailures.length > 0 || linkAffordanceFailures.length > 0) {
  process.exitCode = 1;
}
