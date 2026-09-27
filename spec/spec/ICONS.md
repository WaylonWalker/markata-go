# Icon Shortcodes Specification

The icons plugin expands local SVG icon packs into inline HTML during the Transform stage.

## Goals

- Support Zensical-compatible icon shortcodes in Markdown.
- Keep builds deterministic and usable offline when icon assets are present locally.
- Avoid changing emoji shortcodes such as `:smile:` when no matching icon asset exists.
- Never expand icon syntax inside inline code or fenced code blocks.
- Provide a local asset contract that automatic vendoring can build on separately.

## Asset Layout

The plugin searches these directories by default, in order:

1. `.icons/`
2. `static/.icons/`

SVG paths below an icon root define pack and icon names. For example:

```text
static/.icons/
└── lucide/
    └── smile.svg
```

produces the logical icon name `lucide/smile`.

When the same logical icon exists in more than one configured path, the first configured path wins.

## Markdown Syntax

For an asset named `lucide/smile`, the canonical Zensical-style Markdown syntax is:

```markdown
:lucide-smile:
```

markata-go also accepts the slash form originally proposed by issue #895:

```markdown
:lucide/smile:
```

Both forms resolve to the same icon. Nested paths are flattened by replacing `/` with `-` for the canonical alias. If two different icon paths would produce the same flattened alias, the ambiguous hyphen alias is disabled; the slash forms remain available.

Unknown shortcodes are preserved verbatim. This lets Goldmark's emoji extension continue handling ordinary emoji shortcodes independently.

## Configuration

```toml
[icons]
enabled = true
paths = [".icons", "static/.icons"]
packs = ["lucide", "simple-icons"]
```

Fields:

| Field | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | `true` | Enable icon shortcode expansion. |
| `path` | string | — | Convenience form for a single icon root. |
| `paths` | list[string] | `[".icons", "static/.icons"]` | Ordered local SVG roots. |
| `packs` | list[string] | all packs | Optional allowlist of first path components. |

`paths` takes precedence over `path` when both are set.

## Rendering

Resolved icons are emitted as inline SVG wrapped in a span:

```html
<span class="icon" data-icon="lucide/smile" aria-hidden="true">
  <svg width="1em" height="1em" aria-hidden="true" focusable="false">...</svg>
</span>
```

The wrapper is vertically aligned with surrounding text. Source SVG `width` and `height` attributes are normalized to `1em`; the SVG `viewBox` and drawing attributes are preserved.

Icon shortcodes are decorative by default and are hidden from assistive technology. Authors should include visible text when an icon carries meaning.

## Code Protection

The plugin must not transform shortcodes inside code:

```markdown
Visible :lucide-smile:

`literal :lucide-smile:`

```text
:lucide-smile:
```
```

Only the first shortcode is expanded.

## SVG Safety

Local SVGs are author-controlled inputs, but the plugin rejects assets containing active-content primitives that would be unsafe to inline:

- `<script>`
- `<foreignObject>`
- inline event-handler attributes such as `onload=`
- `javascript:` URLs

Rejected assets behave like missing icons and leave the shortcode unchanged.

## Lifecycle

The plugin runs at `PriorityLast` in the Transform stage. This lets source transforms such as Jinja run first while still expanding icons before Markdown rendering.

## Automatic Pack Vendoring

Downloading icon packs is intentionally separate from shortcode rendering. Issue #898 may populate `.icons/` or `static/.icons/` from configured upstream packs; the rendering contract in this specification does not require network access.
