# Web Awesome Plugin Specification

The `webawesome` plugin integrates Web Awesome custom elements with markata-go content. It provides ergonomic Markdown shortcuts for common components while preserving support for raw `<wa-*>` HTML.

## Goals

- Let authors use content-focused Web Awesome components without writing raw custom-element HTML.
- Load Web Awesome assets only on pages that contain Web Awesome components.
- Support CDN loading and self-hosted vendored Web Awesome distributions.
- Theme Web Awesome components with Web Awesome theme classes and markata-go CSS variables.

## Configuration

```toml
[markata-go]
hooks = ["default", "webawesome"]

[markata-go.webawesome]
enabled = true
version = "3.5.0"
source = "vendor" # "vendor" (default) or "cdn"
cdn_base_url = "https://cdn.jsdelivr.net/npm/@awesome.me/webawesome@3.5.0/dist-cdn"
output_dir = "assets/vendor/webawesome"
theme = "default"
palette = "default"
brand = "blue"
```

## Markdown Syntax

All shortcuts use container syntax. The recommended form is `::: wa-<component>`. The longer `::: webawesome <component>` form remains supported as an alias. Additional classes may be combined with the Web Awesome class; the `wa-*` class or `webawesome <component>` class pair may appear anywhere in the generated `class` attribute.

### Image Comparison

Authors create comparisons with a container block containing exactly two images:

```markdown
::: wa-comparison {position=35 caption="Homepage redesign"}
![Before](/images/before.webp)
![After](/images/after.webp)
:::
```

The long form is also valid:

```markdown
::: webawesome comparison {position=35}
![Before](/images/before.webp)
![After](/images/after.webp)
:::
```

Generated HTML:

```html
<figure class="markata-webawesome-figure">
  <wa-comparison class="markata-webawesome-comparison" position="35">
	<img slot="after" src="/images/before.webp" alt="Before" loading="lazy">
	<img slot="before" src="/images/after.webp" alt="After" loading="lazy">
  </wa-comparison>
  <figcaption>Homepage redesign</figcaption>
</figure>
```

Authoring order is left-to-right: the first image is shown on the left side of the divider at the default position, and the second image is shown on the right.

### Details

```markdown
::: wa-details {summary="Advanced options"}
Hidden content.
:::
```

Generated HTML:

```html
<wa-details summary="Advanced options">
  <p>Hidden content.</p>
</wa-details>
```

### Tabs

````markdown
::: wa-tabs

:::: wa-tab {label="macOS"}
```bash
brew install markata-go
```
::::

:::: wa-tab {label="Linux"}
```bash
curl -fsSL https://example.com/install.sh | sh
```
::::
:::
````

Generated HTML includes `wa-tab-group`, `wa-tab`, and `wa-tab-panel` elements. Tab names are slugified into panel names.

### Copy Button

````markdown
::: wa-copy
`go test ./...`
:::
````

Generated HTML:

```html
<wa-copy-button value="go test ./..."></wa-copy-button>
```

When `value` is omitted, the plugin uses the container body text.

### QR Code

````markdown
::: wa-qr
https://example.com/post
:::
````

Generated HTML:

```html
<wa-qr-code value="https://example.com/post"></wa-qr-code>
```

When `value` is omitted, the plugin uses the container body text.

### Badge And Tag

```markdown
::: wa-badge {variant="brand"}
New
:::

::: wa-tag {variant="success"}
Stable
:::
```

Generated HTML:

```html
<wa-badge variant="brand">New</wa-badge>
<wa-tag variant="success">Stable</wa-tag>
```

### Tooltip

```markdown
::: wa-tooltip {content="Static Site Generator"}
SSG
:::
```

Generated HTML (the slotted body becomes the inline anchor; the `content`
attribute becomes the tooltip popup body, since Web Awesome's `wa-tooltip`
default slot is the popup itself):

```html
<span class="markata-wa-tooltip-anchor" id="wa-tt-XXXXXXXX" tabindex="0">SSG</span><wa-tooltip for="wa-tt-XXXXXXXX">Static Site Generator</wa-tooltip>
```

### Carousel

```markdown
::: wa-carousel {navigation="true" pagination="true"}
![First](/images/one.webp)
First image caption
![Second](/images/two.webp)
Second image caption
:::
```

Generated HTML includes `wa-carousel` and one `wa-carousel-item` per image. Optional plain text or `<figcaption>` content immediately after an image becomes that slide's caption.

### Animated Image

```markdown
::: wa-animated-image
![Demo animation](/images/demo.gif)
:::
```

Generated HTML:

```html
<wa-animated-image src="/images/demo.gif" alt="Demo animation"></wa-animated-image>
```

## Raw HTML

Raw Web Awesome components are supported because markata-go allows raw HTML in Markdown:

```html
<wa-callout variant="brand">Raw Web Awesome also works.</wa-callout>
```

Any rendered page containing `<wa-` receives Web Awesome CSS plus the Web Awesome autoloader, which lazy-imports component modules on demand as `<wa-*>` elements appear in the document.

## Asset Loading

The default `source` is `"vendor"`. Self-hosting is preferred for stable, immutable component versions and to avoid third-party CDN dependencies at build or runtime.

When `source = "vendor"`, the plugin registers a shared archive asset for the Web Awesome npm tarball, which the `cdn_assets` plugin downloads into the shared asset cache and copies into the shared vendor output root. The plugin then loads:

```html
<link rel="stylesheet" href="/assets/vendor/webawesome/styles/themes/default.css">
<script type="module">
  import { setBasePath, startLoader } from "/assets/vendor/webawesome/webawesome.loader.js";
  setBasePath("/assets/vendor/webawesome");
  startLoader();
</script>
```

The vendored payload MUST be the browser-ready Web Awesome `dist-cdn` subtree from the npm tarball so the autoloader and components can resolve their modules and dependent assets without a bundler.

`markata-go assets download` SHOULD prefetch the Web Awesome tarball whenever the `webawesome` plugin is enabled with `source = "vendor"`.

When `source = "cdn"`, the plugin loads the same browser-ready files from the configured CDN base URL (default: `https://cdn.jsdelivr.net/npm/@awesome.me/webawesome@<version>/dist-cdn`):

```html
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@awesome.me/webawesome@3.5.0/dist-cdn/styles/themes/default.css">
<script type="module">
  import { setBasePath, startLoader } from "https://cdn.jsdelivr.net/npm/@awesome.me/webawesome@3.5.0/dist-cdn/webawesome.loader.js";
  setBasePath("https://cdn.jsdelivr.net/npm/@awesome.me/webawesome@3.5.0/dist-cdn");
  startLoader();
</script>
```

The autoloader uses a MutationObserver to discover `<wa-*>` tags in the document and lazy-imports the matching component module on demand. Pages without any `<wa-*>` tags do not load Web Awesome CSS or JavaScript.

## Theming

The plugin adds Web Awesome theme classes to the `<html>` element when needed:

```html
<html class="wa-theme-default wa-palette-default wa-brand-blue">
```

Generated comparison components also use markata-go CSS variables for borders, shadows, divider color, and captions with Web Awesome token fallbacks.
