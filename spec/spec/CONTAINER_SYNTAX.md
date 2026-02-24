# Container Blocks

General-purpose containers (`::: ...`) provide lightweight card-style wrappers that do not enforce any layout, letting authors add semantics, IDs, or extra attributes to groups of Markdown content.

## Opening

- Start a container with a line that begins with `:::` followed by zero or more classes and attributes, for example:

  ```markdown
  :::card
  :::card border {#summary key=value}
  ```

- Classes and IDs are split on whitespace; IDs use the `#` prefix and additional classes can be added inline or via `{...}` attribute blocks. Arbitrary key/value pairs are allowed inside the braces.

## Closing

- Containers only close when a line containing nothing but `:::` (plus optional whitespace) is encountered. Put that closing line on its own (no trailing text or class names) to end the nearest open container.

- Nested containers must each be closed with their own bare `:::` line. The parser treats any `:::` line that carries extra tokens as a new opening, so you cannot annotate the closing line with classes or names.

## Nesting

- Containers may contain other containers, headings, paragraphs, lists, or any other block-level Markdown. Nesting works in a stack-like manner: the closest open container is closed first when a bare `:::` appears.

- Example:

  ```markdown
  :::card
  # Outer header

  :::card
  Inner card content
  :::

  Outer footer
  :::
  ```

  Renders to two nested `<div class="card">` elements where the inner `</div>` always appears before the outer closing tag.

## Relationships with other markdown

- Containers can sit inside list items and blockquotes, but the closing `:::` line must not be indented deeper than its opening line (the parser treats indented `:::` as text). For inline content such as headings, just write them normally inside the container.

## Testing

- The parser behavior is exercised by `pkg/plugins/containers_test.go`, which renders Markdown strings through the `RenderMarkdownPlugin` and asserts that the nested `<div>` structure, headers, and content order stay intact when inner containers open and close.
