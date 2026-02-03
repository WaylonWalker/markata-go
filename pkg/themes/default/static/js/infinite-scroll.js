/**
 * Infinite scroll pagination for markata-go
 * Supports pagination_type: "htmx-infinite"
 *
 * This script works alongside HTMX to provide infinite scroll functionality.
 * It updates the pagination trigger element after each page load to point
 * to the next page, continuing until all pages are loaded.
 */
(function() {
  'use strict';

  // Initialize when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initInfiniteScroll);
  } else {
    initInfiniteScroll();
  }

  // Expose for view transitions to re-initialize
  window.initInfiniteScroll = initInfiniteScroll;

  // Re-initialize after view transitions
  window.addEventListener('view-transition-complete', initInfiniteScroll);

  function initInfiniteScroll() {
    const paginationContainer = document.getElementById('pagination-infinite');
    if (!paginationContainer) return;

    // Get initial state
    let currentPage = parseInt(paginationContainer.dataset.currentPage, 10) || 1;
    const totalPages = parseInt(paginationContainer.dataset.totalPages, 10) || 1;

    // Listen for HTMX after-swap to update the trigger for the next page
    document.body.addEventListener('htmx:afterSwap', function(event) {
      // Only handle swaps into the posts list
      if (!event.detail.target.classList.contains('posts-list')) return;

      currentPage++;

      // Check if there are more pages
      if (currentPage >= totalPages) {
        // No more pages - show end message
        showEndMessage(paginationContainer);
      } else {
        // Update trigger for next page
        updateTrigger(paginationContainer, currentPage, totalPages);
      }
    });
  }

  function updateTrigger(container, currentPage, totalPages) {
    const trigger = container.querySelector('.infinite-scroll-trigger');
    if (!trigger) return;

    // Calculate the next page URL
    // URLs follow the pattern: /feed/ for page 1, /feed/page/N/ for page N
    const currentUrl = window.location.pathname;
    const baseUrl = currentUrl.replace(/\/page\/\d+\/?$/, '').replace(/\/$/, '');
    const nextPageNum = currentPage + 1;
    const nextUrl = baseUrl + '/page/' + nextPageNum + '/';

    // Update the trigger's HTMX attributes
    trigger.setAttribute('hx-get', nextUrl);

    // Re-process the element so HTMX picks up the new URL
    if (window.htmx) {
      htmx.process(trigger);
    }

    // Update data attributes
    container.dataset.currentPage = currentPage;
    container.dataset.nextUrl = nextUrl;
    container.dataset.hasNext = (nextPageNum <= totalPages).toString();
  }

  function showEndMessage(container) {
    // Replace loading indicator with end message
    const loading = container.querySelector('.infinite-scroll-loading');
    const trigger = container.querySelector('.infinite-scroll-trigger');

    if (trigger) {
      trigger.remove();
    }

    if (loading) {
      loading.style.display = 'none';
    }

    // Check if end message already exists
    if (!container.querySelector('.infinite-scroll-end')) {
      const endMessage = document.createElement('div');
      endMessage.className = 'infinite-scroll-end';
      endMessage.innerHTML = '<span>You\'ve reached the end</span>';
      container.appendChild(endMessage);
    }

    container.dataset.hasNext = 'false';
  }
})();
