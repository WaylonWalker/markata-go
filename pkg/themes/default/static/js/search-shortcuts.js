/**
 * Search Shortcuts Module for markata-go
 *
 * Registers search-related keyboard shortcuts with the shortcuts registry.
 * - `/` or `Ctrl/Cmd+K` - Focus search input
 * - `?` - Show shortcuts help modal
 * - `Escape` - Close search/modals
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
   * Detect if the user is on a Mac platform
   */
  function isMacPlatform() {
    if (navigator.userAgentData && navigator.userAgentData.platform) {
      return navigator.userAgentData.platform.toUpperCase().indexOf('MAC') >= 0;
    }
    return navigator.platform.toUpperCase().indexOf('MAC') >= 0;
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

    return closed;
  }

  /**
   * Update the toggle button state in the modal
   */
  function updateToggleButton() {
    var toggleBtn = document.getElementById('shortcuts-toggle');
    if (toggleBtn) {
      var disabled = window.shortcutsRegistry.areDisabled();
      toggleBtn.textContent = disabled ? 'Enable Shortcuts' : 'Disable Shortcuts';
      toggleBtn.setAttribute('aria-pressed', (!disabled).toString());
    }
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
   * Handle click outside modal to close it
   */
  function handleModalBackdropClick(e) {
    var modal = document.getElementById('shortcuts-modal');
    if (modal && e.target === modal) {
      hideShortcutsModal();
    }
  }

  /**
   * Initialize search shortcuts
   */
  function init() {
    // Register / shortcut for search
    window.shortcutsRegistry.register({
      key: '/',
      modifiers: [],
      description: 'Focus search input',
      group: 'search',
      handler: function(e) {
        e.preventDefault();
        focusSearch();
      },
      priority: 100
    });

    // Register Ctrl/Cmd+K for search (need to handle modifiers specially)
    document.addEventListener('keydown', function(e) {
      if (window.shortcutsRegistry.areDisabled()) return;
      if (window.shortcutsRegistry.isInputElement(e.target)) return;

      var modifier = isMacPlatform() ? e.metaKey : e.ctrlKey;
      if (modifier && e.key === 'k') {
        e.preventDefault();
        focusSearch();
      }
    });

    // Register ? for help modal
    window.shortcutsRegistry.register({
      key: '?',
      modifiers: [],
      description: 'Show shortcuts help',
      group: 'help',
      handler: function(e) {
        e.preventDefault();
        showShortcutsModal();
      },
      priority: 100
    });

    // Handle Escape globally (always works, even in inputs)
    document.addEventListener('keydown', function(e) {
      if (e.key === 'Escape') {
        if (closeModals()) {
          e.preventDefault();
        }
      }
    });

    // Setup modal close button
    var closeBtn = document.querySelector('.shortcuts-modal-close');
    if (closeBtn) {
      closeBtn.addEventListener('click', hideShortcutsModal);
    }

    // Setup toggle button
    var toggleBtn = document.getElementById('shortcuts-toggle');
    if (toggleBtn) {
      toggleBtn.addEventListener('click', function() {
        var newDisabled = window.shortcutsRegistry.toggleAll();
        updateToggleButton();
      });
      updateToggleButton();
    }

    // Close modal on backdrop click
    var modal = document.getElementById('shortcuts-modal');
    if (modal) {
      modal.addEventListener('click', handleModalBackdropClick);
    }

    // Update modifier key display
    updateModifierKeyDisplay();
  }

  // Initialize when DOM is ready (or registry is available)
  waitForRegistry(function() {
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', init);
    } else {
      init();
    }
  });

  // Expose for backward compatibility
  window.markataShortcuts = {
    focusSearch: focusSearch,
    showShortcutsModal: showShortcutsModal,
    hideShortcutsModal: hideShortcutsModal,
    toggleShortcuts: function() {
      return window.shortcutsRegistry.toggleAll();
    },
    areShortcutsDisabled: function() {
      return window.shortcutsRegistry.areDisabled();
    }
  };
})();
