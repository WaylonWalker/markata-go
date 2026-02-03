/**
 * History Navigation Shortcuts Module for markata-go
 *
 * Registers history navigation shortcuts with the shortcuts registry.
 * - `h` - Go back to previous page (browser history)
 * - `l` - Go forward to next page (browser history)
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
   * Go back to previous page
   */
  function goBack() {
    window.history.back();
  }

  /**
   * Go forward to next page
   */
  function goForward() {
    window.history.forward();
  }

  /**
   * Initialize history shortcuts
   */
  function init() {
    // h - Go back
    window.shortcutsRegistry.register({
      key: 'h',
      modifiers: [],
      description: 'Go back to previous page',
      group: 'navigation',
      handler: function(e) {
        e.preventDefault();
        goBack();
      },
      priority: 15
    });

    // l - Go forward
    window.shortcutsRegistry.register({
      key: 'l',
      modifiers: [],
      description: 'Go forward to next page',
      group: 'navigation',
      handler: function(e) {
        e.preventDefault();
        goForward();
      },
      priority: 15
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
