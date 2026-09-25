/**
 * Wikilink Hover Previews
 *
 * Shows a small preview card (title, description, date, path) when a wikilink
 * is hovered or keyboard-focused. Events are delegated from the document, so
 * links swapped in by view transitions work without re-binding.
 */
(function() {
  'use strict';

  if (window.__markataWikilinkPreviews) {
    return;
  }
  window.__markataWikilinkPreviews = true;

  var SELECTOR = 'a.wikilink[data-title]';
  var OPEN_DELAY = 180;
  var CLOSE_DELAY = 120;
  var GAP = 8;
  var MARGIN = 8;

  var tooltip = null;
  var currentLink = null;
  var openTimer = null;
  var closeTimer = null;
  var idCounter = 0;

  function clearTimers() {
    clearTimeout(openTimer);
    clearTimeout(closeTimer);
    openTimer = null;
    closeTimer = null;
  }

  function linkPath(link) {
    try {
      var url = new URL(link.href, window.location.href);
      return url.origin === window.location.origin ? url.pathname : url.host + url.pathname;
    } catch (e) {
      return '';
    }
  }

  function addLine(parent, className, text) {
    if (!text) return;
    var el = document.createElement('div');
    el.className = className;
    el.textContent = text;
    parent.appendChild(el);
  }

  function buildTooltip(link) {
    var el = document.createElement('div');
    el.className = 'wikilink-tooltip';
    el.id = 'wikilink-tooltip-' + (++idCounter);
    el.setAttribute('role', 'tooltip');
    addLine(el, 'tooltip-title', link.dataset.title);
    addLine(el, 'tooltip-desc', link.dataset.description);
    var meta = document.createElement('div');
    meta.className = 'tooltip-meta';
    addLine(meta, 'tooltip-date', link.dataset.date);
    addLine(meta, 'tooltip-path', linkPath(link));
    if (meta.childNodes.length) el.appendChild(meta);
    el.addEventListener('pointerenter', function() { clearTimeout(closeTimer); });
    el.addEventListener('pointerleave', scheduleClose);
    return el;
  }

  function positionTooltip(link) {
    if (!tooltip) return;
    // Use the line box nearest the pointer entry for links that wrap.
    var rects = link.getClientRects();
    var rect = rects.length ? rects[0] : link.getBoundingClientRect();
    var vw = document.documentElement.clientWidth;
    var vh = window.innerHeight;
    var tw = tooltip.offsetWidth;
    var th = tooltip.offsetHeight;

    var left = Math.min(Math.max(MARGIN, rect.left), vw - tw - MARGIN);
    var top = rect.bottom + GAP;
    var placement = 'below';
    if (top + th > vh - MARGIN && rect.top - GAP - th >= MARGIN) {
      top = rect.top - GAP - th;
      placement = 'above';
    }
    tooltip.style.left = Math.max(MARGIN, left) + 'px';
    tooltip.style.top = Math.max(MARGIN, top) + 'px';
    tooltip.setAttribute('data-placement', placement);
  }

  function show(link) {
    clearTimers();
    if (currentLink === link && tooltip) return;
    hide();
    currentLink = link;
    tooltip = buildTooltip(link);
    document.body.appendChild(tooltip);
    link.setAttribute('aria-describedby', tooltip.id);
    positionTooltip(link);
  }

  function hide() {
    clearTimers();
    if (currentLink) {
      currentLink.removeAttribute('aria-describedby');
    }
    if (tooltip) {
      tooltip.remove();
    }
    tooltip = null;
    currentLink = null;
  }

  function scheduleOpen(link) {
    clearTimeout(closeTimer);
    if (currentLink === link && tooltip) return;
    clearTimeout(openTimer);
    openTimer = setTimeout(function() { show(link); }, currentLink ? 0 : OPEN_DELAY);
  }

  function scheduleClose() {
    clearTimeout(openTimer);
    clearTimeout(closeTimer);
    closeTimer = setTimeout(hide, CLOSE_DELAY);
  }

  function linkFrom(target) {
    return target && target.closest ? target.closest(SELECTOR) : null;
  }

  document.addEventListener('pointerover', function(e) {
    if (e.pointerType === 'touch') return;
    var link = linkFrom(e.target);
    if (link) scheduleOpen(link);
  });

  document.addEventListener('pointerout', function(e) {
    var link = linkFrom(e.target);
    if (!link) return;
    var to = e.relatedTarget;
    if (to && (link.contains(to) || (tooltip && tooltip.contains(to)))) return;
    scheduleClose();
  });

  document.addEventListener('focusin', function(e) {
    var link = linkFrom(e.target);
    if (link && link.matches(':focus-visible')) {
      show(link);
    } else if (currentLink) {
      hide();
    }
  });

  document.addEventListener('focusout', function(e) {
    if (linkFrom(e.target) === currentLink) scheduleClose();
  });

  document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape' && tooltip) hide();
  });

  window.addEventListener('scroll', function() {
    if (tooltip) hide();
  }, { passive: true });

  document.addEventListener('pointerdown', function(e) {
    if (tooltip && !tooltip.contains(e.target)) hide();
  });

  function cleanup() {
    hide();
    document.querySelectorAll('.wikilink-tooltip').forEach(function(el) {
      el.remove();
    });
  }

  // View transitions call this after swapping content; delegation means only
  // stale tooltips need removing.
  window.initTooltips = cleanup;
  window.addEventListener('view-transition-complete', cleanup);
})();
