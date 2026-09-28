(() => {
  'use strict';

  const monthNames = [
    'January', 'February', 'March', 'April', 'May', 'June',
    'July', 'August', 'September', 'October', 'November', 'December'
  ];
  const weekdayNames = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
  const hoverMedia = window.matchMedia
    ? window.matchMedia('(hover: hover) and (pointer: fine)')
    : null;

  const pad2 = (value) => String(value).padStart(2, '0');
  const dateKey = (year, month, day) => `${year}-${pad2(month + 1)}-${pad2(day)}`;

  function parseDate(value) {
    const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value || '');
    if (!match) return null;

    const year = Number(match[1]);
    const month = Number(match[2]) - 1;
    const day = Number(match[3]);
    const candidate = new Date(year, month, day);

    if (
      candidate.getFullYear() !== year ||
      candidate.getMonth() !== month ||
      candidate.getDate() !== day
    ) {
      return null;
    }

    return { year, month, day };
  }

  function element(tag, className, text) {
    const node = document.createElement(tag);
    if (className) node.className = className;
    if (text !== undefined) node.textContent = text;
    return node;
  }

  function daySummary(posts) {
    if (posts.length === 1) {
      return posts[0].summary || posts[0].title;
    }

    const summaries = posts
      .map((post) => post.summary)
      .filter(Boolean)
      .slice(0, 2);
    if (summaries.length > 0) {
      return summaries.join(' • ');
    }

    const titles = posts.slice(0, 3).map((post) => post.title);
    const rest = posts.length > titles.length ? '…' : '';
    return `${posts.length} posts: ${titles.join(', ')}${rest}`;
  }

  function hydratePreviewImages(details) {
    for (const image of details.querySelectorAll('img[data-calendar-src]')) {
      const src = image.getAttribute('data-calendar-src');
      if (!src) continue;
      image.src = src;
      image.removeAttribute('data-calendar-src');
    }
  }

  function positionPreview(details) {
    const preview = details.querySelector('.calendar-day-posts');
    if (!preview) return;

    preview.removeAttribute('data-align');
    window.requestAnimationFrame(() => {
      if (!details.open) return;
      const rect = preview.getBoundingClientRect();
      const gutter = 12;
      if (rect.left < gutter) {
        preview.setAttribute('data-align', 'start');
      } else if (rect.right > window.innerWidth - gutter) {
        preview.setAttribute('data-align', 'end');
      }
    });
  }

  function closeDetails(root, except = null) {
    for (const details of root.querySelectorAll('.calendar-day details[open]')) {
      if (details === except) continue;
      details.open = false;
      details.removeAttribute('data-calendar-pinned');
      const preview = details.querySelector('.calendar-day-posts');
      if (preview) preview.removeAttribute('data-align');
    }
  }

  function openDetails(root, details, pinned) {
    closeDetails(root, details);
    details.open = true;
    if (pinned) {
      details.setAttribute('data-calendar-pinned', 'true');
    } else {
      details.removeAttribute('data-calendar-pinned');
    }
    hydratePreviewImages(details);
    positionPreview(details);
  }

  function attachDayInteractions(root, details) {
    const summary = details.querySelector(':scope > summary');
    if (!summary) return;

    let closeTimer = 0;
    const cancelClose = () => {
      if (!closeTimer) return;
      window.clearTimeout(closeTimer);
      closeTimer = 0;
    };
    const scheduleClose = () => {
      cancelClose();
      if (details.hasAttribute('data-calendar-pinned')) return;
      closeTimer = window.setTimeout(() => {
        closeTimer = 0;
        if (details.matches(':hover')) return;
        if (details.contains(document.activeElement)) return;
        details.open = false;
      }, 70);
    };

    details.addEventListener('pointerenter', () => {
      cancelClose();
      if (!hoverMedia || !hoverMedia.matches) return;
      openDetails(root, details, false);
    });
    details.addEventListener('pointerleave', () => {
      if (!hoverMedia || !hoverMedia.matches) return;
      scheduleClose();
    });

    summary.addEventListener('focus', () => {
      cancelClose();
      openDetails(root, details, false);
    });
    details.addEventListener('focusout', scheduleClose);

    summary.addEventListener('click', (event) => {
      event.preventDefault();
      cancelClose();
      const shouldPin = !details.hasAttribute('data-calendar-pinned');
      if (shouldPin) {
        openDetails(root, details, true);
      } else {
        details.removeAttribute('data-calendar-pinned');
        details.open = false;
      }
    });

    details.addEventListener('keydown', (event) => {
      if (event.key !== 'Escape' || !details.open) return;
      event.preventDefault();
      details.removeAttribute('data-calendar-pinned');
      details.open = false;
      summary.focus();
    });

    details.addEventListener('toggle', () => {
      if (details.open) {
        hydratePreviewImages(details);
        positionPreview(details);
      } else {
        details.removeAttribute('data-calendar-pinned');
      }
    });
  }

  function renderMonth(year, month, postsByDate) {
    const section = element('section', 'calendar-month');
    section.setAttribute('aria-labelledby', `calendar-${year}-${month + 1}`);

    const heading = element('h3', '', monthNames[month]);
    heading.id = `calendar-${year}-${month + 1}`;
    section.append(heading);

    const weekdayRow = element('div', 'calendar-weekdays');
    weekdayRow.setAttribute('aria-hidden', 'true');
    for (const weekday of weekdayNames) {
      weekdayRow.append(element('span', '', weekday));
    }
    section.append(weekdayRow);

    const grid = element('div', 'calendar-days');
    const firstWeekday = new Date(year, month, 1).getDay();
    const daysInMonth = new Date(year, month + 1, 0).getDate();

    for (let i = 0; i < firstWeekday; i += 1) {
      const blank = element('span', 'calendar-day-empty');
      blank.setAttribute('aria-hidden', 'true');
      grid.append(blank);
    }

    for (let day = 1; day <= daysInMonth; day += 1) {
      const key = dateKey(year, month, day);
      const posts = postsByDate.get(key) || [];
      const cell = element('div', 'calendar-day');

      if (posts.length === 0) {
        const quiet = element('span', 'calendar-day-quiet', String(day));
        quiet.setAttribute('aria-label', `${monthNames[month]} ${day}, ${year}`);
        cell.append(quiet);
        grid.append(cell);
        continue;
      }

      cell.classList.add('calendar-day--active');
      const details = document.createElement('details');
      details.setAttribute('data-calendar-count', String(posts.length));
      const summary = document.createElement('summary');
      const postWord = posts.length === 1 ? 'post' : 'posts';
      const dateLabel = `${monthNames[month]} ${day}, ${year}`;
      summary.setAttribute('aria-label', `${dateLabel}: ${posts.length} ${postWord}`);
      summary.append(element('span', 'calendar-day-number', String(day)));

      const marker = element('span', 'calendar-day-marker');
      marker.setAttribute('aria-hidden', 'true');
      marker.style.setProperty('--calendar-density', String(Math.min(posts.length, 5)));
      summary.append(marker);

      if (posts.length > 1) {
        const count = element('span', 'calendar-day-count', String(posts.length));
        count.setAttribute('aria-hidden', 'true');
        summary.append(count);
      }

      details.append(summary);

      const postList = element('ul', 'calendar-day-posts');
      postList.setAttribute('aria-label', `${dateLabel} posts`);

      const overview = element('li', 'calendar-day-overview');
      const overviewHead = element('div', 'calendar-day-overview-head');
      overviewHead.append(element('strong', 'calendar-day-overview-date', dateLabel));
      overviewHead.append(element('span', 'calendar-day-overview-count', `${posts.length} ${postWord}`));
      overview.append(overviewHead);
      const overviewSummary = daySummary(posts);
      if (overviewSummary) {
        overview.append(element('p', 'calendar-day-overview-summary', overviewSummary));
      }
      postList.append(overview);

      for (const post of posts) {
        const item = element('li', 'calendar-day-post');
        const link = element('a', 'calendar-day-post-link');
        link.href = post.href;

        if (post.image) {
          const image = element('img', 'calendar-day-post-image');
          image.alt = '';
          image.width = 64;
          image.height = 40;
          image.loading = 'lazy';
          image.decoding = 'async';
          image.setAttribute('data-calendar-src', post.image);
          link.append(image);
        }

        const copy = element('span', 'calendar-day-post-copy');
        copy.append(element('strong', 'calendar-day-post-title', post.title));
        if (post.summary) {
          copy.append(element('span', 'calendar-day-post-description', post.summary));
        }
        link.append(copy);
        item.append(link);
        postList.append(item);
      }
      details.append(postList);
      cell.append(details);
      grid.append(cell);
    }

    const usedCells = firstWeekday + daysInMonth;
    const trailingCells = (7 - (usedCells % 7)) % 7;
    for (let i = 0; i < trailingCells; i += 1) {
      const blank = element('span', 'calendar-day-empty');
      blank.setAttribute('aria-hidden', 'true');
      grid.append(blank);
    }

    section.append(grid);
    return section;
  }

  function renderCalendar(root, calendar, datedPosts) {
    const postsByDate = new Map();
    const years = new Set();

    for (const post of datedPosts) {
      const parsed = parseDate(post.date);
      if (!parsed) continue;

      years.add(parsed.year);
      const existing = postsByDate.get(post.date) || [];
      existing.push(post);
      postsByDate.set(post.date, existing);
    }

    if (years.size === 0) return false;

    calendar.replaceChildren();
    for (const year of Array.from(years).sort((a, b) => b - a)) {
      const yearSection = element('section', 'calendar-year');
      const yearHeading = element('h2', '', String(year));
      yearHeading.id = `calendar-year-${year}`;
      yearSection.setAttribute('aria-labelledby', yearHeading.id);
      yearSection.append(yearHeading);

      const months = element('div', 'calendar-months');
      for (let month = 0; month < 12; month += 1) {
        months.append(renderMonth(year, month, postsByDate));
      }
      yearSection.append(months);
      calendar.append(yearSection);
    }

    for (const details of calendar.querySelectorAll('.calendar-day details')) {
      attachDayInteractions(root, details);
    }

    return true;
  }

  function calendarModeFromURL() {
    try {
      return new URLSearchParams(window.location.search).get('view') === 'calendar';
    } catch (error) {
      return false;
    }
  }

  function updateURLMode(mode) {
    try {
      const url = new URL(window.location.href);
      if (mode === 'calendar') {
        url.searchParams.set('view', 'calendar');
      } else if (url.searchParams.get('view') === 'calendar') {
        url.searchParams.delete('view');
      }
      window.history.replaceState(window.history.state, '', url.toString());
    } catch (error) {
      // URL state is progressive enhancement only.
    }
  }

  function primaryNodesFor(root, sourceList) {
    const nodes = Array.from(root.querySelectorAll('[data-calendar-primary]'));
    for (const child of root.children) {
      if (
        child.classList.contains('pagination') ||
        child.classList.contains('pagination-infinite') ||
        child.classList.contains('feed-empty')
      ) {
        if (!nodes.includes(child)) nodes.push(child);
      }
    }
    if (nodes.length === 0) nodes.push(sourceList);
    return nodes;
  }

  function initialize(root) {
    const sourceList = root.querySelector('[data-calendar-list]');
    const calendar = root.querySelector('[data-calendar-years]');
    const switcher = root.querySelector('[data-calendar-switch]');
    if (!sourceList || !calendar || !switcher) return;

    const datedPosts = [];
    for (const item of sourceList.querySelectorAll('[data-calendar-post]')) {
      const date = item.getAttribute('data-date') || '';
      const link = item.querySelector('a[href]');
      if (!parseDate(date) || !link) continue;

      const summary = item.querySelector('[data-calendar-summary]');
      const image = item.querySelector('[data-calendar-image-url]');
      datedPosts.push({
        date,
        href: link.href,
        title: link.textContent.trim() || link.href,
        summary: summary ? summary.textContent.trim() : '',
        image: image ? image.getAttribute('data-calendar-image-url') || '' : ''
      });
    }

    if (!renderCalendar(root, calendar, datedPosts)) return;

    const primaryNodes = primaryNodesFor(root, sourceList);
    const sourceIsPrimary = primaryNodes.includes(sourceList);
    const buttons = Array.from(switcher.querySelectorAll('[data-calendar-mode]'));
    const setMode = (mode, persistURL = true) => {
      const calendarMode = mode === 'calendar';
      calendar.hidden = !calendarMode;
      for (const node of primaryNodes) node.hidden = calendarMode;
      if (!sourceIsPrimary) sourceList.hidden = true;
      if (!calendarMode) closeDetails(root);

      for (const button of buttons) {
        button.setAttribute(
          'aria-pressed',
          button.getAttribute('data-calendar-mode') === mode ? 'true' : 'false'
        );
      }

      if (persistURL && root.hasAttribute('data-calendar-url-state')) {
        updateURLMode(mode);
      }
    };

    for (const button of buttons) {
      button.addEventListener('click', () => setMode(button.getAttribute('data-calendar-mode')));
    }

    let initialMode = root.getAttribute('data-calendar-default') === 'calendar' ? 'calendar' : 'list';
    if (root.hasAttribute('data-calendar-url-state') && calendarModeFromURL()) {
      initialMode = 'calendar';
    }

    switcher.hidden = false;
    setMode(initialMode, false);
  }

  for (const root of document.querySelectorAll('[data-calendar-feed]')) {
    initialize(root);
  }
})();
