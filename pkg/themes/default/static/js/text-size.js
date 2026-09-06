/**
 * Visitor-controlled reading-size preference.
 *
 * The configured theme value is the fallback. A valid local preference wins
 * and is scoped to the current site origin.
 */
(function() {
  'use strict';

  const STORAGE_KEY = 'text-size';
  const SIZES = ['small', 'medium', 'large'];

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

  function init() {
    const size = getStoredSize() || getDefaultSize();
    applySize(size, false);

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
    sizes: SIZES.slice()
  };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init, { once: true });
  } else {
    init();
  }
  window.addEventListener('view-transition-complete', init);
})();
