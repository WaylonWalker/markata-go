/**
 * Conditional CSS Runtime Loader
 *
 * Handles CSS injection after view-transition SPA navigations.
 * When markata-go uses view transitions, only document.body.innerHTML is
 * swapped - <head> content persists. This means:
 *   - CSS loaded on page A stays available on page B (harmless, cached).
 *   - CSS NOT loaded on page A will be missing on page B if page B needs it.
 *
 * This script detects DOM elements requiring conditional CSS and injects
 * the appropriate <link> tags if they are not already present in <head>.
 */
(function() {
  'use strict';

  // Map of CSS base names to DOM selectors that require them.
  var conditionalCSS = {
    'admonitions': '.admonition',
    'code':        'pre > code, .highlight, code[class*="language-"]',
    'chroma':      '.chroma',
    'cards':       '.card, .posts-list',
    'webmentions': '.webmentions',
    'encryption':  '.encrypted-content, [data-encrypted]'
  };

  /**
   * Check if a stylesheet containing the given base name is already loaded.
   */
  function hasCSS(baseName) {
    var links = document.querySelectorAll('link[rel="stylesheet"]');
    for (var i = 0; i < links.length; i++) {
      if (links[i].href && links[i].href.indexOf(baseName) !== -1) {
        return true;
      }
    }
    return false;
  }

  /**
   * Inject a CSS file by base name. Looks for a matching link already in
   * the page to derive the full hashed path; if not found, falls back to
   * /css/{baseName}.css.
   */
  function injectCSS(baseName) {
    var href = '/css/' + baseName + '.css';
    var link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = href;
    document.head.appendChild(link);
  }

  /**
   * Scan the current DOM and inject any missing conditional CSS.
   */
  function loadConditionalCSS() {
    for (var baseName in conditionalCSS) {
      if (!conditionalCSS.hasOwnProperty(baseName)) continue;
      var selector = conditionalCSS[baseName];
      if (document.querySelector(selector) && !hasCSS(baseName)) {
        injectCSS(baseName);
      }
    }

    // Handle decryption.js for encrypted content
    if (document.querySelector('.encrypted-content, [data-encrypted]')) {
      var scripts = document.querySelectorAll('script[src]');
      var hasDecryption = false;
      for (var j = 0; j < scripts.length; j++) {
        if (scripts[j].src.indexOf('decryption') !== -1) {
          hasDecryption = true;
          break;
        }
      }
      if (!hasDecryption) {
        var script = document.createElement('script');
        script.src = '/js/decryption.js';
        script.defer = true;
        document.body.appendChild(script);
      }
    }

    // Handle pagefind re-initialization after navigation
    if (window.loadPagefind && document.getElementById('pagefind-search')) {
      // Pagefind lazy loader already handles this
      window.loadPagefind();
    } else if (window.initPagefindSearch) {
      window.initPagefindSearch();
    }

    // Handle GLightbox re-initialization after navigation
    if (window.initGLightbox && document.querySelector('.glightbox')) {
      window.initGLightbox();
    }
  }

  // Run after view transitions complete
  window.addEventListener('view-transition-complete', loadConditionalCSS);
})();
