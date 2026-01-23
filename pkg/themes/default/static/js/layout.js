/**
 * Layout System JavaScript
 * Handles sidebar toggle, mobile menu, and keyboard navigation
 */

(function() {
  'use strict';

  // DOM elements
  const toggle = document.querySelector('.mobile-menu-toggle');
  const sidebar = document.querySelector('.layout-sidebar');
  const overlay = document.querySelector('.sidebar-overlay');
  const body = document.body;

  if (!toggle || !sidebar) return;

  /**
   * Toggle sidebar open/closed state
   */
  function toggleSidebar() {
    const isOpen = sidebar.dataset.open === 'true';
    setSidebarState(!isOpen);
  }

  /**
   * Set sidebar state
   * @param {boolean} open - Whether sidebar should be open
   */
  function setSidebarState(open) {
    sidebar.dataset.open = open;
    toggle.setAttribute('aria-expanded', open);
    body.classList.toggle('sidebar-open', open);

    if (open) {
      // Focus first link in sidebar for accessibility
      const firstLink = sidebar.querySelector('a');
      if (firstLink) {
        firstLink.focus();
      }
    } else {
      // Return focus to toggle button
      toggle.focus();
    }
  }

  /**
   * Close sidebar
   */
  function closeSidebar() {
    setSidebarState(false);
  }

  // Toggle button click
  toggle.addEventListener('click', toggleSidebar);

  // Overlay click closes sidebar
  if (overlay) {
    overlay.addEventListener('click', closeSidebar);
  }

  // Escape key closes sidebar
  document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape' && sidebar.dataset.open === 'true') {
      closeSidebar();
    }
  });

  // Close sidebar when clicking a link (mobile)
  sidebar.addEventListener('click', function(e) {
    if (e.target.matches('a') && window.innerWidth < 1024) {
      closeSidebar();
    }
  });

  // Handle window resize
  let resizeTimeout;
  window.addEventListener('resize', function() {
    clearTimeout(resizeTimeout);
    resizeTimeout = setTimeout(function() {
      // Close mobile sidebar if window is resized to desktop
      if (window.innerWidth >= 1024 && sidebar.dataset.open === 'true') {
        closeSidebar();
      }
    }, 100);
  });

  // Trap focus within sidebar when open (accessibility)
  sidebar.addEventListener('keydown', function(e) {
    if (e.key !== 'Tab' || sidebar.dataset.open !== 'true') return;

    const focusableElements = sidebar.querySelectorAll(
      'a[href], button, [tabindex]:not([tabindex="-1"])'
    );
    const firstElement = focusableElements[0];
    const lastElement = focusableElements[focusableElements.length - 1];

    if (e.shiftKey && document.activeElement === firstElement) {
      e.preventDefault();
      lastElement.focus();
    } else if (!e.shiftKey && document.activeElement === lastElement) {
      e.preventDefault();
      firstElement.focus();
    }
  });
})();

/**
 * Theme Toggle
 * Handles light/dark theme switching with localStorage persistence
 */
(function() {
  'use strict';

  const toggle = document.querySelector('.theme-toggle');
  if (!toggle) return;

  const STORAGE_KEY = 'theme';
  const DARK_CLASS = 'dark';

  /**
   * Get current theme preference
   * @returns {string} 'dark' or 'light'
   */
  function getTheme() {
    // Check localStorage first
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) return stored;

    // Fall back to system preference
    if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
      return 'dark';
    }
    return 'light';
  }

  /**
   * Set theme
   * @param {string} theme - 'dark' or 'light'
   */
  function setTheme(theme) {
    document.documentElement.dataset.theme = theme;
    document.documentElement.classList.toggle(DARK_CLASS, theme === 'dark');
    localStorage.setItem(STORAGE_KEY, theme);
  }

  /**
   * Toggle between light and dark themes
   */
  function toggleTheme() {
    const current = getTheme();
    setTheme(current === 'dark' ? 'light' : 'dark');
  }

  // Initialize theme on page load
  setTheme(getTheme());

  // Toggle button click
  toggle.addEventListener('click', toggleTheme);

  // Listen for system theme changes
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function(e) {
    // Only auto-switch if user hasn't set a preference
    if (!localStorage.getItem(STORAGE_KEY)) {
      setTheme(e.matches ? 'dark' : 'light');
    }
  });
})();
