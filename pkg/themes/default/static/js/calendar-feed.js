(() => {
  'use strict';

  const monthNames = [
    'January', 'February', 'March', 'April', 'May', 'June',
    'July', 'August', 'September', 'October', 'November', 'December'
  ];
  const weekdayNames = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

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

  function postPreview(post) {
    const item = element('li', 'calendar-preview-post');
    const link = element('a', 'calendar-preview-link');
    link.href = post.href;

    if (post.image) {
      const image = document.createElement('img');
      image.className = 'calendar-preview-image';
      image.src = post.image;
      image.alt = '';
      image.loading = 'lazy';
      image.decoding = 'async';
      image.width = 72;
      image.height = 54;
      link.append(image);
    }

    const copy = element('span', 'calendar-preview-copy');
    copy.append(element('strong', 'calendar-preview-title', post.title));
    if (post.description) {
      copy.append(element('span', 'calendar-preview-description', post.description));
    }
    link.append(copy);
    item.append(link);
    return item;
  }

  function renderPreview(details, year, month, day, posts) {
    const preview = element('div', 'calendar-day-preview');
    const dateLabel = `${monthNames[month]} ${day}, ${year}`;
    const postWord = posts.length === 1 ? 'post' : 'posts';
    preview.append(element('div', 'calendar-preview-summary', `${dateLabel} · ${posts.length} ${postWord}`));

    const postList = element('ul', 'calendar-day-posts');
    for (const post of posts) postList.append(postPreview(post));
    preview.append(postList);
    details.append(preview);
  }

  function renderMonth(year, month, postsByDate) {
    const section = element('section', 'calendar-month');
    section.setAttribute('aria-labelledby', `calendar-${year}-${month + 1}`);

    const heading = element('h3', '', monthNames[month]);
    heading.id = `calendar-${year}-${month + 1}`;
    section.append(heading);

    const weekdayRow = element('div', 'calendar-weekdays');
    weekdayRow.setAttribute('aria-hidden', 'true');
    for (const weekday of weekdayNames) weekdayRow.append(element('span', '', weekday));
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

      const details = document.createElement('details');
      const summary = document.createElement('summary');
      const postWord = posts.length === 1 ? 'post' : 'posts';
      details.dataset.postCount = String(posts.length);
      summary.setAttribute('aria-label', `${monthNames[month]} ${day}, ${year}: ${posts.length} ${postWord}`);
      summary.append(element('span', 'calendar-day-number', String(day)));

      const marker = element('span', 'calendar-day-marker');
      marker.setAttribute('aria-hidden', 'true');
      marker.style.setProperty('--calendar-density', String(Math.min(posts.length, 6)));
      summary.append(marker);

      if (posts.length > 1) {
        const count = element('span', 'calendar-day-count', String(posts.length));
        count.setAttribute('aria-hidden', 'true');
        summary.append(count);
      }

      details.append(summary);
      renderPreview(details, year, month, day, posts);
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
      for (let month = 0; month < 12; month += 1) months.append(renderMonth(year, month, postsByDate));
      yearSection.append(months);
      calendar.append(yearSection);
    }

    const openExclusive = (details) => {
      for (const candidate of root.querySelectorAll('.calendar-day details[open]')) {
        if (candidate !== details) candidate.open = false;
      }
      details.open = true;
    };

    calendar.addEventListener('pointerover', (event) => {
      if (!window.matchMedia('(hover: hover) and (pointer: fine)').matches) return;
      const details = event.target.closest('.calendar-day details');
      if (!details || !calendar.contains(details)) return;
      openExclusive(details);
    });

    calendar.addEventListener('focusin', (event) => {
      const details = event.target.closest('.calendar-day details');
      if (details && calendar.contains(details)) openExclusive(details);
    });

    calendar.addEventListener('toggle', (event) => {
      const opened = event.target;
      if (!(opened instanceof HTMLDetailsElement) || !opened.open) return;
      for (const details of root.querySelectorAll('.calendar-day details[open]')) {
        if (details !== opened) details.open = false;
      }
    }, true);

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
      if (mode === 'calendar') url.searchParams.set('view', 'calendar');
      else if (url.searchParams.get('view') === 'calendar') url.searchParams.delete('view');
      window.history.replaceState(window.history.state, '', url.toString());
    } catch (error) {
      // URL state is progressive enhancement only.
    }
  }

  function primaryNodesFor(root, sourceList) {
    const nodes = Array.from(root.querySelectorAll('[data-calendar-primary]'));
    for (const child of root.children) {
      if (child.classList.contains('pagination') || child.classList.contains('pagination-infinite') || child.classList.contains('feed-empty')) {
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
      datedPosts.push({
        date,
        href: link.href,
        title: link.textContent.trim() || link.href,
        description: item.getAttribute('data-description') || '',
        image: item.getAttribute('data-image') || ''
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

      for (const button of buttons) {
        button.setAttribute('aria-pressed', button.getAttribute('data-calendar-mode') === mode ? 'true' : 'false');
      }

      if (persistURL && root.hasAttribute('data-calendar-url-state')) updateURLMode(mode);
    };

    for (const button of buttons) button.addEventListener('click', () => setMode(button.getAttribute('data-calendar-mode')));

    let initialMode = root.getAttribute('data-calendar-default') === 'calendar' ? 'calendar' : 'list';
    if (root.hasAttribute('data-calendar-url-state') && calendarModeFromURL()) initialMode = 'calendar';

    switcher.hidden = false;
    setMode(initialMode, false);
  }

  for (const root of document.querySelectorAll('[data-calendar-feed]')) initialize(root);
})();
