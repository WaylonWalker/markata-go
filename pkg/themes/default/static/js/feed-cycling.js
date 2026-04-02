/**
 * Feed Cycling Module for markata-go
 *
 * Allows cycling through alternative feeds for the current post using { and }.
 * When a post belongs to multiple feeds (series, tag-based, auto-discovered),
 * this module lets the user switch which feed is shown in the sidebar.
 *
 * Data is embedded in the page as JSON in #feed-sidebar-data.
 * The module re-renders the sidebar title, post list, prev/next links,
 * feed counter, and feed format links when the user cycles feeds.
 *
 * Feed selection persists across [/] navigation via ?feed= URL parameter.
 *
 * Keyboard shortcuts:
 *   { (Shift+[) - Previous feed
 *   } (Shift+]) - Next feed
 */

(function() {
  'use strict';

  var feedState = {
    data: null,        // Parsed sidebarFeedsDataJSON
    currentIndex: 0,   // Current feed index
    initialized: false,
    shortcutsRegistered: false,
    explicitFeedSelection: false
  };

  function currentFeedQuerySlug(feed) {
    var requestedFeed = getFeedParam();
    if (requestedFeed && feed && requestedFeed === feed.slug) {
      return requestedFeed;
    }

    if (feedState.data && feedState.currentIndex !== feedState.data.currentFeedIndex && feed) {
      return feed.slug || '';
    }

    return '';
  }

  function getRotationFeedIndexes() {
    if (!feedState.data || !feedState.data.feeds) return [];

    var slugs = feedState.data.rotationFeedSlugs || [];
    if (!slugs.length) return [];

    var indexes = [];
    for (var i = 0; i < slugs.length; i++) {
      for (var j = 0; j < feedState.data.feeds.length; j++) {
        if (feedState.data.feeds[j].slug === slugs[i]) {
          indexes.push(j);
          break;
        }
      }
    }

    return indexes;
  }

  function getPickerFeedIndexes() {
    var rotationIndexes = getRotationFeedIndexes();
    if (rotationIndexes.length) {
      return rotationIndexes;
    }

    if (!feedState.data || !feedState.data.feeds) return [];

    var indexes = [];
    for (var i = 0; i < feedState.data.feeds.length; i++) {
      if (feedState.data.feeds[i].primary) {
        indexes.push(i);
      }
    }

    return indexes;
  }

  /**
   * Known feed output formats with labels.
   * The engine generates these files for each feed.
   */
  var FEED_FORMATS = [
    { key: 'html', path: '/',          label: 'html' },
    { key: 'rss',  path: '/rss.xml',   label: 'rss' },
    { key: 'atom', path: '/atom.xml',  label: 'atom' },
    { key: 'json', path: '/feed.json', label: 'json' },
    { key: 'txt',  path: '/index.txt', label: 'txt' },
    { key: 'md',   path: '/index.md',  label: 'md' }
  ];

  /**
   * Parse the embedded JSON data from #feed-sidebar-data.
   * Returns null if no data is available.
   */
  function parseData() {
    var el = document.getElementById('feed-sidebar-data');
    if (!el) return null;
    try {
      return JSON.parse(el.textContent);
    } catch (e) {
      return null;
    }
  }

  /**
   * Get the ?feed= parameter from the current URL.
   */
  function getFeedParam() {
    try {
      var params = new URLSearchParams(window.location.search);
      return params.get('feed') || '';
    } catch (e) {
      return '';
    }
  }

  /**
   * Update the URL with ?feed= parameter (without triggering navigation).
   */
  function setFeedParam(slug, forceParam) {
    try {
      var url = new URL(window.location.href);
      var requestedFeed = getFeedParam();
      if (slug && feedState.data && (forceParam || feedState.currentIndex !== feedState.data.currentFeedIndex || requestedFeed === slug)) {
        url.searchParams.set('feed', slug);
      } else {
        // If back to default feed, remove the param for cleaner URLs
        url.searchParams.delete('feed');
      }
      history.replaceState(null, '', url.toString());
    } catch (e) {
      // URL API not supported, skip
    }
  }

  /**
   * Append ?feed=slug to a URL string, preserving existing params.
   */
  function appendFeedParam(href, slug) {
    if (!slug) return href;
    try {
      var url = new URL(href, window.location.origin);
      url.searchParams.set('feed', slug);
      return url.pathname + url.search;
    } catch (e) {
      // Fallback: simple append
      var sep = href.indexOf('?') >= 0 ? '&' : '?';
      return href + sep + 'feed=' + encodeURIComponent(slug);
    }
  }

  /**
   * Update the feed counter display (e.g., "2/5").
   */
  function updateCounter() {
    var counter = document.getElementById('feed-nav-counter');
    if (!counter || !feedState.data) return;

    var rotationIndexes = getRotationFeedIndexes();
    if (!rotationIndexes.length) {
      counter.textContent = (feedState.currentIndex + 1) + '/' + feedState.data.feeds.length;
      return;
    }

    var rotationPosition = rotationIndexes.indexOf(feedState.currentIndex);
    if (rotationPosition === -1) {
      counter.textContent = '-/' + rotationIndexes.length;
      return;
    }

    counter.textContent = (rotationPosition + 1) + '/' + rotationIndexes.length;
  }

  function updateFeedPicker() {
    var select = document.getElementById('feed-nav-select');
    if (!select || !feedState.data) return;

    var pickerIndexes = getPickerFeedIndexes();
    if (!pickerIndexes.length) {
      select.innerHTML = '';
      return;
    }

    var selectedIndex = feedState.currentIndex;
    if (pickerIndexes.indexOf(selectedIndex) === -1) {
      selectedIndex = pickerIndexes[0];
    }

    var currentSlug = feedState.data.feeds[selectedIndex] ? feedState.data.feeds[selectedIndex].slug : '';
    var html = '';
    for (var i = 0; i < pickerIndexes.length; i++) {
      var feed = feedState.data.feeds[pickerIndexes[i]];
      html += '<option value="' + escapeHtml(feed.slug) + '"' + (feed.slug === currentSlug ? ' selected' : '') + '>' + escapeHtml(feed.title) + '</option>';
    }
    select.innerHTML = html;
  }

  /**
   * Build feed format links HTML for a given feed slug.
   */
  function buildFeedLinksHTML(slug) {
    if (!slug) return '';
    var html = '';
    for (var i = 0; i < FEED_FORMATS.length; i++) {
      var fmt = FEED_FORMATS[i];
      if (i > 0) html += ' ';
      html += '<a href="/' + escapeHtml(slug) + escapeHtml(fmt.path) + '" class="feed-nav-format-link" data-feed-format="' + fmt.key + '">' + fmt.label + '</a>';
    }
    return html;
  }

  /**
   * Update the feed format links element.
   */
  function updateFeedLinks(feed) {
    var linksEl = document.getElementById('feed-nav-links');
    if (!linksEl) return;
    if (feed && feed.slug) {
      linksEl.innerHTML = buildFeedLinksHTML(feed.slug);
      linksEl.classList.remove('feed-nav-links--hidden');
    } else {
      linksEl.classList.add('feed-nav-links--hidden');
    }
  }

  /**
   * Re-render the sidebar with the feed at the given index.
   */
  function renderFeed(index, options) {
    if (!feedState.data || !feedState.data.feeds.length) return;

    var renderOptions = options || {};

    var feed = feedState.data.feeds[index];
    if (!feed) return;

    // Update title (include total post count if feed is windowed)
    var titleEl = document.getElementById('feed-nav-title');
    if (titleEl) {
      var titleText = feed.title;
      if (feed.totalPosts && feed.totalPosts > feed.posts.length) {
        titleText += ' (' + feed.totalPosts + ')';
      }
      if (feed.slug) {
        titleEl.innerHTML = '<a href="/' + escapeHtml(feed.slug) + '/">' + escapeHtml(titleText) + '</a>';
      } else {
        titleEl.textContent = titleText;
      }
    }

    // Update post list
    var listEl = document.getElementById('feed-nav-list');
    if (listEl) {
      var html = '';
      var selectedFeedSlug = currentFeedQuerySlug(feed);
      for (var i = 0; i < feed.posts.length; i++) {
        var p = feed.posts[i];
        var isActive = p.active;
        var href = appendFeedParam(p.href, selectedFeedSlug);
        html += '<li class="feed-nav-item' + (isActive ? ' feed-nav-item--active' : '') + '" data-sidebar-transition-path="' + escapeHtml(p.href.replace(/\?.*$/, '')) + '">';
        html += '<a href="' + escapeHtml(href) + '" class="feed-nav-link" data-sidebar-transition-title';
        if (isActive) html += ' aria-current="page"';
        html += '>' + escapeHtml(p.title) + '</a>';
        html += '</li>';
      }
      listEl.innerHTML = html;
    }

    // Update prev/next navigation links in .post-nav
    updatePrevNextLinks(feed);

    // Update hotkey hints for [/] based on new prev/next
    updateHotkeyHints(feed);

    // Update feed format links
    updateFeedLinks(feed);

    // Update counter
    feedState.currentIndex = index;
    updateCounter();
    updateFeedPicker();

    // Update URL with ?feed= param so [/] navigation preserves feed selection
    setFeedParam(feed.slug, !!renderOptions.forceFeedParam);

    if (typeof window.preloadNavigationShortcuts === 'function') {
      window.preloadNavigationShortcuts();
    }

    // Scroll active item into view
    if (window.initFeedSidebarScroll) {
      window.initFeedSidebarScroll();
    }
  }

  /**
   * Update the prev/next links in .post-nav (the data-action elements
   * that [ and ] keys use to navigate).
   * Appends ?feed=slug so the next page knows which feed to restore.
   */
  function updatePrevNextLinks(feed) {
    var postNav = document.querySelector('.post-nav');
    var feedSlug = currentFeedQuerySlug(feed);

    if (!feed.prev && !feed.next) {
      if (postNav) postNav.classList.add('post-nav--hidden');
      return;
    }

    if (!postNav) return;

    postNav.classList.remove('post-nav--hidden');

    // Update or create prev link
    var prevLink = postNav.querySelector('[data-action="prev"]');
    if (feed.prev) {
      var prevHref = appendFeedParam(feed.prev.href, feedSlug);
      if (prevLink) {
        prevLink.href = prevHref;
        prevLink.textContent = feed.prev.title;
      } else {
        prevLink = document.createElement('a');
        prevLink.href = prevHref;
        prevLink.className = 'prev';
        prevLink.setAttribute('data-action', 'prev');
        prevLink.textContent = feed.prev.title;
        postNav.insertBefore(prevLink, postNav.firstChild);
      }
    } else if (prevLink) {
      prevLink.remove();
    }

    // Update or create next link
    var nextLink = postNav.querySelector('[data-action="next"]');
    if (feed.next) {
      var nextHref = appendFeedParam(feed.next.href, feedSlug);
      if (nextLink) {
        nextLink.href = nextHref;
        nextLink.textContent = feed.next.title;
      } else {
        nextLink = document.createElement('a');
        nextLink.href = nextHref;
        nextLink.className = 'next';
        nextLink.setAttribute('data-action', 'next');
        nextLink.textContent = feed.next.title;
        postNav.appendChild(nextLink);
      }
    } else if (nextLink) {
      nextLink.remove();
    }
  }

  /**
   * Update [/] hotkey hint visibility based on new feed's prev/next.
   */
  function updateHotkeyHints(feed) {
    var group = document.querySelector('.feed-nav-hotkey-group--nav');
    if (!group) return;

    var html = '';
    if (feed.prev) {
      html += '<span class="feed-nav-hotkey"><kbd>[</kbd> prev</span>';
    }
    if (feed.next) {
      html += '<span class="feed-nav-hotkey"><kbd>]</kbd> next</span>';
    }
    group.innerHTML = html;
  }

  /**
   * Cycle to the next feed (wrapping around).
   */
  function nextFeed() {
    var rotationIndexes = getRotationFeedIndexes();
    if (rotationIndexes.length <= 1) return;

    var currentRotation = rotationIndexes.indexOf(feedState.currentIndex);
    if (currentRotation === -1) currentRotation = 0;

    var nextRotation = (currentRotation + 1) % rotationIndexes.length;
    feedState.explicitFeedSelection = true;
    renderFeed(rotationIndexes[nextRotation], { forceFeedParam: true });
  }

  /**
   * Cycle to the previous feed (wrapping around).
   */
  function prevFeed() {
    var rotationIndexes = getRotationFeedIndexes();
    if (rotationIndexes.length <= 1) return;

    var currentRotation = rotationIndexes.indexOf(feedState.currentIndex);
    if (currentRotation === -1) currentRotation = 0;

    var prevRotation = (currentRotation - 1 + rotationIndexes.length) % rotationIndexes.length;
    feedState.explicitFeedSelection = true;
    renderFeed(rotationIndexes[prevRotation], { forceFeedParam: true });
  }

  /**
   * Escape HTML special characters to prevent XSS.
   */
  function escapeHtml(str) {
    if (!str) return '';
    var div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }

  /**
   * Register { and } shortcuts with the shortcuts registry.
   */
  function registerShortcuts() {
    if (!window.shortcutsRegistry || feedState.shortcutsRegistered) return;

    // Only register if feed data exists and has multiple feeds
    if (!feedState.data) return;
    if (!feedState.data.rotationFeedSlugs || feedState.data.rotationFeedSlugs.length <= 1) return;

    window.shortcutsRegistry.register({
      key: '{',
      modifiers: [],
      description: 'Previous feed',
      group: 'navigation',
      handler: function(e) {
        e.preventDefault();
        prevFeed();
      },
      priority: 56  // Slightly above [/] (55)
    });

    window.shortcutsRegistry.register({
      key: '}',
      modifiers: [],
      description: 'Next feed',
      group: 'navigation',
      handler: function(e) {
        e.preventDefault();
        nextFeed();
      },
      priority: 56
    });

    feedState.shortcutsRegistered = true;
  }

  /**
   * Initialize feed cycling. Called on page load and after view transitions.
   */
  function init() {
    feedState.data = parseData();
    feedState.shortcutsRegistered = false;
    if (!feedState.data || feedState.data.feeds.length <= 1) {
      feedState.initialized = false;
      return;
    }

    // Default to the server-rendered feed
    feedState.currentIndex = feedState.data.currentFeedIndex;

    // Check URL for ?feed= param (set by prev/next navigation)
    var requestedFeed = getFeedParam();
    if (requestedFeed) {
      for (var i = 0; i < feedState.data.feeds.length; i++) {
        if (feedState.data.feeds[i].slug === requestedFeed) {
          feedState.currentIndex = i;
          break;
        }
      }
    }

    feedState.initialized = true;
    feedState.explicitFeedSelection = !!requestedFeed;

    // Always render the selected feed so sidebar title, picker, and controls
    // match the current feed even when the server rendered fallback chrome.
    renderFeed(feedState.currentIndex, { forceFeedParam: feedState.explicitFeedSelection });

    // Register keyboard shortcuts
    registerShortcuts();

    // Wire up clickable prev/next feed buttons
    var prevBtn = document.getElementById('feed-nav-prev-feed');
    var nextBtn = document.getElementById('feed-nav-next-feed');
    if (prevBtn) {
      prevBtn.addEventListener('click', function(e) {
        e.preventDefault();
        prevFeed();
      });
    }
    if (nextBtn) {
      nextBtn.addEventListener('click', function(e) {
        e.preventDefault();
        nextFeed();
      });
    }

    var select = document.getElementById('feed-nav-select');
    if (select) {
      select.addEventListener('change', function() {
        var slug = select.value;
        var pickerIndexes = getPickerFeedIndexes();
        for (var i = 0; i < pickerIndexes.length; i++) {
          var feedIndex = pickerIndexes[i];
          if (feedState.data.feeds[feedIndex].slug === slug) {
            feedState.explicitFeedSelection = true;
            renderFeed(feedIndex, { forceFeedParam: true });
            select.blur();
            var header = document.querySelector('.feed-nav-header');
            if (header && typeof header.focus === 'function') {
              header.focus({ preventScroll: true });
            }
            break;
          }
        }
      });
    }
  }

  // Expose for view-transitions reinitializeScripts()
  window.initFeedCycling = init;

  function waitForShortcutsRegistry(attempts) {
    attempts = attempts || 0;
    if (window.shortcutsRegistry) {
      registerShortcuts();
    } else if (attempts < 50) {
      setTimeout(function() { waitForShortcutsRegistry(attempts + 1); }, 10);
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function() {
      init();
      waitForShortcutsRegistry();
    });
  } else {
    init();
    waitForShortcutsRegistry();
  }
})();
