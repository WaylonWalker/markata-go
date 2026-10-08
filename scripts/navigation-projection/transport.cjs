// Transport/DOMParser probe only. Does not execute route scripts or swap a page.
const http = require('node:http');
const fs = require('node:fs');
const path = require('node:path');
const zlib = require('node:zlib');
const {execFileSync} = require('node:child_process');
const {parseArgs} = require('node:util');
const args = parseArgs({allowPositionals: true, options: {
  help: {type: 'boolean', short: 'h'}, version: {type: 'boolean'}}});
if (args.values.version) {console.log('navigation-transport-probe 1'); process.exit(0);}
if (args.values.help || args.positionals.length !== 1) {
  const message = 'Usage: node transport.cjs FIXTURES\nMeasure inert HTML fetch + DOMParser; JSON on stdout. Requires existing Chromium and Playwright.\nNo page swap, assets, paint, or history measurement.';
  if (args.values.help) console.log(message); else console.error(message);
  process.exit(args.values.help ? 0 : 2);
}
const playwright = require(process.env.PLAYWRIGHT_MODULE || path.join(
  execFileSync('npm', ['root', '-g'], {encoding: 'utf8'}).trim(),
  'agent-browser/node_modules/playwright-core'));
const fixtures = path.resolve(args.positionals[0]);
const routes = ['vimgrep-open-buffers', 'til', '2025-nas', 'tmux-pop-size', 'bloatware-is-dying'];
const responses = new Map();
for (const route of routes) for (const file of ['index.html', '_nav.html']) {
  const data = fs.readFileSync(path.join(fixtures, route, file));
  responses.set(`/${route}/${file}`, zlib.gzipSync(data, {level: 9}));
}
const server = http.createServer((req, res) => {
  if (req.url === '/') {res.setHeader('Content-Type', 'text/html'); return res.end('<!doctype html><title>Inert navigation probe</title>');}
  const data = responses.get(req.url);
  if (!data) {res.writeHead(404); return res.end();}
  res.writeHead(200, {'Content-Type': 'text/html', 'Content-Encoding': 'gzip',
    'Content-Length': data.length, 'Cache-Control': 'no-store'});
  res.end(data);
});
(async () => {
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  const origin = `http://127.0.0.1:${server.address().port}`;
  const browser = await playwright.chromium.launch({executablePath: process.env.CHROMIUM_EXECUTABLE || '/usr/bin/chromium', headless: true, args: ['--no-sandbox']});
  const rows = [];
  try {
    for (const profile of ['desktop-normal', 'mobile-normal', 'mobile-constrained']) {
      const context = await browser.newContext({viewport: profile.startsWith('mobile') ? {width: 390, height: 844} : {width: 1365, height: 900}});
      const page = await context.newPage();
      await page.goto(origin);
      const cdp = await context.newCDPSession(page);
      await cdp.send('Network.enable');
      await cdp.send('Network.setCacheDisabled', {cacheDisabled: true});
      if (profile === 'mobile-constrained') await cdp.send('Network.emulateNetworkConditions', {
        offline: false, latency: 150, downloadThroughput: 192 * 1024, uploadThroughput: 192 * 1024});
      for (const route of routes) for (let run = 0; run < 3; run++) {
        // Alternate order to reduce warming/order bias.
        for (const file of run % 2 ? ['_nav.html', 'index.html'] : ['index.html', '_nav.html']) {
          const url = `${origin}/${route}/${file}`;
          const result = await page.evaluate(async url => {
            const start = performance.now();
            const response = await fetch(url);
            const text = await response.text();
            const received = performance.now();
            const doc = new DOMParser().parseFromString(text, 'text/html');
            const region = doc.querySelector('#view-transition-page');
            if (!region || !region.querySelector('main')) throw new Error('Missing destination region');
            const finished = performance.now();
            const entry = performance.getEntriesByName(url).at(-1);
            return {fetch_ms: received - start, parse_ms: finished - received,
              fetch_parse_ms: finished - start, encoded_body_bytes: entry.encodedBodySize,
              transfer_bytes: entry.transferSize, region_text_chars: region.textContent.length};
          }, url);
          rows.push({profile, route, run, representation: file === 'index.html' ? 'full' : 'projection', ...result});
        }
      }
      await context.close();
    }
    process.stdout.write(JSON.stringify({method: 'inert fetch + DOMParser; no swap, assets, history, paint, or lifecycle', rows}, null, 2) + '\n');
  } finally {await browser.close(); await new Promise(resolve => server.close(resolve));}
})().catch(error => {console.error(error); server.close(); process.exitCode = 1;});
