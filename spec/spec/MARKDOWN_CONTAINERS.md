# Markdown Container Specification

This document specifies fenced Markdown containers using `:::` delimiters.

## Opening syntax

A container opener MUST:

- begin with at least three colons
- contain non-whitespace content after the colon marker
- use the number of leading colons as the container depth

Examples:

```markdown
::: card
:::: nested-card
::: card {#intro .featured data-kind="note"}
```

Text after the marker MAY provide classes and an attribute block. A named or attributed marker is always an opener, never a closer.

## Closing syntax

A closing marker MUST contain only three or more colons after surrounding whitespace is removed.

```markdown
:::
::::
```

These are not closing markers:

```markdown
::: card
::: {.card}
:::: nested
```

## Nesting semantics

Each container records the number of colons used by its opener.

- A colon-only marker with the same depth closes the current container and consumes the marker.
- A colon-only marker with fewer colons than the current container closes the current container without consuming the marker. The parent container can then process that same marker.
- A colon-only marker with more colons than the current container does not close the current container.

Authors SHOULD use a longer delimiter for each nested level and close each level with its matching delimiter when they want one-level-at-a-time behavior.

Recommended example:

```markdown
::: outer
Outer before.

:::: inner
Inner content.
::::

Outer after.
:::
```

Expected structure:

```html
<div class="outer">
<p>Outer before.</p>
<div class="inner">
<p>Inner content.</p>
</div>
<p>Outer after.</p>
</div>
```

A shorter closing marker inside a deeper container MAY therefore close more than one nested level as the unconsumed marker propagates to ancestors. Implementations MUST preserve this behavior consistently if they implement depth-aware container parsing.

## Attributes

Opening markers MAY contain:

- one or more whitespace-separated classes
- an ID using `#id` inside an attribute block
- extra classes using `.class` inside an attribute block
- key/value attributes such as `data-kind="note"`

Example:

```markdown
:::: panel wide {#example .accent data-state="ready"}
Content
::::
```

The rendered container MUST retain the declared classes, ID, and valid custom attributes.

## Conformance tests

Implementations SHOULD test at minimum:

1. a basic three-colon container
2. nested three/four-colon containers with matching closing markers
3. content after an inner close remains inside the outer container
4. named and attributed `:::` lines are treated as openers
5. colon-only markers are the only closing markers
6. shorter closing markers propagate to the parent rather than being consumed by the deeper container
