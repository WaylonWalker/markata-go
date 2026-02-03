/**
 * Scrolling Shortcuts Module for markata-go
 *
 * Registers scrolling-related keyboard shortcuts with the shortcuts registry.
 * - `j` - Scroll down (on post/article pages)
 * - `k` - Scroll up (on post/article pages)
 * - `d` - Scroll half-page down
 * - `u` - Scroll half-page up
 * - `g g` - Scroll to top
 * - `Shift+G` - Scroll to bottom
 *
 * Note: j/k are primarily handled by navigation module for feed card selection.
 * On post/article pages, j/k scroll the content.
 */

(function() {
  'use strict';

  // Track which keys are currently pressed for smooth held-key scrolling
  var keysPressed = {};
  var scrollAnimationId = null;
  var lastScrollTime = 0;
  var scrollThrottle = 16; // ~60fps (ms between scrolls)

  // Wait for registry to be available
  function waitForRegistry(callback, attempts = 0) {
    if (window.shortcutsRegistry) {
      callback();
    } else if (attempts < 50) {
      setTimeout(function() {
        waitForRegistry(callback, attempts + 1);
      }, 10);
    }
  }

  /**
   * Scroll the page instantly (for held keys)
   * @param {number} amount - Amount to scroll in pixels
   */
  function scrollInstant(amount) {
    window.scrollBy({
      top: amount,
      behavior: 'auto'
    });
  }

  /**
   * Scroll the page smoothly (for single key press)
   * @param {number} amount - Amount to scroll in pixels
   */
  function scrollSmooth(amount) {
    window.scrollBy({
      top: amount,
      behavior: 'smooth'
    });
  }

  /**
   * Scroll by a percentage of the window height
   */
  function scrollByPercent(percent) {
    var amount = window.innerHeight * percent;
    scrollSmooth(amount);
  }

  /**
   * Scroll to top
   */
  function scrollToTop() {
    window.scrollTo({
      top: 0,
      behavior: 'smooth'
    });
  }

  /**
   * Scroll to bottom
   */
  function scrollToBottom() {
    window.scrollTo({
      top: document.documentElement.scrollHeight,
      behavior: 'smooth'
    });
  }

  /**
   * Check if we're on a post/article page
   */
  function isPostPage() {
    // Check for article element or post-specific classes/data attributes
    return document.querySelector('article, [data-type="post"], .post-content') !== null;
  }

  /**
   * Continuous scroll loop - runs while keys are held
   */
  function continuousScroll() {
    var now = Date.now();

    // Throttle scrolling to ~60fps
    if (now - lastScrollTime > scrollThrottle) {
      var shouldScroll = false;
      var scrollAmount = 0;

      // Check which scroll keys are pressed
      if (keysPressed['j']) {
        scrollAmount += 8; // Scroll down (smaller increments for smooth feel)
        shouldScroll = true;
      }
      if (keysPressed['k']) {
        scrollAmount -= 8; // Scroll up
        shouldScroll = true;
      }
      if (keysPressed['d']) {
        scrollAmount += 4; // Half-page scroll is slower
        shouldScroll = true;
      }
      if (keysPressed['u']) {
        scrollAmount -= 4; // Half-page scroll is slower
        shouldScroll = true;
      }

      if (shouldScroll && scrollAmount !== 0) {
        scrollInstant(scrollAmount);
        lastScrollTime = now;
      }
    }

    // Continue loop if any keys are pressed
    if (Object.keys(keysPressed).some(function(k) { return keysPressed[k]; })) {
      scrollAnimationId = requestAnimationFrame(continuousScroll);
    } else {
      scrollAnimationId = null;
    }
  }

  /**
   * Initialize scrolling shortcuts
   */
  function init() {
    // d - Scroll half-page down
    window.shortcutsRegistry.register({
      key: 'd',
      modifiers: [],
      description: 'Scroll half-page down',
      group: 'scrolling',
      handler: function(e) {
        e.preventDefault();
        scrollByPercent(0.5);
      },
      priority: 10
    });

    // u - Scroll half-page up
    window.shortcutsRegistry.register({
      key: 'u',
      modifiers: [],
      description: 'Scroll half-page up',
      group: 'scrolling',
      handler: function(e) {
        e.preventDefault();
        scrollByPercent(-0.5);
      },
      priority: 10
    });

    // j/k scrolling - for post/article pages
    // (navigation module handles j/k for feed card selection)
    if (isPostPage()) {
      // j - Scroll down
      window.shortcutsRegistry.register({
        key: 'j',
        modifiers: [],
        description: 'Scroll down',
        group: 'scrolling',
        handler: function(e) {
          e.preventDefault();
          // Single press uses smooth scroll
          scrollSmooth(100);
        },
        priority: 10
      });

      // k - Scroll up
      window.shortcutsRegistry.register({
        key: 'k',
        modifiers: [],
        description: 'Scroll up',
        group: 'scrolling',
        handler: function(e) {
          e.preventDefault();
          // Single press uses smooth scroll
          scrollSmooth(-100);
        },
        priority: 10
      });
    }

    // Handle two-key sequence: g g for scroll to top
    var lastKeyTime = 0;
    var lastKey = null;
    var KEY_SEQUENCE_TIMEOUT = 500; // ms

    // Track keydown to detect held keys for smooth scrolling
    document.addEventListener('keydown', function(e) {
      if (window.shortcutsRegistry.areDisabled()) return;
      if (window.shortcutsRegistry.isInputElement(e.target)) return;

      var key = e.key.toLowerCase();

      // Track if scroll keys are pressed (for held key detection)
      if (isPostPage() && (key === 'j' || key === 'k')) {
        keysPressed[key] = true;
        if (!scrollAnimationId) {
          scrollAnimationId = requestAnimationFrame(continuousScroll);
        }
        e.preventDefault();
        return;
      }

      if (key === 'd' || key === 'u') {
        keysPressed[key] = true;
        if (!scrollAnimationId) {
          scrollAnimationId = requestAnimationFrame(continuousScroll);
        }
        e.preventDefault();
        return;
      }

      var now = Date.now();
      var timeSinceLastKey = now - lastKeyTime;

      if (key === 'g') {
        if (lastKey === 'g' && timeSinceLastKey < KEY_SEQUENCE_TIMEOUT) {
          // g g - scroll to top
          e.preventDefault();
          scrollToTop();
          lastKey = null;
          lastKeyTime = 0;
        } else {
          lastKey = 'g';
          lastKeyTime = now;
        }
      } else if (e.shiftKey && key === 'g') {
        // Shift+G - scroll to bottom
        e.preventDefault();
        scrollToBottom();
        lastKey = null;
        lastKeyTime = 0;
      } else {
        // Reset sequence on other keys
        lastKey = null;
        lastKeyTime = 0;
      }
    });

    // Track keyup to detect when keys are released
    document.addEventListener('keyup', function(e) {
      var key = e.key.toLowerCase();
      if (keysPressed[key]) {
        delete keysPressed[key];
        // Continue scroll loop if other keys are still pressed
        if (Object.keys(keysPressed).some(function(k) { return keysPressed[k]; })) {
          if (!scrollAnimationId) {
            scrollAnimationId = requestAnimationFrame(continuousScroll);
          }
        }
      }
    });
  }

  // Initialize when registry is available
  waitForRegistry(function() {
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', init);
    } else {
      init();
    }
  });
})();
