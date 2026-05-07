// bleve-search.js — Drop-in search UI powered by the bleve API.
// Activated automatically when window.__markataSearchEndpoint is set
// (injected by `markata-go serve`). Uses the same #pagefind-search
// container and similar CSS classes so existing search styles apply.
(function () {
  'use strict';

  var endpoint = window.__markataSearchEndpoint;
  if (!endpoint) return;

  var root = document.getElementById('pagefind-search');
  if (!root) return;

  var debounceTimer = null;
  var controller = null;
  var resultLimit = 8;
  var activeIndex = -1;

  // Build the search input
  root.innerHTML = '';
  var wrapper = document.createElement('div');
  wrapper.className = 'bleve-search';

  var input = document.createElement('input');
  input.type = 'search';
  input.className = 'bleve-search__input';
  input.id = 'pagefind-search-input';
  input.name = 'search';
  input.placeholder = 'Search…';
  input.setAttribute('autocomplete', 'off');
  input.setAttribute('aria-label', 'Search site');
  wrapper.appendChild(input);

  var results = document.createElement('div');
  results.className = 'bleve-search__results';
  results.setAttribute('role', 'listbox');
  results.setAttribute('aria-label', 'Search results');
  results.hidden = true;
  wrapper.appendChild(results);

  root.appendChild(wrapper);

  input.setAttribute('aria-controls', 'bleve-search-results');
  input.setAttribute('aria-activedescendant', '');
  results.id = 'bleve-search-results';

  input.addEventListener('input', function () {
    var q = input.value.trim();
    if (!q) {
      clearResults();
      return;
    }
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(function () {
      doSearch(q);
    }, 200);
  });

  input.addEventListener('keydown', handleInputKeydown);
  input.addEventListener('blur', function () {
    window.setTimeout(function () {
      if (!wrapper.contains(document.activeElement)) {
        hideResults();
      }
    }, 0);
  });

  results.addEventListener('click', function (event) {
    var link = event.target.closest('.bleve-search__link');
    if (!link) {
      return;
    }
    clearResults(true);
  });

  results.addEventListener('keydown', handleResultKeydown);

  results.addEventListener('focusin', function (event) {
    var link = event.target.closest('.bleve-search__link');
    if (!link) {
      return;
    }
    activeIndex = parseInt(link.getAttribute('data-index'), 10);
    updateActiveDescendant(getResultLinks());
  });

  document.addEventListener('click', function (event) {
    if (!wrapper.contains(event.target)) {
      hideResults();
    }
  });

  function doSearch(q) {
    if (controller) controller.abort();
    controller = new AbortController();

    var url = endpoint + '?q=' + encodeURIComponent(q) + '&limit=' + resultLimit + '&fuzzy=true';

    fetch(url, { signal: controller.signal })
      .then(function (r) { return r.json(); })
      .then(function (data) { renderResults(data, q); })
      .catch(function (err) {
        if (err.name !== 'AbortError') {
          results.innerHTML = '<p class="bleve-search__error">Search unavailable</p>';
          showResults();
        }
      });
  }

  function renderResults(data, query) {
    activeIndex = -1;
    if (!data.results || data.results.length === 0) {
      results.innerHTML = '<p class="bleve-search__empty">No results for "' + escapeHTML(query) + '"</p>';
      showResults();
      return;
    }

    var html = '<ul class="bleve-search__list">';
    for (var i = 0; i < data.results.length; i++) {
      var r = data.results[i];
      var href = r.href || ('/' + r.slug);
      var title = escapeHTML(r.title || r.slug || 'Untitled');
      var media = normalizedMedia(r);
      var description = r.description ? escapeHTML(truncate(r.description, 140)) : '';
      var meta = buildMeta(r);

      html += '<li class="bleve-search__item" role="option">';
      html += '<a href="' + escapeHTML(href) + '" class="bleve-search__link" id="bleve-search-option-' + i + '" data-index="' + i + '">';
      if (media.html) {
        html += media.html;
      }
      html += '<span class="bleve-search__content">';
      html += '<span class="bleve-search__title">' + title + '</span>';
      if (description) {
        html += '<span class="bleve-search__desc">' + description + '</span>';
      }
      if (meta) {
        html += '<span class="bleve-search__meta">' + meta + '</span>';
      }
      html += '</span>';
      html += '</a></li>';
    }
    html += '</ul>';
    if (data.total > resultLimit) {
      html += '<p class="bleve-search__more">' + data.total + ' results</p>';
    }
    results.innerHTML = html;
    showResults();
  }

  function handleInputKeydown(event) {
    var links = getResultLinks();

    if (event.key === 'Escape') {
      clearResults(true);
      return;
    }

    if (!links.length) {
      return;
    }

    if (event.key === 'ArrowDown') {
      event.preventDefault();
      moveActive(1, links);
      return;
    }

    if (event.key === 'ArrowUp') {
      event.preventDefault();
      moveActive(-1, links);
      return;
    }

    if (event.key === 'Enter' && activeIndex >= 0) {
      event.preventDefault();
      links[activeIndex].click();
      return;
    }

    if (event.key === 'Tab') {
      if (activeIndex >= 0 && !event.shiftKey) {
        event.preventDefault();
        links[activeIndex].focus();
      } else if (!event.shiftKey) {
        hideResults();
      }
    }
  }

  function handleResultKeydown(event) {
    var links = getResultLinks();
    if (!links.length) {
      return;
    }

    if (event.key === 'Escape') {
      event.preventDefault();
      clearResults(true);
      return;
    }

    if (event.key === 'ArrowDown') {
      event.preventDefault();
      moveActive(1, links);
      if (activeIndex >= 0) {
        links[activeIndex].focus();
      }
      return;
    }

    if (event.key === 'ArrowUp') {
      event.preventDefault();
      moveActive(-1, links);
      if (activeIndex >= 0) {
        links[activeIndex].focus();
      }
      return;
    }

    if (event.key === 'Tab' && event.shiftKey) {
      event.preventDefault();
      input.focus();
      return;
    }

    if (event.key === 'Tab' && !event.shiftKey) {
      hideResults();
    }
  }

  function moveActive(delta, links) {
    if (!links.length) {
      return;
    }

    activeIndex += delta;
    if (activeIndex < 0) {
      activeIndex = links.length - 1;
    }
    if (activeIndex >= links.length) {
      activeIndex = 0;
    }

    updateActiveDescendant(links);
  }

  function updateActiveDescendant(links) {
    for (var i = 0; i < links.length; i++) {
      var isActive = i === activeIndex;
      links[i].classList.toggle('is-active', isActive);
      links[i].setAttribute('aria-selected', isActive ? 'true' : 'false');
    }

    if (activeIndex >= 0 && links[activeIndex]) {
      input.setAttribute('aria-activedescendant', links[activeIndex].id);
      links[activeIndex].scrollIntoView({ block: 'nearest' });
    } else {
      input.setAttribute('aria-activedescendant', '');
    }
  }

  function getResultLinks() {
    return results.querySelectorAll('.bleve-search__link');
  }

  function showResults() {
    results.hidden = false;
  }

  function hideResults() {
    activeIndex = -1;
    results.hidden = true;
    input.setAttribute('aria-activedescendant', '');
    var links = getResultLinks();
    for (var i = 0; i < links.length; i++) {
      links[i].classList.remove('is-active');
      links[i].setAttribute('aria-selected', 'false');
    }
  }

  function clearResults(blurInput) {
    if (controller) {
      controller.abort();
      controller = null;
    }
    results.innerHTML = '';
    hideResults();
    if (blurInput) {
      input.blur();
    }
  }

  function buildMeta(result) {
    var parts = [];
    if (result.date) {
      parts.push('<span class="bleve-search__meta-item bleve-search__date">' + escapeHTML(formatDate(result.date)) + '</span>');
    }
    if (result.read_time) {
      parts.push('<span class="bleve-search__meta-item">' + escapeHTML(result.read_time) + '</span>');
    }
    return parts.join('<span class="bleve-search__meta-sep">•</span>');
  }

  function formatDate(isoDate) {
    var date = new Date(isoDate);
    if (Number.isNaN(date.getTime())) {
      return isoDate.substring(0, 10);
    }

    return date.toLocaleDateString(undefined, {
      month: 'short',
      day: 'numeric',
      year: 'numeric'
    });
  }

  function normalizedMedia(result) {
    var mediaURL = normalizeString(result.media_url);
    if (!mediaURL) {
      return { html: '' };
    }

    if (result.media_type === 'video') {
      var attrs = ' autoplay muted loop playsinline preload="metadata"';
      var poster = normalizeString(result.poster_url);
      var mime = normalizeString(result.video_mime);
      var html = '<span class="bleve-search__media bleve-search__media--video">';
      html += '<video class="bleve-search__video"' + attrs;
      if (poster) {
        html += ' poster="' + escapeHTML(poster) + '"';
      }
      html += '>';
      html += '<source src="' + escapeHTML(mediaURL) + '"';
      if (mime) {
        html += ' type="' + escapeHTML(mime) + '"';
      }
      html += '>';
      html += '</video>';
      html += '</span>';
      return { html: html };
    }

    return {
      html: '<span class="bleve-search__media"><img src="' + escapeHTML(mediaURL) + '" alt="" loading="lazy"></span>'
    };
  }

  function normalizeString(value) {
    if (!value) return '';
    return String(value).trim();
  }

  function escapeHTML(s) {
    var el = document.createElement('span');
    el.textContent = s;
    return el.innerHTML;
  }

  function truncate(s, n) {
    return s.length > n ? s.substring(0, n) + '…' : s;
  }

  // Expose for view transitions re-initialization
  window.initBleveSearch = function () {
    var newRoot = document.getElementById('pagefind-search');
    if (newRoot && newRoot !== root) {
      root = newRoot;
      root.innerHTML = '';
      root.appendChild(wrapper);
      input.value = '';
      clearResults();
    }
  };

  window.dismissBleveSearch = function () {
    clearResults(true);
  };
})();
