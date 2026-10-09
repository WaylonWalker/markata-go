(function() {
  'use strict';

  function prefersReducedMotion() {
    return window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  }

  function resetVideo(video) {
    try {
      video.pause();
      video.currentTime = 0;
    } catch (_) {
      // ignore
    }
  }

  function playVideo(video) {
    var policy = document.documentElement.dataset.loadingPolicy;
    if (!video || prefersReducedMotion() || policy === 'constrained') return;
    var playPromise = video.play();
    if (playPromise && typeof playPromise.catch === 'function') {
      playPromise.catch(function() {
        // Ignore autoplay failures.
      });
    }
  }

  function bindVideo(video) {
    if (!video || video.dataset.hoverPlayBound === 'true') return;
    if (!window.matchMedia || !window.matchMedia('(hover: hover) and (pointer: fine)').matches) return;
    video.dataset.hoverPlayBound = 'true';

    var card = video.closest('.shot-card, .card, [data-card], .photo-figure');
    if (!card) return;
    var hoverTimer = null;

    card.addEventListener('mouseenter', function() {
      hoverTimer = window.setTimeout(function() { playVideo(video); }, 250);
    });
    card.addEventListener('focusin', function() { playVideo(video); });
    card.addEventListener('mouseleave', function() {
      if (hoverTimer) window.clearTimeout(hoverTimer);
      hoverTimer = null;
      resetVideo(video);
    });
    card.addEventListener('focusout', function(event) {
      if (card.contains(event.relatedTarget)) return;
      resetVideo(video);
    });
  }

  function initHoverPlayVideo() {
    document.querySelectorAll('video[data-hover-play], video[data-adaptive-preview]').forEach(bindVideo);
  }

  window.initHoverPlayVideo = initHoverPlayVideo;

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initHoverPlayVideo);
  } else {
    initHoverPlayVideo();
  }

  window.addEventListener('view-transition-complete', initHoverPlayVideo);
  document.addEventListener('visibilitychange', function() {
    if (!document.hidden) return;
    document.querySelectorAll('video[data-hover-play], video[data-adaptive-preview]').forEach(resetVideo);
  });
})();
