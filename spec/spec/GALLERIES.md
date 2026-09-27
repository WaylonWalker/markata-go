# Galleries

## Gallery page

A post MAY use `template: gallery.html` and a `gallery` frontmatter list. Each item MUST provide `src` and descriptive `alt` text. Items MAY provide `caption`, `width`, and `height`; dimensions are positive pixel counts for the original image. The template MUST render a linked `<img>` for each valid item, with optional `<figcaption>`. The link MUST point to the original media URL. The preview SHOULD use `with_size` for relative and trusted media URLs and leave other URLs intact.

The gallery MUST remain navigable without JavaScript. The default theme MAY enhance these links with a modal image viewer. The viewer MUST support Escape, arrow keys, focus return, and horizontal touch swipes. The viewer MUST show the source item's caption in a `<figcaption>` when one is provided. Previous and next image changes MUST animate horizontally, with motion disabled when the user requests reduced motion. Viewer controls MUST use centered, accessible icons with descriptive labels. Broken media MUST preserve access to the original link. An empty gallery MUST render the post body without an empty image grid.

The default layout uses responsive columns and keeps source order in the document. Sites MAY override the template or CSS.

## Photo grid feed

The existing `feed-photo-grid.html` template SHOULD show publication date and word count for entries with more than 280 characters of content when the counts exist. It MUST preserve keyboard access and the `card_classes` grid hooks.

## Post titles

Default post pages MUST expose a build-time title length category based on the plain title. The category provides a conservative first-paint font size for long titles while allowing wrapping. A browser enhancement MAY measure available width after fonts load and adjust the title without hiding it. The enhancement MUST recheck after resize and view transitions.
