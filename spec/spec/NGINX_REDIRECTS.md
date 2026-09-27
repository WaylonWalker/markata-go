# Nginx Redirect Output Specification

This document extends the static redirects specification with nginx-native output.

## Goals

The redirects plugin MUST be able to use the existing redirects source file to produce direct nginx redirects while preserving the existing static HTML redirect behavior by default.

## Inputs

The source file and parsing rules are the same as described in `REDIRECTS.md`.

Supported rules have:

- an absolute source path beginning with `/`
- a destination beginning with `/`, `http://`, or `https://`
- no wildcard `*`

Comments, blank lines, malformed entries, wildcard entries, and unsupported destinations are ignored.

## Generated nginx output

For every supported redirect, a normal build MUST write an exact-match nginx location rule to:

```text
<output_dir>/redirects.conf
```

Example source:

```text
/old /new
/go/docs https://docs.example.com/start
```

Expected nginx output shape:

```nginx
location = "/old" {
    return 301 "/new";
}

location = "/go/docs" {
    return 301 "https://docs.example.com/start";
}
```

Requirements:

- generated rules MUST use exact-match `location =` blocks
- generated rules MUST use HTTP status `301`
- generated source and destination values MUST be quoted and escaped so user-authored redirect text cannot break nginx configuration syntax
- the file MUST be suitable for inclusion inside an nginx `server` block
- generation MUST use the same accepted redirect set as the HTML fallback so native and fallback behavior do not diverge
- if there are no supported redirects, the plugin MAY omit `redirects.conf`
- missing redirects files MUST continue to be treated as a no-op
- fast builds MUST continue to skip redirects generation

## HTML fallback

HTML fallback generation MUST remain enabled by default for backward compatibility and static-host portability.

The plugin MUST recognize:

```toml
[markata-go.redirects]
html_fallback = false
```

When `html_fallback = false`:

- `redirects.conf` MUST still be generated
- per-source HTML redirect pages MUST NOT be generated
- custom redirect templates need not be loaded because they are unused

When `html_fallback` is absent or `true`, existing HTML redirect behavior MUST remain unchanged.

## Cache behavior

The redirect cache key MUST include the effective HTML fallback setting. Changing `html_fallback` without changing `_redirects` must not incorrectly reuse output from the previous configuration.

If a custom HTML redirect template participates in generation, its content MUST continue to influence the cache key.

## Compatibility

This feature does not add wildcard, conditional, or per-rule status-code semantics. Extra fields in provider-specific `_redirects` formats remain ignored by the shared parser.

Deployments that need those features should use provider-native routing or hand-written nginx configuration in addition to the generated file.

## Deployment

Consumers MAY include the generated file directly from the built site artifact:

```nginx
server {
    root /usr/share/nginx/html;
    include /usr/share/nginx/html/redirects.conf;
}
```

Deployments using release directories or an active-release symlink SHOULD include the file from the same active release that supplies the served HTML.
