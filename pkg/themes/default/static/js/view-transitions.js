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
  let navigationInFlight = false;

  function getNavigationPath(url) {
    if (!url) return '';
    return url.pathname || '';
  }

  function getNavigationKey(url) {
    if (!url) return '';
    return `${url.pathname || ''}${url.search || ''}`;
  }

  function createSharedTransitionToken(path) {
    const token = String(path || '')
      .toLowerCase()
      .replace(/[^a-z0-9_-]+/g, '-')
      .replace(/^-+|-+$/g, '');

    return token || 'post';
  }

  function createSharedTransitionNames(path) {
    const token = createSharedTransitionToken(path);
    return {
      shell: `post-shell-${token}`,
      title: `post-title-${token}`,
    };
  }

  function createSidebarTransitionNames(path) {
    const token = createSharedTransitionToken(path);
    return {
      shell: `sidebar-header-${token}`,
      title: `sidebar-title-${token}`,
    };
  }

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

  function shouldTransitionToURL(url) {
    if (!url) return false;

    if (url.origin !== window.location.origin) return false;
    if (shouldSkipSpecialRoutes(url)) return false;
    if (isNonHTMLDocumentURL(url)) return false;

    if (url.pathname === window.location.pathname &&
        url.search === window.location.search) {
      return false;
    }

    return true;
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

    if (!shouldTransitionToURL(url)) {
      if (config.debug && isNonHTMLDocumentURL(url)) {
        console.log('Skipping non-HTML navigation:', url.href);
      }
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

  function resolveCardRoot(link) {
    if (!link) return null;
    return link.closest('.card, .photo-figure');
  }

  function resolvePrimaryCardLink(cardRoot) {
    if (!cardRoot) return null;

    return cardRoot.querySelector([
      'a.u-url[href]',
      '.card-title a[href]',
      'a.card-cover[href]',
      'a.card-video-link[href]',
      'a.card-image-link[href]',
      'a.card-link[href]',
      'a[href]'
    ].join(', '));
  }

  function resolveCardTitleElement(cardRoot) {
    if (!cardRoot) return null;

    return cardRoot.querySelector([
      '.card-title',
      '.card-caption',
      'figcaption'
    ].join(', '));
  }

  function getSharedTransitionContext(link, targetURL) {
    const cardRoot = resolveCardRoot(link);
    if (!cardRoot) return null;

    const primaryLink = resolvePrimaryCardLink(cardRoot);
    if (!primaryLink) return null;

    let primaryURL;
    try {
      primaryURL = new URL(primaryLink.href, window.location.href);
    } catch (_) {
      return null;
    }

    if (getNavigationKey(primaryURL) !== getNavigationKey(targetURL)) {
      return null;
    }

    return {
      path: getNavigationPath(targetURL),
      names: createSharedTransitionNames(getNavigationPath(targetURL)),
      shell: cardRoot,
      title: resolveCardTitleElement(cardRoot),
      incomingShellSelector: '[data-shared-transition-path]',
      incomingTitleSelector: '[data-shared-transition-title]',
    };
  }

  function getSidebarSharedTransitionContext(link, targetURL) {
    if (!link || !link.closest('.feed-sidebar')) return null;

    const row = link.closest('[data-sidebar-transition-path]');
    if (!row) return null;

    if (row.getAttribute('data-sidebar-transition-path') !== getNavigationPath(targetURL)) {
      return null;
    }

    return {
      path: getNavigationPath(targetURL),
      names: createSidebarTransitionNames(getNavigationPath(targetURL)),
      shell: row,
      title: row.querySelector('[data-sidebar-transition-title]'),
      incomingShellSelector: '[data-sidebar-transition-header]',
      incomingTitleSelector: '[data-shared-transition-title]',
    };
  }

  function markSharedTransitionElement(element, name) {
    if (!element || !name) return;

    element.style.setProperty('view-transition-name', name);
    element.setAttribute('data-shared-transition-active', 'true');
  }

  function setSharedTransitionState(active) {
    if (active) {
      document.documentElement.setAttribute('data-shared-transition-active', 'true');
      return;
    }

    document.documentElement.removeAttribute('data-shared-transition-active');
  }

  function setPostNavigationState(context) {
    if (!context) return;

    document.documentElement.setAttribute('data-post-transition-active', 'true');
    document.documentElement.setAttribute('data-post-transition-source', context.source);
    document.documentElement.setAttribute('data-post-transition-direction', context.direction);
  }

  function clearPostNavigationState() {
    document.documentElement.removeAttribute('data-post-transition-active');
    document.documentElement.removeAttribute('data-post-transition-source');
    document.documentElement.removeAttribute('data-post-transition-direction');
  }

  function resolveSidebarFeedLinks() {
    return Array.from(document.querySelectorAll('.feed-sidebar a[href]')).filter((link) => {
      try {
        const url = new URL(link.href, window.location.href);
        return shouldTransitionToURL(url);
      } catch (_) {
        return false;
      }
    });
  }

  function getSidebarDirection(link) {
    if (!link || !link.closest('.feed-sidebar')) return null;

    const links = resolveSidebarFeedLinks();
    if (!links.length) return null;

    const currentPath = window.location.pathname;
    const activeIndex = links.findIndex((candidate) => {
      try {
        return new URL(candidate.href, window.location.href).pathname === currentPath;
      } catch (_) {
        return false;
      }
    });
    const targetIndex = links.indexOf(link);

    if (activeIndex === -1 || targetIndex === -1 || activeIndex === targetIndex) {
      return null;
    }

    return targetIndex > activeIndex ? 'next' : 'prev';
  }

  function getPostNavigationContext(link) {
    if (!link) return null;

    if (link.closest('.post-nav')) {
      if (link.matches('.next, [data-action="next"]')) {
        return { source: 'post-nav', direction: 'next' };
      }

      if (link.matches('.prev, [data-action="prev"]')) {
        return { source: 'post-nav', direction: 'prev' };
      }
    }

    const sidebarDirection = getSidebarDirection(link);
    if (sidebarDirection) {
      return { source: 'sidebar', direction: sidebarDirection };
    }

    return null;
  }

  function clearSharedTransitionElements(root) {
    if (!root || !root.querySelectorAll) return;

    root.querySelectorAll('[data-shared-transition-active]').forEach((element) => {
      element.style.removeProperty('view-transition-name');
      element.removeAttribute('data-shared-transition-active');
    });
  }

  function findIncomingSharedTransitionTarget(newDoc, context) {
    const selector = context && context.incomingShellSelector ? context.incomingShellSelector : '[data-shared-transition-path]';
    const candidates = newDoc.querySelectorAll(selector);

    for (const candidate of candidates) {
      if (!context || !context.path || !candidate.hasAttribute('data-shared-transition-path')) {
        return candidate;
      }

      if (candidate.getAttribute('data-sidebar-transition-path') === context.path ||
          candidate.getAttribute('data-shared-transition-path') === context.path) {
        return candidate;
      }
    }

    return null;
  }

  function activateSharedTransitionContext(context) {
    if (!context) return;

    setSharedTransitionState(true);
    markSharedTransitionElement(context.shell, context.names.shell);
    markSharedTransitionElement(context.title, context.names.title);
  }

  function prepareIncomingSharedTransition(newDoc, context) {
    if (!context) return;

    const target = findIncomingSharedTransitionTarget(newDoc, context);
    if (!target) return;

    markSharedTransitionElement(target, context.names.shell);
    markSharedTransitionElement(target.querySelector(context.incomingTitleSelector || '[data-shared-transition-title]'), context.names.title);
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

  function syncDocumentElementAttributes(newDoc) {
    if (!newDoc || !newDoc.documentElement) return;

    const currentAttributes = Array.from(document.documentElement.attributes);
    for (const attribute of currentAttributes) {
      if (!newDoc.documentElement.hasAttribute(attribute.name)) {
        document.documentElement.removeAttribute(attribute.name);
      }
    }

    Array.from(newDoc.documentElement.attributes).forEach((attribute) => {
      document.documentElement.setAttribute(attribute.name, attribute.value);
    });
  }

  function stylesheetKey(node) {
    if (!node) return '';
    if (node.tagName === 'LINK') {
      return ['link', node.getAttribute('rel') || '', node.getAttribute('href') || '', node.getAttribute('media') || ''].join('::');
    }
    if (node.tagName === 'STYLE') {
      return ['style', node.textContent || ''].join('::');
    }
    return '';
  }

  function syncHeadStyles(newDoc) {
    if (!newDoc || !newDoc.head) return;

    const selector = 'link[rel="stylesheet"], style';
    const currentNodes = Array.from(document.head.querySelectorAll(selector));
    const nextNodes = Array.from(newDoc.head.querySelectorAll(selector));
    const nextKeys = new Set(nextNodes.map(stylesheetKey).filter(Boolean));

    currentNodes.forEach((node) => {
      const key = stylesheetKey(node);
      if (key && !nextKeys.has(key)) {
        node.remove();
      }
    });

    const head = document.head;
    let previousInserted = null;

    nextNodes.forEach((node) => {
      const key = stylesheetKey(node);
      if (!key) return;

      const existing = Array.from(head.querySelectorAll(selector)).find((candidate) => stylesheetKey(candidate) === key);
      if (existing) {
        previousInserted = existing;
        return;
      }

      const clone = node.cloneNode(true);
      if (previousInserted && previousInserted.parentNode === head) {
        previousInserted.insertAdjacentElement('afterend', clone);
      } else {
        head.appendChild(clone);
      }
      previousInserted = clone;
    });
  }

  function headScriptKey(node) {
    if (!node || node.tagName !== 'SCRIPT') return '';
    const src = node.getAttribute('src') || '';
    if (src) {
      return ['script', node.getAttribute('type') || '', src].join('::');
    }
    // Inline scripts in <head> are keyed by their text content. We do NOT
    // re-execute matching inline scripts; we only add ones that don't already
    // exist on the live document.
    return ['script', node.getAttribute('type') || '', 'inline', node.textContent || ''].join('::');
  }

  /**
   * Sync <script> tags from the new document's <head> into the live document.
   *
   * Without this, navigating from a page that does not load a particular head
   * script (for example the Web Awesome module loader) into a page that does
   * means the script never runs and custom elements (wa-*) never upgrade.
   *
   * We only ADD missing scripts; we do not remove or re-execute scripts that
   * already exist, since module imports are idempotent and re-injecting a
   * loader can re-register custom elements (which throws).
   */
  function syncHeadScripts(newDoc) {
    if (!newDoc || !newDoc.head) return;

    const head = document.head;
    const existingKeys = new Set(
      Array.from(head.querySelectorAll('script')).map(headScriptKey).filter(Boolean)
    );

    Array.from(newDoc.head.querySelectorAll('script')).forEach((node) => {
      const key = headScriptKey(node);
      if (!key || existingKeys.has(key)) return;

      // Re-create the element so the browser actually executes it. Cloning a
      // <script> node parsed by DOMParser does not run; we must construct a
      // fresh element and copy over attributes/contents.
      const fresh = document.createElement('script');
      Array.from(node.attributes).forEach((attr) => {
        fresh.setAttribute(attr.name, attr.value);
      });
      if (!node.hasAttribute('src')) {
        fresh.textContent = node.textContent || '';
      }
      head.appendChild(fresh);
      existingKeys.add(key);
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
    var feedList = document.querySelector('#feed-nav-collapsible');
    var savedScrollTop = feedList ? feedList.scrollTop : 0;

    const replacedPage = replaceElementContents('#view-transition-page', newDoc);
    if (!replacedPage) {
      document.body.innerHTML = newDoc.body.innerHTML;
      return false;
    }

    // Restore feed sidebar scroll position after content swap
    var newFeedList = document.querySelector('#feed-nav-collapsible');
    if (newFeedList && savedScrollTop > 0) {
      newFeedList.scrollTop = savedScrollTop;
    }

    replaceElement('#view-transition-progress', newDoc);
    replaceElementContents('.site-footer', newDoc);
    return true;
  }

  /**
   * Update the current document with content from new document
   */
  function updateDocument(newDoc, metrics) {
    const swapStartedAt = getNow();

    syncDocumentElementAttributes(newDoc);
    syncBodyAttributes(newDoc);
    syncHeadStyles(newDoc);
    syncHeadScripts(newDoc);

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

  function preserveFeedContext(rawUrl, triggerElement) {
    try {
      const currentURL = new URL(window.location.href);
      const targetURL = new URL(rawUrl, window.location.href);
      const currentFeed = currentURL.searchParams.get('feed');

      if (!currentFeed) return targetURL.href;
      if (targetURL.searchParams.get('feed')) return targetURL.href;
      if (!triggerElement) return targetURL.href;

      if (triggerElement.closest('.feed-sidebar, .post-nav')) {
        targetURL.searchParams.set('feed', currentFeed);
      }

      return targetURL.href;
    } catch (_) {
      return rawUrl;
    }
  }

  /**
   * Re-initialize scripts after content replacement
   */
  function reinitializeScripts() {
    function callInit(name, fn) {
      if (typeof fn !== 'function') return;
      try {
        fn();
      } catch (error) {
        if (config.debug) console.warn('View transition init failed:', name, error);
      }
    }

    // Dispatch custom event for other scripts to listen to
    window.dispatchEvent(new CustomEvent('view-transition-complete'));

    // Pagefind Search (navbar) - needs re-init after DOM swap
    callInit('initPagefindSearch', window.initPagefindSearch);
    callInit('initBleveSearch', window.initBleveSearch);

    // Re-initialize feed cycling first so sidebar chrome is hydrated before
    // other scripts inspect feed-aware DOM state.
    callInit('initFeedCycling', window.initFeedCycling);

    // Re-initialize common scripts if they exist
    callInit('initScrollSpy', window.initScrollSpy);
    callInit('initTooltips', window.initTooltips);
    callInit('initMentionCards', window.initMentionCards);
    callInit('initPagination', window.initPagination);
    callInit('initNavigationShortcuts', window.initNavigationShortcuts);

    // Re-scroll feed sidebar active item into view (inline scripts don't re-run after DOM swap)
    callInit('initFeedSidebarScroll', window.initFeedSidebarScroll);

    // Re-bind feed sidebar collapse toggle (tablet/mobile)
    callInit('initSidebarToggle', window.initSidebarToggle);

    // Close hamburger menu after navigation (header is outside #view-transition-page so it persists)
    var openHamburger = document.querySelector('.hamburger-toggle--open');
    if (openHamburger) {
      openHamburger.classList.remove('hamburger-toggle--open');
      openHamburger.setAttribute('aria-expanded', 'false');
      var navGroup = document.querySelector('.mobile-nav-group--open');
      if (navGroup) navGroup.classList.remove('mobile-nav-group--open');
    }

    // Re-initialize mermaid diagrams (module script won't re-execute after DOM swap)
    callInit('initMermaid', window.initMermaid);

    // Re-attach event listeners
    initNavigationInterceptor();
  }

  async function navigateWithViewTransition(url, options) {
    const navOptions = Object.assign({
      source: 'script',
      pushState: true,
      triggerElement: null,
      bypassTransition: false,
    }, options || {});

    if (navigationInFlight) {
      if (config.debug) console.log('Skipping navigation while transition is active:', url);
      return false;
    }

    let targetURL;
    try {
      targetURL = new URL(preserveFeedContext(url, navOptions.triggerElement), window.location.href);
    } catch (_) {
      window.location.href = url;
      return false;
    }

    if (!shouldTransitionToURL(targetURL)) {
      window.location.href = targetURL.href;
      return false;
    }

    if (navOptions.bypassTransition) {
      const metrics = createNavigationMetrics(targetURL.href, navOptions.source);
      const newDoc = await resolveDocument(targetURL.href, metrics);
      updateDocument(newDoc, metrics);

      if (navOptions.pushState) {
        history.pushState(null, '', targetURL.href);
      }

      finalizeNavigationMetrics(metrics);
      return true;
    }

    navigationInFlight = true;

    const metrics = createNavigationMetrics(targetURL.href, navOptions.source);
    const newDoc = await resolveDocument(targetURL.href, metrics);
    const sharedContext = getSidebarSharedTransitionContext(navOptions.triggerElement, targetURL) || getSharedTransitionContext(navOptions.triggerElement, targetURL);
    const postNavigationContext = getPostNavigationContext(navOptions.triggerElement);

    if (sharedContext) {
      activateSharedTransitionContext(sharedContext);
    }

    if (postNavigationContext) {
      setPostNavigationState(postNavigationContext);
    }

    try {
      const transition = document.startViewTransition(() => {
        prepareIncomingSharedTransition(newDoc, sharedContext);
        updateDocument(newDoc, metrics);
        if (navOptions.pushState) {
          history.pushState(null, '', targetURL.href);
        }
      });

      await transition.finished;
      finalizeNavigationMetrics(metrics);
      return true;
    } finally {
      clearSharedTransitionElements(document);
      setSharedTransitionState(false);
      clearPostNavigationState();
      navigationInFlight = false;
    }
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

    const url = preserveFeedContext(link.href, link);

    if (config.debug) console.log('Starting view transition to:', url);

    try {
      await navigateWithViewTransition(url, {
        source: 'click',
        pushState: true,
        triggerElement: link,
      });

      if (config.debug) console.log('View transition completed');
    } catch (error) {
      console.error('View transition failed:', error);
      clearSharedTransitionElements(document);
      setSharedTransitionState(false);
      clearPostNavigationState();
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
    window.navigateWithViewTransition = navigateWithViewTransition;

    // Expose config for runtime inspection
    window.VIEW_TRANSITIONS_CONFIG = config;
  }

  // ── Feed Sidebar: scroll active item into view ──
  // Defined outside init() so it's available even if view transitions are disabled.
  // Uses manual scrollTop math so that only the sidebar container scrolls --
  // scrollIntoView() would scroll every ancestor including the page viewport.
  window.initFeedSidebarScroll = function() {
    var active = document.querySelector('.feed-nav-item--active');
    if (!active) return;
    var container = document.getElementById('feed-nav-collapsible');
    if (!container) return;
    var activeTop = active.offsetTop - container.offsetTop;
    var activeHeight = active.offsetHeight;
    var containerHeight = container.clientHeight;
    // Center the active item within the scrollable container
    container.scrollTop = activeTop - (containerHeight / 2) + (activeHeight / 2);
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
