# Plugin Creation

Use this topic only after checking whether the task can be solved in config, content, templates, CSS, feeds, or an existing built-in feature.

## Decision Rule

Reach for a plugin when the change needs new build-stage behavior, derived content, external data, or output generation that templates alone cannot express.

## Lifecycle Stages

Markata-go runs plugins through these stages:

- Configure
- Validate
- Glob
- Load
- Transform
- Render
- Collect
- Write
- Cleanup

## Matching Work To A Stage

- config parsing or plugin setup -> Configure
- config sanity checks -> Validate
- discover files -> Glob
- parse new content types -> Load
- modify markdown or derived fields before render -> Transform
- generate HTML or wrap rendered content -> Render
- build feeds, archives, or aggregate data -> Collect
- write output files -> Write
- flush caches or cleanup -> Cleanup

## Guidance

- Prefer matching the work to one lifecycle stage instead of spreading logic across many stages.
- Follow the existing plugin style in the current codebase.
- Update spec and user docs alongside plugin behavior.
- Validate whether the task belongs in a site-local customization or in `markata-go` upstream.
- If the task is site-only and mostly presentational, templates are usually cheaper than plugin code.

## Site Repo Vs Upstream Repo

Use this rule:

- if the task changes only one site's content model, templates, output choices, or local integrations, it probably belongs in the site repo
- if the task adds generally useful build behavior, new CLI support, reusable config, or a new built-in plugin, it probably belongs in `markata-go` upstream

## Before Writing A Plugin

Check these lower-cost options first:

1. config change
2. frontmatter field
3. feed definition
4. layout config
5. template or partial override
6. CSS change

If all of those fail to express the behavior, then plugin work is justified.

## Start With

- `markata-go explain plugins`
- plugin development docs for the current version of markata-go
- the closest built-in plugin to the behavior you need
- current `hooks` and `disabled_hooks` settings in site config

## Common Plugin Use Cases

- fetch and cache remote data
- add derived post fields
- create new output files or collection pages
- transform markdown or rendered HTML before publishing

## Operational Debugging Tips

- if a plugin seems missing, check `hooks = ["default"]` and `disabled_hooks`
- use `markata-go build -v` when you need plugin-stage error detail
- isolate plugin issues by temporarily disabling the suspected hook rather than changing templates first
