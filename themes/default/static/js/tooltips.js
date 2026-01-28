/**
 * Wikilink Hover Tooltips
 * Shows a tooltip with title, description, and date when hovering over wikilinks.
 */
(function() {
  'use strict';

  let tooltip = null;

  function createTooltip(link) {
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

  function init() {
    var links = document.querySelectorAll('.wikilink[data-title]');
    links.forEach(function(link) {
      link.addEventListener('mouseenter', function() { createTooltip(link); });
      link.addEventListener('mouseleave', removeTooltip);
    });
  }

  // Initialize immediately if DOM is ready, otherwise wait
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
