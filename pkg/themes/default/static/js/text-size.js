/**
 * Visitor-controlled reading-size preference.
 *
 * The configured theme value is the fallback. A valid local preference wins
 * and is scoped to the current site origin.
 */
(function() {
  'use strict';

  const STORAGE_KEY = 'text-size';
  const FONT_STORAGE_KEY = 'reading-font';
  const SIZES = ['small', 'medium', 'large', 'x-large'];
  const FONTS = ['sans', 'serif'];

  function isValidSize(value) {
    return SIZES.includes(value);
  }

  function getDefaultSize() {
    const configured = window.__markataTextSizeDefault || document.documentElement.dataset.textSize;
    return isValidSize(configured) ? configured : 'large';
  }

  function getStoredSize() {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      return isValidSize(stored) ? stored : null;
    } catch (_) {
      return null;
    }
  }

  function syncControls(size) {
    document.querySelectorAll('[data-text-size-control]').forEach(function(control) {
      control.value = size;
    });
  }

  function applySize(size, persist) {
    if (!isValidSize(size)) {
      return;
    }

    document.documentElement.dataset.textSize = size;
    if (persist) {
      try {
        localStorage.setItem(STORAGE_KEY, size);
      } catch (_) {
        // Ignore storage access failures; the current page still updates.
      }
    }
    syncControls(size);
  }

  function isValidFont(value) {
    return FONTS.includes(value);
  }

  function getDefaultFont() {
    const configured = document.documentElement.dataset.readingFont;
    return isValidFont(configured) ? configured : 'sans';
  }

  function getStoredFont() {
    try {
      const stored = localStorage.getItem(FONT_STORAGE_KEY);
      return isValidFont(stored) ? stored : null;
    } catch (_) {
      return null;
    }
  }

  function syncFontControls(font) {
    document.querySelectorAll('[data-reading-font-toggle]').forEach(function(control) {
      control.setAttribute('aria-pressed', font === 'serif' ? 'true' : 'false');
      control.dataset.readingFontValue = font;
    });
  }

  function applyFont(font, persist) {
    if (!isValidFont(font)) {
      return;
    }
    document.documentElement.dataset.readingFont = font;
    if (persist) {
      try {
        localStorage.setItem(FONT_STORAGE_KEY, font);
      } catch (_) {
        // Ignore storage access failures; the current page still updates.
      }
    }
    syncFontControls(font);
  }

  function currentFont() {
    return document.documentElement.dataset.readingFont || getDefaultFont();
  }

  function init() {
    const size = getStoredSize() || getDefaultSize();
    applySize(size, false);
    applyFont(getStoredFont() || getDefaultFont(), false);

    document.querySelectorAll('[data-reading-font-toggle]').forEach(function(control) {
      if (control.dataset.readingFontBound === 'true') {
        return;
      }
      control.dataset.readingFontBound = 'true';
      control.addEventListener('click', function() {
        const next = currentFont() === 'serif' ? 'sans' : 'serif';
        applyFont(next, true);
        window.dispatchEvent(new CustomEvent('reading-font-change', {
          detail: { font: next }
        }));
      });
    });

    document.querySelectorAll('[data-text-size-control]').forEach(function(control) {
      if (control.dataset.textSizeBound === 'true') {
        return;
      }
      control.dataset.textSizeBound = 'true';
      control.addEventListener('change', function() {
        applySize(control.value, true);
        window.dispatchEvent(new CustomEvent('text-size-change', {
          detail: { size: control.value }
        }));
      });
    });
  }

  window.MarkataTextSize = {
    get: function() {
      return document.documentElement.dataset.textSize || getDefaultSize();
    },
    set: function(size) {
      applySize(size, true);
    },
    sizes: SIZES.slice(),
    getFont: currentFont,
    setFont: function(font) {
      applyFont(font, true);
    },
    fonts: FONTS.slice()
  };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init, { once: true });
  } else {
    init();
  }
  window.addEventListener('view-transition-complete', init);
})();
