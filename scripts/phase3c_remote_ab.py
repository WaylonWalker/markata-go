#!/usr/bin/env python3
"""Isolated, reproducible Phase 3C sidebar source-order experiment.

Loads one live /archive/ HTML snapshot, reorders the sidebar in a byte-preserving
way, and serves the same CSS/JS/fonts/images for baseline and variant locally.
Does not change production or make claims from a DOM-only visibility probe.
"""
import asyncio
import base64
import json
import os
from pathlib import Path
import re
import statistics
import time
from urllib.parse import urlsplit

import brotli
import requests
from aiohttp import web
from playwright.async_api import async_playwright

ORIGIN = "https://waylonwalker.com"
OUT = Path("phase3c-evidence")
OUT.mkdir(exist_ok=True)
PERF = """(() => {
  window.phase3 = { fcp:null, lcp:null, cls:0, lcpTag:null };
  new PerformanceObserver(l => { for(const e of l.getEntries()) {
    if(e.name === 'first-contentful-paint') window.phase3.fcp=e.startTime;
  }}).observe({type:'paint',buffered:true});
  new PerformanceObserver(l => { for(const e of l.getEntries()) {
    window.phase3.lcp=e.startTime; window.phase3.lcpTag=e.element?.tagName || null;
  }}).observe({type:'largest-contentful-paint',buffered:true});
  new PerformanceObserver(l => { for(const e of l.getEntries()) {
    if (!e.hadRecentInput) window.phase3.cls+=e.value;
  }}).observe({type:'layout-shift',buffered:true});
})();"""

def make_pair(html):
    pattern = re.compile(r'<aside\b[^>]*class=["\'][^"\']*\bfeed-sidebar\b[^"\']*["\'][^>]*>', re.I)
    m = pattern.search(html)
    if not m:
        raise RuntimeError("Missing <aside class=feed-sidebar> in current /archive/ snapshot")
    depth, stop = 0, None
    for tag in re.finditer(r'</?aside\b[^>]*>', html[m.start():], re.I):
        depth += -1 if tag.group(0).startswith("</") else 1
        if depth == 0:
            stop = m.start() + tag.end()
            break
    if stop is None: raise RuntimeError("Unbalanced sidebar <aside>")
    markup = html[m.start():stop]
    without = html[:m.start()] + html[stop:]
    main = re.search(r'<main\b[^>]*>', without, re.I)
    if not main: raise RuntimeError("Missing main")
    close = re.search(r'</main\s*>', without[main.end():], re.I)
    if not close: raise RuntimeError("Missing closing main")
    offset = main.end() + close.end()
    moved = without[:offset] + markup + without[offset:]
    if len(moved) != len(html): raise RuntimeError("HTML length changed")
    info = {
        "baseline_html_decoded_bytes": len(html.encode()),
        "after_html_decoded_bytes": len(moved.encode()),
        "baseline_h1_offset": len(html[:html.find('<h1', html.find('<main'))].encode()),
        "after_h1_offset": len(moved[:moved.find('<h1', moved.find('<main'))].encode()),
        "baseline_br_bytes": len(brotli.compress(html.encode(), quality=5)),
        "after_br_bytes": len(brotli.compress(moved.encode(), quality=5)),
    }
    if info["after_h1_offset"] >= info["baseline_h1_offset"]:
        raise RuntimeError("Reorder did not advance main heading")
    return moved, info

print("Capturing production HTML once", flush=True)
r = requests.get(ORIGIN+"/archive/",timeout=90,headers={"User-Agent":"Markata-Phase3C-Benchmark/1.0"})
r.raise_for_status()
html = r.content.decode(r.encoding or "utf-8", errors="replace")
moved, source_info = make_pair(html)
(OUT/"baseline.html").write_text(html)
(OUT/"moved.html").write_text(moved)
source_info["source_url"] = ORIGIN+"/archive/"
source_info["origin_content_encoding"] = r.headers.get("Content-Encoding")
(OUT/"source_info.json").write_text(json.dumps(source_info,indent=2))
print("SOURCE_INFO "+json.dumps(source_info),flush=True)

cache = {}
session = requests.Session()
session.headers["User-Agent"] = "Markata-Phase3C-Benchmark/1.0"
def fetch_asset(path):
    if path in cache: return cache[path]
    try:
        x = session.get(ORIGIN+path,timeout=35,headers={"Accept-Encoding":"gzip, deflate"})
        content_type = x.headers.get("Content-Type","application/octet-stream")
        result = x.status_code, x.content, content_type
        if x.ok and len(x.content) < 20_000_000: cache[path] = result
        return result
    except Exception as ex:
        print("ASSET_FETCH_ERROR",path,str(ex),flush=True)
        return 502,str(ex).encode(),"text/plain"

async def archive(req):
    version = req.match_info["version"]
    payload = html if version=="before" else moved
    compressed = brotli.compress(payload.encode(),quality=5)
    return web.Response(body=compressed,headers={
        "Content-Type":"text/html; charset=utf-8",
        "Content-Encoding":"br","Cache-Control":"no-store","Vary":"Accept-Encoding",
        "X-Phase3C-Variant":version
    })

async def asset(req):
    path = req.path_qs
    status, data, content_type = await asyncio.to_thread(fetch_asset,path)
    return web.Response(status=status,body=data,headers={"Content-Type":content_type,"Cache-Control":"no-store"})

async def test_once(browser, variant, pair_index):
    ctx=await browser.new_context(viewport={"width":390,"height":844},
        is_mobile=True,has_touch=True,service_workers="block")
    await ctx.add_init_script(PERF)
    page=await ctx.new_page()
    cdp=await ctx.new_cdp_session(page)
    await cdp.send("Network.enable")
    await cdp.send("Network.setCacheDisabled",{"cacheDisabled":True})
    await cdp.send("Network.emulateNetworkConditions",{
        "offline":False,"latency":400,"downloadThroughput":51200,"uploadThroughput":51200})
    await cdp.send("Emulation.setCPUThrottlingRate",{"rate":4})
    failed=[]
    cdp.on("Network.loadingFailed",lambda e: failed.append(e.get("errorText","unknown")))
    frames=[]
    latest=None
    start=None
    pending=set()
    async def ack(sid):
        try: await cdp.send("Page.screencastFrameAck",{"sessionId":sid})
        except Exception: pass
    def frame(ev):
        nonlocal latest
        t=round((time.monotonic()-start)*1000) if start is not None else 0
        latest=(ev["data"],t)
        task=asyncio.create_task(ack(ev["sessionId"]))
        pending.add(task)
        task.add_done_callback(pending.discard)
    cdp.on("Page.screencastFrame",frame)
    await cdp.send("Page.enable")
    await cdp.send("Page.startScreencast",{"format":"jpeg","quality":65,"everyNthFrame":1})
    checkpoints=[1000,2000,3000,5000,8000,12000,18000,25000]
    start=time.monotonic()
    async def capture(target):
        delay=target/1000-(time.monotonic()-start)
        if delay>0: await asyncio.sleep(delay)
        snapshot=round((time.monotonic()-start)*1000)
        if latest:
            name=f"{variant}-{pair_index:02d}-{target:05d}.jpg"
            (OUT/name).write_bytes(base64.b64decode(latest[0]))
            frames.append({"target_ms":target,"snapshot_ms":snapshot,
                           "frame_received_ms":latest[1],"file":name})
        else: frames.append({"target_ms":target,"error":"no frame"})
    tasks=[asyncio.create_task(capture(t)) for t in checkpoints]
    nav_error=None
    try:
        await page.goto(f"http://127.0.0.1:8898/{variant}/archive/",wait_until="commit",timeout=50000)
    except Exception as ex: nav_error=str(ex)
    remain=26000-(time.monotonic()-start)*1000
    if remain>0: await asyncio.sleep(remain/1000)
    await asyncio.gather(*tasks,return_exceptions=True)
    try: metrics=await asyncio.wait_for(page.evaluate("() => window.phase3 || {}"),timeout=10)
    except Exception as ex: metrics={"error":str(ex)}
    try: await cdp.send("Page.stopScreencast")
    except Exception: pass
    if pending: await asyncio.gather(*pending,return_exceptions=True)
    await ctx.close()
    return {"variant":variant,"pair":pair_index,"metrics":metrics,
            "frames":frames,"network_errors":failed[:20],"navigation_error":nav_error,
            "observed_ms":round((time.monotonic()-start)*1000)}

async def main():
    app=web.Application()
    app.router.add_get("/{version:before|after}/archive/",archive)
    app.router.add_route("*","/{path:.*}",asset)
    runner=web.AppRunner(app)
    await runner.setup()
    await web.TCPSite(runner,"127.0.0.1",8898).start()
    results=[]
    try:
        async with async_playwright() as p:
            browser=await p.chromium.launch(headless=True,args=["--no-sandbox","--disable-dev-shm-usage"])
            try:
                # prewarm shared server assets by an unrecorded unthrottled navigation
                warm=await browser.new_page()
                try: await warm.goto("http://127.0.0.1:8898/before/archive/",wait_until="domcontentloaded",timeout=120000)
                except Exception as ex: print("WARM_WARNING",str(ex)[:300],flush=True)
                await warm.close()
                for pair in range(1,6):
                    order=["before","after"] if pair%2 else ["after","before"]
                    for v in order:
                        print("START",pair,v,flush=True)
                        result=await test_once(browser,v,pair)
                        results.append(result)
                        (OUT/"results.json").write_text(json.dumps(results,indent=2))
                        print("RESULT",json.dumps({
                            "pair":pair,"variant":v,"fcp":result["metrics"].get("fcp"),
                            "lcp":result["metrics"].get("lcp"),"cls":result["metrics"].get("cls"),
                            "frames":len([f for f in result["frames"] if "file" in f]),
                            "network_errors":len(result["network_errors"]),
                            "navigation_error":result["navigation_error"]}),flush=True)
            finally: await browser.close()
    finally: await runner.cleanup()
    for name in ("before","after"):
        values=[q["metrics"].get("fcp") for q in results if q["variant"]==name
                and isinstance(q["metrics"].get("fcp"),(int,float))]
        print("SUMMARY",name,"runs",len(values),"fcp_median_ms",
              statistics.median(values) if values else "missing",flush=True)
    print("NOTE: FCP is NOT first visible readable main content. Review JPEG frames.",flush=True)
asyncio.run(main())
