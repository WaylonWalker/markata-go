/**
 * Navigation Shortcuts Module for markata-go
 *
 * Registers navigation-related keyboard shortcuts with the shortcuts registry.
 * - `j` or `↓` - Next post (in feeds) / Highlight next card
 * - `k` or `↑` - Previous post (in feeds) / Highlight previous card
 * - `Enter` or `o` - Open highlighted post
 * - `Shift+O` - Open in new tab
 * - `g h` - Go to home
 * - `g s` - Focus search
 * - `s` - Toggle simple/rich feed view (on feed pages only)
 * - `[` - Previous page
 * - `]` - Next page
 * - `y y` - Copy URL to clipboard
 *
 * When feed cards are present, j/k will navigate between cards (with visual highlight).
 * Press o/Enter to open the selected card.
 */

(function() {
  'use strict';

  // Wait for registry to be available
  function waitForRegistry(callback, attempts) {
    attempts = attempts || 0;
    if (window.shortcutsRegistry) {
      callback();
    } else if (attempts < 50) {
      setTimeout(function() {
        waitForRegistry(callback, attempts + 1);
      }, 10);
    }
  }

  var state = {
    selectedCard: null,
    cards: [],
    lastKeyTime: 0,
    lastKey: null,
    keySequenceTimeout: 500, // ms
    jKeyDown: false,
    kKeyDown: false,
    navRepeatTimer: null,
    navRepeatDelay: 80, // ms between navigations when key held (faster)
    initialized: false,  // Track if already initialized
    jkRegistered: false  // Track if j/k shortcuts are registered
  };

  /**
   * Get all post cards on the page
   */
  function getCards() {
    return Array.from(document.querySelectorAll('.card, [data-card]'));
  }

  /**
   * Highlight a card
   */
  function highlightCard(card) {
    // Remove previous highlight
    if (state.selectedCard) {
      state.selectedCard.classList.remove('kb-highlighted');
    }

    // Highlight new card
    state.selectedCard = card;
    card.classList.add('kb-highlighted');
    card.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
  }

  /**
   * Navigate to next post
   */
  function nextPost() {
    state.cards = getCards();
    if (state.cards.length === 0) return;

    var currentIndex = state.cards.indexOf(state.selectedCard);
    var nextIndex = currentIndex + 1;

    if (nextIndex >= state.cards.length) {
      // Try to navigate to next page
      var nextBtn = document.querySelector('[data-action="next"]');
      if (nextBtn && !nextBtn.disabled) {
        nextBtn.click();
      }
      return;
    }

    highlightCard(state.cards[nextIndex]);
  }

  /**
   * Navigate to previous post
   */
  function previousPost() {
    state.cards = getCards();
    if (state.cards.length === 0) return;

    var currentIndex = state.selectedCard ? state.cards.indexOf(state.selectedCard) : -1;
    var prevIndex = currentIndex - 1;

    if (prevIndex < 0) {
      // Try to navigate to previous page
      var prevBtn = document.querySelector('[data-action="prev"]');
      if (prevBtn && !prevBtn.disabled) {
        prevBtn.click();
      }
      return;
    }

    highlightCard(state.cards[prevIndex]);
  }

  /**
   * Open highlighted post
   */
  function openPost(newTab) {
    if (!state.selectedCard) {
      // Highlight first card if none selected
      state.cards = getCards();
      if (state.cards.length > 0) {
        highlightCard(state.cards[0]);
        return;
      }
      return;
    }

    var link = state.selectedCard.querySelector('a');
    if (!link) return;

    if (newTab) {
      window.open(link.href, '_blank');
    } else {
      window.location.href = link.href;
    }
  }

  /**
   * Go to home page
   */
  function goHome() {
    var homeLink = document.querySelector('[data-nav="home"], a[href="/"], nav a:first-child');
    if (homeLink) {
      window.location.href = homeLink.href;
    } else {
      window.location.href = '/';
    }
  }

  /**
   * Focus search
   */
  function focusSearch() {
    var pagefindInput = document.querySelector('.pagefind-ui__search-input');
    if (pagefindInput) {
      pagefindInput.focus();
      return;
    }

    var searchInput = document.querySelector('#pagefind-search input, #search input, [type="search"]');
    if (searchInput) {
      searchInput.focus();
    }
  }

  /**
   * Toggle between simple and rich feed views.
   * On simple feed pages (URL contains /simple/), navigates to the rich feed.
   * On rich feed pages, navigates to the simple feed variant.
   * Only works on feed pages (elements with .feed class).
   */
  function toggleFeedView() {
    var feedEl = document.querySelector('.feed');
    if (!feedEl) return;

    var path = window.location.pathname;
    var isSimple = feedEl.classList.contains('feed-simple');

    if (isSimple) {
      // On simple feed -> go to rich feed: remove /simple/ from path
      var richPath = path.replace(/\/simple\/(page\/\d+\/)?$/, '/');
      // Preserve page number if present
      var pageMatch = path.match(/\/simple\/page\/(\d+)\//);
      if (pageMatch) {
        richPath = path.replace(/\/simple\/page\/(\d+)\//, '/page/' + pageMatch[1] + '/');
      }
      window.location.href = richPath;
    } else {
      // On rich feed -> go to simple feed: insert /simple/ before any /page/ or at end
      var simplePath;
      var richPageMatch = path.match(/\/page\/(\d+)\/$/);
      if (richPageMatch) {
        simplePath = path.replace(/\/page\/(\d+)\/$/, '/simple/page/' + richPageMatch[1] + '/');
      } else {
        // Ensure trailing slash
        simplePath = path.replace(/\/?$/, '/') + 'simple/';
      }
      window.location.href = simplePath;
    }
  }

  /**
   * Navigate to next page.
   * Checks for JS pagination buttons (data-action="next") and also
   * manual/htmx pagination links (.pagination-next).
   */
  function nextPage() {
    // Try JS pagination button first
    var nextBtn = document.querySelector('[data-action="next"]:not(:disabled)');
    if (nextBtn) {
      nextBtn.click();
      return;
    }
    // Try manual/htmx pagination link
    var nextLink = document.querySelector('a.pagination-next');
    if (nextLink) {
      nextLink.click();
    }
  }

  /**
   * Navigate to previous page.
   * Checks for JS pagination buttons (data-action="prev") and also
   * manual/htmx pagination links (.pagination-prev).
   */
  function previousPage() {
    // Try JS pagination button first
    var prevBtn = document.querySelector('[data-action="prev"]:not(:disabled)');
    if (prevBtn) {
      prevBtn.click();
      return;
    }
    // Try manual/htmx pagination link
    var prevLink = document.querySelector('a.pagination-prev');
    if (prevLink) {
      prevLink.click();
    }
  }

  /**
   * Copy current page URL to clipboard
   */
  function copyUrl() {
    var url = window.location.href;
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(url).then(function() {
        showNotification('URL copied to clipboard');
      }).catch(function(err) {
        console.error('Failed to copy URL:', err);
      });
    } else {
      // Fallback for older browsers
      var textarea = document.createElement('textarea');
      textarea.value = url;
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand('copy');
      document.body.removeChild(textarea);
      showNotification('URL copied to clipboard');
    }
  }

  /**
   * Show a brief notification
   */
  function showNotification(text) {
    var notif = document.querySelector('.shortcuts-notification');
    if (!notif) {
      notif = document.createElement('div');
      notif.className = 'shortcuts-notification';
      notif.style.cssText = 'position: fixed; bottom: 20px; right: 20px; ' +
        'background: var(--color-bg); color: var(--color-text); ' +
        'padding: 12px 16px; border-radius: 4px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); ' +
        'font-size: 14px; z-index: 10000; opacity: 0; transition: opacity 0.3s;';
      document.body.appendChild(notif);
    }

    notif.textContent = text;
    notif.style.opacity = '1';

    clearTimeout(notif._timeout);
    notif._timeout = setTimeout(function() {
      notif.style.opacity = '0';
    }, 2000);
  }

  /**
   * Clean up state for re-initialization (for view transitions)
   */
  function cleanup() {
    // Clear any timers
    if (state.navRepeatTimer) {
      clearTimeout(state.navRepeatTimer);
      state.navRepeatTimer = null;
    }

    // Reset key states
    state.jKeyDown = false;
    state.kKeyDown = false;

    // Remove highlight from old card (if it still exists in DOM)
    if (state.selectedCard && state.selectedCard.parentNode) {
      state.selectedCard.classList.remove('kb-highlighted');
    }

    // Clear card references
    state.selectedCard = null;
    state.cards = [];
  }

  /**
   * Initialize navigation shortcuts
   */
  function init() {
    // Clean up previous state
    cleanup();

    // Initialize cards
    state.cards = getCards();
    if (state.cards.length > 0) {
      highlightCard(state.cards[0]);
    }

    // Only register these shortcuts once (they use document-level listeners internally)
    if (!state.initialized) {
      // Enter - Open highlighted post
      window.shortcutsRegistry.register({
        key: 'Enter',
        modifiers: [],
        description: 'Open highlighted post',
        group: 'navigation',
        handler: function(e) {
          e.preventDefault();
          openPost(false);
        },
        priority: 15
      });

      // o - Open highlighted post
      window.shortcutsRegistry.register({
        key: 'o',
        modifiers: [],
        description: 'Open highlighted card',
        group: 'navigation',
        handler: function(e) {
          e.preventDefault();
          openPost(false);
        },
        priority: 15
      });

      // Shift+O - Open in new tab
      window.shortcutsRegistry.register({
        key: 'O',
        modifiers: [],
        description: 'Open highlighted card in new tab',
        group: 'navigation',
        handler: function(e) {
          e.preventDefault();
          openPost(true);
        },
        priority: 15
      });

      // [ and ] - Previous/Next page
      // Only register when pagination elements exist on the page.
      // Priority 55: higher than palette-switcher (50) so pagination wins.
      // Without this guard, palette cycling ([/]) would be shadowed on all
      // feed pages even when no pagination is present.
      if (document.querySelector('[data-action="prev"], [data-action="next"], a.pagination-prev, a.pagination-next')) {
        window.shortcutsRegistry.register({
          key: '[',
          modifiers: [],
          description: 'Previous page',
          group: 'navigation',
          handler: function(e) {
            e.preventDefault();
            previousPage();
          },
          priority: 55
        });

        window.shortcutsRegistry.register({
          key: ']',
          modifiers: [],
          description: 'Next page',
          group: 'navigation',
          handler: function(e) {
            e.preventDefault();
            nextPage();
          },
          priority: 55
        });
      }

      // Handle multi-key sequences: g h, g s, and standalone s for feed toggle
      document.addEventListener('keydown', function(e) {
        if (window.shortcutsRegistry.areDisabled()) return;
        if (window.shortcutsRegistry.isInputElement(e.target)) return;

        var now = Date.now();
        var timeSinceLastKey = now - state.lastKeyTime;

        if (e.key === 'g') {
          state.lastKey = 'g';
          state.lastKeyTime = now;
        } else if (e.key === 'h' && state.lastKey === 'g' && timeSinceLastKey < state.keySequenceTimeout) {
          // g h - go to home
          e.preventDefault();
          goHome();
          state.lastKey = null;
          state.lastKeyTime = 0;
        } else if (e.key === 's' && state.lastKey === 'g' && timeSinceLastKey < state.keySequenceTimeout) {
          // g s - focus search
          e.preventDefault();
          focusSearch();
          state.lastKey = null;
          state.lastKeyTime = 0;
        } else if (e.key === 's' && state.lastKey !== 'g') {
          // Standalone s - toggle simple/rich feed view (only on feed pages)
          if (document.querySelector('.feed')) {
            e.preventDefault();
            toggleFeedView();
          }
          state.lastKey = null;
          state.lastKeyTime = 0;
        } else {
          state.lastKey = null;
          state.lastKeyTime = 0;
        }
      });

      // y y - Copy URL (special handling for repeated key)
      var yKeyTime = 0;
      document.addEventListener('keydown', function(e) {
        if (window.shortcutsRegistry.areDisabled()) return;
        if (window.shortcutsRegistry.isInputElement(e.target)) return;

        if (e.key === 'y') {
          var now = Date.now();
          var timeSinceLastY = now - yKeyTime;

          if (timeSinceLastY < state.keySequenceTimeout) {
            // y y - copy URL
            e.preventDefault();
            copyUrl();
            yKeyTime = 0;
          } else {
            yKeyTime = now;
          }
        } else {
          yKeyTime = 0;
        }
      });

      state.initialized = true;
    }

    // Register j/k shortcuts if we have cards (these use state.cards which gets refreshed)
    // Only register once - the handlers will use the refreshed state.cards
    if (state.cards.length > 0 && !state.jkRegistered) {
      // j - Next card in feed
      window.shortcutsRegistry.register({
        key: 'j',
        modifiers: [],
        description: 'Select next card in feed',
        group: 'navigation',
        handler: function(e) {
          // Refresh cards on each call to handle view transitions
          state.cards = getCards();
          if (state.cards.length === 0) return;

          // Initialize selection if needed
          if (!state.selectedCard || !state.selectedCard.parentNode) {
            highlightCard(state.cards[0]);
          } else {
            e.preventDefault();
            nextPost();
          }
        },
        priority: 20
      });

      // k - Previous card in feed
      window.shortcutsRegistry.register({
        key: 'k',
        modifiers: [],
        description: 'Select previous card in feed',
        group: 'navigation',
        handler: function(e) {
          // Refresh cards on each call to handle view transitions
          state.cards = getCards();
          if (state.cards.length === 0) return;

          // Initialize selection if needed
          if (!state.selectedCard || !state.selectedCard.parentNode) {
            highlightCard(state.cards[0]);
          } else {
            e.preventDefault();
            previousPost();
          }
        },
        priority: 20
      });

      // Handle held j/k for continuous navigation
      document.addEventListener('keydown', function(e) {
        if (window.shortcutsRegistry.areDisabled()) return;
        if (window.shortcutsRegistry.isInputElement(e.target)) return;

        // Refresh cards to handle view transitions
        state.cards = getCards();
        if (state.cards.length === 0) return;

        if (e.key === 'j' && !state.jKeyDown) {
          state.jKeyDown = true;
          // Start repeat timer after initial delay
          state.navRepeatTimer = setTimeout(function repeatJ() {
            if (state.jKeyDown) {
              nextPost();
              state.navRepeatTimer = setTimeout(repeatJ, state.navRepeatDelay);
            }
          }, state.navRepeatDelay);
          e.preventDefault();
        } else if (e.key === 'k' && !state.kKeyDown) {
          state.kKeyDown = true;
          // Start repeat timer after initial delay
          state.navRepeatTimer = setTimeout(function repeatK() {
            if (state.kKeyDown) {
              previousPost();
              state.navRepeatTimer = setTimeout(repeatK, state.navRepeatDelay);
            }
          }, state.navRepeatDelay);
          e.preventDefault();
        }
      });

      // Clear repeat timer when keys are released
      document.addEventListener('keyup', function(e) {
        if (e.key === 'j') {
          state.jKeyDown = false;
          if (state.navRepeatTimer && !state.kKeyDown) {
            clearTimeout(state.navRepeatTimer);
            state.navRepeatTimer = null;
          }
        } else if (e.key === 'k') {
          state.kKeyDown = false;
          if (state.navRepeatTimer && !state.jKeyDown) {
            clearTimeout(state.navRepeatTimer);
            state.navRepeatTimer = null;
          }
        }
      });

      state.jkRegistered = true;
    }
  }

  // Initialize when registry is available
  waitForRegistry(function() {
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', init);
    } else {
      init();
    }
  });

  // Expose init for view transitions
  window.initNavigationShortcuts = init;

  // Re-initialize after view transitions
  window.addEventListener('view-transition-complete', init);
})();
