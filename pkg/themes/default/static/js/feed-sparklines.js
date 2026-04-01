(function () {
  'use strict';

  function bindSparklineReadout(containerSelector, valueSelector, pointSelector) {
    document.querySelectorAll(containerSelector).forEach(function (wrap) {
      var value = wrap.querySelector(valueSelector);
      if (!value) return;

      var defaultValue = value.textContent;
      wrap.querySelectorAll(pointSelector).forEach(function (point) {
        var update = function () {
          value.textContent = point.dataset.label || defaultValue;
        };
        point.addEventListener('mouseenter', update);
        point.addEventListener('focusin', update);
      });

      wrap.addEventListener('mouseleave', function () {
        value.textContent = defaultValue;
      });
    });
  }

  function init() {
    bindSparklineReadout('.feed-sparkline-wrap', '.feed-sparkline-value', '.feed-sparkline-point');
    bindSparklineReadout('.feed-header-sparkline', '.feed-header-sparkline-value', '.feed-header-sparkline-point');
  }

  document.addEventListener('DOMContentLoaded', init);
  window.addEventListener('view-transition-complete', init);
})();
