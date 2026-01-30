/**
 * Keyboard Shortcuts System for markata-go
 *
 * Provides keyboard navigation shortcuts for the generated site:
 * - `/` or `Cmd/Ctrl+K` - Focus search input
 * - `Escape` - Close search/modals
 * - `?` - Show shortcuts help modal
 *
 * Vim-style scrolling:
 * - `j`/`k` - Scroll down/up (~60px)
 * - `gg` - Scroll to top (double-g with timeout)
 * - `G` (Shift+g) - Scroll to bottom
 * - `d`/`u` - Scroll half-page down/up
 *
 * Feed/list navigation:
 * - `j`/`k` or arrows - Navigate posts in feed
 * - `Enter`/`o` - Open highlighted post
 * - `O` (Shift+o) - Open in new tab
 *
 * GitHub-style "go to":
 * - `g h` - Go to home
 * - `g s` - Focus search
 *
 * Utility:
 * - `yy` - Copy URL to clipboard
 * - `[`/`]` - Previous/next post (pagination)
 *
 * Accessibility: Shortcuts can be disabled via localStorage
 * WCAG 2.1.4 compliant - shortcuts are ignored when typing in inputs
 */

(function() {
  'use strict';

  // Storage key for disabled state
  var STORAGE_KEY = 'markata-shortcuts-disabled';

  // Multi-key sequence tracking
  var pendingKey = null;
  var pendingKeyTimeout = null;
  var MULTI_KEY_TIMEOUT = 800; // ms

  // Feed navigation state
  var feedPostLinks = [];
  var highlightedPostIndex = -1;

  /**
   * Check if shortcuts are disabled via localStorage
   * @returns {boolean}
   */
  function areShortcutsDisabled() {
    try {
      return localStorage.getItem(STORAGE_KEY) === 'true';
    } catch (e) {
      // localStorage may be unavailable (e.g., private browsing)
      return false;
    }
  }

  /**
   * Toggle shortcuts enabled/disabled state
   * @returns {boolean} New state (true = disabled)
   */
  function toggleShortcuts() {
    try {
      var currentlyDisabled = areShortcutsDisabled();
      localStorage.setItem(STORAGE_KEY, (!currentlyDisabled).toString());
      updateToggleButton();
      return !currentlyDisabled;
    } catch (e) {
      return false;
    }
  }

  /**
   * Update the toggle button state in the modal
   */
  function updateToggleButton() {
    var toggleBtn = document.getElementById('shortcuts-toggle');
    if (toggleBtn) {
      var disabled = areShortcutsDisabled();
      toggleBtn.textContent = disabled ? 'Enable Shortcuts' : 'Disable Shortcuts';
      toggleBtn.setAttribute('aria-pressed', (!disabled).toString());
    }
  }

  /**
   * Detect if the user is on a Mac platform
   * @returns {boolean}
   */
  function isMacPlatform() {
    // Check userAgentData first (modern browsers)
    if (navigator.userAgentData && navigator.userAgentData.platform) {
      return navigator.userAgentData.platform.toUpperCase().indexOf('MAC') >= 0;
    }
    // Fallback to navigator.platform
    return navigator.platform.toUpperCase().indexOf('MAC') >= 0;
  }

  /**
   * Update the modifier key display in the modal based on platform
   */
  function updateModifierKeyDisplay() {
    var isMac = isMacPlatform();
    var macKeys = document.querySelectorAll('.kbd-mac');
    var winKeys = document.querySelectorAll('.kbd-win');

    macKeys.forEach(function(el) {
      el.style.display = isMac ? 'inline-block' : 'none';
    });

    winKeys.forEach(function(el) {
      el.style.display = isMac ? 'none' : 'inline-block';
    });
  }

  /**
   * Focus the search input (Pagefind)
   */
  function focusSearch() {
    // Try Pagefind's input first
    var pagefindInput = document.querySelector('.pagefind-ui__search-input');
    if (pagefindInput) {
      pagefindInput.focus();
      return true;
    }

    // Fallback to any search input
    var searchInput = document.querySelector('#pagefind-search input, #search input, [type="search"]');
    if (searchInput) {
      searchInput.focus();
      return true;
    }

    return false;
  }

  /**
   * Show the shortcuts help modal
   */
  function showShortcutsModal() {
    var modal = document.getElementById('shortcuts-modal');
    if (modal) {
      modal.classList.add('shortcuts-modal--open');
      modal.setAttribute('aria-hidden', 'false');
      // Focus the close button for accessibility
      var closeBtn = modal.querySelector('.shortcuts-modal-close');
      if (closeBtn) {
        closeBtn.focus();
      }
      // Prevent body scrolling while modal is open
      document.body.style.overflow = 'hidden';
    }
  }

  /**
   * Hide the shortcuts help modal
   */
  function hideShortcutsModal() {
    var modal = document.getElementById('shortcuts-modal');
    if (modal && modal.classList.contains('shortcuts-modal--open')) {
      modal.classList.remove('shortcuts-modal--open');
      modal.setAttribute('aria-hidden', 'true');
      document.body.style.overflow = '';
      return true;
    }
    return false;
  }

  /**
   * Close all open modals and clear search focus
   */
  function closeModals() {
    var closed = false;

    // Close shortcuts modal
    if (hideShortcutsModal()) {
      closed = true;
    }

    // Blur search input if focused
    var activeElement = document.activeElement;
    if (activeElement && (activeElement.tagName === 'INPUT' || activeElement.tagName === 'TEXTAREA')) {
      activeElement.blur();
      closed = true;
    }

    // Clear Pagefind results if present
    var pagefindResults = document.querySelector('.pagefind-ui__results-area');
    if (pagefindResults && pagefindResults.children.length > 0) {
      var pagefindInput = document.querySelector('.pagefind-ui__search-input');
      if (pagefindInput) {
        pagefindInput.value = '';
        pagefindInput.dispatchEvent(new Event('input', { bubbles: true }));
        closed = true;
      }
    }

    // Clear feed highlight
    if (highlightedPostIndex >= 0) {
      clearFeedHighlight();
      closed = true;
    }

    return closed;
  }

  /**
   * Smooth scroll with reduced motion support
   * @param {number} amount - Pixels to scroll (positive = down, negative = up)
   */
  function smoothScroll(amount) {
    var reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    window.scrollBy({
      top: amount,
      behavior: reducedMotion ? 'auto' : 'smooth'
    });
  }

  /**
   * Smooth scroll to top of page
   */
  function smoothScrollToTop() {
    var reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    window.scrollTo({
      top: 0,
      behavior: reducedMotion ? 'auto' : 'smooth'
    });
  }

  /**
   * Smooth scroll to bottom of page
   */
  function smoothScrollToBottom() {
    var reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    window.scrollTo({
      top: document.body.scrollHeight,
      behavior: reducedMotion ? 'auto' : 'smooth'
    });
  }

  /**
   * Clear pending multi-key sequence
   */
  function clearPendingKey() {
    pendingKey = null;
    if (pendingKeyTimeout) {
      clearTimeout(pendingKeyTimeout);
      pendingKeyTimeout = null;
    }
  }

  /**
   * Set pending key for multi-key sequence
   * @param {string} key - The pending key
   */
  function setPendingKey(key) {
    pendingKey = key;
    if (pendingKeyTimeout) {
      clearTimeout(pendingKeyTimeout);
    }
    pendingKeyTimeout = setTimeout(clearPendingKey, MULTI_KEY_TIMEOUT);
  }

  /**
   * Initialize feed navigation by finding post links
   * @returns {boolean} True if feed posts were found
   */
  function initFeedNavigation() {
    // Find all post links in feeds/lists
    feedPostLinks = Array.from(document.querySelectorAll(
      '.post-list article a[href], ' +
      '.feed-list .post-link, ' +
      '.card a[href], ' +
      '.post-card a[href], ' +
      '.reader-entry-title a[href]'
    ));
    return feedPostLinks.length > 0;
  }

  /**
   * Check if the page has feed navigation available
   * @returns {boolean}
   */
  function hasFeedNavigation() {
    return feedPostLinks.length > 0;
  }

  /**
   * Highlight a post in the feed
   * @param {number} index - Index of post to highlight
   */
  function highlightPost(index) {
    // Remove previous highlight
    feedPostLinks.forEach(function(link) {
      link.classList.remove('kb-highlighted');
    });

    // Add highlight to current
    if (index >= 0 && index < feedPostLinks.length) {
      feedPostLinks[index].classList.add('kb-highlighted');
      feedPostLinks[index].scrollIntoView({ behavior: 'smooth', block: 'nearest' });
      highlightedPostIndex = index;
    }
  }

  /**
   * Clear feed highlight
   */
  function clearFeedHighlight() {
    feedPostLinks.forEach(function(link) {
      link.classList.remove('kb-highlighted');
    });
    highlightedPostIndex = -1;
  }

  /**
   * Navigate to the highlighted post
   * @param {boolean} newTab - Open in new tab
   */
  function navigateToHighlighted(newTab) {
    if (highlightedPostIndex >= 0 && highlightedPostIndex < feedPostLinks.length) {
      var url = feedPostLinks[highlightedPostIndex].href;
      if (newTab) {
        window.open(url, '_blank');
      } else {
        window.location.href = url;
      }
    }
  }

  /**
   * Copy current URL to clipboard and show toast notification
   */
  function copyUrlToClipboard() {
    var url = window.location.href;

    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(url).then(function() {
        showToast('URL copied to clipboard');
      }).catch(function() {
        fallbackCopyUrl(url);
      });
    } else {
      fallbackCopyUrl(url);
    }
  }

  /**
   * Fallback URL copy method for older browsers
   * @param {string} url - URL to copy
   */
  function fallbackCopyUrl(url) {
    var textArea = document.createElement('textarea');
    textArea.value = url;
    textArea.style.position = 'fixed';
    textArea.style.left = '-9999px';
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();

    try {
      var successful = document.execCommand('copy');
      if (successful) {
        showToast('URL copied to clipboard');
      } else {
        showToast('Failed to copy URL');
      }
    } catch (err) {
      showToast('Failed to copy URL');
    }

    document.body.removeChild(textArea);
  }

  /**
   * Show a toast notification
   * @param {string} message - Message to display
   */
  function showToast(message) {
    // Remove existing toast if any
    var existingToast = document.querySelector('.kb-toast');
    if (existingToast) {
      existingToast.remove();
    }

    var toast = document.createElement('div');
    toast.className = 'kb-toast';
    toast.textContent = message;
    toast.setAttribute('role', 'status');
    toast.setAttribute('aria-live', 'polite');
    document.body.appendChild(toast);

    // Trigger animation
    setTimeout(function() {
      toast.classList.add('kb-toast--visible');
    }, 10);

    // Remove after delay
    setTimeout(function() {
      toast.classList.remove('kb-toast--visible');
      setTimeout(function() {
        if (toast.parentNode) {
          toast.remove();
        }
      }, 300);
    }, 2000);
  }

  /**
   * Navigate to previous/next post via pagination links
   * @param {string} direction - 'prev' or 'next'
   */
  function navigatePagination(direction) {
    var selector = direction === 'prev'
      ? '.pagination-prev a, .prev-post a, a[rel="prev"]'
      : '.pagination-next a, .next-post a, a[rel="next"]';
    var link = document.querySelector(selector);
    if (link && link.href) {
      window.location.href = link.href;
    }
  }

  /**
   * Check if the current element is an input field where typing should be allowed
   * @param {Element} element
   * @returns {boolean}
   */
  function isInputElement(element) {
    if (!element) return false;

    var tagName = element.tagName;
    if (tagName === 'INPUT' || tagName === 'TEXTAREA' || tagName === 'SELECT') {
      return true;
    }

    // Check for contenteditable
    if (element.isContentEditable) {
      return true;
    }

    return false;
  }

  /**
   * Main keyboard event handler
   * @param {KeyboardEvent} e
   */
  function handleKeyDown(e) {
    var target = e.target;

    // Always allow Escape to work, even in inputs
    if (e.key === 'Escape') {
      clearPendingKey();
      if (closeModals()) {
        e.preventDefault();
      }
      return;
    }

    // Skip other shortcuts when typing in input fields
    if (isInputElement(target)) {
      return;
    }

    // Check if shortcuts are disabled (except for Escape which always works)
    if (areShortcutsDisabled()) {
      return;
    }

    // Detect platform for modifier key
    var modifier = isMacPlatform() ? e.metaKey : e.ctrlKey;

    // Cmd/Ctrl+K - Focus search
    if (modifier && e.key === 'k') {
      e.preventDefault();
      clearPendingKey();
      focusSearch();
      return;
    }

    // Don't process other shortcuts if modifier keys are held (except the ones we handle)
    if (e.ctrlKey || e.metaKey || e.altKey) {
      return;
    }

    // Handle multi-key sequences
    if (pendingKey === 'g') {
      clearPendingKey();
      switch (e.key) {
        case 'g':
          // gg - scroll to top
          e.preventDefault();
          smoothScrollToTop();
          return;
        case 'h':
          // g h - go to home
          e.preventDefault();
          window.location.href = '/';
          return;
        case 's':
          // g s - focus search
          e.preventDefault();
          focusSearch();
          return;
      }
      // Pending 'g' consumed, fall through for other keys
    }

    if (pendingKey === 'y') {
      clearPendingKey();
      if (e.key === 'y') {
        // yy - copy URL to clipboard
        e.preventDefault();
        copyUrlToClipboard();
        return;
      }
      // Pending 'y' consumed, fall through for other keys
    }

    // Handle single-key shortcuts
    switch (e.key) {
      case '/':
        e.preventDefault();
        clearPendingKey();
        focusSearch();
        break;

      case '?':
        e.preventDefault();
        clearPendingKey();
        showShortcutsModal();
        break;

      case 'j':
        e.preventDefault();
        clearPendingKey();
        if (hasFeedNavigation()) {
          // Feed navigation: move to next post
          highlightPost(Math.min(highlightedPostIndex + 1, feedPostLinks.length - 1));
        } else {
          // Vim-style scroll down
          smoothScroll(60);
        }
        break;

      case 'k':
        e.preventDefault();
        clearPendingKey();
        if (hasFeedNavigation()) {
          // Feed navigation: move to previous post
          highlightPost(Math.max(highlightedPostIndex - 1, 0));
        } else {
          // Vim-style scroll up
          smoothScroll(-60);
        }
        break;

      case 'ArrowDown':
        if (hasFeedNavigation()) {
          e.preventDefault();
          clearPendingKey();
          highlightPost(Math.min(highlightedPostIndex + 1, feedPostLinks.length - 1));
        }
        break;

      case 'ArrowUp':
        if (hasFeedNavigation()) {
          e.preventDefault();
          clearPendingKey();
          highlightPost(Math.max(highlightedPostIndex - 1, 0));
        }
        break;

      case 'g':
        // Start 'g' sequence (for gg, gh, gs)
        e.preventDefault();
        setPendingKey('g');
        break;

      case 'G':
        // Shift+G - scroll to bottom
        if (e.shiftKey) {
          e.preventDefault();
          clearPendingKey();
          smoothScrollToBottom();
        }
        break;

      case 'd':
        // Half-page down
        e.preventDefault();
        clearPendingKey();
        smoothScroll(window.innerHeight / 2);
        break;

      case 'u':
        // Half-page up
        e.preventDefault();
        clearPendingKey();
        smoothScroll(-window.innerHeight / 2);
        break;

      case 'y':
        // Start 'y' sequence (for yy)
        e.preventDefault();
        setPendingKey('y');
        break;

      case 'o':
        // Open highlighted post
        if (highlightedPostIndex >= 0) {
          e.preventDefault();
          clearPendingKey();
          navigateToHighlighted(false);
        }
        break;

      case 'O':
        // Open highlighted post in new tab (Shift+o)
        if (e.shiftKey && highlightedPostIndex >= 0) {
          e.preventDefault();
          clearPendingKey();
          navigateToHighlighted(true);
        }
        break;

      case 'Enter':
        // Open highlighted post
        if (highlightedPostIndex >= 0) {
          e.preventDefault();
          clearPendingKey();
          navigateToHighlighted(false);
        }
        break;

      case '[':
        // Previous post (pagination)
        e.preventDefault();
        clearPendingKey();
        navigatePagination('prev');
        break;

      case ']':
        // Next post (pagination)
        e.preventDefault();
        clearPendingKey();
        navigatePagination('next');
        break;
    }
  }

  /**
   * Handle click outside modal to close it
   * @param {MouseEvent} e
   */
  function handleModalBackdropClick(e) {
    var modal = document.getElementById('shortcuts-modal');
    if (modal && e.target === modal) {
      hideShortcutsModal();
    }
  }

  /**
   * Initialize the keyboard shortcuts system
   */
  function init() {
    // Update modifier key display for the current platform
    updateModifierKeyDisplay();

    // Initialize feed navigation
    initFeedNavigation();

    // Add keyboard event listener
    document.addEventListener('keydown', handleKeyDown);

    // Setup modal close button
    var closeBtn = document.querySelector('.shortcuts-modal-close');
    if (closeBtn) {
      closeBtn.addEventListener('click', hideShortcutsModal);
    }

    // Setup toggle button
    var toggleBtn = document.getElementById('shortcuts-toggle');
    if (toggleBtn) {
      toggleBtn.addEventListener('click', toggleShortcuts);
      updateToggleButton();
    }

    // Close modal on backdrop click
    var modal = document.getElementById('shortcuts-modal');
    if (modal) {
      modal.addEventListener('click', handleModalBackdropClick);
    }
  }

  // Initialize when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  // Expose functions for external use if needed
  window.markataShortcuts = {
    focusSearch: focusSearch,
    showShortcutsModal: showShortcutsModal,
    hideShortcutsModal: hideShortcutsModal,
    toggleShortcuts: toggleShortcuts,
    areShortcutsDisabled: areShortcutsDisabled,
    smoothScroll: smoothScroll,
    smoothScrollToTop: smoothScrollToTop,
    smoothScrollToBottom: smoothScrollToBottom,
    copyUrlToClipboard: copyUrlToClipboard,
    showToast: showToast
  };
})();
