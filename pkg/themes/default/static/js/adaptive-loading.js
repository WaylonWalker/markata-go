(function(root, factory) {
  if (typeof module === 'object' && module.exports) module.exports = factory();
  else root.MarkataAdaptiveLoading = factory();
})(typeof window === 'undefined' ? this : window, function() {
  'use strict';

  var KEY = 'markata-loading-mode-v1';
  var SESSION_KEY = 'markata-loading-auto-v1';
  var DISMISS_KEY = 'markata-loading-dismissed-v1';
  var SESSION_TTL = 10 * 60 * 1000;
  var MAX_SAMPLES = 6;

  function state() {
    return { recommendation: 'unknown', poor: 0, good: 0, samples: 0, offline: false };
  }

  function vote(current, kind) {
    var next = Object.assign(state(), current || {});
    if (kind === 'offline') {
      next.offline = true;
      next.recommendation = 'constrained';
      return next;
    }
    if (kind === 'online') {
      next.offline = false;
      return next;
    }
    if (kind === 'poor') {
      next.poor = Math.min(8, next.poor + 1);
      next.good = 0;
      next.samples = Math.min(MAX_SAMPLES, next.samples + 1);
    } else if (kind === 'good') {
      next.good = Math.min(8, next.good + 1);
      next.poor = 0;
      next.samples = Math.min(MAX_SAMPLES, next.samples + 1);
    } else if (kind === 'strong-poor') {
      next.poor = Math.max(2, next.poor + 2);
      next.good = 0;
    } else if (kind === 'hint-good') {
      next.good = Math.min(8, next.good + 1);
      next.poor = 0;
    } else if (kind === 'strong-good') {
      next.good = Math.max(4, next.good + 2);
      next.poor = 0;
    }
    if (next.poor >= 2) next.recommendation = 'constrained';
    else if (next.good >= 4) next.recommendation = 'full-quality';
    else if (next.recommendation === 'unknown') next.recommendation = 'normal';
    return next;
  }

  function effectiveMode(userMode, automatic) {
    if (userMode === 'save-data') return 'constrained';
    if (userMode === 'full-quality') return 'full-quality';
    return automatic || 'unknown';
  }

  function readMode(storage, key) {
    try {
      var mode = storage && storage.getItem(key);
      return mode === 'save-data' || mode === 'full-quality' ? mode : 'auto';
    } catch (_) { return 'auto'; }
  }

  function persistMode(storage, key, mode) {
    try {
      if (!storage) return;
      if (mode === 'auto') storage.removeItem(key);
      else if (mode === 'save-data' || mode === 'full-quality') storage.setItem(key, mode);
    } catch (_) { /* storage may be blocked */ }
  }

  function mismatch(userMode, current) {
    if (current.offline) return userMode === 'full-quality' ? 'constrained' : '';
    if (userMode === 'full-quality' && current.poor >= 4) return 'constrained';
    if (userMode === 'save-data' && current.good >= 5) return 'full-quality';
    return '';
  }

  function networkHint(current, info) {
    if (!info) return current || state();
    var type = String(info.effectiveType || '').toLowerCase();
    if (info.saveData || type === 'slow-2g') return vote(current, 'strong-poor');
    if (type === '2g') return vote(current, 'poor');
    if (type === '4g' && Number(info.downlink) >= 5) return vote(current, 'strong-good');
    if (type === '4g' || Number(info.downlink) >= 1.5) return vote(current, 'hint-good');
    return current || state();
  }

  function restore(serialized, now) {
    try {
      var saved = JSON.parse(serialized);
      if (!saved || now - saved.at > SESSION_TTL || now < saved.at) return state();
      var restored = state();
      restored.poor = Math.max(0, Math.min(8, saved.poor | 0));
      restored.good = Math.max(0, Math.min(8, saved.good | 0));
      restored.samples = Math.max(0, Math.min(MAX_SAMPLES, saved.samples | 0));
      restored.offline = !!saved.offline;
      if (restored.offline) restored.recommendation = 'constrained';
      else if (restored.poor >= 2) restored.recommendation = 'constrained';
      else if (restored.good >= 4) restored.recommendation = 'full-quality';
      else if (restored.poor || restored.good) restored.recommendation = 'normal';
      return restored;
    } catch (_) { return state(); }
  }

  function savedState(current, now) {
    return JSON.stringify({ poor: current.poor, good: current.good, samples: current.samples, offline: current.offline, at: now });
  }

  var api = { state: state, vote: vote, networkHint: networkHint, effectiveMode: effectiveMode, readMode: readMode, persistMode: persistMode, mismatch: mismatch, restore: restore, savedState: savedState };
  if (typeof document === 'undefined') return api;

  var rootElement = document.documentElement;
  var control = document.querySelector('[data-adaptive-loading]');
  if (!control) return api;

  var radios = control.querySelectorAll('input[name="loading-mode"]');
  var status = control.querySelector('[data-loading-status]');
  var message = control.querySelector('[data-loading-mismatch]');
  var actions = control.querySelector('[data-loading-actions]');
  var useAuto = control.querySelector('[data-loading-use-auto]');
  var keepMode = control.querySelector('[data-loading-keep]');
  var details = control.querySelector('details');
  var sessionStore = null;
  var localStore = null;
  try { sessionStore = window.sessionStorage; } catch (_) { /* storage may be blocked */ }
  try { localStore = window.localStorage; } catch (_) { /* storage may be blocked */ }
  var current = restore(readStorage(sessionStore, SESSION_KEY), Date.now());
  var userMode = readMode(localStore, KEY);
  var samples = current.samples;
  if (navigator.onLine && current.offline) current = vote(current, 'online');

  function readStorage(storage, key) {
    try { return storage ? storage.getItem(key) : null; } catch (_) { return null; }
  }

  function writeStorage(storage, key, value) {
    try { if (storage) storage.setItem(key, value); } catch (_) { /* storage may be blocked */ }
  }

  function clearStorage(storage, key) {
    try { if (storage) storage.removeItem(key); } catch (_) { /* storage may be blocked */ }
  }

  function setUserMode(mode) {
    userMode = mode;
    persistMode(localStore, KEY, mode);
    update();
  }

  function update() {
    if (!navigator.onLine) current = vote(current, 'offline');
    var automatic = current.recommendation;
    var effective = effectiveMode(userMode, automatic);
    rootElement.dataset.loadingMode = userMode;
    rootElement.dataset.loadingPolicy = effective;
    rootElement.dataset.loadingEffective = effective;
    rootElement.dataset.loadingRecommendation = automatic;
    rootElement.dataset.adaptiveReady = 'true';
    for (var i = 0; i < radios.length; i++) radios[i].checked = radios[i].value === userMode;
    var labels = { auto: 'Auto', 'save-data': 'Save Data', 'full-quality': 'Full Quality' };
    var summary = control.querySelector('[data-loading-label]');
    if (summary) summary.textContent = 'Loading: ' + labels[userMode];
    var suggestion = mismatch(userMode, current);
    var dismissed = readStorage(sessionStore, DISMISS_KEY);
    if (actions) actions.hidden = !suggestion || dismissed === suggestion;
    if (keepMode) keepMode.textContent = 'Keep ' + labels[userMode];
    if (suggestion && dismissed !== suggestion) {
      if (status) status.hidden = false;
      if (message) {
        message.hidden = false;
        message.textContent = userMode === 'save-data'
          ? 'Save Data is on. Your connection appears fast now.'
          : 'Full Quality is on. Your connection appears constrained.';
      }
    } else {
      if (status) status.hidden = true;
      if (message) message.hidden = true;
    }
  }

  function record(kind) {
    if (samples >= MAX_SAMPLES) return;
    current = vote(current, kind);
    if (kind === 'good' || kind === 'poor') samples++;
    current.samples = samples;
    writeStorage(sessionStore, SESSION_KEY, savedState(current, Date.now()));
    update();
  }

  function connectionHint() {
    var connection = navigator.connection || navigator.mozConnection || navigator.webkitConnection;
    if (!connection) return;
    current = networkHint(current, connection);
    writeStorage(sessionStore, SESSION_KEY, savedState(current, Date.now()));
  }

  function inspect(entry) {
    if (samples >= MAX_SAMPLES || !entry || (entry.entryType !== 'navigation' && entry.initiatorType !== 'img' && entry.initiatorType !== 'video' && entry.initiatorType !== 'audio')) return;
    var bytes = Number(entry.transferSize || entry.encodedBodySize || 0);
    var elapsed = Number(entry.responseEnd - entry.responseStart);
    if (bytes < 24000 || elapsed < 100 || elapsed > 30000) return;
    var rate = bytes * 8 / (elapsed / 1000);
    if (rate < 250000) record('poor');
    else if (rate > 1000000) record('good');
  }

  for (var r = 0; r < radios.length; r++) radios[r].addEventListener('change', function() {
    if (this.checked) setUserMode(this.value);
  });
  if (useAuto) useAuto.addEventListener('click', function() { clearStorage(sessionStore, DISMISS_KEY); setUserMode('auto'); });
  if (keepMode) keepMode.addEventListener('click', function() {
    var recommendation = mismatch(userMode, current);
    if (recommendation) writeStorage(sessionStore, DISMISS_KEY, recommendation);
    if (details) details.open = false;
    update();
  });
  if (status && details) status.addEventListener('click', function() { details.open = true; });
  window.addEventListener('offline', function() {
    current = vote(current, 'offline');
    writeStorage(sessionStore, SESSION_KEY, savedState(current, Date.now()));
    update();
  });
  window.addEventListener('online', function() {
    current = vote(current, 'online');
    writeStorage(sessionStore, SESSION_KEY, savedState(current, Date.now()));
    update();
  });

  connectionHint();
  update();
  function collectTimings() {
    if (!performance.getEntriesByType) return;
    var navigation = performance.getEntriesByType('navigation');
    if (navigation.length) inspect(navigation[0]);
    var entries = performance.getEntriesByType('resource');
    for (var e = 0; e < entries.length && samples < MAX_SAMPLES; e++) inspect(entries[e]);
  }
  if (document.readyState === 'complete') collectTimings();
  else window.addEventListener('load', collectTimings, { once: true });
  return api;
});
