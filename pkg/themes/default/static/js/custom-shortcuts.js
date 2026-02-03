/**
 * Custom Shortcuts Module for markata-go
 *
 * Registers user-defined keyboard shortcuts from markata.toml configuration.
 * Supports key sequences like "g t" for navigation to specific pages.
 *
 * Configuration in markata.toml:
 *   [shortcuts.navigation]
 *   "g t" = "/tags/"
 *   "g a" = "/about/"
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
    pendingKeys: [],
    lastKeyTime: 0,
    keySequenceTimeout: 500 // ms
  };

  /**
   * Parse a key sequence string into an array of keys
   * @param {string} sequence - e.g., "g t" or "Shift+G"
   * @returns {Array<string>} Array of individual keys
   */
  function parseKeySequence(sequence) {
    return sequence.trim().split(/\s+/);
  }

  /**
   * Navigate to a URL
   * @param {string} url - The destination URL
   */
  function navigateTo(url) {
    window.location.href = url;
  }

  /**
   * Get description for a custom shortcut
   * @param {string} url - The destination URL
   * @returns {string} Human-readable description
   */
  function getDescription(url) {
    // Extract a meaningful name from the URL
    var path = url.replace(/^\/+|\/+$/g, '');
    if (!path) {
      return 'Go to home';
    }
    // Capitalize first letter
    var name = path.split('/').pop();
    name = name.charAt(0).toUpperCase() + name.slice(1);
    return 'Go to ' + name;
  }

  /**
   * Initialize custom shortcuts from window.customShortcuts
   */
  function init() {
    if (!window.customShortcuts || !window.customShortcuts.navigation) {
      return;
    }

    var navigation = window.customShortcuts.navigation;
    var shortcuts = [];

    // Parse all shortcuts and group by first key
    for (var sequence in navigation) {
      if (navigation.hasOwnProperty(sequence)) {
        var url = navigation[sequence];
        var keys = parseKeySequence(sequence);
        shortcuts.push({
          keys: keys,
          url: url,
          description: getDescription(url)
        });
      }
    }

    if (shortcuts.length === 0) {
      return;
    }

    // Register each custom shortcut with the registry for display in help modal
    shortcuts.forEach(function(shortcut) {
      var displayKey = shortcut.keys.join(' ');
      window.shortcutsRegistry.register({
        key: displayKey,
        modifiers: [],
        description: shortcut.description,
        group: 'custom navigation',
        // Handler is a no-op since we handle it via keydown listener for sequences
        handler: function() {},
        priority: 5
      });
    });

    // Handle key sequences via keydown listener
    document.addEventListener('keydown', function(e) {
      if (window.shortcutsRegistry.areDisabled()) return;
      if (window.shortcutsRegistry.isInputElement(e.target)) return;

      var now = Date.now();
      var timeSinceLastKey = now - state.lastKeyTime;

      // Reset sequence if too much time passed
      if (timeSinceLastKey > state.keySequenceTimeout) {
        state.pendingKeys = [];
      }

      // Add current key to sequence
      state.pendingKeys.push(e.key);
      state.lastKeyTime = now;

      // Check if current sequence matches any shortcut
      var currentSequence = state.pendingKeys.join(' ');

      for (var i = 0; i < shortcuts.length; i++) {
        var shortcut = shortcuts[i];
        var targetSequence = shortcut.keys.join(' ');

        if (currentSequence === targetSequence) {
          // Full match - navigate
          e.preventDefault();
          navigateTo(shortcut.url);
          state.pendingKeys = [];
          state.lastKeyTime = 0;
          return;
        }

        // Check if current sequence is a prefix of this shortcut
        if (targetSequence.indexOf(currentSequence) === 0) {
          // Partial match - wait for more keys
          e.preventDefault();
          return;
        }
      }

      // No match found - reset if no partial matches
      var hasPartialMatch = shortcuts.some(function(shortcut) {
        var targetSequence = shortcut.keys.join(' ');
        return targetSequence.indexOf(currentSequence) === 0;
      });

      if (!hasPartialMatch) {
        state.pendingKeys = [];
        state.lastKeyTime = 0;
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
