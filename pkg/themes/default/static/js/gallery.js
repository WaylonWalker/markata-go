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

  function show(index) {
    current = (index + links.length) % links.length;
    const link = links[current];
    const id = ++request;
    image.hidden = true;
    image.alt = link.querySelector('img')?.alt || '';
    image.src = link.href;
    status.textContent = `${current + 1} of ${links.length}`;
    image.decode().then(() => {
      if (id === request && dialog.open) image.hidden = false;
    }).catch(() => {
      if (id === request) location.href = link.href;
    });
  }

  function createDialog() {
    if (dialog) return;
    dialog = document.createElement('dialog');
    dialog.className = 'gallery-viewer';
    dialog.setAttribute('aria-label', 'Image gallery');
    dialog.innerHTML = '<button class="gallery-viewer__close" type="button" aria-label="Close image">×</button><button class="gallery-viewer__prev" type="button" aria-label="Previous image">‹</button><img class="gallery-viewer__image" alt=""><button class="gallery-viewer__next" type="button" aria-label="Next image">›</button><p class="gallery-viewer__status" role="status" aria-live="polite"></p>';
    document.body.append(dialog);
    image = dialog.querySelector('img');
    status = dialog.querySelector('[role="status"]');
    dialog.querySelector('.gallery-viewer__close').addEventListener('click', () => dialog.close());
    dialog.querySelector('.gallery-viewer__prev').addEventListener('click', () => show(current - 1));
    dialog.querySelector('.gallery-viewer__next').addEventListener('click', () => show(current + 1));
    dialog.addEventListener('keydown', (event) => {
      if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') {
        event.preventDefault();
        show(current + (event.key === 'ArrowRight' ? 1 : -1));
      }
    });
    dialog.addEventListener('click', (event) => { if (event.target === dialog) dialog.close(); });
    image.addEventListener('touchstart', (event) => {
      const touch = event.changedTouches[0];
      touchStart = { x: touch.screenX, y: touch.screenY };
    }, { passive: true });
    image.addEventListener('touchend', (event) => {
      if (!touchStart) return;
      const touch = event.changedTouches[0];
      const dx = touch.screenX - touchStart.x;
      const dy = touch.screenY - touchStart.y;
      touchStart = null;
      if (Math.abs(dx) > 50 && Math.abs(dx) > Math.abs(dy) * 1.2) show(current + (dx < 0 ? 1 : -1));
    }, { passive: true });
    dialog.addEventListener('close', () => { request++; image.removeAttribute('src'); opener?.focus(); });
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
