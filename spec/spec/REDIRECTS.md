# Redirects Specification

Redirects support URL migrations and content reorganization from one `_redirects` source file. A normal build produces nginx-native rules and, by default, portable HTML fallback pages.

## Goals

The redirects plugin MUST:

- read one configured redirects source file
- use one accepted rule set for nginx and HTML output
- emit an nginx include suitable for direct server-side redirects
- preserve HTML redirect pages by default for static-host compatibility
- avoid leaving stale nginx rules active when redirect input is cleared or removed
- never silently overwrite a user-managed nginx redirect file

## Configuration

```toml
[markata-go.redirects]
redirects_file = "static/_redirects"
redirect_template = "templates/redirect.html" # optional
html_fallback = true                            # default
```

`html_fallback = false` disables creation of per-path HTML redirect pages. It does not disable nginx output.

## Redirects file format

The source format is whitespace-separated and compatible with the simple subset commonly used by Netlify and Cloudflare Pages:

```text
# comments are ignored
/old-path /new-path
/go/docs https://docs.example.com/start
```

A supported rule MUST have:

- a source beginning with `/`
- a destination beginning with `/`, `http://`, or `https://`
- no wildcard `*`

Blank lines, comments, malformed entries, wildcard entries, and unsupported destinations are ignored. Extra provider-specific fields are currently ignored.

## Nginx output

When the redirects source file exists, a normal build MUST write:

```text
<output_dir>/redirects.conf
```

Each supported redirect becomes an exact-match location using HTTP `301`:

```nginx
location = "/old-path" {
    return 301 "/new-path";
}

location = "/go/docs" {
    return 301 "https://docs.example.com/start";
}
```

Requirements:

- generated rules MUST use exact-match `location =` blocks
- generated rules MUST use HTTP `301`
- source and destination strings MUST be quoted and escaped for nginx syntax
- literal `$` characters MUST NOT become nginx variable interpolation
- the generated file MUST be valid for inclusion inside an nginx `server` block
- generated files MUST carry a recognizable ownership header
- an existing but empty redirects source MUST produce a valid header-only `redirects.conf`, clearing any previously generated native rules
- if the redirects source is removed, a normal build MUST remove a stale `redirects.conf` only when the file is recognizable as markata-go generated output
- a user-authored `redirects.conf` MUST NOT be deleted merely because the redirects source is absent
- if `_redirects` exists and `<output_dir>/redirects.conf` already exists without the markata-go ownership header, the build MUST fail rather than overwrite that file

## HTML fallback

HTML fallback pages remain enabled by default.

For a rule:

```text
/old-path /new-path
```

the fallback is written to:

```text
<output_dir>/old-path/index.html
```

The default page MUST provide:

- an immediate meta refresh
- a canonical link to the destination
- a visible fallback link
- a human-readable moved-page message

When `redirect_template` is configured, the custom template is used. If it cannot be read or parsed, the implementation MAY warn and fall back to the built-in template.

On a normal build, a supported source rule MUST recreate its HTML fallback if
that generated page is missing from the output directory.

When `html_fallback = false`, new fallback pages MUST NOT be generated. Because fallback pages share normal site paths and the plugin does not maintain a safe ownership manifest for arbitrary custom templates, changing or removing redirect rules MAY leave old HTML fallback files in an incremental output directory. Users who need those files removed SHOULD run a clean build.

## Fast builds

Fast builds MUST skip redirect generation. Production or deployment validation that depends on `redirects.conf` MUST use a normal build.

## Nginx deployment

The generated file may be included from a server block:

```nginx
server {
    root /usr/share/nginx/html;
    include /usr/share/nginx/html/redirects.conf;

    location / {
        try_files $uri $uri/ /index.html =404;
    }
}
```

Nginx reads included configuration as part of loading its configuration. Replacing a release directory or changing a `current` symlink does not by itself apply changed redirect rules to an already-running nginx process. Deployments that rebuild redirects in place MUST reload or restart nginx after the active redirect configuration changes.

`redirects.conf` is a generated-output path when `_redirects` is present. Sites that intentionally manage their own nginx include at that path MUST rename or remove it before enabling generated nginx redirects; the plugin MUST fail rather than silently clobber it.

## Processing and conflicts

The redirects plugin runs late in the write stage. For HTML fallbacks, it creates the source directory and writes `index.html`. A source that resolves to an existing file path is skipped rather than replacing that file.

Nginx output is generated from the full accepted rule set even when an individual HTML fallback cannot be written, so operators SHOULD treat build warnings about fallback conflicts as deployment diagnostics.

## Limitations

This feature does not currently provide:

- wildcard redirects
- conditional redirects
- per-rule status codes
- query-string matching semantics

Extra `_redirects` fields such as `302` are ignored and generated nginx redirects use `301`. Deployments that need richer semantics should use provider-native routing or hand-written nginx configuration.

## Error handling

| Condition | Behavior |
|---|---|
| Missing redirects source | Remove only stale markata-go-generated `redirects.conf`; otherwise no-op |
| Empty redirects source | Write a header-only `redirects.conf` |
| Existing user-managed `redirects.conf` while `_redirects` exists | Fail rather than overwrite it |
| Malformed or unsupported rule | Skip the rule |
| Nginx output write failure | Fail the redirects write stage |
| HTML fallback write failure | Warn and continue with other fallbacks |
| Custom HTML template read/parse failure | Warn and use the built-in template |

## See also

- [CONFIG.md](./CONFIG.md) - configuration system
- [LIFECYCLE.md](./LIFECYCLE.md) - build lifecycle
- [DEFAULT_PLUGINS.md](./DEFAULT_PLUGINS.md) - built-in plugin set
- `docs/guides/nginx-redirects.md` - user-facing nginx setup
