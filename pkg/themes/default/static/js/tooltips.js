document.addEventListener('DOMContentLoaded', () => {
  const links = document.querySelectorAll('.wikilink[data-title]');
  let tooltip = null;

  function createTooltip(link) {
    tooltip = document.createElement('div');
    tooltip.className = 'wikilink-tooltip';
    tooltip.innerHTML = `
      <div class="tooltip-title">${link.dataset.title}</div>
      ${link.dataset.description ? `<div class="tooltip-desc">${link.dataset.description}</div>` : ''}
      ${link.dataset.date ? `<div class="tooltip-date">${link.dataset.date}</div>` : ''}
    `;
    document.body.appendChild(tooltip);
    positionTooltip(link);
  }

  function positionTooltip(link) {
    const rect = link.getBoundingClientRect();
    tooltip.style.left = rect.left + 'px';
    tooltip.style.top = (rect.bottom + 8) + 'px';
  }

  function removeTooltip() {
    if (tooltip) {
      tooltip.remove();
      tooltip = null;
    }
  }

  links.forEach(link => {
    link.addEventListener('mouseenter', () => createTooltip(link));
    link.addEventListener('mouseleave', removeTooltip);
  });
});
