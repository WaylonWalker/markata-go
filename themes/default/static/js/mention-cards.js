/**
 * Mention Hover Cards for markata-go
 *
 * Displays contextual information (avatar, name, bio) when hovering over @mention links.
 *
 * Features:
 * - Smart positioning (above/below based on viewport)
 * - Data fetching from blogroll cache first, then meta tags from URL
 * - 5-minute cache TTL for performance
 * - 300ms show delay, 200ms hide delay to prevent flickering
 * - Keyboard support (Escape to dismiss, focus shows card)
 * - Disabled on touch devices
 * - Graceful error handling
 *
 * @see https://github.com/example/markata-go/issues/297
 */

(function() {
  'use strict';

  // Configuration
  var SHOW_DELAY = 300;   // ms before showing card
  var HIDE_DELAY = 200;   // ms before hiding card
  var CACHE_TTL = 5 * 60 * 1000;  // 5 minutes in ms

  // State
  var cache = {};
  var currentCard = null;
  var showTimer = null;
  var hideTimer = null;
  var currentTarget = null;

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
   * Get initials from a name or handle
   * @param {string} name - The name or handle
   * @returns {string} - Up to 2 initials
   */
  function getInitials(name) {
    if (!name) return '?';

    // Remove @ prefix if present
    name = name.replace(/^@/, '');

    // Split by common separators
    var parts = name.split(/[\s._-]+/);

    if (parts.length >= 2) {
      return (parts[0][0] + parts[1][0]).toUpperCase();
    }

    return name.substring(0, 2).toUpperCase();
  }

  /**
   * Extract domain from URL for display
   * @param {string} url
   * @returns {string}
   */
  function getDomain(url) {
    try {
      var domain = new URL(url).hostname;
      return domain.replace(/^www\./, '');
    } catch (e) {
      return url;
    }
  }

  /**
   * Look up mention data in blogroll cache
   * @param {string} url - The mention URL
   * @returns {object|null} - Blogroll entry data or null
   */
  function lookupBlogroll(url) {
    if (!window.blogrollData || !Array.isArray(window.blogrollData)) {
      return null;
    }

    var targetDomain = getDomain(url).toLowerCase();

    for (var i = 0; i < window.blogrollData.length; i++) {
      var entry = window.blogrollData[i];
      if (entry.url) {
        var entryDomain = getDomain(entry.url).toLowerCase();
        if (entryDomain === targetDomain) {
          return {
            name: entry.name || entry.title,
            handle: '@' + getDomain(entry.url),
            bio: entry.description || '',
            avatar: entry.avatar || entry.icon || null,
            url: entry.url
          };
        }
      }
    }

    return null;
  }

  /**
   * Fetch meta tags from a URL
   * @param {string} url
   * @returns {Promise<object>}
   */
  function fetchMetaTags(url) {
    return new Promise(function(resolve, reject) {
      // Use a CORS proxy or direct fetch
      // Note: This may fail due to CORS restrictions on many sites
      fetch(url, {
        method: 'GET',
        mode: 'cors',
        credentials: 'omit'
      })
      .then(function(response) {
        if (!response.ok) {
          throw new Error('HTTP ' + response.status);
        }
        return response.text();
      })
      .then(function(html) {
        var parser = new DOMParser();
        var doc = parser.parseFromString(html, 'text/html');

        var data = {
          name: null,
          bio: null,
          avatar: null
        };

        // Try various meta tags for name/title
        var titleMeta = doc.querySelector('meta[property="og:site_name"]') ||
                       doc.querySelector('meta[property="og:title"]') ||
                       doc.querySelector('meta[name="author"]') ||
                       doc.querySelector('title');

        if (titleMeta) {
          data.name = titleMeta.content || titleMeta.textContent;
        }

        // Try various meta tags for description
        var descMeta = doc.querySelector('meta[property="og:description"]') ||
                      doc.querySelector('meta[name="description"]');

        if (descMeta) {
          data.bio = descMeta.content;
        }

        // Try various meta tags for image/avatar
        var imageMeta = doc.querySelector('meta[property="og:image"]') ||
                       doc.querySelector('link[rel="icon"]') ||
                       doc.querySelector('link[rel="apple-touch-icon"]');

        if (imageMeta) {
          var imageUrl = imageMeta.content || imageMeta.href;
          if (imageUrl) {
            // Make relative URLs absolute
            if (imageUrl.startsWith('/')) {
              try {
                var baseUrl = new URL(url);
                imageUrl = baseUrl.origin + imageUrl;
              } catch (e) {
                // Keep as-is
              }
            }
            data.avatar = imageUrl;
          }
        }

        resolve(data);
      })
      .catch(function(err) {
        reject(err);
      });
    });
  }

  /**
   * Get mention data (from cache, blogroll, or fetch)
   * @param {string} url - The mention URL
   * @param {string} handle - The @handle text
   * @returns {Promise<object>}
   */
  function getMentionData(url, handle) {
    // Check cache first
    var cacheKey = url;
    var cached = cache[cacheKey];

    if (cached && (Date.now() - cached.timestamp < CACHE_TTL)) {
      return Promise.resolve(cached.data);
    }

    // Check blogroll
    var blogrollData = lookupBlogroll(url);
    if (blogrollData) {
      cache[cacheKey] = {
        data: blogrollData,
        timestamp: Date.now()
      };
      return Promise.resolve(blogrollData);
    }

    // Fetch from URL
    return fetchMetaTags(url)
      .then(function(metaData) {
        var data = {
          name: metaData.name || getDomain(url),
          handle: handle || '@' + getDomain(url),
          bio: metaData.bio || '',
          avatar: metaData.avatar || null,
          url: url
        };

        cache[cacheKey] = {
          data: data,
          timestamp: Date.now()
        };

        return data;
      })
      .catch(function() {
        // Return fallback data on error
        var data = {
          name: getDomain(url),
          handle: handle || '@' + getDomain(url),
          bio: '',
          avatar: null,
          url: url,
          error: true
        };

        // Still cache even on error to prevent repeated failed requests
        cache[cacheKey] = {
          data: data,
          timestamp: Date.now()
        };

        return data;
      });
  }

  /**
   * Create the hover card element
   * @param {object} data - Mention data
   * @returns {HTMLElement}
   */
  function createCard(data) {
    var card = document.createElement('div');
    card.className = 'mention-card';
    card.setAttribute('role', 'tooltip');
    card.setAttribute('aria-live', 'polite');

    var avatarHtml;
    if (data.avatar) {
      avatarHtml = '<div class="mention-card-avatar">' +
                   '<img src="' + escapeHtml(data.avatar) + '" alt="" ' +
                   'onerror="this.parentNode.innerHTML=\'<span class=mention-card-initials>' +
                   escapeHtml(getInitials(data.name || data.handle)) + '</span>\'">' +
                   '</div>';
    } else {
      avatarHtml = '<div class="mention-card-avatar">' +
                   '<span class="mention-card-initials">' +
                   escapeHtml(getInitials(data.name || data.handle)) +
                   '</span></div>';
    }

    var nameHtml = '<div class="mention-card-name">' + escapeHtml(data.name || data.handle) + '</div>';
    var handleHtml = '<div class="mention-card-handle">' + escapeHtml(data.handle) + '</div>';

    var bioHtml = '';
    if (data.bio) {
      bioHtml = '<div class="mention-card-bio">' + escapeHtml(data.bio) + '</div>';
    }

    var linkHtml = '<a href="' + escapeHtml(data.url) + '" class="mention-card-link" target="_blank" rel="noopener noreferrer">' +
                   '<span aria-hidden="true">→</span> Visit site</a>';

    card.innerHTML =
      '<div class="mention-card-header">' + avatarHtml +
      '<div class="mention-card-content">' + nameHtml + handleHtml + '</div></div>' +
      bioHtml +
      '<div class="mention-card-footer">' + linkHtml + '</div>';

    return card;
  }

  /**
   * Create a loading state card
   * @returns {HTMLElement}
   */
  function createLoadingCard() {
    var card = document.createElement('div');
    card.className = 'mention-card mention-card-loading';
    card.setAttribute('role', 'tooltip');
    card.setAttribute('aria-live', 'polite');
    card.innerHTML = '<div class="mention-card-loading-text">Loading...</div>';
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
      card.classList.add('mention-card--above');
      card.classList.remove('mention-card--below');
    } else {
      card.classList.add('mention-card--below');
      card.classList.remove('mention-card--above');
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
   * Show the hover card for a mention link
   * @param {HTMLElement} target - The mention link element
   */
  function showCard(target) {
    // Don't show if another card is visible for a different target
    if (currentCard && currentTarget !== target) {
      hideCard();
    }

    currentTarget = target;

    var url = target.href;
    var handle = target.textContent.trim();

    // Show loading state
    var loadingCard = createLoadingCard();
    document.body.appendChild(loadingCard);
    currentCard = loadingCard;

    // Position after adding to DOM so we can measure
    requestAnimationFrame(function() {
      if (currentCard === loadingCard) {
        positionCard(loadingCard, target);
        loadingCard.classList.add('mention-card--visible');
      }
    });

    // Fetch data and update card
    getMentionData(url, handle)
      .then(function(data) {
        // Check if we're still showing the card for this target
        if (currentTarget !== target) return;

        var newCard = createCard(data);
        document.body.appendChild(newCard);

        // Position and show
        requestAnimationFrame(function() {
          positionCard(newCard, target);
          newCard.classList.add('mention-card--visible');

          // Remove loading card
          if (loadingCard.parentNode) {
            loadingCard.remove();
          }

          currentCard = newCard;
        });
      })
      .catch(function() {
        // On error, just hide the loading card
        if (loadingCard.parentNode) {
          loadingCard.remove();
        }
        currentCard = null;
      });
  }

  /**
   * Hide the current hover card
   */
  function hideCard() {
    if (currentCard) {
      currentCard.classList.remove('mention-card--visible');

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
   * Handle mouse enter on mention link
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
   * Handle mouse leave on mention link
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
   * Handle focus on mention link (keyboard navigation)
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
   * Handle blur on mention link
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
    document.addEventListener('mouseenter', function(e) {
      if (e.target.closest && e.target.closest('.mention-card')) {
        handleCardMouseEnter(e);
      }
    }, true);

    document.addEventListener('mouseleave', function(e) {
      if (e.target.closest && e.target.closest('.mention-card')) {
        handleCardMouseLeave(e);
      }
    }, true);
  }

  /**
   * Initialize mention cards
   */
  function init() {
    // Skip on touch devices
    if (isTouchDevice()) {
      return;
    }

    // Find all mention links
    var mentions = document.querySelectorAll('a.mention');

    mentions.forEach(function(mention) {
      mention.addEventListener('mouseenter', handleMouseEnter);
      mention.addEventListener('mouseleave', handleMouseLeave);
      mention.addEventListener('focus', handleFocus);
      mention.addEventListener('blur', handleBlur);
    });

    // Global keyboard handler for Escape
    document.addEventListener('keydown', handleKeyDown);

    // Setup card event delegation
    setupCardEventDelegation();
  }

  // Initialize when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  // Expose for external use if needed
  window.mentionCards = {
    show: showCard,
    hide: hideCard,
    clearCache: function() { cache = {}; }
  };
})();
