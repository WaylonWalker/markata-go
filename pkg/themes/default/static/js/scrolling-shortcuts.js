/**
 * Scrolling Shortcuts Module for markata-go
 *
 * Registers scrolling-related keyboard shortcuts with the shortcuts registry.
 * - `j` or `↓` - Scroll down
 * - `k` or `↑` - Scroll up
 * - `d` - Scroll half-page down
 * - `u` - Scroll half-page up
 * - `g g` - Scroll to top
 * - `Shift+G` - Scroll to bottom
 */

(function() {
  'use strict';

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
   * Scroll the page
   * @param {number} amount - Amount to scroll in pixels
   */
  function scroll(amount) {
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
    scroll(amount);
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
   * Initialize scrolling shortcuts
   */
  function init() {
    // j or ↓ - Scroll down
    window.shortcutsRegistry.register({
      key: 'j',
      modifiers: [],
      description: 'Scroll down',
      group: 'scrolling',
      handler: function(e) {
        e.preventDefault();
        scroll(100);
      },
      priority: 10
    });

    // k or ↑ - Scroll up
    window.shortcutsRegistry.register({
      key: 'k',
      modifiers: [],
      description: 'Scroll up',
      group: 'scrolling',
      handler: function(e) {
        e.preventDefault();
        scroll(-100);
      },
      priority: 10
    });

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

    // Handle two-key sequence: g g for scroll to top
    var lastKeyTime = 0;
    var lastKey = null;
    var KEY_SEQUENCE_TIMEOUT = 500; // ms

    document.addEventListener('keydown', function(e) {
      if (window.shortcutsRegistry.areDisabled()) return;
      if (window.shortcutsRegistry.isInputElement(e.target)) return;

      var now = Date.now();
      var timeSinceLastKey = now - lastKeyTime;

      if (e.key === 'g') {
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
      } else if (e.key === 'G' || (e.shiftKey && e.key === 'g')) {
        // Shift+G - scroll to bottom
        if (!window.shortcutsRegistry.areDisabled() && !window.shortcutsRegistry.isInputElement(e.target)) {
          e.preventDefault();
          scrollToBottom();
        }
        lastKey = null;
        lastKeyTime = 0;
      } else {
        // Reset sequence on other keys
        lastKey = null;
        lastKeyTime = 0;
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
