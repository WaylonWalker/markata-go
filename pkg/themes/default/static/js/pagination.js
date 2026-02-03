/**
 * Client-side pagination for markata-go
 * Supports pagination_type: "js"
 */
(function() {
  'use strict';

  // Initialize pagination when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initPagination);
  } else {
    initPagination();
  }

  // Expose for view transitions to re-initialize
  window.initPagination = initPagination;

  // Re-initialize after view transitions
  window.addEventListener('view-transition-complete', initPagination);

  function initPagination() {
    const paginationNav = document.querySelector('.pagination-js');
    if (!paginationNav) return;

    const feedSlug = paginationNav.dataset.feedSlug || '';
    const itemsPerPage = parseInt(paginationNav.dataset.itemsPerPage, 10) || 10;
    const totalPages = parseInt(paginationNav.dataset.totalPages, 10) || 1;

    // Get current page from URL hash or default to 1
    let currentPage = getPageFromHash() || 1;

    const postsList = document.querySelector('.posts-list');
    const allPosts = postsList ? Array.from(postsList.querySelectorAll('.card')) : [];

    if (allPosts.length === 0) return;

    // Store all posts and create pagination state
    const state = {
      feedSlug,
      itemsPerPage,
      totalPages: Math.ceil(allPosts.length / itemsPerPage),
      currentPage,
      allPosts,
      postsList
    };

    // Initial render
    renderPage(state);
    renderPaginationControls(paginationNav, state);
    setupEventListeners(paginationNav, state);

    // Handle browser back/forward
    window.addEventListener('hashchange', function() {
      state.currentPage = getPageFromHash() || 1;
      renderPage(state);
      renderPaginationControls(paginationNav, state);
    });
  }

  function getPageFromHash() {
    const hash = window.location.hash;
    const match = hash.match(/page=(\d+)/);
    return match ? parseInt(match[1], 10) : null;
  }

  function setPageHash(page) {
    if (page === 1) {
      history.pushState(null, '', window.location.pathname);
    } else {
      history.pushState(null, '', '#page=' + page);
    }
  }

  /**
   * Navigate to a page with View Transition API support
   * Falls back to immediate update if View Transitions not supported
   */
  function navigateToPage(page, state, nav) {
    var updatePage = function() {
      state.currentPage = page;
      setPageHash(page);
      renderPage(state);
      renderPaginationControls(nav, state);
    };

    // Use View Transitions if available
    if (document.startViewTransition) {
      document.startViewTransition(updatePage);
    } else {
      updatePage();
    }
  }

  function renderPage(state) {
    const { allPosts, postsList, currentPage, itemsPerPage } = state;

    const startIndex = (currentPage - 1) * itemsPerPage;
    const endIndex = startIndex + itemsPerPage;

    // Hide all posts
    allPosts.forEach(function(post) {
      post.style.display = 'none';
    });

    // Show posts for current page
    for (let i = startIndex; i < endIndex && i < allPosts.length; i++) {
      allPosts[i].style.display = '';
    }

    // Scroll to top of posts list
    if (postsList) {
      postsList.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
  }

  function renderPaginationControls(nav, state) {
    const { currentPage, totalPages } = state;

    const prevBtn = nav.querySelector('[data-action="prev"]');
    const nextBtn = nav.querySelector('[data-action="next"]');
    const pagesContainer = nav.querySelector('.pagination-pages');

    // Update prev/next buttons
    if (prevBtn) {
      prevBtn.disabled = currentPage <= 1;
      prevBtn.classList.toggle('disabled', currentPage <= 1);
    }

    if (nextBtn) {
      nextBtn.disabled = currentPage >= totalPages;
      nextBtn.classList.toggle('disabled', currentPage >= totalPages);
    }

    // Render page numbers
    if (pagesContainer) {
      pagesContainer.innerHTML = '';

      for (let i = 1; i <= totalPages; i++) {
        const pageEl = document.createElement(i === currentPage ? 'span' : 'button');
        pageEl.className = 'pagination-page' + (i === currentPage ? ' current' : '');
        pageEl.textContent = i;

        if (i === currentPage) {
          pageEl.setAttribute('aria-current', 'page');
        } else {
          pageEl.dataset.page = i;
          // Use closure to capture correct page number
          (function(pageNum) {
            pageEl.addEventListener('click', function() {
              navigateToPage(pageNum, state, nav);
            });
          })(i);
        }

        pagesContainer.appendChild(pageEl);
      }
    }
  }

  function setupEventListeners(nav, state) {
    const prevBtn = nav.querySelector('[data-action="prev"]');
    const nextBtn = nav.querySelector('[data-action="next"]');

    if (prevBtn) {
      prevBtn.addEventListener('click', function() {
        if (state.currentPage > 1) {
          navigateToPage(state.currentPage - 1, state, nav);
        }
      });
    }

    if (nextBtn) {
      nextBtn.addEventListener('click', function() {
        if (state.currentPage < state.totalPages) {
          navigateToPage(state.currentPage + 1, state, nav);
        }
      });
    }
  }
})();
