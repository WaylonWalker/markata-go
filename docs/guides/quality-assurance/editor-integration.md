---
title: "Editor Integration"
description: "Use markata-go lint output with your favorite editor's quickfix and problem matching features"
date: 2026-01-24
published: true
tags:
  - guides
  - quality-assurance
  - lint
  - vim
  - neovim
  - vscode
---

# Editor Integration

The `markata-go lint` command outputs issues in a format compatible with many editors' error navigation features. This guide shows how to integrate lint output with popular editors.

## Vim / Neovim Quickfix

Vim and Neovim have built-in quickfix support for navigating errors and warnings.

### Basic Usage

```vim
" Set markata-go as the make program
:set makeprg=markata-go\ lint

" Run lint and populate quickfix
:make docs/**/*.md

" Open quickfix window
:cwindow

" Navigate errors
:cnext      " Go to next error
:cprev      " Go to previous error
:cfirst     " Go to first error
:clast      " Go to last error
```

### Custom Error Format

For perfect parsing of markata-go lint output, add this to your `~/.vimrc` or `init.vim`:

```vim
" Add markata-go lint error format
set errorformat+=%f:
set errorformat+=\ \ %tarning\ [line\ %l\\,\ col\ %c]:\ %m
set errorformat+=\ \ %trror\ [line\ %l\\,\ col\ %c]:\ %m
```

### One-liner Command

Quick way to run lint and populate quickfix:

```vim
:cgetexpr system('markata-go lint docs/**/*.md')
:cwindow
```

### Neovim Lua Configuration

For Neovim users with Lua config:

```lua
-- ~/.config/nvim/lua/lint.lua or init.lua

-- Function to run markata-go lint
local function markata_lint()
  vim.cmd('cgetexpr system("markata-go lint docs/**/*.md")')
  vim.cmd('cwindow')
end

-- Create a user command
vim.api.nvim_create_user_command('MarkataLint', markata_lint, {})

-- Optional: keybinding
vim.keymap.set('n', '<leader>ml', markata_lint, { desc = 'Run markata-go lint' })
```

### Lint Current File

To lint only the current file:

```vim
:cgetexpr system('markata-go lint ' . expand('%'))
```

Or in Lua:

```lua
local function lint_current_file()
  local file = vim.fn.expand('%')
  vim.cmd('cgetexpr system("markata-go lint ' .. file .. '")')
  vim.cmd('cwindow')
end
```

## VS Code Integration

VS Code can use tasks with problem matchers to integrate with markata-go lint.

### Task Configuration

Create `.vscode/tasks.json` in your project:

```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Lint Markdown",
      "type": "shell",
      "command": "markata-go lint docs/**/*.md",
      "group": "build",
      "problemMatcher": {
        "owner": "markata-go",
        "fileLocation": ["relative", "${workspaceFolder}"],
        "pattern": [
          {
            "regexp": "^(.+):$",
            "file": 1
          },
          {
            "regexp": "^\\s+(warning|error)\\s+\\[line\\s+(\\d+),\\s+col\\s+(\\d+)\\]:\\s+(.+)$",
            "severity": 1,
            "line": 2,
            "column": 3,
            "message": 4,
            "loop": true
          }
        ]
      }
    },
    {
      "label": "Lint Current File",
      "type": "shell",
      "command": "markata-go lint ${file}",
      "group": "build",
      "problemMatcher": {
        "owner": "markata-go",
        "fileLocation": ["relative", "${workspaceFolder}"],
        "pattern": [
          {
            "regexp": "^(.+):$",
            "file": 1
          },
          {
            "regexp": "^\\s+(warning|error)\\s+\\[line\\s+(\\d+),\\s+col\\s+(\\d+)\\]:\\s+(.+)$",
            "severity": 1,
            "line": 2,
            "column": 3,
            "message": 4,
            "loop": true
          }
        ]
      }
    }
  ]
}
```

### Running the Task

1. Press `Ctrl+Shift+B` (or `Cmd+Shift+B` on Mac)
2. Select "Lint Markdown" from the task list
3. Errors appear in the Problems panel
4. Click on errors to jump to the file and line

### Keyboard Shortcut

Add a keybinding in `keybindings.json`:

```json
{
  "key": "ctrl+shift+l",
  "command": "workbench.action.tasks.runTask",
  "args": "Lint Markdown"
}
```

## Emacs Integration

Emacs can use `compile-mode` to run lint and navigate errors.

### Basic Usage

```elisp
;; Run markata-go lint
M-x compile RET markata-go lint docs/**/*.md RET

;; Navigate errors
M-g n  ; next-error
M-g p  ; previous-error
```

### Custom Compilation Regexp

Add to your Emacs config:

```elisp
(add-to-list 'compilation-error-regexp-alist-alist
             '(markata-go
               "^\\(.*\\):\n  \\(warning\\|error\\) \\[line \\([0-9]+\\), col \\([0-9]+\\)\\]: \\(.*\\)$"
               1 3 4 2))

(add-to-list 'compilation-error-regexp-alist 'markata-go)
```

### Flycheck Integration

For on-the-fly linting with Flycheck:

```elisp
(flycheck-define-checker markata-go
  "A markdown linter using markata-go."
  :command ("markata-go" "lint" source)
  :error-patterns
  ((warning line-start (file-name) ":\n"
            "  warning [line " line ", col " column "]: " (message) line-end)
   (error line-start (file-name) ":\n"
          "  error [line " line ", col " column "]: " (message) line-end))
  :modes (markdown-mode gfm-mode))

(add-to-list 'flycheck-checkers 'markata-go)
```

## Sublime Text Integration

Create a build system for markata-go lint.

### Build System Configuration

Create `Packages/User/markata-go.sublime-build`:

```json
{
  "cmd": ["markata-go", "lint", "$file"],
  "working_dir": "$project_path",
  "file_regex": "^(.+):\n  (warning|error) \\[line (\\d+), col (\\d+)\\]: (.*)$",
  "selector": "text.html.markdown"
}
```

### Usage

1. Open a Markdown file
2. Press `Ctrl+B` (or `Cmd+B` on Mac)
3. Use `F4` / `Shift+F4` to navigate errors

## Tips and Best Practices

### Lint on Save

Most editors support running commands on file save. Configure your editor to run `markata-go lint` on the current file when saving for immediate feedback.

**Vim/Neovim autocommand:**

```vim
augroup MarkataLint
  autocmd!
  autocmd BufWritePost *.md cgetexpr system('markata-go lint ' . expand('%'))
augroup END
```

**VS Code setting (settings.json):**

```json
{
  "emeraldwalk.runonsave": {
    "commands": [
      {
        "match": "\\.md$",
        "cmd": "markata-go lint ${file}"
      }
    ]
  }
}
```

### Project-wide Lint

For CI/CD or pre-commit hooks, lint all files:

```bash
markata-go lint
```

This uses the glob patterns from your `markata-go.toml` configuration.

### Exit Codes

The lint command uses standard exit codes:

| Code | Meaning |
|------|---------|
| 0 | No issues found |
| 1 | Issues found |
| 2 | Error running lint |

Use these in scripts:

```bash
if markata-go lint; then
  echo "All good!"
else
  echo "Issues found, check the output"
fi
```
