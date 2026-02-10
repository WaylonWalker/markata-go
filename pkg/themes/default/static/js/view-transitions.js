/**
 * View Transitions API - Global Navigation Interceptor
 *
 * Provides smooth animated transitions for all internal navigation.
 * Gracefully degrades on unsupported browsers.
 *
 * Handles:
 * - Wikilinks, card links, nav/footer links, post navigation
 * - Back/forward browser buttons
 * - Script re-initialization after transitions
 * - Error handling and fallbacks
 *
 * Configuration (set window.VIEW_TRANSITIONS_CONFIG in your template):
 * {
 *   enabled: true,              // Enable/disable view transitions
 *   debug: false,               // Log debug messages
 *   skipClasses: [],            // Additional CSS classes to skip
 *   skipSelectors: [],          // Additional selectors to skip (e.g., '[data-no-transition]')
 *   transitionDuration: 300,    // Default transition duration in ms (for prefetching timeout)
 *   updateMeta: true,           // Update meta tags on navigation
 *   scrollToTop: true,          // Scroll to top on navigation (unless hash present)
 * }
 */

(function() {
  'use strict';

  // Default configuration
  const DEFAULT_CONFIG = {
    enabled: true,
    debug: false,
    skipClasses: [],
    skipSelectors: [],
    transitionDuration: 150,
    updateMeta: true,
    scrollToTop: true,
  };

  const HTML_EXTENSIONS = new Set(['.html', '.htm', '.xhtml']);

  function normalizePathname(p) {
    if (typeof p !== 'string') return '';
    let s = p.trim();
    if (!s) return '';
    // Drop origin if an absolute URL was provided.
    try {
      if (s.startsWith('http://') || s.startsWith('https://')) {
        s = new URL(s).pathname;
      }
    } catch (_) {
      // ignore
    }
    if (!s.startsWith('/')) s = '/' + s;
    // Collapse multiple slashes.
    s = s.replace(/\/{2,}/g, '/');
    return s;
  }

  function shouldSkipSpecialRoutes(url) {
    if (!url || !url.pathname) return false;

    const pathname = url.pathname;

    // Random post endpoint performs a client-side redirect.
    // View-transition navigation swaps DOM without executing inline scripts,
    // which prevents the redirect script from running.
    const randomCandidates = new Set(['/random', '/random/']);
    const configured = normalizePathname(window.MARKATA_GO_RANDOM_POST_PATH);
    if (configured) {
      randomCandidates.add(configured);
      if (configured.endsWith('/')) {
        randomCandidates.add(configured.slice(0, -1));
      } else {
        randomCandidates.add(configured + '/');
      }
    }

    if (randomCandidates.has(pathname)) {
      if (config.debug) console.log('Skipping special route navigation:', pathname);
      return true;
    }

    return false;
  }

  function getLowercaseExtension(pathname) {
    const lastSegment = (pathname || '').split('/').pop() || '';
    const dotIndex = lastSegment.lastIndexOf('.');
    if (dotIndex <= 0) return '';
    return lastSegment.slice(dotIndex).toLowerCase();
  }

  function isNonHTMLDocumentURL(url) {
    const ext = getLowercaseExtension(url.pathname);
    if (!ext) return false; // Pretty URLs and extensionless routes are assumed HTML.
    return !HTML_EXTENSIONS.has(ext);
  }

  // Merge with user config
  const config = Object.assign({}, DEFAULT_CONFIG, window.VIEW_TRANSITIONS_CONFIG || {});

  // Exit early if disabled
  if (!config.enabled) {
    if (config.debug) console.log('View Transitions disabled via config');
    return;
  }

  // Feature detection - exit early if not supported
  if (!document.startViewTransition) {
    if (config.debug) console.log('View Transitions API not supported, using standard navigation');
    return;
  }

  if (config.debug) console.log('View Transitions API enabled', config);

  /**
   * Check if a link should use view transitions
   */
  function shouldTransition(link) {
    // Must be an <a> element with href
    if (!link || !link.href) return false;

    // Parse URL
    let url;
    try {
      url = new URL(link.href);
    } catch (e) {
      return false;
    }

    // Only handle same-origin links
    if (url.origin !== window.location.origin) return false;

    // Skip special routes that must use full navigation
    if (shouldSkipSpecialRoutes(url)) return false;

    // Only transition between HTML documents
    // (Non-HTML resources like .md/.txt/.xml/.json should use native browser navigation.)
    if (isNonHTMLDocumentURL(url)) {
      if (config.debug) console.log('Skipping non-HTML navigation:', url.href);
      return false;
    }

    // Skip if same page (just hash changes)
    if (url.pathname === window.location.pathname &&
        url.search === window.location.search) {
      return false;
    }

    // Skip external links (target=_blank)
    if (link.target === '_blank') return false;

    // Skip non-default navigation targets
    if (link.target && link.target !== '_self') return false;

    // Skip download links
    if (link.hasAttribute('download')) return false;

    // Skip links with rel=external
    if (link.rel && link.rel.includes('external')) return false;

    // Skip TOC links (they have their own smooth scroll handling)
    if (link.classList.contains('toc-link')) return false;

    // Skip HTMX-managed links (they handle their own updates)
    if (link.hasAttribute('hx-get') || link.hasAttribute('hx-post')) return false;

    // Skip GLightbox links (inline lightbox should not trigger navigation)
    if (link.classList.contains('glightbox-mermaid') || link.hasAttribute('data-glightbox')) return false;

    // Skip links that explicitly opt-out
    if (link.dataset.noTransition) return false;

    // Skip user-configured classes
    for (const className of config.skipClasses) {
      if (link.classList.contains(className)) {
        if (config.debug) console.log('Skipping link with class:', className);
        return false;
      }
    }

    // Skip user-configured selectors
    for (const selector of config.skipSelectors) {
      if (link.matches(selector)) {
        if (config.debug) console.log('Skipping link matching selector:', selector);
        return false;
      }
    }

    return true;
  }

  /**
   * Navigate to a URL with view transition
   */
  async function navigateWithTransition(url) {
    // Fetch new page content with cache bypass to ensure fresh content
    // This prevents stale content (e.g., wrong pagination active state) after navigation
    const response = await fetch(url, {
      cache: 'no-store',
      headers: {
        'Accept': 'text/html,application/xhtml+xml',
      },
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    const contentType = (response.headers.get('content-type') || '').toLowerCase();
    if (!contentType.includes('text/html') && !contentType.includes('application/xhtml+xml')) {
      throw new Error(`Non-HTML navigation (content-type: ${contentType || 'unknown'})`);
    }

    const html = await response.text();

    // Parse new document
    const parser = new DOMParser();
    const newDoc = parser.parseFromString(html, 'text/html');

    // Update document
    updateDocument(newDoc);
  }

  /**
   * Update the current document with content from new document
   */
  function updateDocument(newDoc) {
    // Update title
    document.title = newDoc.title;

    // Update meta tags (description, og tags, etc.)
    if (config.updateMeta) {
      updateMetaTags(newDoc);
    }

    // Replace body content
    document.body.innerHTML = newDoc.body.innerHTML;

    // Re-initialize scripts
    reinitializeScripts();

    // Scroll to top (or to hash if present)
    if (config.scrollToTop) {
      if (window.location.hash) {
        const target = document.querySelector(window.location.hash);
        if (target) {
          target.scrollIntoView({ behavior: 'smooth' });
        }
      } else {
        window.scrollTo(0, 0);
      }
    }
  }

  /**
   * Update meta tags from new document
   */
  function updateMetaTags(newDoc) {
    const metaSelectors = [
      'meta[name="description"]',
      'meta[property^="og:"]',
      'meta[name^="twitter:"]',
      'link[rel="canonical"]'
    ];

    metaSelectors.forEach(selector => {
      const oldMeta = document.querySelector(selector);
      const newMeta = newDoc.querySelector(selector);

      if (newMeta) {
        if (oldMeta) {
          oldMeta.replaceWith(newMeta.cloneNode(true));
        } else {
          document.head.appendChild(newMeta.cloneNode(true));
        }
      } else if (oldMeta) {
        oldMeta.remove();
      }
    });
  }

  /**
   * Re-initialize scripts after content replacement
   */
  function reinitializeScripts() {
    // Dispatch custom event for other scripts to listen to
    window.dispatchEvent(new CustomEvent('view-transition-complete'));

    // Pagefind Search (navbar) - needs re-init after DOM swap
    if (window.initPagefindSearch && typeof window.initPagefindSearch === 'function') {
      window.initPagefindSearch();
    }

    // Re-initialize common scripts if they exist
    if (window.initScrollSpy && typeof window.initScrollSpy === 'function') {
      window.initScrollSpy();
    }

    if (window.initTooltips && typeof window.initTooltips === 'function') {
      window.initTooltips();
    }

    if (window.initMentionCards && typeof window.initMentionCards === 'function') {
      window.initMentionCards();
    }

    if (window.initPagination && typeof window.initPagination === 'function') {
      window.initPagination();
    }

    if (window.initNavigationShortcuts && typeof window.initNavigationShortcuts === 'function') {
      window.initNavigationShortcuts();
    }

    // Re-attach event listeners
    initNavigationInterceptor();
  }

  /**
   * Handle navigation click with view transition
   */
  async function handleNavigationClick(event) {
    // Respect other handlers and normal browser behaviors
    if (event.defaultPrevented) return;
    if (event.button !== 0) return; // Only left clicks
    if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;

    const link = event.target.closest('a');

    if (!shouldTransition(link)) return;

    // Prevent default navigation
    event.preventDefault();

    const url = link.href;

    if (config.debug) console.log('Starting view transition to:', url);

    try {
      // Start view transition
      const transition = document.startViewTransition(async () => {
        await navigateWithTransition(url);
        // Update URL after content is loaded
        history.pushState(null, '', url);
      });

      // Wait for transition to complete
      await transition.finished;

      if (config.debug) console.log('View transition completed');
    } catch (error) {
      console.error('View transition failed:', error);
      // Fallback to normal navigation
      window.location.href = url;
    }
  }

  /**
   * Handle back/forward button navigation
   */
  async function handlePopState(event) {
    if (config.debug) console.log('Handling popstate to:', window.location.href);

    // If we land on a special route, force a full reload so its scripts run.
    try {
      const currentURL = new URL(window.location.href);
      if (shouldSkipSpecialRoutes(currentURL)) {
        window.location.reload();
        return;
      }
    } catch (_) {
      // ignore
    }

    try {
      const transition = document.startViewTransition(async () => {
        await navigateWithTransition(window.location.href);
      });

      await transition.finished;
    } catch (error) {
      console.error('View transition failed on popstate:', error);
      // Fallback to reload
      window.location.reload();
    }
  }

  /**
   * Initialize click interceptor for navigation links
   */
  function initNavigationInterceptor() {
    // Use event delegation on document for better performance
    document.removeEventListener('click', handleNavigationClick);
    document.addEventListener('click', handleNavigationClick);
  }

  /**
   * Initialize view transitions on page load
   */
  function init() {
    if (config.debug) console.log('View Transitions API initialized');

    // Intercept navigation clicks
    initNavigationInterceptor();

    // Handle browser back/forward buttons
    window.addEventListener('popstate', handlePopState);

    // Expose reinitialization function for other scripts
    window.reinitViewTransitions = initNavigationInterceptor;

    // Expose config for runtime inspection
    window.VIEW_TRANSITIONS_CONFIG = config;
  }

  // Initialize when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
