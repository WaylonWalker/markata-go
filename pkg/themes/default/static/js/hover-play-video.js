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
    if (!video || prefersReducedMotion()) return;
    var playPromise = video.play();
    if (playPromise && typeof playPromise.catch === 'function') {
      playPromise.catch(function() {
        // Ignore autoplay failures.
      });
    }
  }

  function bindVideo(video) {
    if (!video || video.dataset.hoverPlayBound === 'true') return;
    video.dataset.hoverPlayBound = 'true';

    var card = video.closest('.shot-card');
    if (!card) return;

    card.addEventListener('mouseenter', function() { playVideo(video); });
    card.addEventListener('focusin', function() { playVideo(video); });
    card.addEventListener('mouseleave', function() { resetVideo(video); });
    card.addEventListener('focusout', function(event) {
      if (card.contains(event.relatedTarget)) return;
      resetVideo(video);
    });
  }

  function initHoverPlayVideo() {
    document.querySelectorAll('video[data-hover-play]').forEach(bindVideo);
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
    document.querySelectorAll('video[data-hover-play]').forEach(resetVideo);
  });
})();
