#!/usr/bin/env -S uv run --with pyyaml python
"""Generate built-in family manifests and lock entries from google/fonts.

This maintenance script does not run during site builds. It expects a pinned
google/fonts checkout and already-generated WOFF2 tiers in the catalog.
"""

from __future__ import annotations

import argparse
import hashlib
from pathlib import Path

import yaml

REPOSITORY = "https://github.com/google/fonts"


def sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def validate_tiers(family_id: str, tiers: dict[str, dict]) -> None:
    """Reject distinct generated subsets that contain exactly the same bytes."""
    seen: dict[str, tuple[str, list[str]]] = {}
    for name, tier in tiers.items():
        digest = tier["sha256"]
        ranges = tier.get("unicode_range", [])
        previous = seen.get(digest)
        if previous and previous[1] != ranges:
            raise SystemExit(
                f"{family_id}: tiers {previous[0]!r} and {name!r} have "
                "different unicode ranges but identical content"
            )
        seen[digest] = (name, ranges)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--google-fonts", type=Path, required=True)
    parser.add_argument("--catalog-root", type=Path, default=Path("internal/fontcatalog"))
    parser.add_argument("--lockfile", type=Path, default=Path("internal/fontcatalog/markata-fonts.lock.yaml"))
    args = parser.parse_args()

    catalog = yaml.safe_load((args.catalog_root / "markata-fontpacks.yaml").read_text())
    lock = yaml.safe_load(args.lockfile.read_text())
    profiles = catalog["subset_profiles"]
    revision = lock["revision"]
    repository = lock.get("repository", REPOSITORY)
    for family_id in sorted(catalog["font_sources"]):
        locked = lock["sources"].get(family_id)
        if not locked:
            raise SystemExit(f"{family_id}: missing source in lockfile")
        family = catalog["font_sources"][family_id]["family"]
        upstream_dir = locked["directory"]
        family_dir = args.catalog_root / family_id
        source_name = locked["files"]["normal"]["source"]
        source = args.google_fonts / upstream_dir / source_name
        license_id = locked["license"]["id"]
        license_path = family_dir / locked["license"]["file"]
        if not license_path.exists():
            license_path = family_dir / "OFL.txt"
        if not source.exists() or not license_path.exists():
            raise SystemExit(f"missing source or license for {family_id}")
        for name, file in locked["files"].items():
            actual = sha256(args.google_fonts / upstream_dir / file["source"])
            if actual != file["sha256"]:
                raise SystemExit(f"{family_id}: lock hash mismatch for source file {name}")
        variable = "[" in source_name
        face_weight = [300, 900] if variable else [400, 400]
        tiers = {}
        tier_names = sorted(p for p in profiles if (family_dir / f"{family_id}-{p}.woff2").exists())
        if "full" not in tier_names:
            tier_names.append("full")
        for tier in tier_names:
            output = family_dir / f"{family_id}-{tier}.woff2"
            if not output.exists():
                raise SystemExit(f"missing generated tier: {output}")
            entry = {"file": output.name, "profile": tier, "sha256": sha256(output), "bytes": output.stat().st_size}
            if tier != "full":
                entry["unicode_range"] = profiles[tier]["unicode"]
            tiers[tier] = entry
        validate_tiers(family_id, tiers)
        manifest = {
            "schema": "markata.font/v1", "id": family_id, "family": family, "scope": "builtin",
            "source": {"provider": locked["provider"], "repository": repository, "revision": revision, "directory": upstream_dir, "files": {name: file["sha256"] for name, file in locked["files"].items()}},
            "license": {"id": license_id, "file": license_path.name, "sha256": sha256(license_path)},
            "faces": {"normal": {"style": "normal", "variable": variable, "weight": face_weight, "source_file": source_name}},
            "tiers": tiers,
        }
        (family_dir / "manifest.yaml").write_text(yaml.safe_dump(manifest, sort_keys=False))
        lock["sources"][family_id] = {
            "provider": locked["provider"], "family": family, "repository": repository,
            "revision": revision, "directory": upstream_dir,
            "files": {name: {"source": file["source"], "sha256": sha256(args.google_fonts / upstream_dir / file["source"])} for name, file in locked["files"].items()},
            "license": {"id": license_id, "file": license_path.name, "sha256": sha256(license_path)},
        }
    args.lockfile.write_text(yaml.safe_dump(lock, sort_keys=False))


if __name__ == "__main__":
    main()
