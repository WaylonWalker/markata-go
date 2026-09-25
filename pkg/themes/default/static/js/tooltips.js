/**
 * Hover Cards
 *
 * One hover/focus preview component shared by everything that needs a small
 * floating card: wikilinks, glossary terms, and author-defined hovers.
 *
 * Built-in sources (first match wins):
 *   a.wikilink[data-title]        title, description, date, and path
 *   a.glossary-term[title]        term and definition (native title is suppressed)
 *   a[data-link-preview]          external link title, description, image, site, path
 *   [data-hover-card="#id"]       rich card cloned from a <template> or element
 *   [data-hover-title], [data-hover-description]   simple text card
 *
 * Extra sources can be added with:
 *   window.MarkataHoverCards.register(selector, function(el) { return node; });
 *
 * Popups drawn by other scripts can share the placement rules with:
 *   window.MarkataHoverCards.placement(target, width, height)
 *   // -> { top, left, placement: 'above' | 'below' } in viewport pixels
 *
 * Events are delegated from the document, so content swapped in by view
 * transitions works without re-binding.
 */
(function() {
  'use strict';

  if (window.MarkataHoverCards) {
    return;
  }

  var OPEN_DELAY = 180;
  var CLOSE_DELAY = 150;
  var GAP = 8;
  var MARGIN = 8;
  var CUSTOM_SELECTOR = '[data-hover-card], [data-hover-title], [data-hover-description]';

  var sources = [];
  var selector = '';
  var tooltip = null;
  var currentTarget = null;
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
    if (!link.href) return '';
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

  function textCard(title, desc, metaItems) {
    var frag = document.createDocumentFragment();
    var wrap = document.createElement('div');
    addLine(wrap, 'tooltip-title', title);
    addLine(wrap, 'tooltip-desc', desc);
    var meta = document.createElement('div');
    meta.className = 'tooltip-meta';
    (metaItems || []).forEach(function(item) {
      addLine(meta, item[0], item[1]);
    });
    if (meta.childNodes.length) wrap.appendChild(meta);
    if (!wrap.childNodes.length) return null;
    while (wrap.firstChild) frag.appendChild(wrap.firstChild);
    return frag;
  }

  // Moves a native title into a data attribute so the browser tooltip does
  // not stack on top of the hover card. No-JS readers keep the title.
  function takeTitle(el) {
    var title = el.getAttribute('title');
    if (title) {
      el.setAttribute('data-hover-native-title', title);
      el.removeAttribute('title');
    }
    return el.getAttribute('data-hover-native-title') || '';
  }

  function register(sel, build, options) {
    sources.push({ selector: sel, build: build, className: (options && options.className) || '' });
    selector = sources.map(function(s) { return s.selector; }).join(', ');
  }

  function sourceFor(el) {
    for (var i = 0; i < sources.length; i++) {
      if (el.matches(sources[i].selector)) return sources[i];
    }
    return null;
  }

  function targetFrom(node) {
    if (!selector || !node || !node.closest) return null;
    var el = node.closest(selector);
    if (el && tooltip && tooltip.contains(el)) return null;
    return el;
  }

  function buildTooltip(el, source) {
    var content = source.build(el);
    if (!content) return null;
    var card = document.createElement('div');
    card.className = 'hover-card' + (source.className ? ' ' + source.className : '');
    card.id = 'hover-card-' + (++idCounter);
    card.setAttribute('role', 'tooltip');
    card.appendChild(content);
    card.addEventListener('pointerenter', function() { clearTimeout(closeTimer); });
    card.addEventListener('pointerleave', scheduleClose);
    return card;
  }

  var pointer = null;

  function lineRects(el) {
    var rects = Array.prototype.filter.call(el.getClientRects(), function(r) {
      return r.width > 0 && r.height > 0;
    });
    return rects.length ? rects : [el.getBoundingClientRect()];
  }

  // Index of the line box under the pointer, or -1 when the pointer is not
  // over the target (keyboard focus, stale position).
  function pointerLine(rects, pt) {
    if (!pt) return -1;
    for (var i = 0; i < rects.length; i++) {
      var r = rects[i];
      if (pt.x >= r.left - 2 && pt.x <= r.right + 2 && pt.y >= r.top - 2 && pt.y <= r.bottom + 2) {
        return i;
      }
    }
    return -1;
  }

  /**
   * Computes where a popup of size width x height should sit next to target,
   * in viewport coordinates. Shared by every hover popup in the theme.
   *
   * The card opens on the side of the target the pointer is on: hovering the
   * first line of a wrapped link opens above that line, hovering the last line
   * opens below it, and middle lines pick the nearer end. Single-line targets
   * and keyboard focus open below. The card flips to the other side when the
   * preferred side lacks room, and it never covers any line of the target.
   */
  function placement(target, width, height, pt) {
    pt = pt === undefined ? pointer : pt;
    var rects = lineRects(target);
    var first = rects[0];
    var last = rects[rects.length - 1];
    var vw = document.documentElement.clientWidth;
    var vh = window.innerHeight;
    var line = pointerLine(rects, pt);

    var prefer = 'below';
    if (rects.length > 1 && line >= 0) {
      if (line === 0) {
        prefer = 'above';
      } else if (line < rects.length - 1) {
        prefer = pt.y - first.top < last.bottom - pt.y ? 'above' : 'below';
      }
    }

    var roomAbove = first.top - GAP - MARGIN;
    var roomBelow = vh - last.bottom - GAP - MARGIN;
    var side = prefer;
    if (side === 'below' && height > roomBelow && roomAbove > roomBelow) side = 'above';
    if (side === 'above' && height > roomAbove && roomBelow > roomAbove) side = 'below';

    var anchor = side === 'above' ? first : last;
    var top = side === 'above' ? first.top - GAP - height : last.bottom + GAP;
    var left = Math.min(anchor.left, vw - width - MARGIN);
    return {
      top: Math.max(MARGIN, top),
      left: Math.max(MARGIN, left),
      placement: side
    };
  }

  function positionTooltip(el) {
    if (!tooltip) return;
    var pos = placement(el, tooltip.offsetWidth, tooltip.offsetHeight);
    tooltip.style.left = pos.left + 'px';
    tooltip.style.top = pos.top + 'px';
    tooltip.setAttribute('data-placement', pos.placement);
  }

  function show(el) {
    clearTimers();
    if (currentTarget === el && tooltip) return;
    hide();
    var source = sourceFor(el);
    if (!source) return;
    var card = buildTooltip(el, source);
    if (!card) return;
    currentTarget = el;
    tooltip = card;
    document.body.appendChild(tooltip);
    el.setAttribute('aria-describedby', tooltip.id);
    positionTooltip(el);
  }

  function hide() {
    clearTimers();
    if (currentTarget) {
      currentTarget.removeAttribute('aria-describedby');
    }
    if (tooltip) {
      tooltip.remove();
    }
    tooltip = null;
    currentTarget = null;
  }

  function scheduleOpen(el) {
    clearTimeout(closeTimer);
    if (currentTarget === el && tooltip) return;
    clearTimeout(openTimer);
    openTimer = setTimeout(function() { show(el); }, currentTarget ? 0 : OPEN_DELAY);
  }

  function scheduleClose() {
    clearTimeout(openTimer);
    clearTimeout(closeTimer);
    closeTimer = setTimeout(hide, CLOSE_DELAY);
  }

  // Custom hover targets are often plain spans; make them keyboard reachable.
  function prepare(root) {
    (root || document).querySelectorAll(CUSTOM_SELECTOR).forEach(function(el) {
      if (el.tabIndex < 0 && !el.hasAttribute('tabindex')) {
        el.setAttribute('tabindex', '0');
      }
    });
  }

  register('a.wikilink[data-title]', function(link) {
    return textCard(
      link.dataset.title,
      link.dataset.description || link.dataset.preview,
      [['tooltip-date', link.dataset.date], ['tooltip-path', linkPath(link)]]
    );
  }, { className: 'wikilink-tooltip' });

  register('a.glossary-term', function(link) {
    var desc = link.dataset.hoverDescription || takeTitle(link);
    if (!desc) return null;
    var title = link.dataset.hoverTitle || link.textContent.trim();
    return textCard(title, desc, [['tooltip-kind', 'Glossary'], ['tooltip-path', linkPath(link)]]);
  }, { className: 'hover-card--glossary' });

  register('a[data-link-preview]', function(link) {
    var d = link.dataset;
    takeTitle(link);
    var host = d.linkHost || '';
    var frag = document.createDocumentFragment();
    if (d.linkImage) {
      var img = document.createElement('img');
      img.className = 'hover-card-image';
      img.src = d.linkImage;
      img.alt = '';
      img.decoding = 'async';
      img.referrerPolicy = 'no-referrer';
      // Stay hidden until loaded so slow or broken images never leave a blank
      // block, then re-place the card for its new height.
      img.hidden = true;
      img.addEventListener('load', function() {
        img.hidden = false;
        if (tooltip && tooltip.contains(img)) positionTooltip(currentTarget);
      });
      img.addEventListener('error', function() { img.remove(); });
      frag.appendChild(img);
    }
    var text = textCard(d.linkTitle || host, d.linkDescription, []);
    if (text) frag.appendChild(text);
    var meta = document.createElement('div');
    meta.className = 'tooltip-meta';
    var siteEl = document.createElement('span');
    siteEl.className = 'tooltip-site';
    if (d.linkIcon) {
      var icon = document.createElement('img');
      icon.className = 'tooltip-icon';
      icon.src = d.linkIcon;
      icon.alt = '';
      icon.width = 14;
      icon.height = 14;
      icon.referrerPolicy = 'no-referrer';
      icon.addEventListener('error', function() { icon.remove(); });
      siteEl.appendChild(icon);
    }
    siteEl.appendChild(document.createTextNode((d.linkSite || host) + ' \u2197'));
    meta.appendChild(siteEl);
    var path = '';
    try {
      var u = new URL(link.href);
      path = u.pathname === '/' && !u.search ? '' : u.pathname + u.search;
    } catch (e) {
      path = '';
    }
    addLine(meta, 'tooltip-path', path);
    frag.appendChild(meta);
    return frag;
  }, { className: 'hover-card--external' });

  register('[data-hover-card]', function(el) {
    var ref = el.getAttribute('data-hover-card');
    var src = null;
    try {
      src = ref ? document.querySelector(ref) : null;
    } catch (e) {
      src = null;
    }
    if (!src) return null;
    var body = document.createElement('div');
    body.className = 'hover-card-body';
    body.appendChild(src.content ? src.content.cloneNode(true) : src.cloneNode(true));
    body.querySelectorAll('[id]').forEach(function(n) { n.removeAttribute('id'); });
    if (src.hidden && body.firstElementChild) body.firstElementChild.hidden = false;
    return body;
  }, { className: 'hover-card--rich' });

  register('[data-hover-title], [data-hover-description]', function(el) {
    return textCard(el.dataset.hoverTitle, el.dataset.hoverDescription || takeTitle(el), []);
  });

  function trackPointer(e) {
    pointer = e.pointerType === 'touch' ? null : { x: e.clientX, y: e.clientY };
  }

  document.addEventListener('pointermove', trackPointer, { passive: true });
  document.addEventListener('pointerdown', trackPointer, { passive: true, capture: true });

  document.addEventListener('pointerover', function(e) {
    trackPointer(e);
    if (e.pointerType === 'touch') return;
    var el = targetFrom(e.target);
    if (el) scheduleOpen(el);
  });

  document.addEventListener('pointerout', function(e) {
    var el = targetFrom(e.target);
    if (!el) return;
    var to = e.relatedTarget;
    if (to && (el.contains(to) || (tooltip && tooltip.contains(to)))) return;
    scheduleClose();
  });

  document.addEventListener('focusin', function(e) {
    if (tooltip && tooltip.contains(e.target)) {
      clearTimeout(closeTimer);
      return;
    }
    var el = targetFrom(e.target);
    if (el && el.matches(':focus-visible')) {
      pointer = null;
      show(el);
    } else if (currentTarget) {
      hide();
    }
  });

  document.addEventListener('focusout', function(e) {
    var to = e.relatedTarget;
    if (to && tooltip && tooltip.contains(to)) return;
    if (targetFrom(e.target) === currentTarget || (tooltip && tooltip.contains(e.target))) {
      scheduleClose();
    }
  });

  document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape' && tooltip) {
      var target = currentTarget;
      var hadFocus = tooltip.contains(document.activeElement);
      hide();
      if (hadFocus && target) target.focus();
    }
  });

  window.addEventListener('scroll', function() {
    if (tooltip) hide();
  }, { passive: true });

  document.addEventListener('pointerdown', function(e) {
    if (tooltip && !tooltip.contains(e.target)) hide();
  });

  function cleanup() {
    hide();
    document.querySelectorAll('.hover-card').forEach(function(el) {
      el.remove();
    });
    prepare(document);
  }

  window.MarkataHoverCards = {
    register: register,
    placement: placement,
    show: show,
    hide: hide,
    refresh: cleanup
  };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function() { prepare(document); });
  } else {
    prepare(document);
  }

  // View transitions call this after swapping content; delegation means only
  // stale cards need removing and new custom targets need tabindex.
  window.initTooltips = cleanup;
  window.addEventListener('view-transition-complete', cleanup);
})();
