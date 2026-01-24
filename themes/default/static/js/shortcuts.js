/**
 * Keyboard Shortcuts System for markata-go
 * 
 * Provides keyboard navigation shortcuts for the generated site:
 * - `/` or `Cmd/Ctrl+K` - Focus search input
 * - `Escape` - Close search/modals
 * - `?` - Show shortcuts help modal
 * 
 * Accessibility: Shortcuts can be disabled via localStorage
 * WCAG 2.1.4 compliant - shortcuts are ignored when typing in inputs
 */

(function() {
  'use strict';

  // Storage key for disabled state
  var STORAGE_KEY = 'markata-shortcuts-disabled';

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
    
    return closed;
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
      focusSearch();
      return;
    }
    
    // Don't process other shortcuts if modifier keys are held (except the ones we handle)
    if (e.ctrlKey || e.metaKey || e.altKey) {
      return;
    }
    
    // Handle single-key shortcuts
    switch (e.key) {
      case '/':
        e.preventDefault();
        focusSearch();
        break;
        
      case '?':
        e.preventDefault();
        showShortcutsModal();
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
    areShortcutsDisabled: areShortcutsDisabled
  };
})();
