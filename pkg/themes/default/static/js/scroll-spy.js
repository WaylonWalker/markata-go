/**
 * Scroll Spy for Table of Contents
 * Highlights the current section in the TOC as user scrolls
 */

(function() {
  'use strict';

  const tocLinks = document.querySelectorAll('.toc-link');
  if (!tocLinks.length) return;

  // Get all heading IDs from TOC links
  const headingIds = Array.from(tocLinks).map(link => {
    const href = link.getAttribute('href');
    return href ? href.substring(1) : null;
  }).filter(Boolean);

  // Get corresponding heading elements
  const headings = headingIds.map(id => document.getElementById(id)).filter(Boolean);

  if (!headings.length) return;

  const ACTIVE_CLASS = 'toc-link--active';
  const OFFSET = 100; // Offset from top of viewport

  let ticking = false;

  /**
   * Get the current active heading based on scroll position
   * @returns {Element|null} The currently visible heading
   */
  function getCurrentHeading() {
    const scrollTop = window.scrollY;
    let current = null;

    for (const heading of headings) {
      const rect = heading.getBoundingClientRect();
      const offsetTop = rect.top + scrollTop;

      if (offsetTop - OFFSET <= scrollTop) {
        current = heading;
      } else {
        break;
      }
    }

    return current;
  }

  /**
   * Update TOC link active states
   */
  function updateActiveLink() {
    const currentHeading = getCurrentHeading();

    tocLinks.forEach(link => {
      const href = link.getAttribute('href');
      const isActive = currentHeading && href === `#${currentHeading.id}`;

      link.classList.toggle(ACTIVE_CLASS, isActive);

      // Also highlight parent items if nested
      const parentItem = link.closest('.toc-item');
      if (parentItem) {
        const parentList = parentItem.closest('.toc-list--nested');
        if (parentList) {
          const parentLink = parentList.previousElementSibling;
          if (parentLink && parentLink.classList.contains('toc-link')) {
            // Keep parent highlighted if any child is active
            const hasActiveChild = parentList.querySelector(`.${ACTIVE_CLASS}`);
            parentLink.classList.toggle(ACTIVE_CLASS, isActive || hasActiveChild);
          }
        }
      }
    });

    ticking = false;
  }

  /**
   * Request animation frame for scroll handler
   */
  function onScroll() {
    if (!ticking) {
      requestAnimationFrame(updateActiveLink);
      ticking = true;
    }
  }

  // Listen for scroll events
  window.addEventListener('scroll', onScroll, { passive: true });

  // Initial update
  updateActiveLink();

  // Smooth scroll to heading when clicking TOC links
  tocLinks.forEach(link => {
    link.addEventListener('click', function(e) {
      const href = this.getAttribute('href');
      if (!href || !href.startsWith('#')) return;

      const target = document.getElementById(href.substring(1));
      if (!target) return;

      e.preventDefault();

      // Calculate scroll position with offset
      const offsetTop = target.getBoundingClientRect().top + window.scrollY - OFFSET + 20;

      window.scrollTo({
        top: offsetTop,
        behavior: 'smooth'
      });

      // Update URL hash without jumping
      history.pushState(null, '', href);

      // Update active state immediately
      setTimeout(updateActiveLink, 100);
    });
  });
})();
