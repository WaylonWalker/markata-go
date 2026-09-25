/**
 * Table of contents: scroll spy + animated jumps
 *
 * - Highlights the section currently being read and slides a marker along
 *   the TOC rail to it.
 * - Clicking a TOC link glides to the heading: long jumps first cut to just
 *   short of the target, then ease the last stretch (~150-250ms) so the
 *   reader sees which way the page moved without a long, nauseating scroll.
 *   The destination is re-measured every frame and re-checked after landing,
 *   so late layout shifts (images, embeds, fonts) cannot leave the reader in
 *   the wrong place. prefers-reduced-motion jumps instantly.
 *
 * Listeners are bound once and read the current page state, so repeated
 * initialisation after view transitions is safe.
 */
(function() {
  'use strict';

  if (window.__markataScrollSpy) {
    return;
  }
  window.__markataScrollSpy = true;

  var ACTIVE_CLASS = 'toc-link--active';
  var PARENT_CLASS = 'toc-link--parent-active';
  var TARGET_CLASS = 'toc-target';
  var NEAR_FRACTION = 0.6;
  var reduceMotion = window.matchMedia ? window.matchMedia('(prefers-reduced-motion: reduce)') : { matches: false };

  var links = [];
  var entries = [];
  var activeLink = null;
  var frame = null;
  var glide = null;
  var lockUntil = 0;
  // Heading chosen by a TOC click; stays active until the reader scrolls.
  var clickedHeading = null;
  var targetTimer = null;
  var settleToken = 0;

  function decodeHash(hash) {
    try {
      return decodeURIComponent(hash);
    } catch (e) {
      return hash;
    }
  }

  function targetFor(link) {
    var href = link.getAttribute('href');
    if (!href || href.charAt(0) !== '#' || href.length < 2) return null;
    return document.getElementById(decodeHash(href.slice(1)));
  }

  // Distance from the viewport top where a heading counts as "reached".
  function topOffset() {
    var offset = 16;
    var header = document.querySelector('.site-header');
    if (header) {
      var position = window.getComputedStyle(header).position;
      if (position === 'sticky' || position === 'fixed') {
        offset += Math.max(0, header.getBoundingClientRect().bottom);
      }
    }
    return offset;
  }

  function collect() {
    links = Array.prototype.slice.call(document.querySelectorAll('.toc-link'));
    entries = [];
    links.forEach(function(link) {
      var heading = targetFor(link);
      if (!heading) return;
      entries.push({ link: link, heading: heading });
    });
    entries.sort(function(a, b) {
      var pos = a.heading.compareDocumentPosition(b.heading);
      return pos & Node.DOCUMENT_POSITION_FOLLOWING ? -1 : pos & Node.DOCUMENT_POSITION_PRECEDING ? 1 : 0;
    });
    ensureMarkers();
  }

  function ensureMarkers() {
    document.querySelectorAll('.toc').forEach(function(toc) {
      var list = toc.querySelector(':scope > .toc-list');
      if (!list || list.querySelector(':scope > .toc-marker')) return;
      var marker = document.createElement('span');
      marker.className = 'toc-marker';
      marker.setAttribute('aria-hidden', 'true');
      list.insertBefore(marker, list.firstChild);
    });
  }

  function currentEntry() {
    if (!entries.length) return null;
    // Scrollbar drags send no intent events, so also drop the clicked
    // heading once it has left the viewport.
    if (clickedHeading) {
      var cr = clickedHeading.getBoundingClientRect();
      if (cr.bottom < 0 || cr.top > window.innerHeight) clickedHeading = null;
    }
    if (clickedHeading) {
      for (var k = 0; k < entries.length; k++) {
        if (entries[k].heading === clickedHeading) return entries[k];
      }
      clickedHeading = null;
    }
    var threshold = topOffset() + Math.min(120, window.innerHeight * 0.2);
    var current = null;
    for (var i = 0; i < entries.length; i++) {
      if (entries[i].heading.getBoundingClientRect().top <= threshold) {
        current = entries[i];
      } else {
        break;
      }
    }
    // At the very bottom, the last heading may never reach the threshold.
    var doc = document.documentElement;
    if (window.innerHeight + window.scrollY >= doc.scrollHeight - 2) {
      for (var j = entries.length - 1; j >= 0; j--) {
        if (entries[j].heading.getBoundingClientRect().top < window.innerHeight) {
          current = entries[j];
          break;
        }
      }
    }
    return current || entries[0];
  }

  function moveMarker(link) {
    var list = link ? link.closest('.toc') && link.closest('.toc').querySelector(':scope > .toc-list') : null;
    document.querySelectorAll('.toc-marker').forEach(function(marker) {
      if (!list || marker.parentNode !== list) {
        marker.classList.remove('toc-marker--visible');
        return;
      }
      var listRect = list.getBoundingClientRect();
      var rect = link.getBoundingClientRect();
      if (!rect.height) return;
      marker.style.transform = 'translateY(' + (rect.top - listRect.top) + 'px)';
      marker.style.height = rect.height + 'px';
      marker.classList.add('toc-marker--visible');
    });
  }

  // Keep the active link inside the TOC's own scroll area without touching
  // the page scroll position.
  function revealInToc(link) {
    var scroller = link.closest('.toc');
    if (!scroller || scroller.scrollHeight <= scroller.clientHeight) return;
    if (scroller.matches(':hover')) return;
    var sRect = scroller.getBoundingClientRect();
    var lRect = link.getBoundingClientRect();
    var pad = 48;
    if (lRect.top < sRect.top + pad) {
      scroller.scrollTop -= (sRect.top + pad) - lRect.top;
    } else if (lRect.bottom > sRect.bottom - pad) {
      scroller.scrollTop += lRect.bottom - (sRect.bottom - pad);
    }
  }

  function setActive(link) {
    if (link === activeLink) {
      if (link) moveMarker(link);
      return;
    }
    activeLink = link;
    links.forEach(function(l) {
      l.classList.remove(ACTIVE_CLASS, PARENT_CLASS);
      l.removeAttribute('aria-current');
    });
    if (!link) {
      moveMarker(null);
      return;
    }
    link.classList.add(ACTIVE_CLASS);
    link.setAttribute('aria-current', 'location');
    var parentItem = link.closest('.toc-list--nested');
    while (parentItem) {
      var owner = parentItem.closest('.toc-item');
      var parentLink = owner ? owner.querySelector(':scope > .toc-link') : null;
      if (parentLink) parentLink.classList.add(PARENT_CLASS);
      parentItem = owner ? owner.parentElement.closest('.toc-list--nested') : null;
    }
    moveMarker(link);
    revealInToc(link);
  }

  function update() {
    frame = null;
    if (!entries.length) return;
    if (performance.now() < lockUntil) return;
    var entry = currentEntry();
    setActive(entry ? entry.link : null);
  }

  function schedule() {
    if (frame === null && entries.length) {
      frame = requestAnimationFrame(update);
    }
  }

  function maxScroll() {
    return Math.max(0, document.documentElement.scrollHeight - window.innerHeight);
  }

  function destinationFor(heading) {
    var y = window.scrollY + heading.getBoundingClientRect().top - topOffset();
    return Math.max(0, Math.min(maxScroll(), Math.round(y)));
  }

  function jump(y) {
    window.scrollTo({ top: y, left: window.scrollX, behavior: 'instant' });
  }

  function cancelGlide() {
    if (glide) {
      cancelAnimationFrame(glide.raf);
      glide = null;
    }
  }

  // The reader took over: stop gliding and never "correct" their scroll.
  function onUserScrollIntent() {
    clickedHeading = null;
    if (!glide && !lockUntil) return;
    cancelGlide();
    settleToken++;
    lockUntil = 0;
    schedule();
  }

  function settle(heading, token) {
    // Re-check after late layout shifts (lazy images, embeds, web fonts).
    [120, 450].forEach(function(delay) {
      setTimeout(function() {
        if (token !== settleToken) return;
        var drift = heading.getBoundingClientRect().top - topOffset();
        if (Math.abs(drift) > 3 && !(drift > 0 && window.scrollY >= maxScroll() - 1)) {
          jump(destinationFor(heading));
        }
        if (delay === 450) {
          lockUntil = 0;
          schedule();
        }
      }, delay);
    });
  }

  function markTarget(heading) {
    if (!heading.hasAttribute('tabindex')) {
      heading.setAttribute('tabindex', '-1');
    }
    heading.focus({ preventScroll: true });
    heading.classList.remove(TARGET_CLASS);
    // Restart the highlight animation on repeat clicks.
    void heading.offsetWidth;
    heading.classList.add(TARGET_CLASS);
    clearTimeout(targetTimer);
    targetTimer = setTimeout(function() {
      heading.classList.remove(TARGET_CLASS);
      targetTimer = null;
    }, 1200);
  }

  function glideTo(heading) {
    cancelGlide();
    var token = ++settleToken;
    var end = destinationFor(heading);
    var start = window.scrollY;
    var distance = end - start;

    if (reduceMotion.matches || Math.abs(distance) < 2) {
      jump(end);
      markTarget(heading);
      settle(heading, token);
      return;
    }

    // Cut most of a long trip, then ease the final stretch for direction.
    var near = Math.round(window.innerHeight * NEAR_FRACTION);
    if (Math.abs(distance) > near) {
      start = end - (distance > 0 ? near : -near);
      jump(start);
    }
    var travel = Math.abs(end - start);
    var duration = 150 + Math.min(100, (travel / window.innerHeight) * 160);
    var t0 = performance.now();

    glide = { raf: 0 };
    var step = function(now) {
      if (!glide) return;
      var p = Math.min(1, (now - t0) / duration);
      var eased = 1 - Math.pow(1 - p, 3);
      var goal = destinationFor(heading);
      jump(Math.round(start + (goal - start) * eased));
      if (p < 1) {
        glide.raf = requestAnimationFrame(step);
      } else {
        glide = null;
        markTarget(heading);
        settle(heading, token);
      }
    };
    glide.raf = requestAnimationFrame(step);
  }

  function onClick(e) {
    if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    var link = e.target && e.target.closest ? e.target.closest('.toc-link') : null;
    if (!link) return;
    var heading = targetFor(link);
    if (!heading) return;
    e.preventDefault();

    var href = link.getAttribute('href');
    if (window.location.hash !== href) {
      history.pushState(history.state, '', href);
    }
    // Move the marker straight to the clicked entry and hold it there while
    // the page travels.
    lockUntil = performance.now() + 1000;
    clickedHeading = heading;
    setActive(link);
    glideTo(heading);
  }

  function init() {
    cancelGlide();
    activeLink = null;
    lockUntil = 0;
    clickedHeading = null;
    collect();
    if (entries.length) {
      update();
    }
  }

  document.addEventListener('click', onClick);
  window.addEventListener('scroll', schedule, { passive: true });
  window.addEventListener('resize', function() {
    if (activeLink) moveMarker(activeLink);
    schedule();
  });
  ['wheel', 'touchstart', 'pointerdown'].forEach(function(type) {
    window.addEventListener(type, function(e) {
      if (type === 'pointerdown' && e.target && e.target.closest && e.target.closest('.toc')) return;
      onUserScrollIntent();
    }, { passive: true });
  });
  window.addEventListener('keydown', function(e) {
    if (/^(Arrow|Page|Home|End| )/.test(e.key)) onUserScrollIntent();
  });
  // Drawer open/close changes TOC geometry; refresh the marker afterwards.
  document.addEventListener('transitionend', function(e) {
    if (activeLink && e.target && e.target.matches && e.target.matches('.doc-sidebar, .toc')) {
      moveMarker(activeLink);
    }
  });

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  window.initScrollSpy = init;
  window.addEventListener('view-transition-complete', init);
})();
