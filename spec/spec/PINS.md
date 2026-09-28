# Pins feed

The default build MUST provide a pinboard feed at `/pins/` for published posts whose `link` frontmatter value is non-empty. The feed MUST use the ordinary feed collection and rendering pipeline so configured feed behavior, incremental builds, and output handling remain consistent.

## Feed selection

The built-in feed uses the filter `published == true and link`, sorted by date newest first. A site-defined feed with slug `pins` takes precedence and replaces the built-in definition. If a publishable post already owns the `pins` slug, the implicit feed MUST yield to that post rather than create an output-path conflict. Disabling subscription feeds disables this implicit feed along with the implicit root and archive feeds.

The feed renders HTML using the `pins.html` template. It does not create subscription formats by default. A site's explicit `pins` feed may configure any supported format, template, sorting, and pagination options.

## Pinboard presentation

The default pins template MUST render a responsive, varied-height board of cards using theme color tokens. Each card MUST distinguish the internal post from its external destination: its title links to the post, and a separate source link opens the `link` URL. Cards MAY show the post image, description or body excerpt, date, and tags when present. The source URL MUST be exposed as readable host text, and outbound links MUST use safe target attributes when opened in a new tab.

The feed MUST remain usable on narrow screens and by keyboard. Focus indicators MUST be visible. An empty feed MUST use the standard feed empty state.

Pin cards SHOULD lift subtly on pointer hover and keyboard focus, with a restrained image zoom and source-mark movement when present. These effects MUST be disabled when reduced motion is requested.

## Link metadata compatibility

The `link` frontmatter field selects posts for the default Pins feed but MUST NOT, by itself, change the rendering of an individual post or cause an external metadata request. Sites may continue using `link` as ordinary metadata without opting into embed behavior.
