/* markata-go serve settings sidebar.
 * Injected only by `markata-go serve`; production builds never load it.
 * Lists every config setting. Edits preview live (the dev server rebuilds
 * with them held in memory); Bake writes them into the config file that
 * already holds each setting, and Reset drops them.
 */
(function () {
  'use strict';
  if (window.markataDevSettings) return;

  var endpoint = window.__markataSettingsEndpoint;
  if (!endpoint) return;
  var previewEndpoint = endpoint + '/preview';

  var STATE_KEY = 'markata-dev-settings:state';
  var FLASH_KEY = 'markata-dev-settings:flash';

  var CSS = [
    ':host{all:initial;--bg:var(--color-surface,#1d2029);--fg:var(--color-text,#e8e8ee);--muted:var(--color-text-muted,#9aa0ad);',
    '--line:var(--color-border,rgba(127,127,127,.28));--accent:var(--color-primary,#7aa2f7);--page:var(--color-background,#15171e);',
    '--ok:var(--color-success,#4caf7a);--bad:var(--color-error,#e5534b);--warn:var(--color-warning,#d9a43a);}',
    '*{box-sizing:border-box}',
    '[hidden]{display:none!important}',
    '.toggle{position:fixed;left:max(12px,env(safe-area-inset-left));bottom:max(12px,env(safe-area-inset-bottom));z-index:2147483000;',
    'width:40px;height:40px;border-radius:50%;border:1px solid var(--line);background:var(--bg);color:var(--fg);display:grid;place-items:center;',
    'cursor:pointer;opacity:.72;box-shadow:0 6px 18px rgba(0,0,0,.28);transition:opacity .15s,transform .15s;padding:0}',
    '.toggle:hover,.toggle:focus-visible{opacity:1}',
    '.toggle:hover svg{transform:rotate(30deg)}',
    '.toggle svg{transition:transform .2s}',
    '.toggle:focus-visible,button:focus-visible,input:focus-visible,select:focus-visible,textarea:focus-visible,summary:focus-visible{outline:2px solid var(--accent);outline-offset:2px}',
    '.toggle[aria-expanded="true"]{opacity:1}',
    '.toggle .dot{position:absolute;top:-3px;right:-3px;min-width:17px;height:17px;padding:0 4px;border-radius:9px;background:var(--accent);color:var(--page);',
    'font:600 10px/17px system-ui,sans-serif;text-align:center}',
    '.panel{position:fixed;top:0;right:0;bottom:0;z-index:2147483001;width:min(440px,100vw);display:flex;flex-direction:column;',
    'background:var(--bg);color:var(--fg);border-left:1px solid var(--line);box-shadow:-18px 0 40px rgba(0,0,0,.3);',
    'font:13px/1.45 system-ui,-apple-system,"Segoe UI",Roboto,sans-serif;transform:translateX(102%);visibility:hidden;',
    'transition:transform .2s ease,visibility 0s linear .2s}',
    '.panel.open{transform:none;visibility:visible;transition:transform .2s ease}',
    '@media (prefers-reduced-motion:reduce){.panel,.panel.open,.toggle,.toggle svg{transition:none}}',
    '@media (max-width:640px){.panel{width:100vw;border-left:0;height:100dvh}.panel.open~.toggle{display:none}}',
    'header{padding:14px 16px 10px;border-bottom:1px solid var(--line);display:grid;gap:10px}',
    '.titlebar{display:flex;align-items:center;gap:8px}',
    'h2{margin:0;font-size:15px;font-weight:650;flex:1}',
    '.badge{font-size:10px;font-weight:600;letter-spacing:.04em;text-transform:uppercase;color:var(--accent);border:1px solid currentColor;border-radius:4px;padding:1px 5px}',
    '.icon{background:none;border:0;color:var(--muted);cursor:pointer;padding:8px;border-radius:6px;display:grid;place-items:center}',
    '.icon:hover{color:var(--fg);background:rgba(127,127,127,.12)}',
    '.tools{display:flex;gap:8px}',
    'input,select,textarea{font:inherit;color:var(--fg);background:var(--page);border:1px solid var(--line);border-radius:6px;padding:7px 9px;width:100%;min-height:34px}',
    '@media (max-width:640px){input,select,textarea{font-size:16px}}',
    'textarea{resize:vertical;min-height:60px}',
    '.tools select{width:auto;flex:none}',
    '.files{color:var(--muted);font-size:11.5px;overflow-wrap:anywhere}',
    '.body{flex:1;overflow:auto;overscroll-behavior:contain;padding:0 0 12px}',
    'details{border-bottom:1px solid var(--line)}',
    'summary{list-style:none;cursor:pointer;padding:11px 16px;display:flex;align-items:center;gap:10px;font-weight:600;position:sticky;top:0;background:var(--bg);z-index:1}',
    'summary::-webkit-details-marker{display:none}',
    'summary::before{content:"";width:6px;height:6px;border-right:1.5px solid var(--muted);border-bottom:1.5px solid var(--muted);transform:rotate(-45deg);transition:transform .15s}',
    'details[open]>summary::before{transform:rotate(45deg)}',
    'summary .count{margin-left:auto;color:var(--muted);font-weight:400;font-size:11.5px}',
    'summary .count.changed{color:var(--accent);font-weight:600}',
    '.field{padding:9px 16px 11px 18px;border-left:2px solid transparent;display:grid;gap:5px}',
    '.field.changed{border-left-color:var(--accent);background:rgba(127,127,127,.06)}',
    '.row{display:flex;align-items:center;gap:10px}',
    '.label{font-weight:550;flex:1;min-width:0}',
    '.key{font:11px/1.3 ui-monospace,SFMono-Regular,Menlo,monospace;color:var(--muted);overflow-wrap:anywhere}',
    '.doc{color:var(--muted);font-size:12px}',
    '.meta{display:flex;flex-wrap:wrap;gap:4px 10px;font-size:11px;color:var(--muted)}',
    '.meta .set{color:var(--fg)}',
    '.meta .env{color:var(--warn)}',
    '.readonly{color:var(--muted);font-style:italic}',
    '.revert{background:none;border:0;color:var(--accent);cursor:pointer;font:inherit;font-size:11px;padding:0}',
    '.switch{position:relative;width:38px;height:22px;flex:none}',
    '.switch input{position:absolute;inset:0;opacity:0;margin:0;cursor:pointer;min-height:0;width:100%;height:100%}',
    '.switch span{position:absolute;inset:0;border-radius:11px;background:var(--line);transition:background .15s;pointer-events:none}',
    '.switch span::after{content:"";position:absolute;top:3px;left:3px;width:16px;height:16px;border-radius:50%;background:var(--fg);transition:transform .15s}',
    '.switch input:checked+span{background:var(--accent)}',
    '.switch input:checked+span::after{transform:translateX(16px);background:var(--page)}',
    '.switch input:indeterminate+span::after{transform:translateX(8px);opacity:.55}',
    '.switch input:focus-visible+span{outline:2px solid var(--accent);outline-offset:2px}',
    'footer{border-top:1px solid var(--line);padding:10px 16px max(10px,env(safe-area-inset-bottom));display:grid;gap:8px}',
    '.actions{display:flex;align-items:center;gap:8px;min-width:0}',
    '.summary{flex:1;color:var(--muted);white-space:nowrap}',
    'button.btn{font:inherit;font-weight:600;border-radius:6px;padding:7px 12px;cursor:pointer;border:1px solid var(--line);background:none;color:var(--fg);min-height:34px;min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}',
    'button.btn.primary{background:var(--accent);border-color:var(--accent);color:var(--page)}',
    'button.btn.primary.armed{background:var(--warn);border-color:var(--warn)}',
    'button.btn:disabled{opacity:.45;cursor:default}',
    '.status{font-size:12px;overflow-wrap:anywhere}',
    '.status:empty{display:none}',
    '.status.error{color:var(--bad)}',
    '.status.ok{color:var(--ok)}',
    '.status.warn{color:var(--warn)}',
    '.empty{padding:28px 16px;color:var(--muted);text-align:center}',
    '.field.invalid{border-left-color:var(--bad)}',
    '.field.invalid input,.field.invalid select,.field.invalid textarea{border-color:var(--bad)}',
    '.error{color:var(--bad);font-size:12px}',
    '.hint{color:var(--muted);font-size:11.5px}',
    '.chip{position:fixed;left:calc(max(12px,env(safe-area-inset-left)) + 48px);bottom:max(12px,env(safe-area-inset-bottom));z-index:2147483000;',
    'display:flex;align-items:center;gap:6px;padding:4px 4px 4px 12px;min-height:40px;border-radius:20px;border:1px solid var(--line);background:var(--bg);color:var(--fg);',
    'font:12.5px/1.2 system-ui,-apple-system,"Segoe UI",Roboto,sans-serif;box-shadow:0 6px 18px rgba(0,0,0,.28);max-width:calc(100vw - 72px)}',
    '.chip span{white-space:nowrap;overflow:hidden;text-overflow:ellipsis}',
    '.chip button{font:inherit;font-weight:600;border-radius:16px;padding:6px 10px;cursor:pointer;border:1px solid var(--line);background:none;color:var(--fg);min-height:30px}',
    '.chip button.primary{background:var(--accent);border-color:var(--accent);color:var(--page)}',
    '.panel.open~.chip{display:none}'
  ].join('');

  var ICON_GEAR = '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"/></svg>';
  var ICON_CLOSE = '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M18 6 6 18M6 6l12 12"/></svg>';

  function has(obj, key) { return Object.prototype.hasOwnProperty.call(obj, key); }
  function readJSON(key, fallback) {
    try {
      var raw = sessionStorage.getItem(key);
      return raw ? JSON.parse(raw) : fallback;
    } catch (e) {
      return fallback;
    }
  }
  function writeJSON(key, value) {
    try {
      if (value === null) sessionStorage.removeItem(key);
      else sessionStorage.setItem(key, JSON.stringify(value));
    } catch (e) { /* storage unavailable */ }
  }
  function el(tag, attrs, children) {
    var node = document.createElement(tag);
    if (attrs) {
      Object.keys(attrs).forEach(function (name) {
        var value = attrs[name];
        if (value === undefined || value === null || value === false) return;
        if (name === 'text') node.textContent = value;
        else if (name === 'className') node.className = value;
        else node.setAttribute(name, value === true ? '' : value);
      });
    }
    (children || []).forEach(function (child) {
      if (child) node.appendChild(typeof child === 'string' ? document.createTextNode(child) : child);
    });
    return node;
  }
  function same(a, b) {
    return JSON.stringify(a === undefined ? null : a) === JSON.stringify(b === undefined ? null : b);
  }
  function sectionLabel(key) {
    if (!key) return 'Site';
    return key.replace(/_/g, ' ').replace(/^./, function (c) { return c.toUpperCase(); });
  }
  function plural(n, word) { return n + ' ' + word + (n === 1 ? '' : 's'); }

  var host = document.createElement('markata-dev-settings');
  var root = host.attachShadow({ mode: 'open' });
  var style = document.createElement('style');
  style.textContent = CSS;
  root.appendChild(style);

  var panel = el('aside', { className: 'panel', id: 'panel', role: 'dialog', 'aria-label': 'Site settings', 'aria-modal': 'false' });
  var toggle = el('button', { className: 'toggle', type: 'button', 'aria-expanded': 'false', 'aria-controls': 'panel',
    title: 'Site settings (serve only) \u2014 Alt+,', 'aria-label': 'Site settings' });
  toggle.innerHTML = ICON_GEAR;
  var dot = el('span', { className: 'dot', hidden: true, 'aria-hidden': 'true' });
  toggle.appendChild(dot);

  var closeBtn = el('button', { className: 'icon', type: 'button', 'aria-label': 'Close settings' });
  closeBtn.innerHTML = ICON_CLOSE;
  var search = el('input', { type: 'search', placeholder: 'Search settings', 'aria-label': 'Search settings', autocomplete: 'off', spellcheck: 'false' });
  var filter = el('select', { 'aria-label': 'Filter settings' }, [
    el('option', { value: 'all', text: 'All' }),
    el('option', { value: 'set', text: 'In config' }),
    el('option', { value: 'changed', text: 'Changed' })
  ]);
  var files = el('div', { className: 'files' });
  var body = el('div', { className: 'body' });
  var summaryText = el('span', { className: 'summary', text: 'No changes' });
  var discardBtn = el('button', { className: 'btn', type: 'button', text: 'Reset', disabled: true, title: 'Drop all unsaved changes and rebuild with your config files' });
  var bakeBtn = el('button', { className: 'btn primary', type: 'button', text: 'Bake', disabled: true, title: 'Write the previewed changes into your config files' });
  var status = el('div', { className: 'status', role: 'status', 'aria-live': 'polite' });

  panel.appendChild(el('header', null, [
    el('div', { className: 'titlebar' }, [el('h2', { text: 'Site settings' }), el('span', { className: 'badge', text: 'serve only' }), closeBtn]),
    el('div', { className: 'tools' }, [search, filter]),
    files
  ]));
  panel.appendChild(body);
  panel.appendChild(el('footer', null, [el('div', { className: 'actions' }, [summaryText, discardBtn, bakeBtn]), status]));
  var chipText = el('span');
  var chipReset = el('button', { type: 'button', text: 'Reset', title: 'Drop unsaved changes' });
  var chipReview = el('button', { className: 'primary', type: 'button', text: 'Review', title: 'Open settings to bake or reset' });
  var chip = el('div', { className: 'chip', role: 'status', hidden: true }, [chipText, chipReset, chipReview]);

  root.appendChild(panel);
  root.appendChild(toggle);
  root.appendChild(chip);

  var fields = [];
  var byKey = {};
  // staged: every edited value; invalid: keys whose value failed checks and
  // is not previewed; previewed: what the server is applying right now.
  var staged = {};
  var invalid = {};
  var previewed = {};
  var previewTimer = null;
  var previewSeq = 0;
  var openSections = {};
  var sectionNodes = {};
  var rows = {};
  var loaded = false;
  var loading = null;
  var armedTimer = null;

  function setStatus(text, kind) {
    status.textContent = text || '';
    status.className = 'status' + (kind ? ' ' + kind : '');
  }

  function stagedKeys() {
    return Object.keys(staged).filter(function (key) { return !loaded || byKey[key]; });
  }

  function validKeys() {
    return stagedKeys().filter(function (key) { return !has(invalid, key); });
  }

  function disarm() {
    if (armedTimer) clearTimeout(armedTimer);
    armedTimer = null;
    bakeBtn.classList.remove('armed');
    bakeBtn.textContent = 'Bake';
    bakeBtn.removeAttribute('title');
  }

  function updateSummary() {
    var n = stagedKeys().length;
    var bad = Object.keys(invalid).length;
    var live = Object.keys(previewed).length;
    summaryText.textContent = n ? plural(n, 'unsaved change') + (bad ? ' (' + bad + ' invalid)' : '') : 'No changes';
    summaryText.title = n ? 'Previewing live; Bake saves them, Reset drops them' : '';
    discardBtn.disabled = !n && !live;
    bakeBtn.disabled = !validKeys().length || !loaded;
    dot.hidden = !n;
    dot.textContent = n ? String(n) : '';
    chip.hidden = !live;
    chipText.textContent = 'Previewing ' + plural(live, 'unsaved change');
    disarm();
  }

  function updateCounts() {
    Object.keys(sectionNodes).forEach(function (key) {
      var s = sectionNodes[key];
      var changed = s.fields.filter(function (f) { return has(staged, f.key); }).length;
      s.count.textContent = changed ? changed + ' changed' : String(s.fields.length);
      s.count.classList.toggle('changed', changed > 0);
    });
  }

  function refreshRow(field) {
    var row = rows[field.key];
    if (row) row.replaceWith(renderField(field));
  }

  // check mirrors the server's validation so mistakes show before a rebuild.
  function check(field, value) {
    if (value === null || value === undefined) return '';
    if (field.kind === 'int' || field.kind === 'float') {
      if (typeof value !== 'number' || !isFinite(value)) return 'Enter a number';
      if (field.kind === 'int' && Math.floor(value) !== value) return 'Enter a whole number';
      if (field.min !== undefined && field.min !== null && value < field.min) return 'Must be at least ' + field.min;
      if (field.max !== undefined && field.max !== null && value > field.max) return 'Must be at most ' + field.max;
    }
    if (field.kind === 'string' && field.closed && value !== '' && field.options.indexOf(value) < 0 && value !== field.value) {
      return 'Choose one of the listed values';
    }
    if (field.kind === 'string' && value.length > 4096) return 'Too long (max 4096 characters)';
    return '';
  }

  function setInvalid(key, message) {
    if (message) invalid[key] = message;
    else delete invalid[key];
    var row = rows[key];
    if (!row) return;
    row.classList.toggle('invalid', !!message);
    var err = row.querySelector('.error');
    if (message) {
      if (!err) {
        err = el('div', { className: 'error', role: 'alert' });
        row.insertBefore(err, row.querySelector('.meta'));
      }
      err.textContent = message;
    } else if (err) {
      err.remove();
    }
  }

  // stage records an edit; commit also previews it live (after a short
  // debounce so quick successive changes share one rebuild).
  function stage(field, value, commit) {
    var wasStaged = has(staged, field.key);
    if (same(value, field.value)) delete staged[field.key];
    else staged[field.key] = value;
    setInvalid(field.key, has(staged, field.key) ? check(field, value) : '');
    if (wasStaged !== has(staged, field.key)) {
      var row = rows[field.key];
      if (row) {
        row.classList.toggle('changed', has(staged, field.key));
        var meta = row.querySelector('.meta');
        if (meta) meta.replaceWith(renderMeta(field));
      }
    }
    updateSummary();
    updateCounts();
    if (commit) schedulePreview();
  }

  function schedulePreview() {
    clearTimeout(previewTimer);
    previewTimer = setTimeout(sendPreview, 300);
  }

  function previewChanges() {
    return validKeys().map(function (key) { return { key: key, value: staged[key] }; });
  }

  // keyFromError finds the setting a server error message is about.
  function keyFromError(message) {
    var best = '';
    Object.keys(staged).forEach(function (key) {
      if (message.indexOf(key) >= 0 && key.length > best.length) best = key;
    });
    return best;
  }

  function sendPreview() {
    clearTimeout(previewTimer);
    previewTimer = null;
    var changes = previewChanges();
    var desired = {};
    changes.forEach(function (c) { desired[c.key] = c.value; });
    if (same(desired, previewed)) return Promise.resolve(true);
    var seq = ++previewSeq;
    setStatus(changes.length ? 'Previewing\u2026 rebuilding' : 'Resetting\u2026 rebuilding');
    return post(previewEndpoint, changes).then(function (result) {
      if (seq !== previewSeq) return false;
      var data = result.data || {};
      if (!result.ok) {
        var key = keyFromError(data.error || '');
        if (key && has(staged, key) && !has(invalid, key)) {
          setInvalid(key, data.error);
          updateSummary();
          return sendPreview();
        }
        setStatus(data.error || 'Preview failed', 'error');
        return false;
      }
      previewed = toMap(data.preview);
      var warn = (data.warnings || []).join(' ');
      setStatus(changes.length ? 'Previewing ' + plural(changes.length, 'unsaved change') + ' \u2014 rebuilding\u2026' + (warn ? ' ' + warn : '') :
        'Reset to your config files \u2014 rebuilding\u2026', warn ? 'warn' : '');
      updateSummary();
      return true;
    }, function (err) {
      setStatus(err.message, 'error');
      return false;
    });
  }

  function post(url, changes) {
    return fetch(url, {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({ changes: changes })
    }).then(function (res) {
      return res.json().then(function (data) { return { ok: res.ok, data: data }; });
    });
  }

  function toMap(list) {
    var map = {};
    (list || []).forEach(function (c) { map[c.key] = c.value; });
    return map;
  }

  function currentValue(field) {
    return has(staged, field.key) ? staged[field.key] : field.value;
  }

  function control(field) {
    var value = currentValue(field);
    var id = 'f-' + field.key.replace(/[^a-z0-9_-]/gi, '-');
    if (!field.editable) {
      var text = field.sensitive ? 'Hidden (sensitive) \u2014 edit it in the config file' :
        field.unsupported ? 'Not read from config files' :
        (field.summary ? field.summary + ' \u2014 edit in the config file' : 'Edit in the config file');
      return { node: el('div', { className: 'readonly', text: text }), id: null };
    }
    if (field.kind === 'bool') {
      var box = el('input', { type: 'checkbox', id: id, role: 'switch' });
      box.checked = value === true;
      box.indeterminate = value === null || value === undefined;
      box.addEventListener('change', function () { stage(field, box.checked, true); });
      return { node: el('span', { className: 'switch' }, [box, el('span')]), id: id, inline: true };
    }
    if (field.kind === 'int' || field.kind === 'float') {
      var bounded = field.max !== undefined && field.max !== null && field.max <= 3;
      var num = el('input', {
        type: 'number', id: id, inputmode: field.kind === 'int' ? 'numeric' : 'decimal',
        step: field.kind === 'int' ? '1' : (bounded ? '0.05' : 'any'), min: field.min, max: field.max
      });
      num.value = value === null || value === undefined ? '' : String(value);
      var readNum = function (commit) {
        if (num.value === '') { stage(field, field.value, commit); return; }
        stage(field, Number(num.value), commit);
      };
      num.addEventListener('input', function () { readNum(false); });
      num.addEventListener('change', function () { readNum(true); });
      return { node: num, id: id };
    }
    if (field.kind === 'list') {
      var items = value || [];
      var area = el('textarea', { id: id, rows: String(Math.min(6, Math.max(2, items.length + 1))), spellcheck: 'false', placeholder: 'One item per line' });
      area.value = items.join('\n');
      var readList = function (commit) {
        stage(field, area.value.split('\n').map(function (s) { return s.trim(); }).filter(Boolean), commit);
      };
      area.addEventListener('input', function () { readList(false); });
      area.addEventListener('change', function () { readList(true); });
      return { node: area, id: id };
    }
    var str = value === null || value === undefined ? '' : String(value);
    var options = field.options || [];
    if (field.closed && options.length) {
      var select = el('select', { id: id });
      var opts = options.slice();
      // Keep a saved value the list does not name (e.g. a palette alias).
      if (field.value && opts.indexOf(field.value) < 0) opts.unshift(field.value);
      if (!field.value || !str) select.appendChild(el('option', { value: '', text: 'Default' }));
      opts.forEach(function (opt) { select.appendChild(el('option', { value: opt, text: opt })); });
      select.value = str;
      select.addEventListener('change', function () { stage(field, select.value, true); });
      return { node: select, id: id };
    }
    var long = /(^|[._])(description|text|copyright|message|hint)$/.test(field.key) || str.length > 60 || str.indexOf('\n') >= 0;
    var input = long ? el('textarea', { id: id, rows: '2' }) : el('input', { type: 'text', id: id, spellcheck: 'false' });
    input.value = str;
    input.addEventListener('input', function () { stage(field, input.value, false); });
    input.addEventListener('change', function () { stage(field, input.value, true); });
    if (!long) {
      input.addEventListener('keydown', function (e) { if (e.key === 'Enter') stage(field, input.value, true); });
    }
    if (options.length && !long) {
      var listId = id + '-options';
      input.setAttribute('list', listId);
      var datalist = el('datalist', { id: listId });
      options.forEach(function (opt) { datalist.appendChild(el('option', { value: opt })); });
      return { node: el('div', null, [input, datalist]), id: id };
    }
    return { node: input, id: id };
  }

  function renderMeta(field) {
    var meta = el('div', { className: 'meta' });
    meta.appendChild(field.source ? el('span', { className: 'set', text: 'set in ' + field.source }) : el('span', { text: 'default' }));
    if (field.editable && field.target !== field.source) meta.appendChild(el('span', { text: 'writes to ' + field.target }));
    if (field.env) meta.appendChild(el('span', { className: 'env', text: field.env + ' overrides this' }));
    if (has(staged, field.key)) {
      var revert = el('button', { className: 'revert', type: 'button', text: 'Revert' });
      revert.addEventListener('click', function () {
        delete staged[field.key];
        delete invalid[field.key];
        refreshRow(field);
        updateSummary();
        updateCounts();
        schedulePreview();
      });
      meta.appendChild(revert);
    }
    return meta;
  }

  function renderField(field) {
    var ctl = control(field);
    var row = el('div', { className: 'field' + (has(staged, field.key) ? ' changed' : ''), 'data-key': field.key });
    var label = el(ctl.id ? 'label' : 'span', { className: 'label', for: ctl.id, text: field.label });
    row.appendChild(el('div', { className: 'row' }, [label, ctl.inline ? ctl.node : null]));
    row.appendChild(el('div', { className: 'key', text: field.key }));
    if (field.doc) row.appendChild(el('div', { className: 'doc', text: field.doc }));
    if (!ctl.inline) row.appendChild(ctl.node);
    if (has(invalid, field.key)) {
      row.classList.add('invalid');
      row.appendChild(el('div', { className: 'error', role: 'alert', text: invalid[field.key] }));
    }
    row.appendChild(renderMeta(field));
    rows[field.key] = row;
    return row;
  }

  function matches(field, words, mode) {
    if (mode === 'set' && !field.source) return false;
    if (mode === 'changed' && !has(staged, field.key)) return false;
    if (!words.length) return true;
    var hay = (field.key + ' ' + field.label + ' ' + (field.doc || '')).toLowerCase();
    return words.every(function (word) { return hay.indexOf(word) >= 0; });
  }

  function render() {
    var q = search.value.trim().toLowerCase();
    var words = q ? q.split(/\s+/) : [];
    var mode = filter.value;
    var filtering = !!words.length || mode !== 'all';
    body.textContent = '';
    rows = {};
    sectionNodes = {};
    var groups = [];
    var index = {};
    fields.forEach(function (field) {
      if (!matches(field, words, mode)) return;
      if (!has(index, field.section)) {
        index[field.section] = groups.length;
        groups.push({ key: field.section, fields: [] });
      }
      groups[index[field.section]].fields.push(field);
    });
    if (!groups.length) {
      body.appendChild(el('div', { className: 'empty', text: loaded ? 'No settings match.' : 'Loading settings\u2026' }));
      return;
    }
    groups.forEach(function (group) {
      var details = el('details', { 'data-section': group.key });
      details.open = filtering || !!openSections[group.key];
      var count = el('span', { className: 'count' });
      details.appendChild(el('summary', null, [sectionLabel(group.key), count]));
      var list = el('div');
      var filled = false;
      function fill() {
        if (filled) return;
        filled = true;
        group.fields.forEach(function (field) { list.appendChild(renderField(field)); });
      }
      if (details.open) fill();
      details.addEventListener('toggle', function () {
        if (details.open) fill();
        if (!filtering) {
          openSections[group.key] = details.open;
          saveState();
        }
      });
      details.appendChild(list);
      body.appendChild(details);
      sectionNodes[group.key] = { node: details, count: count, fields: group.fields };
    });
    updateCounts();
  }

  function load() {
    if (loading) return loading;
    loading = fetch(endpoint, { headers: { Accept: 'application/json' }, credentials: 'same-origin' })
      .then(function (res) { return res.json().then(function (data) { return { ok: res.ok, data: data }; }); })
      .then(function (result) {
        if (!result.ok) throw new Error(result.data.error || 'Could not load settings');
        fields = result.data.fields || [];
        byKey = {};
        fields.forEach(function (f) { byKey[f.key] = f; });
        previewed = toMap(result.data.preview);
        Object.keys(previewed).forEach(function (key) {
          var f = byKey[key];
          if (f && f.editable && !has(staged, key)) staged[key] = previewed[key];
        });
        Object.keys(staged).forEach(function (key) {
          var f = byKey[key];
          if (!f || !f.editable || same(staged[key], f.value)) delete staged[key];
        });
        var list = result.data.files || [];
        files.textContent = list.length ? 'Config: ' + list.join(', ') : 'No config file yet \u2014 baking creates markata-go.toml';
        loaded = true;
        render();
        updateSummary();
      })
      .catch(function (err) {
        loading = null;
        setStatus(err.message, 'error');
        body.textContent = '';
        body.appendChild(el('div', { className: 'empty', text: 'Could not load settings.' }));
      });
    return loading;
  }

  function saveState() {
    writeJSON(STATE_KEY, {
      open: isOpen(),
      q: search.value,
      filter: filter.value,
      sections: openSections,
      scroll: body.scrollTop
    });
  }

  function isOpen() { return panel.classList.contains('open'); }

  function open(section) {
    panel.classList.add('open');
    toggle.setAttribute('aria-expanded', 'true');
    saveState();
    return load().then(function () {
      if (section !== undefined && section !== null) {
        openSections[section] = true;
        if (!sectionNodes[section]) {
          search.value = '';
          filter.value = 'all';
        }
        render();
        var s = sectionNodes[section];
        if (s) {
          s.node.open = true;
          s.node.scrollIntoView({ block: 'start' });
        }
        saveState();
      }
      if (window.matchMedia('(min-width: 641px)').matches) search.focus({ preventScroll: true });
    });
  }

  function close() {
    panel.classList.remove('open');
    toggle.setAttribute('aria-expanded', 'false');
    saveState();
    toggle.focus({ preventScroll: true });
  }

  function bake() {
    var keys = validKeys();
    if (!keys.length) return;
    var targets = [];
    keys.forEach(function (key) {
      var f = byKey[key];
      if (f && targets.indexOf(f.target) < 0) targets.push(f.target);
    });
    if (!bakeBtn.classList.contains('armed')) {
      bakeBtn.classList.add('armed');
      bakeBtn.textContent = 'Bake into ' + targets.join(', ') + '?';
      bakeBtn.title = 'Click again to write ' + plural(keys.length, 'setting') + ' into ' + targets.join(', ');
      armedTimer = setTimeout(disarm, 4000);
      return;
    }
    disarm();
    bakeBtn.disabled = true;
    bakeBtn.textContent = 'Baking\u2026';
    setStatus('');
    var changes = keys.map(function (key) { return { key: key, value: staged[key] }; });
    post(endpoint, changes)
      .then(function (result) {
        var data = result.data || {};
        if (!result.ok) {
          var key = keyFromError(data.error || '');
          if (key) setInvalid(key, data.error);
          throw new Error(data.error || 'Bake failed');
        }
        var written = [];
        (data.changed || []).forEach(function (c) {
          if (written.indexOf(c.target) < 0) written.push(c.target);
          var f = byKey[c.key];
          if (f) { f.value = staged[c.key]; f.source = c.target; }
          delete staged[c.key];
        });
        previewed = toMap(data.preview);
        var text = 'Baked ' + plural((data.changed || []).length, 'setting') + ' into ' + written.join(', ') + '.';
        var warn = (data.warnings || []).join(' ');
        writeJSON(FLASH_KEY, { text: text, warn: warn });
        setStatus(text + ' Rebuilding\u2026' + (warn ? ' ' + warn : ''), warn ? 'warn' : 'ok');
        render();
        updateSummary();
      })
      .catch(function (err) {
        setStatus(err.message, 'error');
        updateSummary();
      });
  }

  function reset() {
    staged = {};
    invalid = {};
    render();
    updateSummary();
    if (!Object.keys(previewed).length) { setStatus(''); return; }
    writeJSON(FLASH_KEY, { text: 'Reset to your config files.', warn: '' });
    sendPreview();
  }

  toggle.addEventListener('click', function () { if (isOpen()) close(); else open(); });
  closeBtn.addEventListener('click', close);
  search.addEventListener('input', function () { render(); saveState(); });
  filter.addEventListener('change', function () { render(); saveState(); });
  var scrollTimer = null;
  body.addEventListener('scroll', function () {
    clearTimeout(scrollTimer);
    scrollTimer = setTimeout(saveState, 150);
  }, { passive: true });
  discardBtn.addEventListener('click', reset);
  chipReset.addEventListener('click', reset);
  chipReview.addEventListener('click', function () {
    filter.value = 'changed';
    search.value = '';
    open();
    render();
    saveState();
  });
  bakeBtn.addEventListener('click', bake);
  panel.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') { e.stopPropagation(); close(); }
  });
  document.addEventListener('keydown', function (e) {
    if (e.altKey && !e.ctrlKey && !e.metaKey && e.code === 'Comma') {
      e.preventDefault();
      if (isOpen()) close(); else open();
    }
  });

  function mount() {
    document.body.appendChild(host);
    var state = readJSON(STATE_KEY, null);
    var flash = readJSON(FLASH_KEY, null);
    writeJSON(FLASH_KEY, null);
    openSections = (state && state.sections) || { '': true };
    if (state) {
      search.value = state.q || '';
      filter.value = state.filter || 'all';
    }
    updateSummary();
    if (state && state.open) {
      panel.classList.add('open');
      toggle.setAttribute('aria-expanded', 'true');
    }
    // Load right away so the gear badge and preview bar reflect the server.
    load().then(function () { if (state && state.open) body.scrollTop = state.scroll || 0; });
    if (flash) setStatus(flash.text + (flash.warn ? ' ' + flash.warn : ''), flash.warn ? 'warn' : 'ok');
  }

  window.markataDevSettings = { open: open, close: close, isOpen: isOpen, reset: reset };
  if (document.body) mount();
  else document.addEventListener('DOMContentLoaded', mount);
})();
