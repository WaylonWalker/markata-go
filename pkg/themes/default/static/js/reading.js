/**
 * Reading experience enhancements for post pages.
 *
 * - Scroll-spy for the "On this page" TOC, promoted to a persistent margin
 *   TOC when the viewport has room beside the article.
 * - Sidenotes: footnotes become margin notes on wide viewports and tap/hover
 *   popovers elsewhere. Works on the standard goldmark footnote markup, so
 *   existing `[^1]` content needs no changes.
 * - Code block chrome: language badge, optional title and a copy button.
 * - Back-to-top button after scrolling past the first screen.
 *
 * Everything re-initialises cleanly after view transitions.
 */
(function() {
  'use strict';

  var MARGIN_TOC_WIDTH = 240;
  var MARGIN_GAP = 40;
  var SIDENOTE_WIDTH = 220;
  var HEADER_OFFSET = 96;

  var cleanups = [];

  function onCleanup(fn) {
    cleanups.push(fn);
  }

  function runCleanups() {
    while (cleanups.length) {
      try { cleanups.pop()(); } catch (_) { /* ignore */ }
    }
  }

  function prefersReducedMotion() {
    return window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  }

  function debounce(fn, wait) {
    var t;
    return function() {
      clearTimeout(t);
      t = setTimeout(fn, wait);
    };
  }

  /* ------------------------------------------------------------------------
   * Margin TOC + scroll spy
   * --------------------------------------------------------------------- */
  function initToc(article) {
    var sidebar = document.querySelector('.doc-sidebar');
    var links = sidebar ? sidebar.querySelectorAll('.toc-link') : [];
    if (!sidebar || !links.length) return;

    var headings = [];
    links.forEach(function(link) {
      var href = link.getAttribute('href') || '';
      if (href.charAt(0) !== '#') return;
      var el = document.getElementById(decodeURIComponent(href.slice(1)));
      if (el) headings.push({ el: el, link: link });
    });
    if (!headings.length) return;

    var configuredSide = sidebar.classList.contains('doc-sidebar--left') ? 'left' : 'right';
    var hasSidenotes = !!article.querySelector('.footnote-ref');

    function placeMarginToc() {
      var rect = article.getBoundingClientRect();
      var side = configuredSide;
      // Sidenotes live in the right margin, so the TOC yields to the left.
      if (hasSidenotes && side === 'right') side = 'left';

      var room = side === 'right' ? window.innerWidth - rect.right : rect.left;
      var fits = window.innerWidth >= 1201 && room >= MARGIN_TOC_WIDTH + MARGIN_GAP + 16;

      sidebar.classList.toggle('doc-sidebar--margin', fits);
      document.body.classList.toggle('has-margin-toc', fits);
      if (!fits) {
        sidebar.style.removeProperty('--margin-toc-left');
        sidebar.style.removeProperty('--margin-toc-right');
        sidebar.classList.remove('doc-sidebar--margin-left', 'doc-sidebar--margin-right');
        return;
      }
      sidebar.classList.toggle('doc-sidebar--margin-left', side === 'left');
      sidebar.classList.toggle('doc-sidebar--margin-right', side === 'right');
      if (side === 'right') {
        sidebar.style.setProperty('--margin-toc-left', Math.round(rect.right + MARGIN_GAP) + 'px');
        sidebar.style.removeProperty('--margin-toc-right');
      } else {
        sidebar.style.setProperty('--margin-toc-left', Math.round(rect.left - MARGIN_GAP - MARGIN_TOC_WIDTH) + 'px');
        sidebar.style.removeProperty('--margin-toc-right');
      }
    }

    var active = null;
    function setActive(entry) {
      if (active === entry) return;
      if (active) {
        active.link.classList.remove('toc-link--active');
        active.link.removeAttribute('aria-current');
      }
      active = entry;
      if (!active) return;
      active.link.classList.add('toc-link--active');
      active.link.setAttribute('aria-current', 'location');
      // Keep the active link visible inside a scrolling TOC.
      var toc = sidebar.querySelector('.toc');
      if (toc && toc.scrollHeight > toc.clientHeight) {
        var lr = active.link.getBoundingClientRect();
        var tr = toc.getBoundingClientRect();
        if (lr.top < tr.top || lr.bottom > tr.bottom) {
          active.link.scrollIntoView({ block: 'nearest' });
        }
      }
    }

    var ticking = false;
    function update() {
      ticking = false;
      var current = null;
      var limit = HEADER_OFFSET + 8;
      for (var i = 0; i < headings.length; i++) {
        if (headings[i].el.getBoundingClientRect().top <= limit) {
          current = headings[i];
        } else {
          break;
        }
      }
      // At the very bottom, highlight the last heading.
      if (window.innerHeight + window.scrollY >= document.documentElement.scrollHeight - 2) {
        current = headings[headings.length - 1];
      }
      setActive(current);
    }

    function onScroll() {
      if (ticking) return;
      ticking = true;
      requestAnimationFrame(update);
    }

    var onResize = debounce(function() { placeMarginToc(); update(); }, 80);

    window.addEventListener('scroll', onScroll, { passive: true });
    window.addEventListener('resize', onResize);
    onCleanup(function() {
      window.removeEventListener('scroll', onScroll);
      window.removeEventListener('resize', onResize);
      sidebar.classList.remove('doc-sidebar--margin', 'doc-sidebar--margin-left', 'doc-sidebar--margin-right');
      document.body.classList.remove('has-margin-toc');
    });

    var ro = null;
    if (window.ResizeObserver) {
      ro = new ResizeObserver(onResize);
      ro.observe(article);
      onCleanup(function() { ro.disconnect(); });
    }

    placeMarginToc();
    update();
  }

  /* ------------------------------------------------------------------------
   * Sidenotes
   * --------------------------------------------------------------------- */
  function initSidenotes(article) {
    var refs = article.querySelectorAll('.footnote-ref');
    if (!refs.length) return;

    // Idempotent re-init: drop notes from a previous pass.
    article.querySelectorAll('.sidenote').forEach(function(n) { n.remove(); });
    article.querySelectorAll('.has-sidenote').forEach(function(n) { n.classList.remove('has-sidenote'); });

    var footnotes = article.querySelector('.footnotes');
    var notes = [];
    var openNote = null;

    function closeOpen() {
      if (!openNote) return;
      openNote.classList.remove('sidenote--open');
      openNote.ref.setAttribute('aria-expanded', 'false');
      openNote = null;
    }

    refs.forEach(function(ref, index) {
      var href = ref.getAttribute('href') || '';
      if (href.charAt(0) !== '#') return;
      var target = document.getElementById(decodeURIComponent(href.slice(1)));
      if (!target) return;

      var host = ref.closest('p, li, blockquote, td, h1, h2, h3, h4, h5, h6, .admonition') || ref.parentElement;
      if (!host || host.closest('.footnotes')) return;

      var note = document.createElement('span');
      note.className = 'sidenote';
      note.setAttribute('role', 'note');
      note.id = 'sidenote-' + (index + 1);
      var body = document.createElement('div');
      body.className = 'sidenote__body';
      body.innerHTML = target.innerHTML;
      body.querySelectorAll('.footnote-backref').forEach(function(b) { b.remove(); });
      var num = document.createElement('span');
      num.className = 'sidenote__number';
      num.textContent = ref.textContent.trim();
      note.appendChild(num);
      note.appendChild(body);
      note.ref = ref;

      host.classList.add('has-sidenote');
      host.appendChild(note);

      ref.classList.add('footnote-ref--sidenote');
      ref.setAttribute('aria-controls', note.id);
      ref.setAttribute('aria-expanded', 'false');

      notes.push(note);

      function toggle(e) {
        // On wide screens the note is already visible in the margin; let the
        // link behave normally there.
        if (document.body.classList.contains('has-sidenotes-margin')) return;
        e.preventDefault();
        if (openNote === note) { closeOpen(); return; }
        closeOpen();
        note.classList.add('sidenote--open');
        ref.setAttribute('aria-expanded', 'true');
        openNote = note;
        // The popover opens below its paragraph; keep it on screen.
        if (note.scrollIntoView) {
          note.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
        }
      }
      ref.addEventListener('click', toggle);
      onCleanup(function() { ref.removeEventListener('click', toggle); });
    });

    if (!notes.length) return;

    function layout() {
      var rect = article.getBoundingClientRect();
      var fits = window.innerWidth >= 1201 && window.innerWidth - rect.right >= SIDENOTE_WIDTH + MARGIN_GAP + 16;
      document.body.classList.toggle('has-sidenotes-margin', fits);
      article.classList.toggle('post--sidenotes-margin', fits);
      if (fits) closeOpen();
      if (footnotes) footnotes.classList.toggle('footnotes--as-sidenotes', fits);
      if (!fits) return;

      // Stack notes so they never overlap: push each note below the previous.
      var lastBottom = -Infinity;
      notes.forEach(function(note) {
        note.style.marginTop = '';
        var r = note.getBoundingClientRect();
        if (r.top < lastBottom + 12) {
          note.style.marginTop = Math.round(lastBottom + 12 - r.top) + 'px';
          r = note.getBoundingClientRect();
        }
        lastBottom = r.bottom;
      });
    }

    var onResize = debounce(layout, 80);
    window.addEventListener('resize', onResize);
    function onDocClick(e) {
      if (!openNote) return;
      if (openNote.contains(e.target) || openNote.ref.contains(e.target)) return;
      closeOpen();
    }
    function onKey(e) {
      if (e.key === 'Escape') closeOpen();
    }
    document.addEventListener('click', onDocClick);
    document.addEventListener('keydown', onKey);
    onCleanup(function() {
      window.removeEventListener('resize', onResize);
      document.removeEventListener('click', onDocClick);
      document.removeEventListener('keydown', onKey);
      document.body.classList.remove('has-sidenotes-margin');
    });

    var ro = null;
    if (window.ResizeObserver) {
      ro = new ResizeObserver(onResize);
      ro.observe(article);
      onCleanup(function() { ro.disconnect(); });
    }
    layout();
  }

  /* ------------------------------------------------------------------------
   * Code block chrome
   * --------------------------------------------------------------------- */
  function copyText(text) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      return navigator.clipboard.writeText(text);
    }
    return new Promise(function(resolve, reject) {
      var ta = document.createElement('textarea');
      ta.value = text;
      ta.setAttribute('readonly', '');
      ta.style.position = 'absolute';
      ta.style.left = '-9999px';
      document.body.appendChild(ta);
      ta.select();
      var ok = document.execCommand('copy');
      document.body.removeChild(ta);
      ok ? resolve() : reject(new Error('copy failed'));
    });
  }

  function codeText(pre) {
    // Strip line-number columns (chroma table layout) when present.
    var code = pre.querySelector('code') || pre;
    var lines = code.querySelectorAll('.cl');
    if (lines.length) {
      return Array.prototype.map.call(lines, function(l) { return l.textContent; }).join('').replace(/\n$/, '');
    }
    return code.textContent.replace(/\n$/, '');
  }

  function initCodeChrome(article) {
    var pres = article.querySelectorAll('pre');
    pres.forEach(function(pre) {
      if (pre.closest('.mermaid, .chartjs-container, .csv-table, .post-copy, .sidenote')) return;
      if (pre.querySelector('code.language-mermaid, code.language-chartjs, code.language-contribution-graph')) return;
      var code = pre.querySelector('code');
      if (!pre.classList.contains('chroma') && !code) return;

      // Server-rendered fences arrive wrapped in .code-block; wrap anything else.
      var wrapper = pre.closest('.code-block');
      if (!wrapper) {
        wrapper = document.createElement('div');
        wrapper.className = 'code-block';
        pre.parentNode.insertBefore(wrapper, pre);
        wrapper.appendChild(pre);
      }
      if (wrapper.querySelector('.code-block__copy')) return;

      var header = wrapper.querySelector('.code-block__header');
      if (!header) {
        header = document.createElement('div');
        header.className = 'code-block__header code-block__header--bare';
        wrapper.insertBefore(header, wrapper.firstChild);
      }

      var btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'code-block__copy';
      btn.setAttribute('aria-label', 'Copy code');
      btn.innerHTML = '<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M9 9V4a1 1 0 0 1 1-1h9a1 1 0 0 1 1 1v11a1 1 0 0 1-1 1h-5v-2h4V5h-7v4H9Zm-4 3h9a1 1 0 0 1 1 1v7a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1v-7a1 1 0 0 1 1-1Zm1 2v5h7v-5H6Z" fill="currentColor"/></svg><span>Copy</span>';
      btn.addEventListener('click', function() {
        copyText(codeText(pre)).then(function() {
          btn.classList.add('code-block__copy--done');
          btn.querySelector('span').textContent = 'Copied';
          setTimeout(function() {
            btn.classList.remove('code-block__copy--done');
            btn.querySelector('span').textContent = 'Copy';
          }, 1600);
        }).catch(function() {
          btn.querySelector('span').textContent = 'Failed';
        });
      });
      header.appendChild(btn);
    });
  }

  /* ------------------------------------------------------------------------
   * Back to top
   * --------------------------------------------------------------------- */
  function initBackToTop() {
    var existing = document.querySelector('.back-to-top');
    if (existing) existing.remove();

    var btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'back-to-top';
    btn.setAttribute('aria-label', 'Back to top');
    btn.innerHTML = '<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M12 5l-7 7 1.4 1.4L11 8.8V20h2V8.8l4.6 4.6L19 12l-7-7Z" fill="currentColor"/></svg>';
    btn.addEventListener('click', function() {
      window.scrollTo({ top: 0, behavior: prefersReducedMotion() ? 'auto' : 'smooth' });
      var main = document.getElementById('main-content');
      if (main) main.focus({ preventScroll: true });
    });
    document.body.appendChild(btn);

    var ticking = false;
    function update() {
      ticking = false;
      btn.classList.toggle('back-to-top--visible', window.scrollY > window.innerHeight * 1.2);
    }
    function onScroll() {
      if (ticking) return;
      ticking = true;
      requestAnimationFrame(update);
    }
    window.addEventListener('scroll', onScroll, { passive: true });
    onCleanup(function() {
      window.removeEventListener('scroll', onScroll);
      btn.remove();
    });
    update();
  }

  /* ------------------------------------------------------------------------
   * Init
   * --------------------------------------------------------------------- */
  function init() {
    runCleanups();
    var article = document.querySelector('article.post');
    if (!article) return;
    initToc(article);
    initSidenotes(article);
    initCodeChrome(article);
    initBackToTop();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  // view-transitions.js calls initReading after each DOM swap.
  window.initReading = init;
})();
