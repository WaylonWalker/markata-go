# Pins Feed Specification

## Overview

Markata-Go provides a zero-configuration visual link feed at `/pins/` for public link posts.

## Selection

A pin MUST be generated only from a post that:

- is published;
- is not draft, skipped, or private; and
- has a non-empty string `link` frontmatter field.

Private post metadata MUST NOT be exposed by the pins feed.

## Ordering

Pins with dates MUST be sorted newest first. Undated pins MUST follow dated pins. Ties SHOULD be deterministic.

## Card metadata

Each pin SHOULD expose, when available:

- semantic post title;
- external `link` destination;
- destination hostname;
- local post URL for notes/commentary;
- authored description;
- publication date; and
- representative image metadata.

The external destination and local post URL MUST remain distinguishable in the rendered UI.

## Image aliases

The built-in implementation checks common image frontmatter aliases in this order:

`image`, `cover`, `cover_image`, `featured_image`, `thumbnail`, `og_image`, `social_image`, `hero_image`.

Images are optional.

## Rendering

The default theme MUST provide a responsive visual layout suitable for mixed image aspect ratios and text-only cards. Cards MUST remain keyboard accessible and MUST clearly identify links that navigate to an external destination.

Theme authors MAY override `pins.html`.

## Empty sites

When no posts qualify, Markata-Go MUST NOT create `/pins/index.html`.
