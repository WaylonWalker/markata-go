/**
 * Tag Hover Cards for markata-go
 *
 * Displays contextual information (post count, reading time) when hovering over #tag hashtag links.
 *
 * Features:
 * - Smart positioning (above/below based on viewport)
 * - Data from data-* attributes (instant, no network requests)
 * - 300ms show delay, 200ms hide delay to prevent flickering
 * - Keyboard support (Escape to dismiss, focus shows card)
 * - Disabled on touch devices
 * - Graceful error handling
 *
 * @see https://github.com/WaylonWalker/markata-go/issues/848
 */

(function() {
  'use strict';

  // Configuration
  var SHOW_DELAY = 300;   // ms before showing card
  var HIDE_DELAY = 200;   // ms before hiding card

  // State
  var currentCard = null;
  var showTimer = null;
  var hideTimer = null;
  var currentTarget = null;
  var boundTags = null;  // WeakSet to track bound elements
  var cardDelegationSetup = false;  // Track if card delegation is set up

  /**
   * Check if device supports touch (likely mobile)
   * @returns {boolean}
   */
  function isTouchDevice() {
    return ('ontouchstart' in window) ||
           (navigator.maxTouchPoints > 0) ||
           (navigator.msMaxTouchPoints > 0);
  }

  /**
   * Get tag data from data attributes
   * @param {HTMLElement} link - The tag link element
   * @returns {object}
   */
  function getTagData(link) {
    return {
      tag: link.dataset.tag || '',
      count: parseInt(link.dataset.count, 10) || 0,
      readingTime: parseInt(link.dataset.readingTime, 10) || 0,
      readingTimeText: link.dataset.readingTimeText || 'unknown',
      url: link.href
    };
  }

  /**
   * Create the hover card element
   * @param {object} data - Tag data
   * @returns {HTMLElement}
   */
  function createCard(data) {
    var card = document.createElement('div');
    card.className = 'tag-card';
    card.setAttribute('role', 'tooltip');
    card.setAttribute('aria-live', 'polite');

    var tagHtml = '<div class="tag-card-tag">#' + escapeHtml(data.tag) + '</div>';

    var statsHtml = '<div class="tag-card-stats">';
    statsHtml += '<div class="tag-card-stat">';
    statsHtml += '<span class="tag-card-stat-label">Posts:</span> ';
    statsHtml += '<span class="tag-card-stat-value">' + escapeHtml(String(data.count)) + '</span>';
    statsHtml += '</div>';
    statsHtml += '<div class="tag-card-stat">';
    statsHtml += '<span class="tag-card-stat-label">Reading time:</span> ';
    statsHtml += '<span class="tag-card-stat-value">' + escapeHtml(data.readingTimeText) + '</span>';
    statsHtml += '</div>';
    statsHtml += '</div>';

    var linkHtml = '<a href="' + escapeHtml(data.url) + '" class="tag-card-link">' +
                   '<span aria-hidden="true">→</span> View tag</a>';

    card.innerHTML =
      tagHtml +
      statsHtml +
      '<div class="tag-card-footer">' + linkHtml + '</div>';

    return card;
  }

  /**
   * Escape HTML to prevent XSS
   * @param {string} str
   * @returns {string}
   */
  function escapeHtml(str) {
    if (!str) return '';
    var div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }

  /**
   * Position the card relative to the target element
   * @param {HTMLElement} card
   * @param {HTMLElement} target
   */
  function positionCard(card, target) {
    var rect = target.getBoundingClientRect();
    var cardRect = card.getBoundingClientRect();
    var viewportHeight = window.innerHeight;
    var viewportWidth = window.innerWidth;
    var scrollY = window.scrollY || window.pageYOffset;
    var scrollX = window.scrollX || window.pageXOffset;

    // Default: position below
    var top = rect.bottom + scrollY + 8;
    var left = rect.left + scrollX;

    // Check if card would go below viewport
    if (rect.bottom + cardRect.height + 16 > viewportHeight) {
      // Position above instead
      top = rect.top + scrollY - cardRect.height - 8;
      card.classList.add('tag-card--above');
      card.classList.remove('tag-card--below');
    } else {
      card.classList.add('tag-card--below');
      card.classList.remove('tag-card--above');
    }

    // Check if card would go past right edge
    if (left + cardRect.width > viewportWidth + scrollX - 16) {
      left = viewportWidth + scrollX - cardRect.width - 16;
    }

    // Check if card would go past left edge
    if (left < scrollX + 16) {
      left = scrollX + 16;
    }

    card.style.top = top + 'px';
    card.style.left = left + 'px';
  }

  /**
   * Show the hover card for a tag link
   * @param {HTMLElement} target - The tag link element
   */
  function showCard(target) {
    // Don't show if another card is visible for a different target
    if (currentCard && currentTarget !== target) {
      hideCard();
    }

    currentTarget = target;

    // Get data instantly from data attributes
    var data = getTagData(target);

    var card = createCard(data);
    document.body.appendChild(card);

    // Position and show
    requestAnimationFrame(function() {
      positionCard(card, target);
      card.classList.add('tag-card--visible');
      currentCard = card;
    });
  }

  /**
   * Hide the current hover card
   */
  function hideCard() {
    if (currentCard) {
      currentCard.classList.remove('tag-card--visible');

      // Remove after transition
      var cardToRemove = currentCard;
      setTimeout(function() {
        if (cardToRemove.parentNode) {
          cardToRemove.remove();
        }
      }, 150);

      currentCard = null;
      currentTarget = null;
    }
  }

  /**
   * Clear all timers
   */
  function clearTimers() {
    if (showTimer) {
      clearTimeout(showTimer);
      showTimer = null;
    }
    if (hideTimer) {
      clearTimeout(hideTimer);
      hideTimer = null;
    }
  }

  /**
   * Handle mouse enter on tag link
   * @param {Event} e
   */
  function handleMouseEnter(e) {
    var target = e.currentTarget;

    clearTimers();

    showTimer = setTimeout(function() {
      showCard(target);
    }, SHOW_DELAY);
  }

  /**
   * Handle mouse leave on tag link
   * @param {Event} e
   */
  function handleMouseLeave(e) {
    clearTimers();

    hideTimer = setTimeout(function() {
      hideCard();
    }, HIDE_DELAY);
  }

  /**
   * Handle mouse enter on the card itself (to keep it visible)
   * @param {Event} e
   */
  function handleCardMouseEnter(e) {
    clearTimers();
  }

  /**
   * Handle mouse leave on the card itself
   * @param {Event} e
   */
  function handleCardMouseLeave(e) {
    clearTimers();

    hideTimer = setTimeout(function() {
      hideCard();
    }, HIDE_DELAY);
  }

  /**
   * Handle focus on tag link (keyboard navigation)
   * @param {Event} e
   */
  function handleFocus(e) {
    var target = e.currentTarget;

    clearTimers();

    showTimer = setTimeout(function() {
      showCard(target);
    }, SHOW_DELAY);
  }

  /**
   * Handle blur on tag link
   * @param {Event} e
   */
  function handleBlur(e) {
    clearTimers();

    hideTimer = setTimeout(function() {
      hideCard();
    }, HIDE_DELAY);
  }

  /**
   * Handle Escape key to dismiss card
   * @param {KeyboardEvent} e
   */
  function handleKeyDown(e) {
    if (e.key === 'Escape' && currentCard) {
      clearTimers();
      hideCard();
    }
  }

  /**
   * Setup event delegation for dynamically added cards
   */
  function setupCardEventDelegation() {
    // Only set up once (these are document-level listeners that survive view transitions)
    if (cardDelegationSetup) return;

    document.addEventListener('mouseenter', function(e) {
      if (e.target.closest && e.target.closest('.tag-card')) {
        handleCardMouseEnter(e);
      }
    }, true);

    document.addEventListener('mouseleave', function(e) {
      if (e.target.closest && e.target.closest('.tag-card')) {
        handleCardMouseLeave(e);
      }
    }, true);

    cardDelegationSetup = true;
  }

  /**
   * Clean up before re-initialization (for view transitions)
   */
  function cleanup() {
    // Clear any pending timers
    clearTimers();

    // Hide and remove any visible card
    if (currentCard && currentCard.parentNode) {
      currentCard.remove();
    }
    currentCard = null;
    currentTarget = null;

    // Remove any orphaned tag cards from the DOM
    document.querySelectorAll('.tag-card').forEach(function(el) {
      el.remove();
    });

    // Reset the WeakSet - old DOM elements are gone after view transition
    boundTags = new WeakSet();
  }

  /**
   * Initialize tag cards
   */
  function init() {
    // Skip on touch devices
    if (isTouchDevice()) {
      return;
    }

    // Clean up first to handle view transitions properly
    cleanup();

    // Find all hashtag tag links
    var tags = document.querySelectorAll('a.hashtag-tag');

    tags.forEach(function(tag) {
      // Skip if already bound (shouldn't happen after cleanup, but defensive)
      if (boundTags && boundTags.has(tag)) return;

      tag.addEventListener('mouseenter', handleMouseEnter);
      tag.addEventListener('mouseleave', handleMouseLeave);
      tag.addEventListener('focus', handleFocus);
      tag.addEventListener('blur', handleBlur);
      if (!boundTags) boundTags = new WeakSet();
      boundTags.add(tag);
    });

    // Global keyboard handler for Escape (only add once)
    if (!init.keydownBound) {
      document.addEventListener('keydown', handleKeyDown);
      init.keydownBound = true;
    }

    // Setup card event delegation (only once)
    setupCardEventDelegation();
  }

  // Initialize when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  // Expose for external use if needed
  window.tagCards = {
    show: showCard,
    hide: hideCard,
    init: init
  };

  // Expose init for view transitions
  window.initTagCards = init;

  // Re-initialize after view transitions
  window.addEventListener('view-transition-complete', init);
})();
