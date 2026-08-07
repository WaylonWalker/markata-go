#!/usr/bin/env -S uv run --with pyyaml python
"""Regenerate identical fontpack comparison pages from pages/post/test.md."""

from __future__ import annotations

import argparse
from pathlib import Path

import yaml

START = "<!-- fontpack-demo-links:start -->"
END = "<!-- fontpack-demo-links:end -->"


def split_frontmatter(text: str) -> tuple[str, str]:
    if not text.startswith("---\n"):
        raise ValueError("demo template must start with YAML frontmatter")
    end = text.find("\n---\n", 4)
    if end < 0:
        raise ValueError("demo template has no closing frontmatter delimiter")
    return text[: end + 5], text[end + 5 :].lstrip("\n")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--catalog", type=Path, default=Path("markata-fontpacks.yaml"))
    parser.add_argument("--directory", type=Path, default=Path("pages/post"))
    args = parser.parse_args()

    catalog = yaml.safe_load(args.catalog.read_text())
    packs = sorted(catalog["fontpacks"])
    template_path = args.directory / "test.md"
    links = [START, "## Font pack comparisons", ""]
    links.extend(f"- [{catalog['fontpacks'][pack]['name']}](/test-{pack}/)" for pack in packs)
    links.extend([END, ""])
    template_text = template_path.read_text()
    frontmatter, body = split_frontmatter(template_text)
    if START in body and END in body:
        _, remainder = body.split(START, 1)
        _, body = remainder.split(END, 1)
    body = body.lstrip("\n")
    canonical_body = "\n".join(links) + "\n" + body
    template_text = frontmatter + "\n" + canonical_body
    template_path.write_text(template_text)

    # Keep every generated page byte-identical except for slug and fontpack.
    for pack in packs:
        slug = f"test-{pack}"
        page_frontmatter = frontmatter.replace("slug: test\n", f"slug: {slug}\nfontpack: {pack}\n", 1)
        (args.directory / f"{slug}.md").write_text(page_frontmatter + "\n" + canonical_body)


if __name__ == "__main__":
    main()
