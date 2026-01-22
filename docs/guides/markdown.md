---
title: "Markdown Features"
description: "Guide to supported Markdown syntax including GFM, admonitions, wikilinks, and table of contents"
date: 2024-01-15
published: true
template: doc.html
tags:
  - documentation
  - markdown
  - content
---

# Markdown Features

markata-go supports standard Markdown (CommonMark), GitHub Flavored Markdown (GFM) extensions, and several powerful additions like admonitions, wikilinks, and automatic table of contents generation. This guide covers everything you need to know about writing content in markata-go.

## Table of Contents

- [Basic Markdown](#basic-markdown)
- [Extended Markdown (GFM)](#extended-markdown-gfm)
- [Code Blocks](#code-blocks)
- [Admonitions](#admonitions)
- [Wikilinks](#wikilinks)
- [Table of Contents](#table-of-contents)
- [Heading Anchors](#heading-anchors)
- [Footnotes](#footnotes)

---

## Basic Markdown

markata-go supports all standard Markdown syntax as defined by CommonMark.

### Headings

Use `#` characters to create headings (levels 1-6):

**Input:**

```markdown
# Heading 1
## Heading 2
### Heading 3
#### Heading 4
##### Heading 5
###### Heading 6
```

**Output:**

```html
<h1>Heading 1</h1>
<h2>Heading 2</h2>
<h3>Heading 3</h3>
<h4>Heading 4</h4>
<h5>Heading 5</h5>
<h6>Heading 6</h6>
```

### Paragraphs

Paragraphs are separated by blank lines:

**Input:**

```markdown
This is the first paragraph. It can span
multiple lines in the source.

This is the second paragraph.
```

**Live example:**

This is the first paragraph. It can span
multiple lines in the source.

This is the second paragraph.

**Output:**

```html
<p>This is the first paragraph. It can span
multiple lines in the source.</p>

<p>This is the second paragraph.</p>
```

### Emphasis and Strong

**Input:**

```markdown
*Italic text* or _italic text_

**Bold text** or __bold text__

***Bold and italic*** or ___bold and italic___
```

**Live example:**

*Italic text* or _italic text_

**Bold text** or __bold text__

***Bold and italic*** or ___bold and italic___

**Output:**

```html
<p><em>Italic text</em> or <em>italic text</em></p>

<p><strong>Bold text</strong> or <strong>bold text</strong></p>

<p><strong><em>Bold and italic</em></strong> or <strong><em>bold and italic</em></strong></p>
```

### Links

**Input:**

```markdown
[Link text](https://example.com)

[Link with title](https://example.com "Example Site")

<https://example.com>

[Reference link][ref]

[ref]: https://example.com
```

**Live example:**

[Link text](https://example.com)

[Link with title](https://example.com "Example Site")

<https://example.com>

[Reference link][ref]

[ref]: https://example.com

**Output:**

```html
<p><a href="https://example.com">Link text</a></p>

<p><a href="https://example.com" title="Example Site">Link with title</a></p>

<p><a href="https://example.com">https://example.com</a></p>

<p><a href="https://example.com">Reference link</a></p>
```

### Images

**Input:**

```markdown
![Alt text](/images/photo.jpg)

![Alt text](/images/photo.jpg "Image title")

[![Clickable image](/images/photo.jpg)](https://example.com)
```

**Output:**

```html
<p><img src="/images/photo.jpg" alt="Alt text"></p>

<p><img src="/images/photo.jpg" alt="Alt text" title="Image title"></p>

<p><a href="https://example.com"><img src="/images/photo.jpg" alt="Clickable image"></a></p>
```

### Unordered Lists

**Input:**

```markdown
- Item one
- Item two
  - Nested item
  - Another nested item
- Item three

* Also works with asterisks
* Like this
```

**Live example:**

- Item one
- Item two
  - Nested item
  - Another nested item
- Item three

* Also works with asterisks
* Like this

**Output:**

```html
<ul>
  <li>Item one</li>
  <li>Item two
    <ul>
      <li>Nested item</li>
      <li>Another nested item</li>
    </ul>
  </li>
  <li>Item three</li>
</ul>
```

### Ordered Lists

**Input:**

```markdown
1. First item
2. Second item
   1. Nested numbered item
   2. Another nested item
3. Third item

1. Numbers don't have to be sequential
1. Markdown will number them correctly
1. This becomes 3
```

**Live example:**

1. First item
2. Second item
   1. Nested numbered item
   2. Another nested item
3. Third item

1. Numbers don't have to be sequential
1. Markdown will number them correctly
1. This becomes 3

**Output:**

```html
<ol>
  <li>First item</li>
  <li>Second item
    <ol>
      <li>Nested numbered item</li>
      <li>Another nested item</li>
    </ol>
  </li>
  <li>Third item</li>
</ol>
```

### Blockquotes

**Input:**

```markdown
> This is a blockquote.
> It can span multiple lines.

> Blockquotes can be nested.
>
> > Like this inner quote.

> Blockquotes can contain **other** *Markdown* elements.
>
> - Including lists
> - Like this
```

**Live examples:**

> This is a blockquote.
> It can span multiple lines.

> Blockquotes can be nested.
>
> > Like this inner quote.

> Blockquotes can contain **other** *Markdown* elements.
>
> - Including lists
> - Like this

**Output:**

```html
<blockquote>
  <p>This is a blockquote.
  It can span multiple lines.</p>
</blockquote>

<blockquote>
  <p>Blockquotes can be nested.</p>
  <blockquote>
    <p>Like this inner quote.</p>
  </blockquote>
</blockquote>
```

### Inline Code

**Input:**

```markdown
Use `backticks` for inline code.

Use `` `backticks` `` to show literal backticks.

The `fmt.Println()` function prints to stdout.
```

**Live example:**

Use `backticks` for inline code.

Use `` `backticks` `` to show literal backticks.

The `fmt.Println()` function prints to stdout.

**Output:**

```html
<p>Use <code>backticks</code> for inline code.</p>

<p>Use <code>`backticks`</code> to show literal backticks.</p>

<p>The <code>fmt.Println()</code> function prints to stdout.</p>
```

### Horizontal Rules

**Input:**

```markdown
Content above

---

Content below

***

Also works with asterisks

___

And underscores
```

**Live example:**

Content above

---

Content below

***

Also works with asterisks

___

And underscores

**Output:**

```html
<p>Content above</p>
<hr>
<p>Content below</p>
<hr>
<p>Also works with asterisks</p>
<hr>
<p>And underscores</p>
```

---

## Extended Markdown (GFM)

markata-go supports GitHub Flavored Markdown extensions.

### Tables

Tables use pipes (`|`) and hyphens (`-`) to define structure. Use colons (`:`) for alignment.

**Input:**

```markdown
| Feature      | Status      | Notes                    |
|--------------|:-----------:|-------------------------:|
| Tables       | Supported   | Left-aligned by default  |
| Alignment    | Supported   | Center with `:`          |
| Right align  | Supported   | Colon on right           |
```

**Live example:**

| Feature      | Status      | Notes                    |
|--------------|:-----------:|-------------------------:|
| Tables       | Supported   | Left-aligned by default  |
| Alignment    | Supported   | Center with `:`          |
| Right align  | Supported   | Colon on right           |

**Output:**

```html
<table>
  <thead>
    <tr>
      <th>Feature</th>
      <th style="text-align: center">Status</th>
      <th style="text-align: right">Notes</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>Tables</td>
      <td style="text-align: center">Supported</td>
      <td style="text-align: right">Left-aligned by default</td>
    </tr>
    <tr>
      <td>Alignment</td>
      <td style="text-align: center">Supported</td>
      <td style="text-align: right">Center with <code>:</code></td>
    </tr>
    <tr>
      <td>Right align</td>
      <td style="text-align: center">Supported</td>
      <td style="text-align: right">Colon on right</td>
    </tr>
  </tbody>
</table>
```

**Alignment reference:**

| Syntax | Alignment |
|--------|-----------|
| `\|---\|` | Left (default) |
| `\|:---\|` | Left (explicit) |
| `\|:---:\|` | Center |
| `\|---:\|` | Right |

### Strikethrough

**Input:**

```markdown
~~This text is struck through~~

Use strikethrough for ~~old information~~ corrections.
```

**Live example:**

~~This text is struck through~~

Use strikethrough for ~~old information~~ corrections.

**Output:**

```html
<p><del>This text is struck through</del></p>

<p>Use strikethrough for <del>old information</del> corrections.</p>
```

### Task Lists

**Input:**

```markdown
- [x] Completed task
- [x] Another completed task
- [ ] Incomplete task
- [ ] Another incomplete task
  - [x] Nested completed
  - [ ] Nested incomplete
```

**Live example:**

- [x] Completed task
- [x] Another completed task
- [ ] Incomplete task
- [ ] Another incomplete task
  - [x] Nested completed
  - [ ] Nested incomplete

**Output:**

```html
<ul class="task-list">
  <li class="task-list-item">
    <input type="checkbox" checked disabled> Completed task
  </li>
  <li class="task-list-item">
    <input type="checkbox" checked disabled> Another completed task
  </li>
  <li class="task-list-item">
    <input type="checkbox" disabled> Incomplete task
  </li>
  <li class="task-list-item">
    <input type="checkbox" disabled> Another incomplete task
    <ul class="task-list">
      <li class="task-list-item">
        <input type="checkbox" checked disabled> Nested completed
      </li>
      <li class="task-list-item">
        <input type="checkbox" disabled> Nested incomplete
      </li>
    </ul>
  </li>
</ul>
```

### Autolinks

URLs and email addresses are automatically converted to links:

**Input:**

```markdown
Visit https://example.com for more info.

Contact us at support@example.com.

www.example.com also works.
```

**Live example:**

Visit https://example.com for more info.

Contact us at support@example.com.

www.example.com also works.

**Output:**

```html
<p>Visit <a href="https://example.com">https://example.com</a> for more info.</p>

<p>Contact us at <a href="mailto:support@example.com">support@example.com</a>.</p>

<p><a href="http://www.example.com">www.example.com</a> also works.</p>
```

---

## Code Blocks

Fenced code blocks support syntax highlighting for many languages.

### Basic Code Block

**Input:**

````markdown
```
Plain code block without language
No syntax highlighting applied
```
````

**Output:**

```html
<pre><code>Plain code block without language
No syntax highlighting applied
</code></pre>
```

### Syntax Highlighting

Specify the language after the opening fence for syntax highlighting:

**Input:**

````markdown
```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
```
````

**Output:**

```html
<pre><code class="language-go">package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
</code></pre>
```

With syntax highlighting enabled, the output includes highlight spans:

```html
<pre><code class="language-go hljs">
<span class="hljs-keyword">package</span> main

<span class="hljs-keyword">import</span> <span class="hljs-string">"fmt"</span>

<span class="hljs-function"><span class="hljs-keyword">func</span> <span class="hljs-title">main</span><span class="hljs-params">()</span></span> {
    fmt.<span class="hljs-title function_">Println</span>(<span class="hljs-string">"Hello, World!"</span>)
}
</code></pre>
```

### Supported Languages

markata-go supports syntax highlighting for many languages including:

| Category | Languages |
|----------|-----------|
| **Web** | html, css, javascript, typescript, jsx, tsx, json, xml |
| **Backend** | go, python, ruby, java, c, cpp, rust, php |
| **Shell** | bash, sh, zsh, powershell, shell |
| **Data** | yaml, toml, ini, sql |
| **Markup** | markdown, latex, tex |
| **Config** | dockerfile, nginx, apache |
| **Other** | diff, plaintext, text |

### Configuration

Configure syntax highlighting in `markata-go.toml`:

```toml
[markata-go.markdown.highlight]
enabled = true
theme = "github-dark"    # Highlight.js theme
line_numbers = false     # Line numbers (if supported)
```

Available themes include: `github-dark`, `github-light`, `monokai`, `dracula`, `nord`, `solarized-dark`, `solarized-light`, and more.

### Language Aliases

Common aliases are supported:

| Alias | Language |
|-------|----------|
| `js` | javascript |
| `ts` | typescript |
| `py` | python |
| `rb` | ruby |
| `sh` | bash |
| `yml` | yaml |
| `md` | markdown |

---

## Admonitions

Admonitions (also called callouts) are visually distinct blocks for notes, warnings, tips, and other highlighted content.

### Basic Syntax

Use `!!!` followed by the type and optional title:

**Input:**

```markdown
!!! note "Optional Title"
    Admonition content here.
    Can span multiple lines.
    
    Supports **Markdown** formatting.

!!! warning
    Warning without custom title uses type as title.
```

**Live examples:**

!!! note "Optional Title"

    Admonition content here.
    Can span multiple lines.
    
    Supports **Markdown** formatting.

!!! warning

    Warning without custom title uses type as title.

**Output:**

```html
<div class="admonition note">
  <p class="admonition-title">Optional Title</p>
  <p>Admonition content here.
  Can span multiple lines.</p>
  <p>Supports <strong>Markdown</strong> formatting.</p>
</div>

<div class="admonition warning">
  <p class="admonition-title">Warning</p>
  <p>Warning without custom title uses type as title.</p>
</div>
```

### Admonition Types

| Type | Default Title | Typical Color | Use Case |
|------|---------------|---------------|----------|
| `note` | Note | Blue | Additional information |
| `info` | Info | Blue | Background context |
| `tip` | Tip | Green | Helpful suggestions |
| `hint` | Hint | Green | Subtle guidance |
| `success` | Success | Green | Positive outcomes |
| `warning` | Warning | Yellow/Orange | Potential issues |
| `caution` | Caution | Yellow/Orange | Proceed carefully |
| `danger` | Danger | Red | Critical warnings |
| `error` | Error | Red | Error conditions |
| `bug` | Bug | Red | Known issues |
| `example` | Example | Purple | Code examples |
| `quote` | Quote | Gray | Quotations |
| `abstract` | Abstract | Cyan | Summaries |
| `aside` | (none) | Gray | Sidebar/marginal notes |

### Examples of Each Type

Each example below shows the markdown syntax followed by how it renders.

**Note:**

```markdown
!!! note
    This is a general note providing additional context or information
    that might be helpful but isn't critical.
```

!!! note

    This is a general note providing additional context or information
    that might be helpful but isn't critical.

**Tip:**

```markdown
!!! tip "Pro Tip"
    Use keyboard shortcuts to speed up your workflow.
    Press `Ctrl+Shift+P` to open the command palette.
```

!!! tip "Pro Tip"

    Use keyboard shortcuts to speed up your workflow.
    Press `Ctrl+Shift+P` to open the command palette.

**Warning:**

```markdown
!!! warning "Deprecation Notice"
    This API will be removed in version 3.0.
    Please migrate to the new API before upgrading.
```

!!! warning "Deprecation Notice"

    This API will be removed in version 3.0.
    Please migrate to the new API before upgrading.

**Danger:**

```markdown
!!! danger "Data Loss Warning"
    This operation cannot be undone. Make sure you have
    a backup before proceeding.
```

!!! danger "Data Loss Warning"

    This operation cannot be undone. Make sure you have
    a backup before proceeding.

**Info:**

```markdown
!!! info
    This feature was introduced in version 2.5 and requires
    Go 1.21 or later.
```

!!! info

    This feature was introduced in version 2.5 and requires
    Go 1.21 or later.

**Example:**

```markdown
!!! example "Usage Example"
    ```go
    result := myFunction(input)
    fmt.Println(result)
    ```
```

!!! example "Usage Example"

    ```go
    result := myFunction(input)
    fmt.Println(result)
    ```

**Success:**

```markdown
!!! success "Build Complete"
    Your site has been successfully built and deployed.
```

!!! success "Build Complete"

    Your site has been successfully built and deployed.

**Bug:**

```markdown
!!! bug "Known Issue"
    There is a known issue with Safari where animations may flicker.
    A fix is planned for the next release.
```

!!! bug "Known Issue"

    There is a known issue with Safari where animations may flicker.
    A fix is planned for the next release.

**Quote:**

```markdown
!!! quote "Albert Einstein"
    Imagination is more important than knowledge. Knowledge is limited.
    Imagination encircles the world.
```

!!! quote "Albert Einstein"

    Imagination is more important than knowledge. Knowledge is limited.
    Imagination encircles the world.

**Abstract:**

```markdown
!!! abstract "Summary"
    This article covers the basics of static site generation,
    including configuration, templating, and deployment strategies.
```

!!! abstract "Summary"

    This article covers the basics of static site generation,
    including configuration, templating, and deployment strategies.

### Collapsible Admonitions

Use `???` for collapsed (closed by default) or `???+` for expanded (open by default):

**Input:**

```markdown
??? note "Click to expand"
    This content is hidden by default.
    Click the title to reveal it.

???+ tip "Expanded by default"
    This content is visible initially.
    Click to collapse.
```

**Live examples:**

??? note "Click to expand"

    This content is hidden by default.
    Click the title to reveal it.

???+ tip "Expanded by default"

    This content is visible initially.
    Click to collapse.

**Output:**

```html
<details class="admonition note">
  <summary class="admonition-title">Click to expand</summary>
  <p>This content is hidden by default.
  Click the title to reveal it.</p>
</details>

<details class="admonition tip" open>
  <summary class="admonition-title">Expanded by default</summary>
  <p>This content is visible initially.
  Click to collapse.</p>
</details>
```

### Nested Content in Admonitions

Admonitions can contain any Markdown content:

```markdown
!!! example "Complete Example"
    Here's a full working example:
    
    ```python
    def greet(name):
        return f"Hello, {name}!"
    
    print(greet("World"))
    ```
    
    **Output:**
    
    ```
    Hello, World!
    ```
    
    - Works with Python 3.6+
    - Requires no dependencies
```

**Live example:**

!!! example "Complete Example"

    Here's a full working example:
    
    ```python
    def greet(name):
        return f"Hello, {name}!"
    
    print(greet("World"))
    ```
    
    **Output:**
    
    ```
    Hello, World!
    ```
    
    - Works with Python 3.6+
    - Requires no dependencies

### Aside (Marginal Notes)

The `aside` type creates sidebar or marginal notes:

```markdown
!!! aside
    This is a marginal note that floats to the side.

!!! aside "Definition"
    A **static site generator** converts source files into static HTML.
```

!!! aside

    This is a marginal note that floats to the side.

!!! aside "Definition"

    A **static site generator** converts source files into static HTML.

Control positioning with inline modifiers:

```markdown
!!! aside inline
    Floats to the left of content.

!!! aside inline end
    Floats to the right of content (default).
```

See [Advanced Usage](../advanced-usage.md#admonitions-and-callouts) for styling and customization options.

---

## Wikilinks

Wikilinks provide an easy way to link between posts using `[[slug]]` syntax.

### Basic Syntax

**Input:**

```markdown
Check out [[getting-started]] for installation instructions.

See also: [[quickstart]], [[advanced-usage]]
```

**Live example:**

Check out [[getting-started]] for installation instructions.

See also: [[quickstart]], [[advanced-usage]]

**Output (when posts exist):**

```html
<p>Check out <a href="/getting-started/" class="wikilink">Getting Started</a> for installation instructions.</p>

<p>See also: <a href="/quickstart/" class="wikilink">Quickstart</a>, <a href="/advanced-usage/" class="wikilink">Advanced Usage</a></p>
```

### Custom Link Text

Use the pipe (`|`) to specify custom link text:

**Input:**

```markdown
Read the [[getting-started|installation guide]] first.

For more details, see [[advanced-usage|the advanced guide]].
```

**Live example:**

Read the [[getting-started|installation guide]] first.

For more details, see [[advanced-usage|the advanced guide]].

**Output:**

```html
<p>Read the <a href="/getting-started/" class="wikilink">installation guide</a> first.</p>

<p>For more details, see <a href="/advanced-usage/" class="wikilink">the advanced guide</a>.</p>
```

### Linking to Sections

Link to specific headings within posts:

```markdown
See [[feeds-guide#feed-configuration]] for feed setup.

Check the [[templates-guide#available-variables|template variables section]].
```

**Live example:**

See [[feeds-guide#feed-configuration]] for feed setup.

Check the [[templates-guide#available-variables|template variables section]].

### Wikilink Resolution

markata-go resolves wikilinks by:

1. Finding a post where `slug == link_target`
2. If found: renders as `<a href="{post.href}">{text}</a>`
3. If not found: leaves as literal `[[link]]` and warns

### Broken Link Handling

When a wikilink target doesn't exist:

**Input:**

```markdown
See [[nonexistent-post]] for more.
```

**Live example:**

See [[nonexistent-post]] for more.

**Output:**

```html
<p>See [[nonexistent-post]] for more.</p>
```

**Build warning:**

```
Warning: Broken wikilink in posts/my-post.md: [[nonexistent-post]] (post not found)
```

Enable strict mode to fail builds on broken links:

```toml
[markata-go]
strict_wikilinks = true
```

### Syntax Reference

| Syntax | Description | Output |
|--------|-------------|--------|
| `[[slug]]` | Basic link (auto-title) | Link with post title |
| `[[slug\|Text]]` | Custom text | Link with "Text" |
| `[[slug#section]]` | Section link | Link to heading anchor |
| `[[slug#section\|Text]]` | Section with text | Custom text to section |

---

## Table of Contents

markata-go can automatically generate a table of contents from your headings.

### Enabling TOC

Add `toc: true` to your frontmatter:

```yaml
---
title: "My Long Article"
toc: true
---

# Introduction

## Getting Started

### Prerequisites

### Installation

## Core Concepts

## Advanced Topics
```

### TOC Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `toc` | bool | `false` | Enable TOC generation |
| `toc_depth` | int | `2` | Maximum heading level (1-6) |
| `toc_min_items` | int | `2` | Minimum headings to show TOC |

```yaml
---
title: "Deep Dive Article"
toc: true
toc_depth: 3       # Include h1, h2, h3
toc_min_items: 3   # Only show if 3+ headings
---
```

### Generated TOC Structure

The TOC is available as `post.Extra.toc_html`:

**Example output:**

```html
<nav class="toc">
  <ul>
    <li><a href="#introduction">Introduction</a></li>
    <li>
      <a href="#getting-started">Getting Started</a>
      <ul>
        <li><a href="#prerequisites">Prerequisites</a></li>
        <li><a href="#installation">Installation</a></li>
      </ul>
    </li>
    <li><a href="#core-concepts">Core Concepts</a></li>
    <li><a href="#advanced-topics">Advanced Topics</a></li>
  </ul>
</nav>
```

### Using TOC in Templates

Include the TOC in your post template:

```django
{% if post.Extra.toc_html %}
<aside class="toc-sidebar">
    {{ post.Extra.toc_html|safe }}
</aside>
{% endif %}

<div class="post-content">
    {{ body|safe }}
</div>
```

See [Templates](./templates.md) for more template examples.

### Configuration

Configure TOC defaults in `markata-go.toml`:

```toml
[markata-go.toc]
enabled = true
default_depth = 2
min_items = 2
```

---

## Heading Anchors

markata-go automatically generates anchor IDs for all headings, enabling direct linking to sections.

### Auto-Generated IDs

Headings are automatically assigned IDs based on their text:

**Input:**

```markdown
## Hello World
## What's New?
## 2024 Updates
## FAQ
```

**Output:**

```html
<h2 id="hello-world">Hello World</h2>
<h2 id="whats-new">What's New?</h2>
<h2 id="2024-updates">2024 Updates</h2>
<h2 id="faq">FAQ</h2>
```

!!! tip "Try it"

    Every heading on this page has an auto-generated ID. Hover over any heading to see the anchor link, or try linking to [#auto-generated-ids](#auto-generated-ids).

### ID Generation Rules

| Heading | Generated ID |
|---------|--------------|
| `## Hello World` | `hello-world` |
| `## What's New?` | `whats-new` |
| `## C++ Tutorial` | `c-tutorial` |
| `## 100% Complete` | `100-complete` |
| `## Leading Space` | `leading-space` |

### Duplicate Handling

When multiple headings have the same text, numbers are appended:

**Input:**

```markdown
## FAQ
## FAQ
## FAQ
```

**Output:**

```html
<h2 id="faq">FAQ</h2>
<h2 id="faq-1">FAQ</h2>
<h2 id="faq-2">FAQ</h2>
```

### Custom IDs

Override the auto-generated ID with custom syntax:

**Input:**

```markdown
## My Section {#custom-id}

## Installation Guide {#install}

## Frequently Asked Questions {#faq-section}
```

**Output:**

```html
<h2 id="custom-id">My Section</h2>
<h2 id="install">Installation Guide</h2>
<h2 id="faq-section">Frequently Asked Questions</h2>
```

### Anchor Links

markata-go can add clickable anchor links to headings:

```html
<h2 id="my-section">
  My Section
  <a href="#my-section" class="heading-anchor">#</a>
</h2>
```

Configure anchor link appearance:

```toml
[markata-go.markdown.anchors]
enabled = true
position = "end"    # "start" or "end"
symbol = "#"        # or "link", "paragraph", etc.
```

### Linking to Headings

Link to headings using their IDs:

```markdown
See [the installation section](#installation) for setup instructions.

Jump to [FAQ](#faq) for common questions.
```

Or use wikilinks with section anchors:

```markdown
See [[getting-started#installation]] for setup.
```

---

## Footnotes

Footnotes let you add references without interrupting the flow of your content.

### Basic Syntax

**Input:**

```markdown
Here's a sentence with a footnote.[^1]

Another sentence with a named footnote.[^note]

[^1]: This is the first footnote content.

[^note]: This is a named footnote. It can be referenced
    multiple times and span multiple lines.
    
    It can even contain multiple paragraphs.
```

**Live example:**

Here's a sentence with a footnote.[^example-1]

Another sentence with a named footnote.[^example-note]

[^example-1]: This is the first footnote content.

[^example-note]: This is a named footnote. It can be referenced
    multiple times and span multiple lines.
    
    It can even contain multiple paragraphs.

**Output:**

```html
<p>Here's a sentence with a footnote.<sup id="fnref:1"><a href="#fn:1">1</a></sup></p>

<p>Another sentence with a named footnote.<sup id="fnref:note"><a href="#fn:note">2</a></sup></p>

<div class="footnotes">
  <hr>
  <ol>
    <li id="fn:1">
      <p>This is the first footnote content. <a href="#fnref:1">&#8617;</a></p>
    </li>
    <li id="fn:note">
      <p>This is a named footnote. It can be referenced
      multiple times and span multiple lines.</p>
      <p>It can even contain multiple paragraphs. <a href="#fnref:note">&#8617;</a></p>
    </li>
  </ol>
</div>
```

### Inline Footnotes

For short notes, use inline syntax:

```markdown
Here's an inline footnote.^[This is the footnote content inline.]
```

**Live example:**

Here's an inline footnote.^[This is the footnote content inline.]

### Footnote Best Practices

1. **Placement**: Footnote definitions can be anywhere in the document, but are typically placed at the end.

2. **Naming**: Use descriptive names for complex documents:
   ```markdown
   [^citation-smith-2024]: Smith, J. (2024). *The Article*. Journal.
   [^definition-ssr]: Server-Side Rendering: Generating HTML on the server.
   ```

3. **Multi-paragraph footnotes**: Indent continuation lines with 4 spaces:
   ```markdown
   [^long-note]: First paragraph of the footnote.
   
       Second paragraph must be indented.
       
       Third paragraph too.
   ```

---

## See Also

- [Templates](./templates.md) - Using TOC and content in templates
- [Advanced Usage](../advanced-usage.md#admonitions-and-callouts) - Advanced admonition customization
- [Dynamic Content](./dynamic-content.md) - Using Jinja in Markdown
- [Frontmatter](./frontmatter.md) - Frontmatter fields and metadata
- [Configuration](./configuration.md) - Markdown configuration options
