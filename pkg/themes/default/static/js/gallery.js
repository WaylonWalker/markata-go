(() => {
  if (!window.HTMLDialogElement) return;

  let dialog;
  let image;
  let status;
  let links = [];
  let current = 0;
  let opener;
  let request = 0;
  let touchStart;

  function show(index, direction = 0) {
    if (!links.length) return;
    current = (index + links.length) % links.length;
    const link = links[current];
    const id = ++request;
    image.hidden = true;
    image.alt = link.querySelector('img')?.alt || '';
    image.src = link.href;
    const sourceCaption = link.closest('figure')?.querySelector('figcaption');
    const caption = dialog.querySelector('.gallery-viewer__caption');
    caption.textContent = sourceCaption?.textContent.trim() || '';
    caption.hidden = !caption.textContent;
    status.textContent = `${current + 1} of ${links.length}`;
    image.dataset.direction = direction > 0 ? 'next' : direction < 0 ? 'previous' : 'none';
    image.decode().then(() => {
      if (id === request && dialog.open) {
        image.hidden = false;
        image.classList.remove('gallery-viewer__image--entering');
        if (direction !== 0) {
          // Restart the transition even when a reader moves quickly between images.
          void image.offsetWidth;
          image.classList.add('gallery-viewer__image--entering');
        }
      }
    }).catch(() => {
      if (id === request) location.href = link.href;
    });
  }

  function createDialog() {
    if (dialog) return;
    dialog = document.createElement('dialog');
    dialog.className = 'gallery-viewer';
    dialog.setAttribute('aria-label', 'Image gallery');
    dialog.innerHTML = '<button class="gallery-viewer__close" type="button" aria-label="Close image"><svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="m6 6 12 12M18 6 6 18" /></svg></button><button class="gallery-viewer__prev" type="button" aria-label="Previous image"><svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="m15 18-6-6 6-6" /></svg></button><figure class="gallery-viewer__figure"><img class="gallery-viewer__image" alt=""><figcaption class="gallery-viewer__caption" hidden></figcaption></figure><button class="gallery-viewer__next" type="button" aria-label="Next image"><svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="m9 18 6-6-6-6" /></svg></button><p class="gallery-viewer__status" role="status" aria-live="polite"></p>';
    document.body.append(dialog);
    image = dialog.querySelector('img');
    status = dialog.querySelector('[role="status"]');
    dialog.querySelector('.gallery-viewer__close').addEventListener('click', () => dialog.close());
    dialog.querySelector('.gallery-viewer__prev').addEventListener('click', () => show(current - 1, -1));
    dialog.querySelector('.gallery-viewer__next').addEventListener('click', () => show(current + 1, 1));
    dialog.addEventListener('keydown', (event) => {
      if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') {
        event.preventDefault();
        const direction = event.key === 'ArrowRight' ? 1 : -1;
        show(current + direction, direction);
      }
    });
    dialog.addEventListener('click', (event) => { if (event.target === dialog) dialog.close(); });
    dialog.querySelector('.gallery-viewer__figure').addEventListener('pointerdown', (event) => {
      if (!event.isPrimary || event.pointerType === 'mouse') return;
      touchStart = { x: event.clientX, y: event.clientY, pointerId: event.pointerId };
    }, { passive: true });
    dialog.querySelector('.gallery-viewer__figure').addEventListener('pointerup', (event) => {
      if (!touchStart || event.pointerId !== touchStart.pointerId) return;
      const dx = event.clientX - touchStart.x;
      const dy = event.clientY - touchStart.y;
      touchStart = null;
      if (Math.abs(dx) > 50 && Math.abs(dx) > Math.abs(dy) * 1.2) {
        const direction = dx < 0 ? 1 : -1;
        show(current + direction, direction);
      }
    });
    dialog.querySelector('.gallery-viewer__figure').addEventListener('pointercancel', () => { touchStart = null; });
    dialog.addEventListener('close', () => { request++; image.removeAttribute('src'); image.classList.remove('gallery-viewer__image--entering'); opener?.focus(); });
  }

  document.addEventListener('click', (event) => {
    const link = event.target.closest?.('[data-gallery-image]');
    if (!link || event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
    links = [...link.closest('[data-gallery]').querySelectorAll('[data-gallery-image]')];
    if (!links.length) return;
    event.preventDefault();
    opener = link;
    createDialog();
    dialog.showModal();
    show(links.indexOf(link));
    dialog.querySelector('.gallery-viewer__close').focus();
  });
})();
