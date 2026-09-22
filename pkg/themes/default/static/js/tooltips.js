/**
 * Link Hover Tooltips
 * Shows an instant popover with title, description, and date when hovering
 * over wikilinks, internal links enriched with data-title by the
 * wikilink_hover plugin, and glossary terms (data-preview="glossary").
 */
(function() {
  'use strict';

  var HIDE_GRACE_MS = 140;

  let tooltip = null;
  let hideTimer = null;
  let boundLinks = new WeakSet();  // Track which links have listeners

  function cancelHide() {
    if (hideTimer) {
      clearTimeout(hideTimer);
      hideTimer = null;
    }
  }

  function createTooltip(link) {
    cancelHide();
    // Always clean up any existing tooltip first
    removeTooltip();

    var isGlossary = link.dataset.preview === 'glossary';

    tooltip = document.createElement('div');
    tooltip.className = 'wikilink-tooltip' + (isGlossary ? ' wikilink-tooltip--glossary' : '');
    tooltip.setAttribute('role', 'tooltip');
    function row(cls, text, tag) {
      if (!text) return;
      var el = document.createElement(tag || 'div');
      el.className = cls;
      el.textContent = text;
      tooltip.appendChild(el);
    }
    if (isGlossary) {
      row('tooltip-eyebrow', link.dataset.eyebrow || 'Glossary');
    }
    row('tooltip-title', link.dataset.title || '');
    row('tooltip-desc', link.dataset.description || '');
    row('tooltip-date', link.dataset.date || '');
    if (isGlossary && link.getAttribute('href')) {
      var more = document.createElement('a');
      more.className = 'tooltip-more';
      more.href = link.getAttribute('href');
      more.textContent = 'Read the full definition \u2192';
      tooltip.appendChild(more);
    }

    // Let the pointer travel into the card without dismissing it.
    tooltip.addEventListener('mouseenter', cancelHide);
    tooltip.addEventListener('mouseleave', scheduleHide);

    document.body.appendChild(tooltip);
    positionTooltip(link);
  }

  function positionTooltip(link) {
    if (!tooltip) return;
    var rect = link.getBoundingClientRect();
    var width = tooltip.offsetWidth || 300;
    var height = tooltip.offsetHeight || 0;
    var left = Math.max(8, Math.min(rect.left, window.innerWidth - width - 8));
    var top = rect.bottom + 8;
    if (top + height > window.innerHeight - 8) {
      top = Math.max(8, rect.top - height - 8);
      tooltip.classList.add('wikilink-tooltip--above');
    }
    tooltip.style.left = left + 'px';
    tooltip.style.top = top + 'px';
  }

  function removeTooltip() {
    cancelHide();
    if (tooltip) {
      tooltip.remove();
      tooltip = null;
    }
  }

  function scheduleHide() {
    cancelHide();
    hideTimer = setTimeout(removeTooltip, HIDE_GRACE_MS);
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

    var links = document.querySelectorAll('.wikilink[data-title], a[data-preview][data-title]');
    links.forEach(function(link) {
      // Skip if already bound (shouldn't happen after cleanup, but defensive)
      if (boundLinks.has(link)) return;

      // The title attribute is only a no-JS fallback; drop it so the slow
      // native browser tooltip does not double up with the popover.
      if (link.hasAttribute('title') && link.dataset.description) {
        link.removeAttribute('title');
      }

      link.addEventListener('mouseenter', function() { createTooltip(link); });
      link.addEventListener('mouseleave', scheduleHide);
      link.addEventListener('focus', function() { createTooltip(link); });
      link.addEventListener('blur', scheduleHide);
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
