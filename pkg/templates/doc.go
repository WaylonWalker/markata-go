// Package templates provides a Jinja2-like template engine for markata-go.
//
// The package wraps the pongo2 library, which implements Jinja2/Django-style
// template syntax in Go. It provides a template engine, context management,
// and custom filters for common operations.
//
// # Template Engine
//
// The Engine type manages template loading, caching, and rendering:
//
//	engine, err := templates.NewEngine("templates/")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Render a template file
//	html, err := engine.Render("post.html", ctx)
//
//	// Render a template string
//	html, err := engine.RenderString("{{ post.title }}", ctx)
//
// # Template Context
//
// The Context type holds data available to templates:
//
//	ctx := templates.NewContext(post, articleHTML, config)
//	ctx.Set("custom_key", "custom_value")
//
// Available variables in templates:
//   - post: The current post object
//   - body: Rendered article HTML
//   - config: Site configuration
//   - title, tags, slug, etc.: Shortcuts to post fields
//   - site_title, site_url, etc.: Shortcuts to config fields
//
// # Template Syntax
//
// The template syntax is compatible with Jinja2/Django:
//
//	Variables:      {{ post.title }}
//	Loops:          {% for tag in post.tags %}{{ tag }}{% endfor %}
//	Conditions:     {% if post.published %}...{% endif %}
//	Filters:        {{ post.title | upper }}
//	Blocks:         {% block content %}...{% endblock %}
//	Inheritance:    {% extends "base.html" %}
//	Includes:       {% include "partials/header.html" %}
//
// # Custom Filters
//
// The package provides custom filters for common operations:
//
// Date formatting:
//   - rss_date: Format for RSS feeds (RFC1123Z)
//   - atom_date: Format for Atom feeds (RFC3339)
//   - date_format: Custom date format
//
// String manipulation:
//   - slugify: Convert to URL-safe slug
//   - truncate: Truncate string with ellipsis
//   - truncatewords: Truncate by word count
//   - striptags: Remove HTML tags
//
// Collections:
//   - length: Length of string/slice
//   - first/last: First/last element
//   - join: Join with separator
//   - reverse: Reverse string/slice
//   - sort: Sort slice
//
// Other:
//   - default_if_none: Default value for nil/empty
//   - urlencode: URL-encode string
//   - absolute_url: Convert to absolute URL
//   - linebreaks/linebreaksbr: Convert newlines to HTML
//
// # Example Templates
//
// Base template (base.html):
//
//	<!DOCTYPE html>
//	<html>
//	<head>
//	    <title>{% block title %}{{ site_title }}{% endblock %}</title>
//	</head>
//	<body>
//	    {% block content %}{% endblock %}
//	</body>
//	</html>
//
// Post template (post.html):
//
//	{% extends "base.html" %}
//	{% block title %}{{ post.title }} | {{ site_title }}{% endblock %}
//	{% block content %}
//	<article>
//	    <h1>{{ post.title }}</h1>
//	    <time>{{ post.date | date_format:"January 2, 2006" }}</time>
//	    <div class="tags">
//	        {% for tag in post.tags %}
//	        <span class="tag">{{ tag }}</span>
//	        {% endfor %}
//	    </div>
//	    <div class="content">{{ body | safe }}</div>
//	</article>
//	{% endblock %}
package templates
