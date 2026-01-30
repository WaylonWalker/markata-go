---
title: "Keyboard Navigation"
description: "Complete guide to keyboard shortcuts for navigating markata-go sites"
date: 2024-01-15
published: true
slug: /docs/guides/keyboard-navigation/
tags:
  - documentation
  - accessibility
  - keyboard
  - navigation
---

# Keyboard Navigation

markata-go sites include comprehensive keyboard shortcuts for power users and those who prefer keyboard navigation. All shortcuts are designed to be intuitive, following conventions from popular tools like Vim and GitHub.

> **Accessibility:** Shortcuts respect user preferences and can be disabled. They never interfere with form inputs or accessibility tools.

## Quick Reference

Press `?` at any time to see the full shortcuts help modal.

---

## Scrolling

Navigate through page content using vim-style shortcuts:

| Shortcut | Action |
|----------|--------|
| `j` | Scroll down (~2 lines) |
| `k` | Scroll up (~2 lines) |
| `d` | Scroll half-page down |
| `u` | Scroll half-page up |
| `g g` | Scroll to top of page |
| `Shift+G` | Scroll to bottom of page |

**Note:** On feed/list pages with multiple posts, `j`/`k` will navigate between posts instead of scrolling.

---

## Feed Navigation

When viewing a page with multiple posts (index, archive, tag pages), keyboard navigation lets you browse through posts:

| Shortcut | Action |
|----------|--------|
| `j` or `Down Arrow` | Highlight next post |
| `k` or `Up Arrow` | Highlight previous post |
| `Enter` or `o` | Open highlighted post |
| `Shift+O` | Open highlighted post in new tab |

The highlighted post is visually indicated with an outline. Press `Escape` to clear the highlight.

---

## Go-To Shortcuts

GitHub-style two-key sequences for quick navigation:

| Shortcut | Action |
|----------|--------|
| `g h` | Go to home page |
| `g s` | Focus search input |

Press the first key (`g`), then the second key within 800ms.

---

## Utility Shortcuts

| Shortcut | Action |
|----------|--------|
| `/` | Focus search input |
| `Cmd/Ctrl+K` | Focus search (alternative) |
| `y y` | Copy current URL to clipboard |
| `[` | Go to previous page (pagination) |
| `]` | Go to next page (pagination) |
| `?` | Show shortcuts help modal |
| `Escape` | Close modals, clear highlight, blur inputs |

---

## Accessibility Features

### Respects User Preferences

- **Reduced Motion:** When `prefers-reduced-motion` is enabled, smooth scrolling is disabled for instant navigation.
- **Input Context:** Shortcuts are automatically disabled when typing in text inputs, textareas, or contenteditable elements.

### Disable Shortcuts

If keyboard shortcuts interfere with your workflow or assistive technology:

1. Press `?` to open the shortcuts modal
2. Click "Disable Shortcuts"

This preference is saved in your browser and persists across visits.

### Re-enable Shortcuts

1. Press `?` (this still works even when shortcuts are disabled)
2. Click "Enable Shortcuts"

---

## Customization

### For Site Owners

Keyboard shortcuts are part of the default theme. If you need to customize or extend them:

1. **Override the JavaScript:** Copy `shortcuts.js` from the theme to your `static/js/` directory and modify as needed.

2. **Add to the modal:** Override the `partials/shortcuts-modal.html` template to add your custom shortcuts.

3. **Styling:** The keyboard highlight class `.kb-highlighted` can be customized in your CSS:

```css
/* Custom highlight style */
.kb-highlighted {
  outline: 3px solid var(--color-primary);
  outline-offset: 4px;
  background-color: rgba(var(--color-primary-rgb), 0.1);
}
```

### JavaScript API

The shortcuts system exposes functions for programmatic use:

```javascript
// Show toast notification
markataShortcuts.showToast('Custom message');

// Programmatic scrolling
markataShortcuts.smoothScroll(100);  // Scroll down 100px
markataShortcuts.smoothScrollToTop();
markataShortcuts.smoothScrollToBottom();

// Copy URL
markataShortcuts.copyUrlToClipboard();

// Check if shortcuts are disabled
if (!markataShortcuts.areShortcutsDisabled()) {
  // Shortcuts are enabled
}

// Modal control
markataShortcuts.showShortcutsModal();
markataShortcuts.hideShortcutsModal();
```

---

## Browser Compatibility

Keyboard shortcuts work in all modern browsers:

- Chrome/Edge 80+
- Firefox 75+
- Safari 13+

The clipboard API (`yy` to copy URL) requires a secure context (HTTPS or localhost).

---

## See Also

- [Themes Guide](/docs/guides/themes/) - Theme customization including keyboard shortcuts
- [Configuration Guide](/docs/guides/configuration/) - Site configuration options
- [Accessibility](/docs/guides/accessibility/) - Accessibility features and best practices
