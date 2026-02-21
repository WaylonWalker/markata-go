/**
 * Palette Switcher
 * A dropdown UI for selecting color palette families with separate dark/light toggle.
 * Works with markata-go's theme switcher system.
 */

(function() {
  'use strict';

  const STORAGE_KEY = 'selected-palette';
  const FAMILY_KEY = 'selected-family';
  const MODE_KEY = 'color-mode'; // 'light' or 'dark'
  const AESTHETIC_KEY = 'selected-aesthetic';

  // Aesthetic list is populated from CSS manifest dynamically
  let AESTHETICS = [];

  // Variant suffixes to strip when computing family names
  const VARIANT_SUFFIXES = [
    '-light', '-dark',
    '-dawn', '-moon',           // rose-pine
    '-day', '-storm',           // tokyo-night
    '-latte', '-frappe', '-macchiato', '-mocha',  // catppuccin
    '-mirage',                  // ayu
    '-wave', '-dragon', '-lotus', // kanagawa
    '-operandi', '-vivendi',    // modus
  ];

  /**
   * Get the palette manifest from CSS custom property
   */
  function getManifest() {
    const styles = getComputedStyle(document.documentElement);
    let manifest = styles.getPropertyValue('--palette-manifest').trim();

    if (!manifest || manifest === 'none' || manifest === '') {
      return [];
    }

    try {
      if (manifest.startsWith("'") && manifest.endsWith("'")) {
        manifest = manifest.slice(1, -1);
      }
      if (manifest.startsWith('"') && manifest.endsWith('"')) {
        manifest = manifest.slice(1, -1);
      }
      manifest = manifest.replace(/\\'/g, "'");

      const parsed = JSON.parse(manifest);
      return parsed;
    } catch (e) {
      console.warn('[palette-switcher] Failed to parse manifest:', e);
      return [];
    }
  }

  /**
   * Check if switcher is enabled
   */
  function isSwitcherEnabled() {
    const styles = getComputedStyle(document.documentElement);
    const value = styles.getPropertyValue('--palette-switcher-enabled').trim();
    return value === '1' || value === 'true' || value === '"1"';
  }

  /**
   * Compute the family name from a palette name
   */
  function getFamilyName(paletteName) {
    let family = paletteName.toLowerCase();

    // Sort suffixes by length (longest first) to avoid partial matches
    const sortedSuffixes = [...VARIANT_SUFFIXES].sort((a, b) => b.length - a.length);

    for (const suffix of sortedSuffixes) {
      if (family.endsWith(suffix)) {
        family = family.slice(0, -suffix.length);
        break;
      }
    }

    return family;
  }

  /**
   * Get display name for a family
   */
  function getFamilyDisplayName(familyName) {
    // Special case mappings
    const specialNames = {
      'catppuccin': 'Catppuccin',
      'gruvbox': 'Gruvbox',
      'rose-pine': 'Rose Pine',
      'tokyo-night': 'Tokyo Night',
      'nord': 'Nord',
      'dracula': 'Dracula',
      'solarized': 'Solarized',
      'one': 'One',
      'ayu': 'Ayu',
      'everforest': 'Everforest',
      'kanagawa': 'Kanagawa',
      'modus': 'Modus',
      'night-owl': 'Night Owl',
      'whitesur': 'WhiteSur',
      'graphite': 'Graphite',
    };

    if (specialNames[familyName]) {
      return specialNames[familyName];
    }

    // Default: capitalize each word
    return familyName
      .split('-')
      .map(word => word.charAt(0).toUpperCase() + word.slice(1))
      .join(' ');
  }

  /**
   * Group palettes by family
   */
  function groupByFamily(palettes) {
    const families = new Map();

    for (const p of palettes) {
      const family = getFamilyName(p.name);
      if (!families.has(family)) {
        families.set(family, {
          name: family,
          displayName: getFamilyDisplayName(family),
          light: [],
          dark: [],
          all: []
        });
      }

      const group = families.get(family);
      group.all.push(p);

      if (p.variant === 'light') {
        group.light.push(p);
      } else {
        group.dark.push(p);
      }
    }

    return families;
  }

  /**
   * Get the best palette for a family given the mode preference
   */
  function getBestPalette(family, mode) {
    const variants = mode === 'light' ? family.light : family.dark;

    // If we have variants for this mode, use the first one
    if (variants.length > 0) {
      return variants[0];
    }

    // Fallback to any available variant
    return family.all[0];
  }

  /**
   * Get current color mode preference
   */
  function getColorMode() {
    const stored = localStorage.getItem(MODE_KEY);
    if (stored) return stored;

    // Check system preference
    if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
      return 'dark';
    }
    return 'light';
  }

  /**
   * Set the color mode
   */
  function setColorMode(mode) {
    localStorage.setItem(MODE_KEY, mode);

    const root = document.documentElement;
    root.dataset.theme = mode;
    root.classList.toggle('dark', mode === 'dark');

    // Update toggle button
    updateModeToggle(mode);

    // If we have a current family, switch to appropriate variant
    const currentFamily = localStorage.getItem(FAMILY_KEY);
    if (currentFamily) {
      const manifest = getManifest();
      const families = groupByFamily(manifest);
      const family = families.get(currentFamily);

      if (family) {
        const palette = getBestPalette(family, mode);
        if (palette) {
          applyPalette(palette.name);
        }
      }
    }
  }

  /**
   * Apply a palette (without changing mode)
   */
  function applyPalette(paletteName) {
    const root = document.documentElement;
    root.dataset.palette = paletteName;
    localStorage.setItem(STORAGE_KEY, paletteName);

    // Store the family
    const family = getFamilyName(paletteName);
    localStorage.setItem(FAMILY_KEY, family);

    // Update display
    updateFamilyDisplay(family);

    // Dispatch event
    window.dispatchEvent(new CustomEvent('palette-change', {
      detail: { palette: paletteName, family: family }
    }));
  }

  /**
   * Select a family (picks appropriate variant for current mode)
   */
  function selectFamily(familyName, families) {
    const family = families.get(familyName);
    if (!family) return;

    const mode = getColorMode();
    const palette = getBestPalette(family, mode);

    if (palette) {
      applyPalette(palette.name);
    }
  }

  /**
   * Update the mode toggle button display
   */
  function updateModeToggle(mode) {
    const toggle = document.querySelector('.palette-mode-toggle');
    if (toggle) {
      const icon = mode === 'dark' ? '\u263E' : '\u2600'; // moon/sun
      const label = mode === 'dark' ? 'Dark' : 'Light';
      toggle.innerHTML = `<span class="mode-icon">${icon}</span>`;
      toggle.title = `${label} mode - click to toggle`;
      toggle.setAttribute('aria-label', `Current: ${label} mode. Click to switch.`);
    }
  }

  /**
   * Update the family dropdown display
   */
  function updateFamilyDisplay(familyName) {
    const btn = document.querySelector('.palette-family-btn span');
    if (btn) {
      btn.textContent = getFamilyDisplayName(familyName);
    }

    // Update active state in dropdown
    document.querySelectorAll('.palette-family-option').forEach(opt => {
      opt.classList.toggle('active', opt.dataset.family === familyName);
    });
  }

  /**
   * Enhance an existing HTML-based theme switcher
   */
  function enhanceExistingSwitcher(container, families) {
    // Wire up the existing mode toggle
    const modeToggle = container.querySelector('.palette-mode-toggle');
    if (modeToggle) {
      modeToggle.addEventListener('click', () => {
        const current = getColorMode();
        setColorMode(current === 'dark' ? 'light' : 'dark');
      });
    }

    // Populate the palette family dropdown if it exists
    const familyList = container.querySelector('.palette-family-list');
    const familyBtn = container.querySelector('.palette-family-btn');
    const dropdown = container.querySelector('.palette-family-dropdown');

    if (familyList && familyBtn && dropdown) {
      // Sort families alphabetically
      const sortedFamilies = Array.from(families.entries()).sort((a, b) =>
        a[1].displayName.localeCompare(b[1].displayName)
      );

      // Populate family list
      for (const [familyName, family] of sortedFamilies) {
        const option = document.createElement('button');
        option.className = 'palette-family-option';
        option.type = 'button';
        option.dataset.family = familyName;
        option.setAttribute('role', 'option');

        const hasLight = family.light.length > 0;
        const hasDark = family.dark.length > 0;
        let variantIndicator = '';
        if (hasLight && hasDark) {
          variantIndicator = '<span class="variant-indicator both">\u2600\u263E</span>';
        } else if (hasLight) {
          variantIndicator = '<span class="variant-indicator light">\u2600</span>';
        } else {
          variantIndicator = '<span class="variant-indicator dark">\u263E</span>';
        }

        option.innerHTML = '<span class="family-name">' + family.displayName + '</span>' + variantIndicator;

        option.addEventListener('click', () => {
          selectFamily(familyName, families);
          dropdown.hidden = true;
          familyBtn.setAttribute('aria-expanded', 'false');
        });

        familyList.appendChild(option);
      }

      // Toggle dropdown
      familyBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        const isOpen = !dropdown.hidden;
        dropdown.hidden = isOpen;
        familyBtn.setAttribute('aria-expanded', !isOpen);
        if (!isOpen) {
          const searchInput = dropdown.querySelector('input');
          if (searchInput) searchInput.focus();
        }
      });

      // Search functionality
      const searchInput = dropdown.querySelector('.palette-search input');
      if (searchInput) {
        searchInput.addEventListener('input', (e) => {
          const query = e.target.value.toLowerCase();
          familyList.querySelectorAll('.palette-family-option').forEach(opt => {
            const name = opt.querySelector('.family-name').textContent.toLowerCase();
            opt.hidden = !name.includes(query);
          });
        });
      }

      // Close on outside click
      document.addEventListener('click', (e) => {
        if (!container.contains(e.target)) {
          dropdown.hidden = true;
          familyBtn.setAttribute('aria-expanded', 'false');
        }
      });

      // Close on Escape
      document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && !dropdown.hidden) {
          dropdown.hidden = true;
          familyBtn.setAttribute('aria-expanded', 'false');
        }
      });

      // Store sorted family names for cycling
      sortedFamilyNames = sortedFamilies.map(([name]) => name);
    }

    // Initialize state
    const mode = getColorMode();
    updateModeToggle(mode);

    // Apply saved palette or default
    const savedPalette = localStorage.getItem(STORAGE_KEY);
    const savedFamily = localStorage.getItem(FAMILY_KEY);

    if (savedPalette) {
      applyPalette(savedPalette);
      const manifest2 = getManifest();
      const savedP = manifest2.find(p => p.name === savedPalette);
      if (savedP && savedP.variant !== mode) {
        setColorMode(savedP.variant);
      }
    } else if (savedFamily && families.has(savedFamily)) {
      selectFamily(savedFamily, families);
    } else {
      // Default to first available family
      const sortedFamilies = Array.from(families.entries()).sort((a, b) =>
        a[1].displayName.localeCompare(b[1].displayName)
      );
      if (sortedFamilies.length > 0) {
        selectFamily(sortedFamilies[0][0], families);
      }
    }
  }

  /**
   * Create the palette switcher UI
   */
  function createSwitcher() {
    if (!isSwitcherEnabled()) {
      return;
    }

    const manifest = getManifest();
    if (manifest.length === 0) {
      return;
    }

    const families = groupByFamily(manifest);

    // Check if we have an existing theme-switcher from HTML template
    const existingSwitcher = document.querySelector('.theme-switcher');
    if (existingSwitcher) {
      // Enhance existing switcher instead of creating new one
      enhanceExistingSwitcher(existingSwitcher, families);
      return;
    }

    // Create new container (fallback if no HTML template)
    const container = document.createElement('div');
    container.className = 'palette-switcher';

    // Mode toggle button
    const modeToggle = document.createElement('button');
    modeToggle.className = 'palette-mode-toggle';
    modeToggle.type = 'button';
    modeToggle.addEventListener('click', () => {
      const current = getColorMode();
      setColorMode(current === 'dark' ? 'light' : 'dark');
    });
    container.appendChild(modeToggle);

    // Family selector
    const familyWrapper = document.createElement('div');
    familyWrapper.className = 'palette-family-wrapper';

    const familyBtn = document.createElement('button');
    familyBtn.className = 'palette-family-btn';
    familyBtn.type = 'button';
    familyBtn.setAttribute('aria-haspopup', 'listbox');
    familyBtn.setAttribute('aria-expanded', 'false');
    familyBtn.innerHTML = `<span>Theme</span> <span class="palette-arrow">\u25BC</span>`;

    const dropdown = document.createElement('div');
    dropdown.className = 'palette-family-dropdown';
    dropdown.setAttribute('role', 'listbox');
    dropdown.hidden = true;

    // Search input
    const searchDiv = document.createElement('div');
    searchDiv.className = 'palette-search';
    searchDiv.innerHTML = `<input type="text" placeholder="Search..." aria-label="Search palettes">`;
    dropdown.appendChild(searchDiv);

    // Family list
    const list = document.createElement('div');
    list.className = 'palette-family-list';

    // Sort families alphabetically
    const sortedFamilies = Array.from(families.entries()).sort((a, b) =>
      a[1].displayName.localeCompare(b[1].displayName)
    );

    for (const [familyName, family] of sortedFamilies) {
      const option = document.createElement('button');
      option.className = 'palette-family-option';
      option.type = 'button';
      option.dataset.family = familyName;
      option.setAttribute('role', 'option');

      // Show available variants
      const hasLight = family.light.length > 0;
      const hasDark = family.dark.length > 0;
      let variantIndicator = '';
      if (hasLight && hasDark) {
        variantIndicator = `<span class="variant-indicator both">\u2600\u263E</span>`;
      } else if (hasLight) {
        variantIndicator = `<span class="variant-indicator light">\u2600</span>`;
      } else {
        variantIndicator = `<span class="variant-indicator dark">\u263E</span>`;
      }

      option.innerHTML = `
        <span class="family-name">${family.displayName}</span>
        ${variantIndicator}
      `;

      option.addEventListener('click', () => {
        selectFamily(familyName, families);
        closeDropdown();
      });

      list.appendChild(option);
    }

    dropdown.appendChild(list);
    familyWrapper.appendChild(familyBtn);
    familyWrapper.appendChild(dropdown);
    container.appendChild(familyWrapper);

    // Toggle dropdown
    familyBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      if (dropdown.hidden) {
        openDropdown();
      } else {
        closeDropdown();
      }
    });

    // Search functionality
    const searchInput = searchDiv.querySelector('input');
    searchInput.addEventListener('input', (e) => {
      const query = e.target.value.toLowerCase();
      list.querySelectorAll('.palette-family-option').forEach(opt => {
        const name = opt.querySelector('.family-name').textContent.toLowerCase();
        opt.hidden = !name.includes(query);
      });
    });

    // Close handlers
    document.addEventListener('click', (e) => {
      if (!container.contains(e.target)) {
        closeDropdown();
      }
    });

    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        closeDropdown();
      }
    });

    function openDropdown() {
      dropdown.hidden = false;
      familyBtn.setAttribute('aria-expanded', 'true');
      searchInput.focus();
    }

    function closeDropdown() {
      dropdown.hidden = true;
      familyBtn.setAttribute('aria-expanded', 'false');
    }

    // Insert into page
    insertSwitcher(container);

    // Initialize state
    const mode = getColorMode();
    updateModeToggle(mode);

    // Apply saved palette or default
    const savedPalette = localStorage.getItem(STORAGE_KEY);
    const savedFamily = localStorage.getItem(FAMILY_KEY);

    if (savedPalette) {
      applyPalette(savedPalette);
      // Make sure mode matches the saved palette
      const manifest2 = getManifest();
      const savedP = manifest2.find(p => p.name === savedPalette);
      if (savedP && savedP.variant !== mode) {
        // Update mode to match saved palette
        setColorMode(savedP.variant);
      }
    } else if (savedFamily && families.has(savedFamily)) {
      selectFamily(savedFamily, families);
    } else {
      // Default to first available family
      const firstFamily = sortedFamilies[0];
      if (firstFamily) {
        selectFamily(firstFamily[0], families);
      }
    }
  }

  /**
   * Insert switcher into page
   */
  function insertSwitcher(container) {
    const headerNav = document.querySelector('.site-header .site-nav');
    if (headerNav) {
      headerNav.appendChild(container);
      return;
    }

    const nav = document.querySelector('nav');
    if (nav) {
      nav.appendChild(container);
      return;
    }

    // Fallback: fixed position
    container.classList.add('palette-switcher-fixed');
    document.body.appendChild(container);
  }

  /**
   * Wait for stylesheets to load
   */
  function waitForStylesheets(callback, maxAttempts = 20) {
    let attempts = 0;

    function check() {
      attempts++;
      const styles = getComputedStyle(document.documentElement);
      const manifest = styles.getPropertyValue('--palette-manifest').trim();

      if (manifest && manifest.length > 10) {
        callback();
      } else if (attempts < maxAttempts) {
        setTimeout(check, 100);
      } else {
        callback();
      }
    }

    check();
  }

  function init() {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        waitForStylesheets(() => {
          createSwitcher();
        });
      });
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  // Store sorted families for cycling
  let sortedFamilyNames = [];

  /**
   * Get the aesthetic manifest from CSS custom property
   */
  function getAestheticManifest() {
    const styles = getComputedStyle(document.documentElement);
    let manifest = styles.getPropertyValue('--aesthetic-manifest').trim();

    if (!manifest || manifest === 'none' || manifest === '') {
      return [];
    }

    try {
      if (manifest.startsWith("'") && manifest.endsWith("'")) {
        manifest = manifest.slice(1, -1);
      }
      if (manifest.startsWith('"') && manifest.endsWith('"')) {
        manifest = manifest.slice(1, -1);
      }
      manifest = manifest.replace(/\\'/g, "'");

      return JSON.parse(manifest);
    } catch (e) {
      console.warn('[palette-switcher] Failed to parse aesthetic manifest:', e);
      return [];
    }
  }

  function getDefaultAesthetic() {
    const styles = getComputedStyle(document.documentElement);
    let def = styles.getPropertyValue('--aesthetic-default').trim();
    if (def.startsWith("'") || def.startsWith('"')) {
      def = def.slice(1, -1);
    }
    return def || 'balanced';
  }

  /**
   * Get the current aesthetic
   */
  function getAesthetic() {
    const stored = localStorage.getItem(AESTHETIC_KEY);
    const manifest = getAestheticManifest();
    const manifestNames = manifest.map(m => m.name);

    if (AESTHETICS.length === 0) {
      AESTHETICS = manifestNames;
    }

    if (stored && manifestNames.includes(stored)) {
      return stored;
    }
    return getDefaultAesthetic();
  }

  /**
   * Set the aesthetic and apply CSS properties
   */
  function setAesthetic(aesthetic) {
    if (AESTHETICS.length === 0) {
      AESTHETICS = getAestheticManifest().map(m => m.name);
    }

    if (!AESTHETICS.includes(aesthetic)) {
      console.warn('[palette-switcher] Unknown aesthetic:', aesthetic);
      return;
    }

    const root = document.documentElement;

    // Set data attribute for CSS selectors (aesthetic.css relies on this)
    root.dataset.aesthetic = aesthetic;

    // Persist selection
    localStorage.setItem(AESTHETIC_KEY, aesthetic);

    // Update the dropdown if it exists
    const select = document.getElementById('aesthetic-select');
    if (select && select.value !== aesthetic) {
      select.value = aesthetic;
    }

    // Dispatch event
    window.dispatchEvent(new CustomEvent('aesthetic-change', {
      detail: { aesthetic: aesthetic }
    }));
  }

  /**
   * Cycle to next/previous aesthetic
   */
  function cycleAesthetic(direction) {
    if (AESTHETICS.length === 0) {
      AESTHETICS = getAestheticManifest().map(m => m.name);
    }
    if (AESTHETICS.length === 0) return;

    const currentAesthetic = getAesthetic();
    const currentIndex = AESTHETICS.indexOf(currentAesthetic);

    let newIndex;
    if (currentIndex === -1) {
      newIndex = 0;
    } else if (direction === 'next') {
      newIndex = (currentIndex + 1) % AESTHETICS.length;
    } else {
      newIndex = (currentIndex - 1 + AESTHETICS.length) % AESTHETICS.length;
    }

    const newAesthetic = AESTHETICS[newIndex];
    setAesthetic(newAesthetic);

    // Show notification with capitalized name
    const displayName = newAesthetic.charAt(0).toUpperCase() + newAesthetic.slice(1);
    showNotification(`Aesthetic: ${displayName}`);
  }

  /**
   * Initialize aesthetic on page load
   */
  function initAesthetic() {
    const aesthetic = getAesthetic();
    setAesthetic(aesthetic);
  }

  // Initialize aesthetic immediately
  initAesthetic();

  /**
   * Cycle to next/previous family
   */
  function cycleFamily(direction) {
    if (sortedFamilyNames.length === 0) {
      const manifest = getManifest();
      const families = groupByFamily(manifest);
      sortedFamilyNames = Array.from(families.keys()).sort((a, b) =>
        getFamilyDisplayName(a).localeCompare(getFamilyDisplayName(b))
      );
    }

    const currentFamily = localStorage.getItem(FAMILY_KEY) || sortedFamilyNames[0];
    const currentIndex = sortedFamilyNames.indexOf(currentFamily);

    let newIndex;
    if (direction === 'next') {
      newIndex = (currentIndex + 1) % sortedFamilyNames.length;
    } else {
      newIndex = (currentIndex - 1 + sortedFamilyNames.length) % sortedFamilyNames.length;
    }

    const manifest = getManifest();
    const families = groupByFamily(manifest);
    selectFamily(sortedFamilyNames[newIndex], families);

    // Show brief notification
    showNotification(getFamilyDisplayName(sortedFamilyNames[newIndex]));
  }

  /**
   * Show a brief notification
   */
  function showNotification(text) {
    let notif = document.querySelector('.palette-notification');
    if (!notif) {
      notif = document.createElement('div');
      notif.className = 'palette-notification';
      document.body.appendChild(notif);
    }

    notif.textContent = text;
    notif.classList.add('visible');

    clearTimeout(notif._timeout);
    notif._timeout = setTimeout(() => {
      notif.classList.remove('visible');
    }, 1000);
  }

  /**
   * Setup keyboard shortcuts using the registry
   */
  function setupKeyboardShortcuts() {
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

    waitForRegistry(function() {
      // [ = previous family
      window.shortcutsRegistry.register({
        key: '[',
        modifiers: [],
        description: 'Previous palette',
        group: 'theme',
        handler: function(e) {
          e.preventDefault();
          cycleFamily('prev');
        },
        priority: 50
      });

      // ] = next family
      window.shortcutsRegistry.register({
        key: ']',
        modifiers: [],
        description: 'Next palette',
        group: 'theme',
        handler: function(e) {
          e.preventDefault();
          cycleFamily('next');
        },
        priority: 50
      });

      // { (Shift+[) = previous aesthetic
      window.shortcutsRegistry.register({
        key: '{',
        modifiers: [],
        description: 'Previous aesthetic',
        group: 'theme',
        handler: function(e) {
          e.preventDefault();
          cycleAesthetic('prev');
        },
        priority: 50
      });

      // } (Shift+]) = next aesthetic
      window.shortcutsRegistry.register({
        key: '}',
        modifiers: [],
        description: 'Next aesthetic',
        group: 'theme',
        handler: function(e) {
          e.preventDefault();
          cycleAesthetic('next');
        },
        priority: 50
      });

      // \ = toggle dark/light
      window.shortcutsRegistry.register({
        key: '\\',
        modifiers: [],
        description: 'Toggle dark/light mode',
        group: 'theme',
        handler: function(e) {
          e.preventDefault();
          const current = getColorMode();
          setColorMode(current === 'dark' ? 'light' : 'dark');
          showNotification(current === 'dark' ? 'Light Mode' : 'Dark Mode');
        },
        priority: 50
      });
    });
  }

  // Initialize keyboard shortcuts
  setupKeyboardShortcuts();

  // Expose API
  window.markata = window.markata || {};
  window.markata.paletteSwitcher = {
    setColorMode,
    getColorMode,
    applyPalette,
    getManifest,
    cycleFamily,
    nextFamily: () => cycleFamily('next'),
    prevFamily: () => cycleFamily('prev'),
    getAesthetic,
    setAesthetic,
    cycleAesthetic,
    nextAesthetic: () => cycleAesthetic('next'),
    prevAesthetic: () => cycleAesthetic('prev'),
  };
})();
