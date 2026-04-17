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
  wrapper.appendChild(results);

  root.appendChild(wrapper);

  input.addEventListener('input', function () {
    var q = input.value.trim();
    if (!q) {
      results.innerHTML = '';
      return;
    }
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(function () {
      doSearch(q);
    }, 200);
  });

  function doSearch(q) {
    if (controller) controller.abort();
    controller = new AbortController();

    var url = endpoint + '?q=' + encodeURIComponent(q) + '&limit=20&fuzzy=true';

    fetch(url, { signal: controller.signal })
      .then(function (r) { return r.json(); })
      .then(function (data) { renderResults(data, q); })
      .catch(function (err) {
        if (err.name !== 'AbortError') {
          results.innerHTML = '<p class="bleve-search__error">Search unavailable</p>';
        }
      });
  }

  function renderResults(data, query) {
    if (!data.results || data.results.length === 0) {
      results.innerHTML = '<p class="bleve-search__empty">No results for "' + escapeHTML(query) + '"</p>';
      return;
    }

    var html = '<ul class="bleve-search__list">';
    for (var i = 0; i < data.results.length; i++) {
      var r = data.results[i];
      var href = r.href || ('/' + r.slug);
      html += '<li class="bleve-search__item" role="option">';
      html += '<a href="' + escapeHTML(href) + '" class="bleve-search__link">';
      html += '<span class="bleve-search__title">' + escapeHTML(r.title || r.slug) + '</span>';
      if (r.description) {
        html += '<span class="bleve-search__desc">' + escapeHTML(truncate(r.description, 120)) + '</span>';
      }
      if (r.date) {
        var d = r.date.substring(0, 10);
        html += '<span class="bleve-search__date">' + d + '</span>';
      }
      html += '</a></li>';
    }
    html += '</ul>';
    if (data.total > 20) {
      html += '<p class="bleve-search__more">' + data.total + ' results</p>';
    }
    results.innerHTML = html;
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
      results.innerHTML = '';
    }
  };
})();
