package servecontrol

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/servefix"
)

// NewWebHandler serves a local view of the same runtime consumed by the TUI.
// Callers must bind the HTTP server to a loopback address for local use.
func NewWebHandler(runtime *Runtime) http.Handler {
	return NewWebHandlerWithSourceRoot(runtime, "")
}

// NewWebHandlerWithSourceRoot adds previewable, stale-safe source fixes when
// a site root is available.
func NewWebHandlerWithSourceRoot(runtime *Runtime, sourceRoot string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/_markata/", dashboardHandler)
	mux.HandleFunc("/_markata/api/state", stateHandler(runtime))
	mux.HandleFunc("/_markata/api/actions", actionsHandler(runtime))
	mux.HandleFunc("/_markata/api/fixes/preview", fixPreviewHandler(sourceRoot))
	mux.HandleFunc("/_markata/api/fixes/apply", fixApplyHandler(sourceRoot))
	return mux
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/_markata/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if _, err := w.Write([]byte(webHTML)); err != nil {
		return
	}
}

func stateHandler(runtime *Runtime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if err := json.NewEncoder(w).Encode(runtime.Snapshot()); err != nil {
			return
		}
	}
}

func actionsHandler(runtime *Runtime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		if !sameOriginLocalRequest(r, scheme) {
			http.Error(w, "action requires same-origin request", http.StatusForbidden)
			return
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			http.Error(w, "expected application/json", http.StatusUnsupportedMediaType)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		var action ActionRequest
		if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
			http.Error(w, "invalid action", 400)
			return
		}
		if err := runtime.Trigger(action); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

func fixPreviewHandler(sourceRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !sameOriginLocalRequest(r, requestScheme(r)) {
			http.Error(w, "fix preview requires a same-origin local request", http.StatusForbidden)
			return
		}
		var request fixRequest
		if err := decodeJSON(w, r, &request); err != nil {
			http.Error(w, "invalid fix request", 400)
			return
		}
		if sourceRoot == "" {
			http.Error(w, "source fixes are unavailable", 404)
			return
		}
		plan, err := servefix.PlanFile(sourceRoot, request.Path)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		original, err := os.ReadFile(plan.File)
		if err != nil {
			http.Error(w, "could not read fix source", 400)
			return
		}
		selection := request.Selection
		if len(selection.IDs) == 0 && len(selection.Categories) == 0 && !selection.All {
			selection.All = true
		}
		preview, edits, err := servefix.Preview(plan, original, selection)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if err := json.NewEncoder(w).Encode(fixPreview{Path: request.Path, Digest: plan.Digest, Edits: edits, Before: string(original), After: string(preview), Selection: selection}); err != nil {
			return
		}
	}
}

func fixApplyHandler(sourceRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !sameOriginLocalRequest(r, requestScheme(r)) {
			http.Error(w, "fix apply requires a same-origin local request", http.StatusForbidden)
			return
		}
		var request fixApplyRequest
		if err := decodeJSON(w, r, &request); err != nil {
			http.Error(w, "invalid fix request", 400)
			return
		}
		if sourceRoot == "" {
			http.Error(w, "source fixes are unavailable", 404)
			return
		}
		plan, err := servefix.PlanFile(sourceRoot, request.Path)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if plan.Digest != request.Digest {
			http.Error(w, servefix.ErrStale.Error(), http.StatusConflict)
			return
		}
		applied, err := servefix.Apply(sourceRoot, plan, request.Selection)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(map[string]any{"applied": applied}); err != nil {
			return
		}
	}
}

type fixRequest struct {
	Path      string             `json:"path"`
	Selection servefix.Selection `json:"selection"`
}
type fixApplyRequest struct {
	Path      string             `json:"path"`
	Selection servefix.Selection `json:"selection"`
	Digest    string             `json:"digest"`
}
type fixPreview struct {
	Path      string             `json:"path"`
	Digest    string             `json:"digest"`
	Edits     []servefix.Edit    `json:"edits"`
	Before    string             `json:"before"`
	After     string             `json:"after"`
	Selection servefix.Selection `json:"selection"`
}

func requestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return fmt.Errorf("expected application/json")
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	return json.NewDecoder(r.Body).Decode(dst)
}

func sameOriginLocalRequest(r *http.Request, scheme string) bool {
	if !loopbackAddress(r.RemoteAddr) || !localHost(r.Host) {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return r.Header.Get("Sec-Fetch-Site") == "" || r.Header.Get("Sec-Fetch-Site") == "same-origin"
	}
	parsed, err := url.Parse(origin)
	return err == nil && parsed.Scheme == scheme && parsed.Host == r.Host
}

func loopbackAddress(address string) bool {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func localHost(address string) bool {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	} else if port != "" {
		if _, err := strconv.Atoi(port); err != nil {
			return false
		}
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

const webHTML = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Serve Control Center · Markata</title>
<style>
:root{--paper:#f4f1e9;--card:#fffdf7;--ink:#26302f;--muted:#677471;--line:#d5dbd3;--accent:#d45531;--green:#287e62;--gold:#956819;--red:#b74742;--mono:ui-monospace,'SFMono-Regular',Consolas,'Liberation Mono',monospace;--sans:ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif}
*{box-sizing:border-box}body{margin:0;background:var(--paper);color:var(--ink);font:15px/1.45 var(--sans)}button,input{font:inherit}button{cursor:pointer}button:focus-visible,input:focus-visible,a:focus-visible{outline:3px solid var(--accent);outline-offset:2px}.app{max-width:1600px;min-height:100vh;margin:auto;display:grid;grid-template-columns:220px minmax(0,1fr)}.rail{height:100vh;position:sticky;top:0;background:var(--ink);color:var(--paper);padding:25px 15px;display:flex;flex-direction:column}.brand{padding:0 12px 22px;border-bottom:1px solid var(--muted);font-size:19px;font-weight:700;letter-spacing:-.04em}.brand b{display:inline-grid;place-items:center;width:28px;height:28px;background:var(--accent);font:500 17px var(--mono);transform:rotate(-8deg);margin-right:8px}.brand small{display:block;margin:11px 0 0 38px;color:var(--paper);opacity:.72;font:10px var(--mono);letter-spacing:.11em;text-transform:uppercase}.nav{display:grid;gap:5px;margin-top:23px}.tab{border:0;background:transparent;color:var(--paper);opacity:.76;text-align:left;padding:12px;border-radius:5px;font:12px var(--mono);display:flex;justify-content:space-between}.tab:hover{background:var(--muted);opacity:1}.tab.active{background:var(--paper);color:var(--ink);font-weight:500;opacity:1}.rail-foot{margin-top:auto;padding:16px 12px;border-top:1px solid var(--muted);color:var(--paper);opacity:.72;font:11px/1.8 var(--mono)}.main{min-width:0;padding:30px clamp(18px,3vw,46px) 70px}.mast{display:flex;justify-content:space-between;gap:20px;align-items:start;border-bottom:1px solid #aeb9af;padding-bottom:25px}.eyebrow,.kicker,.metric-label,.detail-label{font:10px var(--mono);text-transform:uppercase;letter-spacing:.13em}.eyebrow,.kicker{color:var(--accent)}h1{font-size:clamp(28px,3vw,42px);letter-spacing:-.055em;line-height:1;margin:7px 0}h2{font-size:24px;letter-spacing:-.04em;margin:3px 0 0}h3{font-size:21px;line-height:1.2;letter-spacing:-.04em;margin:9px 0;overflow-wrap:anywhere}.mast p{color:var(--muted);margin:8px 0 0}.mast-actions{display:flex;gap:9px;align-items:center;flex-wrap:wrap;justify-content:end}.button{border:1px solid #aeb9af;background:var(--card);border-radius:4px;padding:8px 11px;color:var(--ink);font-weight:600;font-size:12px}.button:hover{color:var(--accent);border-color:var(--accent)}.primary{background:var(--accent);border-color:var(--accent);color:white}.primary:hover{background:#ae3f21;color:white}.site-link{font:11px var(--mono);color:var(--green);text-decoration:none}.site-link:hover{text-decoration:underline}.metrics{margin-top:24px;background:var(--card);border:1px solid var(--line);border-radius:8px;display:grid;grid-template-columns:repeat(5,minmax(0,1fr));overflow:hidden}.metric{padding:18px;border-right:1px solid var(--line);min-width:0}.metric:last-child{border:0}.metric-label{color:var(--muted)}.metric-value{font-size:26px;font-weight:600;letter-spacing:-.04em;line-height:1.2;margin-top:7px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.metric-note{color:var(--muted);font:10px var(--mono);overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.metric.warning .metric-value{color:var(--gold)}.metric.failed .metric-value{color:var(--red)}.metric.success .metric-value{color:var(--green)}.content-head{display:flex;justify-content:space-between;align-items:end;gap:15px;margin:31px 0 13px}.filter{width:min(280px,100%);padding:9px 10px;background:var(--card);border:1px solid #aeb9af;border-radius:4px;font:12px var(--mono)}.workspace{display:grid;grid-template-columns:minmax(0,1fr) minmax(280px,36%);gap:15px;align-items:start}.list,.detail{background:var(--card);border:1px solid var(--line);border-radius:8px;min-width:0;overflow:hidden}.list{min-height:270px}.detail{position:sticky;top:20px;max-height:calc(100vh - 40px);overflow:auto}.item{display:block;width:100%;text-align:left;border:0;border-bottom:1px solid var(--line);padding:15px 18px;background:transparent;color:var(--ink)}.item:last-child{border:0}.item:hover{background:#f6f5ed}.item.selected{background:#eaf0e9;border-left:3px solid var(--green);padding-left:15px}.item-top{display:flex;gap:8px;align-items:center;min-width:0}.item-title{font-weight:600;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.item-meta{margin:5px 0 0 20px;color:var(--muted);font:11px var(--mono);overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.item-tail{margin-left:auto;color:var(--muted);font:10px var(--mono);white-space:nowrap}.pill{padding:3px 6px;border-radius:3px;background:#e7e9e2;color:#53615b;font:10px var(--mono);text-transform:uppercase}.pill.success,.pill.ready{background:#dcefe6;color:var(--green)}.pill.running{background:#dceaf4;color:#356b91}.pill.warning,.pill.warn{background:#f6edcd;color:var(--gold)}.pill.failed,.pill.error{background:#f8e4e0;color:var(--red)}.detail-head{padding:21px;border-bottom:1px solid var(--line)}.detail-path{font:11px var(--mono);color:var(--green);overflow-wrap:anywhere}.detail-body{padding:18px 21px}.detail-label{color:var(--muted);margin:17px 0 6px}.detail-label:first-child{margin-top:0}.detail-text{white-space:pre-wrap;overflow-wrap:anywhere}.detail-actions{display:flex;gap:7px;flex-wrap:wrap;margin-top:17px}.step,.log-line{border-bottom:1px solid var(--line);padding:7px 0;font:11px/1.5 var(--mono);overflow-wrap:anywhere}.step{display:flex;gap:8px}.step:last-child,.log-line:last-child{border:0}.time{color:var(--muted);margin-right:8px}.log-line.error{color:var(--red)}.log-line.warning{color:var(--gold)}.empty{text-align:center;padding:45px 20px;color:var(--muted)}.empty strong{display:block;font-size:18px;color:var(--ink)}.notice{display:none;margin-top:14px;padding:10px 12px;background:#f8e4e0;color:var(--red);border-radius:4px;font:12px var(--mono)}.notice.show{display:block}.mobile-nav{display:none}.sr-only{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap}
@media(max-width:1050px){.metrics{grid-template-columns:repeat(3,1fr)}.metric:nth-child(3){border:0}.metric:nth-child(n+4){border-top:1px solid var(--line)}}@media(max-width:780px){.app{display:block}.rail{display:none}.main{padding:20px 15px 80px}.mast{flex-wrap:wrap}.mast-actions{justify-content:start}.metrics{grid-template-columns:repeat(2,1fr)}.metric:nth-child(3){border-right:1px solid var(--line)}.metric:nth-child(even){border-right:0}.metric:nth-child(n+3){border-top:1px solid var(--line)}.workspace{display:block}.detail{position:static;max-height:none;margin-top:14px}.mobile-nav{display:flex;position:fixed;bottom:0;left:0;right:0;gap:2px;padding:6px;background:var(--ink)}.mobile-nav .tab{font-size:10px;justify-content:center;flex:1;padding:10px 2px}.mobile-nav .tab span:last-child{display:none}.content-head{flex-wrap:wrap}.filter{width:100%}}
</style></head><body><div class="app"><aside class="rail"><div class="brand"><b>M</b>markata<small>Serve control center</small></div><nav class="nav" id="desktop-nav" aria-label="Sections"></nav><div class="rail-foot">LOCAL SESSION<br><span id="sync-label">Connecting…</span></div></aside><main class="main"><header class="mast"><div><div class="eyebrow">Development / live session</div><h1>Serve Control Center</h1><p>Build activity, page health, and the details behind every warning.</p></div><div class="mast-actions"><a class="site-link" id="site-link" hidden target="_blank" rel="noopener">Open site ↗</a><button class="button primary" id="build-button" type="button">Trigger build ↗</button></div></header><section class="metrics" id="metrics" aria-label="Current state"></section><div class="notice" id="notice" role="alert"></div><div class="content-head"><div><div class="kicker" id="kicker">Activity</div><h2 id="title">Jobs</h2></div><label><span class="sr-only">Filter current section</span><input class="filter" id="filter" type="search" placeholder="Filter this view…"></label></div><div class="workspace"><section class="list" id="list" aria-label="Results"></section><aside class="detail" id="detail" aria-label="Selected item"></aside></div></main></div><nav class="mobile-nav" id="mobile-nav" aria-label="Sections"></nav>
<script>
(() => {
'use strict';
const state={snapshot:null,section:'jobs',selected:'',filter:''},tabs=[['jobs','Jobs'],['diagnostics','Diagnostics'],['pages','Pages'],['feeds','Feeds'],['logs','Logs']];
const el=id=>document.getElementById(id),arr=x=>Array.isArray(x)?x:[],esc=x=>String(x==null?'':x).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c])),when=x=>x&&!isNaN(new Date(x))?new Date(x).toLocaleTimeString():'';
function field(name,value){return value==null||value===''?'':'<div class="detail-label">'+esc(name)+'</div><div class="detail-text">'+esc(value)+'</div>'}
function metric(name,value,note,kind){return '<div class="metric '+esc(kind||'')+'"><div class="metric-label">'+esc(name)+'</div><div class="metric-value">'+esc(value)+'</div><div class="metric-note">'+esc(note)+'</div></div>'}
function key(item,i){return state.section==='jobs'?'job:'+item.id:state.section==='pages'?'page:'+(item.path||item.file||item.slug):state.section==='feeds'?'feed:'+item.name:state.section+':'+i}
function feedPath(item){const name=String(item.path||item.name||'').replace(/^\/+|\/+$/g,'');return name?'/'+name+'/':'/'}
function title(item){return state.section==='jobs'?item.name||item.type||item.id:state.section==='diagnostics'?item.message||item.code:state.section==='pages'?item.path||item.file||item.slug:state.section==='feeds'?item.title||feedPath(item):item.message}
function count(section){return arr((state.snapshot||{})[section]).length}
function nav(target){target.innerHTML=tabs.map(([id,name])=>'<button class="tab '+(state.section===id?'active':'')+'" data-section="'+id+'" type="button">'+name+'<span>'+count(id)+'</span></button>').join('')}
function status(){const s=state.snapshot||{},server=s.server||{},site=s.site||{},jobs=arr(s.jobs),diags=arr(s.diagnostics),warnings=diags.filter(d=>d.severity==='warning').length,errors=diags.filter(d=>d.severity==='error').length,serverState=server.status||(state.snapshot?'ready':'connecting'),siteState=site.status||'waiting';el('metrics').innerHTML=metric('Server',serverState,server.address||'Local session',serverState==='failed'?'failed':'success')+metric('Site',siteState,(site.page_count||0)+' pages'+(site.message?' · '+site.message:''),siteState==='failed'?'failed':'success')+metric('Jobs',jobs.length,jobs.filter(j=>j.state==='running').length+' running')+metric('Warnings',warnings,'Session history',warnings?'warning':'')+metric('Errors',errors,'Session history',errors?'failed':'success');const url=server.address?(String(server.address).startsWith('http')?server.address:'http://'+server.address):'';el('site-link').hidden=!url;if(url)el('site-link').href=url}
function applyTheme(theme){const names={background:'--paper',surface:'--card',text:'--ink',muted:'--muted',primary:'--accent',success:'--green',warning:'--gold',error:'--red',border:'--line'};for(const [key,variable] of Object.entries(names)){const color=theme&&theme[key];if(color&&CSS.supports('color',color))document.documentElement.style.setProperty(variable,color)}}
function knownPage(path){return arr((state.snapshot||{}).pages).some(p=>p.path===path)}
function row(item,k){const kind=state.section==='jobs'?item.state:state.section==='diagnostics'?item.severity:state.section==='pages'?(item.state||item.status):state.section==='feeds'?(item.status||'feed'):item.level;const meta=state.section==='jobs'?[item.trigger,item.id].filter(Boolean).join(' · '):state.section==='diagnostics'?[item.code,item.file||item.page,item.line?':'+item.line:''].filter(Boolean).join(' · '):state.section==='pages'?(item.file||item.path||''):state.section==='feeds'?[item.path,arr(item.entries).length+' posts'].filter(Boolean).join(' · '):(item.job_id||'');return '<button class="item '+(state.selected===k?'selected':'')+'" data-key="'+esc(k)+'" type="button"><div class="item-top"><span class="pill '+esc(kind)+'">'+esc(kind||'item')+'</span><span class="item-title">'+esc(title(item))+'</span><span class="item-tail">'+esc(when(item.ended_at||item.time))+'</span></div><div class="item-meta">'+esc(meta)+'</div></button>'}
function logLine(l){return '<div class="log-line '+esc(l.level)+'"><span class="time">'+esc(when(l.time))+'</span>'+esc(l.message)+'</div>'}
function detail(item){if(!item){el('detail').innerHTML='<div class="empty"><strong>Nothing selected</strong>Choose an item to inspect it.</div>';return}let head='',body='';if(state.section==='jobs'){head='<span class="pill '+esc(item.state)+'">'+esc(item.state)+'</span><h3>'+esc(title(item))+'</h3><div class="detail-path">'+esc(item.id)+'</div>';body=field('Trigger',item.trigger)+field('Type',item.type)+field('Started',item.started_at)+field('Finished',item.ended_at);if(arr(item.steps).length)body+='<div class="detail-label">Steps</div>'+item.steps.map(x=>'<div class="step"><span class="pill '+esc(x.state)+'">'+esc(x.state)+'</span>'+esc(x.name||x.id)+'</div>').join('');if(arr(item.pages).length)body+='<div class="detail-label">Affected pages</div>'+item.pages.map(path=>'<div class="step">'+(knownPage(path)?'<button class="button" type="button" data-page="'+esc(path)+'">'+esc(path)+' →</button>':esc(path))+'</div>').join('');if(arr(item.diagnostics).length)body+='<div class="detail-label">Diagnostics</div>'+item.diagnostics.map(d=>'<div class="step"><span class="pill '+esc(d.severity)+'">'+esc(d.code||d.severity)+'</span>'+esc(d.message)+'</div>').join('');if(arr(item.logs).length)body+='<div class="detail-label">Recent logs</div>'+item.logs.slice(-20).map(logLine).join('');body+='<div class="detail-actions"><button class="button" type="button" data-rerun="'+esc(item.id)+'">Rerun job ↗</button><button class="button" type="button" data-job-logs="'+esc(item.id)+'">View logs →</button></div>'}
else if(state.section==='diagnostics'){head='<span class="pill '+esc(item.severity)+'">'+esc(item.severity)+'</span><h3>'+esc(item.message)+'</h3><div class="detail-path">'+esc([item.file||item.page,item.line].filter(Boolean).join(':'))+'</div>';body=field('Code',item.code)+field('Explanation',item.explanation)+field('Suggested fix',item.suggested_fix)+field('Job',item.job_id)+field('Step',item.step_id);body+='<div class="detail-actions">';if(item.job_id)body+='<button class="button" type="button" data-job="'+esc(item.job_id)+'">View job →</button>';if(knownPage(item.page||item.file))body+='<button class="button" type="button" data-page="'+esc(item.page||item.file)+'">View page →</button>';if(/\.(md|markdown)$/i.test(item.file||item.page||''))body+='<button class="button" type="button" data-fix-preview="'+esc(item.file||item.page)+'">Preview safe fixes</button>';body+='</div>'}
else if(state.section==='pages'){const path=item.path||item.file||item.slug||'',diags=arr(item.diagnostics);head='<span class="pill '+esc(item.state||item.status)+'">'+esc(item.state||item.status||'page')+'</span><h3>'+esc(path)+'</h3><div class="detail-path">'+esc(item.file||path)+'</div>';body=field('Last job',item.job_id||item.last_job_id)+'<div class="detail-label">Diagnostics</div>'+(diags.length?diags.map(d=>'<div class="step"><span class="pill '+esc(d.severity)+'">'+esc(d.code||d.severity)+'</span>'+esc(d.message)+'</div>').join(''):'<div class="detail-text">No current diagnostics.</div>');body+='<div class="detail-actions">';if(/\.(md|markdown)$/i.test(path))body+='<button class="button" type="button" data-rebuild="'+esc(path)+'">Rebuild page ↗</button>';if(item.url){try{const url=new URL(item.url,location.origin);if(url.origin===location.origin)body+='<a class="button" href="'+esc(url.href)+'" target="_blank" rel="noopener">Open page ↗</a>'}catch{}}body+='</div>'}
else if(state.section==='feeds'){head='<span class="pill '+esc(item.status||'feed')+'">'+esc(item.status||'feed')+'</span><h3>'+esc(title(item))+'</h3><div class="detail-path">'+esc(feedPath(item))+'</div>';body=field('Name',item.name)+field('Source/config',item.path)+field('Entries',arr(item.entries).length);if(arr(item.entries).length)body+='<div class="detail-label">Posts</div>'+item.entries.slice(0,250).map(entry=>'<div class="step">'+(knownPage(entry.path)?'<button class="button" type="button" data-page="'+esc(entry.path)+'">'+esc(entry.title||entry.path)+' →</button>':esc(entry.title||entry.path))+'</div>').join('')}
else{head='<span class="pill '+esc(item.level)+'">'+esc(item.level||'log')+'</span><h3>Log entry</h3><div class="detail-path">'+esc(when(item.time))+'</div>';body=field('Message',item.message)+field('Job',item.job_id)+field('Step',item.step_id)}
el('detail').innerHTML='<div class="detail-head">'+head+'</div><div class="detail-body">'+body+'</div>'}
function list(){const query=state.filter.toLowerCase(),matches=arr((state.snapshot||{})[state.section]).map((item,i)=>({item,k:key(item,i)})).filter(({item})=>!query||JSON.stringify(item).toLowerCase().includes(query)),items=state.section==='logs'?matches.slice(-200):matches;if(state.section==='logs')el('kicker').textContent='Session output · latest '+items.length+' of '+matches.length;if(!items.some(x=>x.k===state.selected))state.selected=items.length?items[0].k:'';el('list').innerHTML=items.length?items.map(({item,k})=>row(item,k)).join(''):'<div class="empty"><strong>No matching items</strong>Try another filter or wait for the next build.</div>';detail((items.find(x=>x.k===state.selected)||{}).item)}
function render(){nav(el('desktop-nav'));nav(el('mobile-nav'));status();el('title').textContent=tabs.find(x=>x[0]===state.section)[1];el('kicker').textContent=state.section==='diagnostics'?'Problems to resolve':state.section==='pages'?'Built content':state.section==='feeds'?'Generated feeds':state.section==='logs'?'Session output':'Activity';list()}
function notice(message){el('notice').textContent=message;el('notice').classList.toggle('show',!!message)}
async function poll(){try{const res=await fetch('/_markata/api/state',{cache:'no-store'});if(!res.ok)throw Error('State request failed ('+res.status+')');state.snapshot=await res.json();applyTheme(state.snapshot.theme);el('sync-label').textContent='Live · updated '+when(new Date());notice('');render()}catch(err){el('sync-label').textContent='Connection interrupted';notice(err.message)}}
async function action(request){try{const res=await fetch('/_markata/api/actions',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(request)});if(!res.ok)throw Error((await res.text()).trim()||'Action failed');notice('');await poll()}catch(err){notice(err.message)}}
async function previewFix(path){try{const res=await fetch('/_markata/api/fixes/preview',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({path,selection:{all:true}})});if(!res.ok)throw Error((await res.text()).trim()||'Could not preview fixes');const preview=await res.json();if(!preview.edits.length){notice('No safe automatic fixes are available for this source.');return}state.fixPreview=preview;const body=el('detail').querySelector('.detail-body');body.insertAdjacentHTML('beforeend','<div class="detail-label">Review changes before applying</div><pre style="white-space:pre-wrap;overflow-wrap:anywhere;max-height:280px;overflow:auto">'+esc(preview.edits.map(x=>x.message+'\n− '+x.before+'\n+ '+x.after).join('\n\n'))+'</pre><button class="button primary" type="button" data-fix-apply>Apply previewed fixes</button>');notice('Preview only · the source file has not changed.')}catch(err){notice(err.message)}}
async function applyFix(){const preview=state.fixPreview;if(!preview)return;try{const res=await fetch('/_markata/api/fixes/apply',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({path:preview.path,digest:preview.digest,selection:preview.selection})});if(!res.ok)throw Error((await res.text()).trim()||'Could not apply fixes');state.fixPreview=null;notice('Fixes applied · rebuilding');await fetch('/_markata/api/actions',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({kind:'build'})});await poll()}catch(err){notice(err.message)}}
document.addEventListener('click',e=>{const t=e.target.closest('[data-section]');if(t){state.section=t.dataset.section;state.selected='';state.filter='';el('filter').value='';render();return}const select=e.target.closest('[data-key]');if(select){state.selected=select.dataset.key;list();return}const preview=e.target.closest('[data-fix-preview]');if(preview){previewFix(preview.dataset.fixPreview);return}if(e.target.closest('[data-fix-apply]')){applyFix();return}const rerun=e.target.closest('[data-rerun]');if(rerun){action({kind:'rerun',job_id:rerun.dataset.rerun});return}const rebuild=e.target.closest('[data-rebuild]');if(rebuild){action({kind:'rebuild-page',page:rebuild.dataset.rebuild});return}const logs=e.target.closest('[data-job-logs]');if(logs){state.section='logs';state.filter=logs.dataset.jobLogs;el('filter').value=state.filter;render();return}const page=e.target.closest('[data-page]');if(page){state.section='pages';state.filter=page.dataset.page;el('filter').value=state.filter;state.selected='page:'+page.dataset.page;render();return}const job=e.target.closest('[data-job]');if(job){state.section='jobs';state.filter=job.dataset.job;el('filter').value=state.filter;render()}});
el('build-button').addEventListener('click',()=>action({kind:'build'}));el('filter').addEventListener('input',e=>{state.filter=e.target.value;list()});poll();setInterval(poll,2000);
})();
</script></body></html>`
