---
title: "Complex Test Post with Many Features"
description: "A comprehensive post for testing all markdown features and rendering performance"
date: 2024-01-15T10:30:00Z
published: true
draft: false
template: post.html
slug: complex-benchmark-post
tags:
  - benchmark
  - testing
  - performance
  - go
  - markdown
author: "Test Author"
category: "Performance Testing"
---

# Complex Test Post with Many Features

This post contains various markdown elements to test rendering performance comprehensively.

## Code Blocks

### Go Code

```go
package main

import (
    "fmt"
    "os"
)

func main() {
    fmt.Println("Hello, World!")
    os.Exit(0)
}

func fibonacci(n int) int {
    if n <= 1 {
        return n
    }
    return fibonacci(n-1) + fibonacci(n-2)
}
```

### Python Code

```python
def hello():
    """A simple hello function."""
    print("Hello, World!")

class Calculator:
    def add(self, a, b):
        return a + b

    def multiply(self, a, b):
        return a * b

if __name__ == "__main__":
    hello()
    calc = Calculator()
    print(calc.add(2, 3))
```

### JavaScript Code

```javascript
function hello() {
    console.log("Hello, World!");
}

const fibonacci = (n) => {
    if (n <= 1) return n;
    return fibonacci(n - 1) + fibonacci(n - 2);
};

class MyClass {
    constructor(name) {
        this.name = name;
    }

    greet() {
        return `Hello, ${this.name}!`;
    }
}

hello();
```

## Tables

| Feature | Description | Status |
|---------|-------------|--------|
| Tables | Markdown tables | Supported |
| Code Blocks | Syntax highlighting | Supported |
| Lists | Ordered and unordered | Supported |
| Blockquotes | Quote formatting | Supported |
| Links | Internal and external | Supported |
| Images | Image embedding | Supported |

### Complex Table

| Column 1 | Column 2 | Column 3 | Column 4 | Column 5 |
|:---------|:--------:|:---------|:--------:|----------:|
| Left | Center | Left | Center | Right |
| Data 1 | Data 2 | Data 3 | Data 4 | Data 5 |
| More | More | More | More | More |
| Even more | data | in | this | table |

## Blockquotes

> This is a blockquote with some important information
> that spans multiple lines for emphasis.
>
> It can contain **bold** and *italic* text as well.

### Nested Blockquotes

> Level 1 quote
> > Level 2 nested quote
> > > Level 3 deeply nested quote

## Lists

### Unordered List

- First item
- Second item with **bold**
- Third item with `code`
- Fourth item with [link](https://example.com)
  - Nested item 1
  - Nested item 2
    - Deeply nested item
- Fifth item

### Ordered List

1. First numbered item
2. Second numbered item
3. Third numbered item
   1. Nested numbered 1
   2. Nested numbered 2
4. Fourth numbered item

### Task List

- [x] Completed task
- [ ] Incomplete task
- [x] Another completed task
- [ ] Another incomplete task

## Links and Images

Here's an [external link](https://example.com) and an [internal link](/docs/guides/).

![Example image](https://example.com/image.png "Image title")

## Inline Elements

This paragraph contains `inline code`, **bold text**, *italic text*, ~~strikethrough~~, and ***bold italic*** text.

You can also use <sub>subscript</sub> and <sup>superscript</sup> text.

## Horizontal Rules

---

Above and below are horizontal rules.

***

Another style of horizontal rule.

## Footnotes

Here's a sentence with a footnote[^1].

And another one with a different footnote[^2].

[^1]: This is the first footnote.
[^2]: This is the second footnote with more content.

## Definition Lists

Term 1
: Definition for term 1

Term 2
: Definition for term 2
: Another definition for term 2

## Math (if supported)

Inline math: $E = mc^2$

Block math:

$$
\frac{d}{dx}\left( \int_{a}^{x} f(t)\,dt\right) = f(x)
$$

## Final Section

This concludes the complex benchmark post. It contains:

- Multiple code blocks with different languages
- Various table formats
- Nested quotes and lists
- Task lists
- Links and images
- Inline formatting
- Footnotes
- Definition lists
- Math expressions

This ensures comprehensive testing of the markdown rendering pipeline.
