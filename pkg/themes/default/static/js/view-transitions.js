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
    transitionDuration: 120,
    updateMeta: true,
    scrollToTop: true,
  };

  const HTML_EXTENSIONS = new Set(['.html', '.htm', '.xhtml']);
  const prefetchedDocuments = new Map();

  function getNow() {
    if (window.performance && typeof window.performance.now === 'function') {
      return window.performance.now();
    }

    return Date.now();
  }

  function roundTiming(value) {
    return Math.round(value * 10) / 10;
  }

  function createNavigationMetrics(url, source) {
    return {
      url: url,
      source: source,
      prefetched: false,
      fetchMs: 0,
      parseMs: 0,
      swapMs: 0,
      reinitMs: 0,
      totalMs: 0,
      startedAt: getNow(),
    };
  }

  function finalizeNavigationMetrics(metrics) {
    if (!metrics) return;

    metrics.totalMs = roundTiming(getNow() - metrics.startedAt);

    const finalized = {
      url: metrics.url,
      source: metrics.source,
      prefetched: metrics.prefetched,
      fetchMs: roundTiming(metrics.fetchMs),
      parseMs: roundTiming(metrics.parseMs),
      swapMs: roundTiming(metrics.swapMs),
      reinitMs: roundTiming(metrics.reinitMs),
      totalMs: metrics.totalMs,
    };

    window.__lastViewTransitionMetrics = finalized;
    window.dispatchEvent(new CustomEvent('view-transition-timing', { detail: finalized }));

    if (config.debug) {
      console.log('[view-transitions] timing', finalized);
    }

    return finalized;
  }

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
   * Fetch document for transition
   */
  function buildFetchOptions(extraOptions) {
    const options = Object.assign({}, extraOptions || {});
    options.headers = Object.assign({
      'Accept': 'text/html,application/xhtml+xml',
    }, options.headers || {});

    return options;
  }

  async function fetchDocument(url, options) {
    const metrics = options && options.metrics;
    const fetchOptions = options && options.fetchOptions;
    const fetchStartedAt = getNow();

    // Fetch new page content.
    // Allow default caching so prefetching and back/forward navigation is instantaneous.
    const response = await fetch(url, buildFetchOptions(fetchOptions));

    if (metrics) {
      metrics.fetchMs += getNow() - fetchStartedAt;
    }

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    const contentType = (response.headers.get('content-type') || '').toLowerCase();
    if (!contentType.includes('text/html') && !contentType.includes('application/xhtml+xml')) {
      throw new Error(`Non-HTML navigation (content-type: ${contentType || 'unknown'})`);
    }

    const html = await response.text();

    // Parse new document
    const parseStartedAt = getNow();
    const parser = new DOMParser();
    const newDoc = parser.parseFromString(html, 'text/html');

    if (metrics) {
      metrics.parseMs += getNow() - parseStartedAt;
    }

    return newDoc;
  }

  async function resolveDocument(url, metrics) {
    if (prefetchedDocuments.has(url)) {
      metrics.prefetched = true;

      const prefetchedDocument = prefetchedDocuments.get(url);
      prefetchedDocuments.delete(url);
      return prefetchedDocument;
    }

    return fetchDocument(url, { metrics: metrics });
  }

  function syncBodyAttributes(newDoc) {
    if (!newDoc || !newDoc.body) return;

    const currentAttributes = Array.from(document.body.attributes);
    for (const attribute of currentAttributes) {
      if (!newDoc.body.hasAttribute(attribute.name)) {
        document.body.removeAttribute(attribute.name);
      }
    }

    Array.from(newDoc.body.attributes).forEach((attribute) => {
      document.body.setAttribute(attribute.name, attribute.value);
    });
  }

  function replaceElementContents(selector, newDoc) {
    const currentElement = document.querySelector(selector);
    const nextElement = newDoc.querySelector(selector);

    if (!currentElement || !nextElement) {
      return false;
    }

    currentElement.innerHTML = nextElement.innerHTML;
    return true;
  }

  function replaceElement(selector, newDoc) {
    const currentElement = document.querySelector(selector);
    const nextElement = newDoc.querySelector(selector);

    if (!currentElement || !nextElement) {
      return false;
    }

    currentElement.replaceWith(nextElement.cloneNode(true));
    return true;
  }

  function updateLayoutRegions(newDoc) {
    // Preserve feed sidebar scroll position across DOM swap so
    // the list doesn't jump to top before scrollIntoView runs.
    var feedSidebar = document.querySelector('.feed-sidebar');
    var savedScrollTop = feedSidebar ? feedSidebar.scrollTop : 0;

    const replacedPage = replaceElementContents('#view-transition-page', newDoc);
    if (!replacedPage) {
      document.body.innerHTML = newDoc.body.innerHTML;
      return false;
    }

    // Restore feed sidebar scroll position after content swap
    var newFeedSidebar = document.querySelector('.feed-sidebar');
    if (newFeedSidebar && savedScrollTop > 0) {
      newFeedSidebar.scrollTop = savedScrollTop;
    }

    replaceElement('#view-transition-progress', newDoc);
    return true;
  }

  /**
   * Update the current document with content from new document
   */
  function updateDocument(newDoc, metrics) {
    const swapStartedAt = getNow();

    syncBodyAttributes(newDoc);

    // Update title
    document.title = newDoc.title;

    // Update meta tags (description, og tags, etc.)
    if (config.updateMeta) {
      updateMetaTags(newDoc);
    }

    // Replace the main layout region instead of the entire body when possible.
    updateLayoutRegions(newDoc);

    // Re-execute inline module scripts (e.g. mermaid, chartjs).
    // Replaced fragments do not run <script> tags, so we clone module scripts
    // into fresh elements which the browser will evaluate.
    document.body.querySelectorAll('script[type="module"]').forEach(old => {
      const fresh = document.createElement('script');
      fresh.type = 'module';
      fresh.textContent = old.textContent;
      old.replaceWith(fresh);
    });

    if (metrics) {
      metrics.swapMs += getNow() - swapStartedAt;
    }

    // Re-initialize scripts
    const reinitStartedAt = getNow();
    reinitializeScripts();

    if (metrics) {
      metrics.reinitMs += getNow() - reinitStartedAt;
    }

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

    // Re-scroll feed sidebar active item into view (inline scripts don't re-run after DOM swap)
    if (window.initFeedSidebarScroll && typeof window.initFeedSidebarScroll === 'function') {
      window.initFeedSidebarScroll();
    }

    // Re-initialize feed cycling (parses new page's feed data)
    if (window.initFeedCycling && typeof window.initFeedCycling === 'function') {
      window.initFeedCycling();
    }

    // Re-bind feed sidebar collapse toggle (tablet/mobile)
    if (window.initSidebarToggle && typeof window.initSidebarToggle === 'function') {
      window.initSidebarToggle();
    }

    // Close hamburger menu after navigation (header is outside #view-transition-page so it persists)
    var openHamburger = document.querySelector('.hamburger-toggle--open');
    if (openHamburger) {
      openHamburger.classList.remove('hamburger-toggle--open');
      openHamburger.setAttribute('aria-expanded', 'false');
      var navGroup = document.querySelector('.mobile-nav-group--open');
      if (navGroup) navGroup.classList.remove('mobile-nav-group--open');
    }

    // Re-initialize mermaid diagrams (module script won't re-execute after DOM swap)
    if (window.initMermaid && typeof window.initMermaid === 'function') {
      window.initMermaid();
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
      // 1. Fetch the new document first so the screen doesn't freeze
      // waiting for the network request.
      const metrics = createNavigationMetrics(url, 'click');
      const newDoc = await resolveDocument(url, metrics);

      // 2. Start view transition with the parsed document
      const transition = document.startViewTransition(() => {
        updateDocument(newDoc, metrics);
        // Update URL after content is loaded
        history.pushState(null, '', url);
      });

      // Wait for transition to complete
      await transition.finished;

      finalizeNavigationMetrics(metrics);

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
      const url = window.location.href;
      const metrics = createNavigationMetrics(url, 'popstate');
      const newDoc = await resolveDocument(url, metrics);

      const transition = document.startViewTransition(() => {
        updateDocument(newDoc, metrics);
      });

      await transition.finished;

      finalizeNavigationMetrics(metrics);
    } catch (error) {
      console.error('View transition failed on popstate:', error);
      // Fallback to reload
      window.location.reload();
    }
  }

  /**
   * Prefetch a URL to make subsequent navigation instant
   */
  function prefetchUrl(url) {
    // Only prefetch if we're not heavily resource constrained
    if (navigator.connection && navigator.connection.saveData) {
      return;
    }

    if (!url || prefetchedDocuments.has(url)) return;

    if (config.debug) console.log('Prefetching URL:', url);

    const prefetchedDocument = fetchDocument(url, {
      fetchOptions: { priority: 'low' },
    }).catch((error) => {
      prefetchedDocuments.delete(url);

      if (config.debug) {
        console.warn('[view-transitions] prefetch failed', url, error);
      }

      throw error;
    });

    prefetchedDocuments.set(url, prefetchedDocument);
  }

  /**
   * Handle hovering over links to prefetch them
   */
  function handleLinkHover(event) {
    const link = event.target.closest('a');
    if (!link || !shouldTransition(link)) return;

    prefetchUrl(link.href);
  }

  /**
   * Initialize click interceptor for navigation links
   */
  function initNavigationInterceptor() {
    // Use event delegation on document for better performance
    document.removeEventListener('click', handleNavigationClick);
    document.addEventListener('click', handleNavigationClick);

    // Add prefetch on hover/focus for instant transitions
    document.removeEventListener('mouseover', handleLinkHover);
    document.addEventListener('mouseover', handleLinkHover, { passive: true });

    document.removeEventListener('focusin', handleLinkHover);
    document.addEventListener('focusin', handleLinkHover, { passive: true });
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
    window.prefetchViewTransitionUrl = prefetchUrl;

    // Expose config for runtime inspection
    window.VIEW_TRANSITIONS_CONFIG = config;
  }

  // ── Feed Sidebar: scroll active item into view ──
  // Defined outside init() so it's available even if view transitions are disabled.
  window.initFeedSidebarScroll = function() {
    var active = document.querySelector('.feed-nav-item--active');
    if (active) {
      active.scrollIntoView({ block: 'center', behavior: 'instant' });
    }
  };

  // Initialize when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function() {
      init();
      window.initFeedSidebarScroll();
    });
  } else {
    init();
    window.initFeedSidebarScroll();
  }
})();
