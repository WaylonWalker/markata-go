// Usage: node scripts/adaptive-loading.benchmark.cjs BEFORE_URL AFTER_URL RESULTS.json
// Servers must use the same content-encoding. CDP totals and Lighthouse are
// deliberately measured separately. Three fresh browser contexts per case.
const fs = require('node:fs');
const path = require('node:path');
const {execFileSync} = require('node:child_process');
const {chromium} = require(process.env.PLAYWRIGHT_MODULE || path.join(execFileSync('npm',['root','-g'],{encoding:'utf8'}).trim(),'agent-browser/node_modules/playwright-core'));
const [before, after, output] = process.argv.slice(2);
const routes = (process.env.BENCHMARK_ROUTES || 'vimgrep-open-buffers,til,2025-nas,tmux-pop-size,bloatware-is-dying').split(',');
const modes = (process.env.BENCHMARK_MODES || 'auto-constrained,auto-normal,save-data-constrained,desktop-normal').split(',');
(async () => {
 const browser = await chromium.launch({executablePath:'/usr/bin/chromium',headless:true,args:['--no-sandbox']});
 const results=[];
 try {
  for (const mode of modes) for (const route of routes) for (let run=0;run<3;run++) for (const version of (process.env.BENCHMARK_VERSIONS || 'before,after').split(',')) {
   // Manual Save Data has the same baseline as Auto constrained.
   const context=await browser.newContext({viewport: mode === 'desktop-normal' ? {width:1365,height:900} : {width:390,height:844}});
   await context.addInitScript(mode => {
    Object.defineProperty(navigator,'connection',{value:undefined});
    if (mode.startsWith('save-data')) localStorage.setItem('markata-loading-mode-v1','save-data');
    window.bench={lcp:0,cls:0};
    new PerformanceObserver(list => {for(const e of list.getEntries()) {window.bench.lcp=e.startTime; window.bench.lcpURL=e.url; window.bench.lcpTag=e.element && e.element.tagName;}}).observe({type:'largest-contentful-paint',buffered:true});
    new PerformanceObserver(list => {for(const e of list.getEntries()) if(!e.hadRecentInput) window.bench.cls+=e.value;}).observe({type:'layout-shift',buffered:true});
   },mode);
   const page=await context.newPage(); const cdp=await context.newCDPSession(page);
   await cdp.send('Network.enable'); await cdp.send('Network.setCacheDisabled',{cacheDisabled:true});
   await cdp.send('Network.emulateNetworkConditions',{offline:false,latency:mode.includes('constrained')?150:0,downloadThroughput:mode.includes('constrained')?192*1024:-1,uploadThroughput:-1});
   const types=new Map(); let bytes=0,media=0,requests=0; const failures=[];
   cdp.on('Network.requestWillBeSent',e=>{types.set(e.requestId,e.type);requests++;});
   cdp.on('Network.loadingFinished',e=>{bytes+=e.encodedDataLength;if(types.get(e.requestId)==='Media') media+=e.encodedDataLength;});
   cdp.on('Network.loadingFailed',e=>failures.push(e.errorText));
   await page.goto((version==='before'?before:after)+'/'+route+'/',{waitUntil:'load',timeout:90000});
   await page.waitForTimeout(2500);
   const initial=await page.evaluate(()=>({...window.bench,policy:document.documentElement.dataset.loadingPolicy,recommendation:document.documentElement.dataset.loadingRecommendation,
     evidence:sessionStorage.getItem('markata-loading-auto-v1'),timings:performance.getEntriesByType('resource').filter(e=>e.initiatorType==='img'||e.initiatorType==='video').map(e=>({name:e.name,bytes:e.transferSize,duration:e.duration,elapsed:e.responseEnd-e.responseStart})),
     lcpElement:performance.getEntriesByType('largest-contentful-paint').map(e=>e.url)}));
   // Record initial paint before scrolling; totals include identical feed/gallery exploration.
   if(route==='til'||route==='2025-nas') {for(let i=0;i<4;i++){await page.evaluate(()=>scrollBy(0,700));await page.waitForTimeout(300);} await page.waitForTimeout(1500);}
   const result={version,mode,route,run,bytes,media,requests,failures,...initial};
   results.push(result); fs.writeFileSync(output,JSON.stringify(results,null,2));
   console.log(JSON.stringify({version,mode,route,run,bytes,media,requests,lcp:initial.lcp,policy:initial.policy}));
   await context.close();
  }
 } finally {await browser.close();}
})().catch(e=>{console.error(e);process.exitCode=1;});
