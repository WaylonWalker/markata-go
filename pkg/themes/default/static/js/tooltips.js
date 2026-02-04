/**
 * Wikilink Hover Tooltips
 * Shows a tooltip with title, description, and date when hovering over wikilinks.
 */
(function() {
  'use strict';

  let tooltip = null;
  let boundLinks = new WeakSet();  // Track which links have listeners

  function createTooltip(link) {
    // Always clean up any existing tooltip first
    removeTooltip();

    tooltip = document.createElement('div');
    tooltip.className = 'wikilink-tooltip';
    tooltip.innerHTML =
      '<div class="tooltip-title">' + (link.dataset.title || '') + '</div>' +
      (link.dataset.description ? '<div class="tooltip-desc">' + link.dataset.description + '</div>' : '') +
      (link.dataset.date ? '<div class="tooltip-date">' + link.dataset.date + '</div>' : '');
    document.body.appendChild(tooltip);
    positionTooltip(link);
  }

  function positionTooltip(link) {
    if (!tooltip) return;
    var rect = link.getBoundingClientRect();
    tooltip.style.left = rect.left + 'px';
    tooltip.style.top = (rect.bottom + 8) + 'px';
  }

  function removeTooltip() {
    if (tooltip) {
      tooltip.remove();
      tooltip = null;
    }
  }

  /**
   * Clean up before re-initialization (for view transitions)
   */
  function cleanup() {
    // Remove any existing tooltip
    removeTooltip();

    // Also remove any orphaned tooltips that might be left in the DOM
    document.querySelectorAll('.wikilink-tooltip').forEach(function(el) {
      el.remove();
    });

    // Reset the WeakSet - old DOM elements are gone after view transition
    boundLinks = new WeakSet();
  }

  function init() {
    // Clean up first to handle view transitions properly
    cleanup();

    var links = document.querySelectorAll('.wikilink[data-title]');
    links.forEach(function(link) {
      // Skip if already bound (shouldn't happen after cleanup, but defensive)
      if (boundLinks.has(link)) return;

      link.addEventListener('mouseenter', function() { createTooltip(link); });
      link.addEventListener('mouseleave', removeTooltip);
      boundLinks.add(link);
    });
  }

  // Initialize immediately if DOM is ready, otherwise wait
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  // Expose for view transitions to re-initialize
  window.initTooltips = init;

  // Re-initialize after view transitions
  window.addEventListener('view-transition-complete', init);
})();
