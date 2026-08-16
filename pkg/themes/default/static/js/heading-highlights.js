/* Draw accurate backgrounds for wrapped heading highlights. */
(function () {
  "use strict";

  // canonical-document-v1 contour input; never infer this from consumer CSS.
  var canonicalHighlightRadius = 10.08;

  var selector = "html[data-fontpack] :is(.post-header h1, .post-content h1, .post-content h2) .heading-highlight";
  var scheduled = false;
  var observers = [];

  function schedule() {
    if (scheduled) return;
    scheduled = true;
    requestAnimationFrame(function () {
      scheduled = false;
      document.querySelectorAll(selector).forEach(measure);
    });
  }

  function roundedRectPath(x, y, width, height, radius, corners) {
    var right = x + width;
    var bottom = y + height;
    var tl = corners.tl ? radius : 0;
    var tr = corners.tr ? radius : 0;
    var br = corners.br ? radius : 0;
    var bl = corners.bl ? radius : 0;
    return [
      "M", x + tl, y,
      "L", right - tr, y,
      tr ? "Q" : "L", right, y, right, y + tr,
      "L", right, bottom - br,
      br ? "Q" : "L", right, bottom, right - br, bottom,
      "L", x + bl, bottom,
      bl ? "Q" : "L", x, bottom, x, bottom - bl,
      "L", x, y + tl,
      tl ? "Q" : "L", x, y, x + tl, y,
      "Z"
    ].join(" ");
  }

  function overlapsAt(rect, other, x, radius) {
    return other && other.right > x - radius && other.left < x + radius;
  }

  function measure(wrapper) {
    var mark = wrapper.querySelector("mark");
    if (!mark) return;

    wrapper.classList.remove("is-measured");
    var old = wrapper.querySelector(".heading-highlight__contour");
    if (old) old.remove();

    var range = document.createRange();
    range.selectNodeContents(mark);
    var rawRects = Array.from(range.getClientRects()).filter(function (rect) {
      return rect.width > 0 && rect.height > 0;
    });
    if (!rawRects.length) return;

    var style = getComputedStyle(mark);
    var padLeft = parseFloat(style.paddingLeft) || 0;
    var padRight = parseFloat(style.paddingRight) || 0;
    var base = wrapper.getBoundingClientRect();
    var rects = rawRects.map(function (rect) {
      return {
        left: rect.left - base.left - padLeft,
        right: rect.right - base.left + padRight,
        top: rect.top - base.top,
        bottom: rect.bottom - base.top
      };
    });
    var radius = canonicalHighlightRadius;
    var paths = [];

    rects.forEach(function (rect, index) {
      var previous = rects[index - 1];
      var next = rects[index + 1];
      var samePreviousLine = previous && Math.abs(previous.bottom - rect.top) < 2;
      var sameNextLine = next && Math.abs(next.top - rect.bottom) < 2;
      var corners = {
        tl: !samePreviousLine || !overlapsAt(rect, previous, rect.left, radius),
        tr: !samePreviousLine || !overlapsAt(rect, previous, rect.right, radius),
        bl: !sameNextLine || !overlapsAt(rect, next, rect.left, radius),
        br: !sameNextLine || !overlapsAt(rect, next, rect.right, radius)
      };
      paths.push(roundedRectPath(rect.left, rect.top, rect.right - rect.left, rect.bottom - rect.top, radius, corners));
    });

    var svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    svg.setAttribute("class", "heading-highlight__contour");
    svg.setAttribute("aria-hidden", "true");
    var path = document.createElementNS("http://www.w3.org/2000/svg", "path");
    path.setAttribute("d", paths.join(" "));
    path.setAttribute("fill", "var(--heading-mark-bg, var(--color-mark-bg))");
    svg.appendChild(path);
    wrapper.appendChild(svg);
    wrapper.classList.add("is-measured");
  }

  function observe() {
    if (window.ResizeObserver) {
      document.querySelectorAll(selector).forEach(function (element) {
        var observer = new ResizeObserver(schedule);
        observer.observe(element);
        observers.push(observer);
      });
    }
    if (document.fonts && document.fonts.ready) document.fonts.ready.then(schedule);
    new MutationObserver(function (records) {
      var relevant = records.some(function (record) {
        if (record.type === "attributes" || record.type === "characterData") return true;
        return Array.from(record.addedNodes).concat(Array.from(record.removedNodes)).some(function (node) {
          return !(node.nodeType === 1 && node.classList.contains("heading-highlight__contour"));
        });
      });
      if (relevant) schedule();
    }).observe(document.documentElement, { attributes: true, characterData: true, childList: true, subtree: true });
    window.addEventListener("resize", schedule, { passive: true });
    schedule();
  }

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", observe);
  else observe();
})();
