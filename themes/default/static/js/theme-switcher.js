/**
 * Multi-Palette Theme Switcher
 * Provides a dropdown UI for selecting from all available color palettes.
 * Integrates with the existing light/dark theme toggle.
 *
 * CSS Variables used:
 * - --palette-manifest: JSON array of available palettes
 * - --palette-switcher-enabled: Set to 1 when switcher is enabled
 * - --palette-light: Default light mode palette name
 * - --palette-dark: Default dark mode palette name
 */
(function() {
  'use strict';

  // Check if switcher is enabled
  const styles = getComputedStyle(document.documentElement);
  const switcherEnabled = styles.getPropertyValue('--palette-switcher-enabled').trim() === '1';

  if (!switcherEnabled) {
    return;
  }

  const STORAGE_KEY_PALETTE = 'palette';
  const STORAGE_KEY_THEME = 'theme';

  /**
   * Parse the palette manifest from CSS custom property
   * @returns {Array} Array of palette entries
   */
  function getPaletteManifest() {
    const manifestStr = styles.getPropertyValue('--palette-manifest').trim();
    if (!manifestStr) {
      return [];
    }
    try {
      // Remove surrounding quotes if present
      const cleaned = manifestStr.replace(/^'|'$/g, '');
      return JSON.parse(cleaned);
    } catch (e) {
      console.error('Failed to parse palette manifest:', e);
      return [];
    }
  }

  /**
   * Get default palette names from CSS
   * @returns {{light: string, dark: string}}
   */
  function getDefaultPalettes() {
    return {
      light: styles.getPropertyValue('--palette-light').trim().replace(/"/g, '') || 'default-light',
      dark: styles.getPropertyValue('--palette-dark').trim().replace(/"/g, '') || 'default-dark'
    };
  }

  /**
   * Get current theme (light/dark)
   * @returns {string}
   */
  function getCurrentTheme() {
    const stored = localStorage.getItem(STORAGE_KEY_THEME);
    if (stored) return stored;
    if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
      return 'dark';
    }
    return 'light';
  }

  /**
   * Get currently selected palette name
   * @returns {string|null}
   */
  function getCurrentPalette() {
    return localStorage.getItem(STORAGE_KEY_PALETTE);
  }

  /**
   * Set the active palette
   * @param {string} paletteName - Normalized palette name
   */
  function setPalette(paletteName) {
    const root = document.documentElement;

    // Set data-palette attribute
    root.dataset.palette = paletteName;

    // Persist selection
    localStorage.setItem(STORAGE_KEY_PALETTE, paletteName);

    // Dispatch event for other scripts
    window.dispatchEvent(new CustomEvent('palette-change', {
      detail: { palette: paletteName }
    }));

    // Update dropdown selection if it exists
    const dropdown = document.querySelector('.palette-switcher-select');
    if (dropdown && dropdown.value !== paletteName) {
      dropdown.value = paletteName;
    }
  }

  /**
   * Clear palette selection (use theme default)
   */
  function clearPalette() {
    const root = document.documentElement;
    delete root.dataset.palette;
    localStorage.removeItem(STORAGE_KEY_PALETTE);

    window.dispatchEvent(new CustomEvent('palette-change', {
      detail: { palette: null }
    }));
  }

  /**
   * Group palettes by base name for the dropdown
   * @param {Array} palettes
   * @returns {Map}
   */
  function groupPalettesByBase(palettes) {
    const groups = new Map();
    for (const p of palettes) {
      const base = p.baseName || p.name;
      if (!groups.has(base)) {
        groups.set(base, []);
      }
      groups.get(base).push(p);
    }
    return groups;
  }

  /**
   * Create the palette switcher dropdown UI
   * @returns {HTMLElement}
   */
  function createSwitcherUI() {
    const manifest = getPaletteManifest();
    if (manifest.length === 0) {
      return null;
    }

    const container = document.createElement('div');
    container.className = 'palette-switcher';

    const label = document.createElement('label');
    label.className = 'palette-switcher-label';
    label.textContent = 'Theme';
    label.setAttribute('for', 'palette-switcher-select');

    const select = document.createElement('select');
    select.className = 'palette-switcher-select';
    select.id = 'palette-switcher-select';
    select.setAttribute('aria-label', 'Select color palette');

    // Add default option
    const defaultOpt = document.createElement('option');
    defaultOpt.value = '';
    defaultOpt.textContent = 'Auto (System)';
    select.appendChild(defaultOpt);

    // Group palettes by base name
    const groups = groupPalettesByBase(manifest);

    // Sort groups alphabetically
    const sortedGroups = Array.from(groups.entries()).sort((a, b) => a[0].localeCompare(b[0]));

    for (const [baseName, palettes] of sortedGroups) {
      if (palettes.length === 1) {
        // Single palette, no optgroup needed
        const opt = document.createElement('option');
        opt.value = palettes[0].name;
        opt.textContent = palettes[0].displayName;
        select.appendChild(opt);
      } else {
        // Multiple variants, use optgroup
        const group = document.createElement('optgroup');
        // Capitalize base name for display
        group.label = baseName.charAt(0).toUpperCase() + baseName.slice(1);

        for (const p of palettes) {
          const opt = document.createElement('option');
          opt.value = p.name;
          // Show variant in parentheses
          const variantLabel = p.variant === 'light' ? 'Light' : p.variant === 'dark' ? 'Dark' : p.variant;
          opt.textContent = `${p.displayName}`;
          select.appendChild(opt);
        }

        // Note: Optgroups are appended to select, but options go directly to select
        // This is intentional - we want flat list with all options
      }
    }

    // Set current selection
    const currentPalette = getCurrentPalette();
    if (currentPalette) {
      select.value = currentPalette;
    }

    // Handle selection change
    select.addEventListener('change', function(e) {
      const value = e.target.value;
      if (value) {
        setPalette(value);
      } else {
        clearPalette();
      }
    });

    container.appendChild(label);
    container.appendChild(select);

    return container;
  }

  /**
   * Insert the switcher into the page
   */
  function insertSwitcher() {
    const switcher = createSwitcherUI();
    if (!switcher) {
      return;
    }

    // Try to insert in header first
    const headerNav = document.querySelector('.site-nav');
    if (headerNav) {
      headerNav.appendChild(switcher);
      return;
    }

    // Try header container
    const headerContainer = document.querySelector('.site-header .container');
    if (headerContainer) {
      headerContainer.appendChild(switcher);
      return;
    }

    // Fallback to header itself
    const header = document.querySelector('.site-header');
    if (header) {
      header.appendChild(switcher);
    }
  }

  /**
   * Initialize palette from storage on page load
   */
  function initializePalette() {
    const savedPalette = getCurrentPalette();
    if (savedPalette) {
      const root = document.documentElement;
      root.dataset.palette = savedPalette;
    }
  }

  // Initialize
  initializePalette();

  // Insert UI when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', insertSwitcher);
  } else {
    insertSwitcher();
  }

  // Listen for theme changes to potentially update palette
  window.addEventListener('theme-change', function(e) {
    // When theme changes and no specific palette is selected,
    // the CSS will automatically use the appropriate default
    const currentPalette = getCurrentPalette();
    if (!currentPalette) {
      // No custom palette selected, theme change handles it
      return;
    }
    // If a palette is selected, keep it active
  });

  // Expose API globally
  window.markata = window.markata || {};
  window.markata.paletteSwitcher = {
    getPalettes: getPaletteManifest,
    getCurrent: getCurrentPalette,
    set: setPalette,
    clear: clearPalette,
    getDefaults: getDefaultPalettes
  };
})();
