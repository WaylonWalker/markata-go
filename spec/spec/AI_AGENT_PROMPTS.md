# AI Agent Prompt Templates Specification

## Overview

The repository includes reusable prompt templates and a generator script for
OpenCode-based GPT-5.4 maintenance workflows.

The generator produces:

- one multi-issue prompt for a supplied issue set
- one single-issue prompt for each supplied issue

The prompt content is template-driven so maintainers can revise the prompt text
without rewriting Python logic.

## Inputs

The generator MUST accept one or more GitHub issue numbers.

The generator SHOULD accept:

- a repository override in `OWNER/REPO` form
- an output directory override
- a worktree root hint for generated single-issue prompts
- a base branch hint for generated single-issue prompts
- an option to also write the rendered content back to the canonical prompt
  filenames in the repository root

## GitHub Metadata

The generator MUST resolve each issue number to:

- issue number
- issue title
- issue URL

The generator uses the GitHub CLI for this lookup.

## Templates

The repository stores prompt templates as text files with placeholder tokens.

Required templates:

- `continue-prompt-gpt-54.template.txt`
- `continue-prompt-gpt-54-single-issue.template.txt`

The generator MUST replace placeholders using the fetched issue metadata and
derived execution hints.

## Output Files

For a supplied issue set, the generator MUST create:

- `continue-prompt-gpt-54-issues-<issue-list>.txt`

For each supplied issue, the generator MUST create:

- `continue-prompt-gpt-54-issue-<issue-number>.txt`

When root prompt writing is enabled, the generator SHOULD also write:

- `continue-prompt-gpt-54.txt`
- `continue-prompt-gpt-54-single-issue.txt` when exactly one issue is supplied

## Derived Hints

For single-issue prompts, the generator SHOULD derive a suggested branch name
from the issue title.

If the title uses a conventional prefix such as `feat(...)` or `fix(...)`, the
branch name SHOULD preserve that prefix.

The generator SHOULD derive a suggested worktree path from the configured
worktree root and the suggested branch name.
