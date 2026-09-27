# Nginx-native redirects

The built-in redirects plugin can generate nginx-native redirect rules from the same `static/_redirects` file used for static HTML fallbacks.

## Default behavior

Given:

```text
/old-post /new-post
/go/docs https://docs.example.com/start
```

A normal build writes:

```text
public/
├── redirects.conf
├── old-post/index.html
└── go/docs/index.html
```

`redirects.conf` contains exact-match nginx rules:

```nginx
location = "/old-post" {
    return 301 "/new-post";
}

location = "/go/docs" {
    return 301 "https://docs.example.com/start";
}
```

HTML fallback pages remain enabled by default so the same build artifact continues to work on static hosts that do not consume nginx configuration.

## Configuration

```toml
[markata-go.redirects]
redirects_file = "static/_redirects"
redirect_template = "templates/redirect.html" # optional
html_fallback = true
```

Set `html_fallback = false` when the deployment is guaranteed to use the generated nginx rules and you do not want per-path HTML redirect pages:

```toml
[markata-go.redirects]
html_fallback = false
```

The nginx output itself is always generated when the redirects file contains at least one supported redirect.

## Include from nginx

Include the generated file from inside the nginx `server` block that serves the built site:

```nginx
server {
    listen 80;
    server_name example.com;
    root /usr/share/nginx/html;

    include /usr/share/nginx/html/redirects.conf;

    location / {
        try_files $uri $uri/ /index.html =404;
    }
}
```

The include path must match the deployed `output_dir`. If your deployment uses release directories or a `current` symlink, point nginx at the active release's `redirects.conf` just as you do for the site root.

## Supported rules

The native output intentionally follows the redirects plugin's existing parser:

- source paths must start with `/`
- destinations must be `/absolute-path`, `http://...`, or `https://...`
- comments and blank lines are ignored
- malformed rules are ignored
- wildcard rules containing `*` are skipped
- extra fields such as platform-specific status codes are currently ignored
- generated nginx redirects use HTTP `301`

This keeps nginx output and HTML fallback behavior aligned. Use hand-written nginx configuration or the hosting provider's native redirect system for wildcard, conditional, or status-code-specific rules.

## Fast builds

As with the existing HTML redirect generation, the redirects plugin is skipped in `--fast` mode. Run a normal build before publishing or validating `redirects.conf`.
