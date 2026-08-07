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

REVISION = "c28e08582e7bd36751febb3391142a5eb18bbb34"
REPOSITORY = "https://github.com/google/fonts"

FAMILIES = {
    "inter": ("Inter", "ofl/inter", "Inter[opsz,wght].ttf", "OFL-1.1"),
    "inter-tight": ("Inter Tight", "ofl/intertight", "InterTight[wght].ttf", "OFL-1.1"),
    "ibm-plex-sans-condensed": ("IBM Plex Sans Condensed", "ofl/ibmplexsanscondensed", "IBMPlexSansCondensed-Regular.ttf", "OFL-1.1"),
    "ibm-plex-serif": ("IBM Plex Serif", "ofl/ibmplexserif", "IBMPlexSerif-Regular.ttf", "OFL-1.1"),
    "syne": ("Syne", "ofl/syne", "Syne[wght].ttf", "OFL-1.1"),
    "sedgwick-ave-display": ("Sedgwick Ave Display", "ofl/sedgwickavedisplay", "SedgwickAveDisplay-Regular.ttf", "OFL-1.1"),
    "permanent-marker": ("Permanent Marker", "apache/permanentmarker", "PermanentMarker-Regular.ttf", "Apache-2.0"),
    "rubik-dirt": ("Rubik Dirt", "ofl/rubikdirt", "RubikDirt-Regular.ttf", "OFL-1.1"),
    "victor-mono": ("Victor Mono", "ofl/victormono", "VictorMono[wght].ttf", "OFL-1.1"),
    "trade-winds": ("Trade Winds", "ofl/tradewinds", "TradeWinds-Regular.ttf", "OFL-1.1"),
    "ibm-plex-sans": ("IBM Plex Sans", "ofl/ibmplexsans", "IBMPlexSans[wdth,wght].ttf", "OFL-1.1"),
    "jetbrains-mono": ("JetBrains Mono", "ofl/jetbrainsmono", "JetBrainsMono[wght].ttf", "OFL-1.1"),
    "special-elite": ("Special Elite", "apache/specialelite", "SpecialElite-Regular.ttf", "Apache-2.0"),
    "bitter": ("Bitter", "ofl/bitter", "Bitter[wght].ttf", "OFL-1.1"),
    "crimson-pro": ("Crimson Pro", "ofl/crimsonpro", "CrimsonPro[wght].ttf", "OFL-1.1"),
    "courier-prime": ("Courier Prime", "ofl/courierprime", "CourierPrime-Regular.ttf", "OFL-1.1"),
    "fraunces": ("Fraunces", "ofl/fraunces", "Fraunces[SOFT,WONK,opsz,wght].ttf", "OFL-1.1"),
    "literata": ("Literata", "ofl/literata", "Literata[opsz,wght].ttf", "OFL-1.1"),
    "cormorant-garamond": ("Cormorant Garamond", "ofl/cormorantgaramond", "CormorantGaramond[wght].ttf", "OFL-1.1"),
    "ibm-plex-mono": ("IBM Plex Mono", "ofl/ibmplexmono", "IBMPlexMono-Regular.ttf", "OFL-1.1"),
    "manrope": ("Manrope", "ofl/manrope", "Manrope[wght].ttf", "OFL-1.1"),
    "instrument-serif": ("Instrument Serif", "ofl/instrumentserif", "InstrumentSerif-Regular.ttf", "OFL-1.1"),
    "geist-mono": ("Geist Mono", "ofl/geistmono", "GeistMono[wght].ttf", "OFL-1.1"),
    "bungee": ("Bungee", "ofl/bungee", "Bungee-Regular.ttf", "OFL-1.1"),
    "source-serif-4": ("Source Serif 4", "ofl/sourceserif4", "SourceSerif4[opsz,wght].ttf", "OFL-1.1"),
    "caveat-brush": ("Caveat Brush", "ofl/caveatbrush", "CaveatBrush-Regular.ttf", "OFL-1.1"),
    "finger-paint": ("Finger Paint", "ofl/fingerpaint", "FingerPaint-Regular.ttf", "OFL-1.1"),
    "rock-salt": ("Rock Salt", "apache/rocksalt", "RockSalt-Regular.ttf", "Apache-2.0"),
}

RANGES = {
    "display-core": ["U+0020-007E", "U+00A0", "U+00A2-00A5", "U+00A9", "U+00AE", "U+00B0", "U+00D7", "U+00F7", "U+2010-2027", "U+2030", "U+2032-2033", "U+2044", "U+20AC", "U+2122", "U+2190-2199"],
    "prose-core": ["U+0020-007E", "U+00A0-00FF", "U+0100-017F", "U+0180-024F", "U+1E00-1EFF", "U+2000-206F", "U+20A0-20CF", "U+2100-214F", "U+2190-21FF", "U+2150-218F"],
    "code-core": ["U+0020-007E", "U+00A0", "U+00AC", "U+00B0", "U+00B1", "U+00B7", "U+00D7", "U+00F7", "U+2010-2027", "U+2190-21FF", "U+2200-22FF", "U+2300-23FF", "U+2500-259F", "U+25A0-25FF"],
    "latin-ext": ["U+0180-024F", "U+1E00-1EFF", "U+2C60-2C7F", "U+A720-A7FF", "U+AB30-AB6F"],
}


def sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--google-fonts", type=Path, required=True)
    parser.add_argument("--catalog-root", type=Path, default=Path("internal/fontcatalog"))
    parser.add_argument("--lockfile", type=Path, default=Path("internal/fontcatalog/markata-fonts.lock.yaml"))
    args = parser.parse_args()

    lock = yaml.safe_load(args.lockfile.read_text())
    for family_id, (family, upstream_dir, source_name, license_id) in FAMILIES.items():
        family_dir = args.catalog_root / family_id
        source = args.google_fonts / upstream_dir / source_name
        license_path = family_dir / ("LICENSE.txt" if license_id == "Apache-2.0" else "OFL.txt")
        if not license_path.exists():
            license_path = family_dir / "OFL.txt"
        if not source.exists() or not license_path.exists():
            raise SystemExit(f"missing source or license for {family_id}")
        variable = "[" in source_name
        face_weight = [300, 900] if variable else [400, 400]
        tiers = {}
        tier_names = ("display-core", "latin-ext", "full") if family_id in {"finger-paint", "rock-salt"} else (*RANGES, "full")
        for tier in tier_names:
            output = family_dir / f"{family_id}-{tier}.woff2"
            if not output.exists():
                raise SystemExit(f"missing generated tier: {output}")
            entry = {"file": output.name, "profile": tier, "sha256": sha256(output), "bytes": output.stat().st_size}
            if tier != "full":
                entry["unicode_range"] = RANGES[tier]
            tiers[tier] = entry
        manifest = {
            "schema": "markata.font/v1", "id": family_id, "family": family, "scope": "builtin",
            "source": {"provider": "google-fonts", "repository": REPOSITORY, "revision": REVISION, "directory": upstream_dir, "files": {"normal": sha256(source)}},
            "license": {"id": license_id, "file": license_path.name, "sha256": sha256(license_path)},
            "faces": {"normal": {"style": "normal", "variable": variable, "weight": face_weight, "source_file": source_name}},
            "tiers": tiers,
        }
        (family_dir / "manifest.yaml").write_text(yaml.safe_dump(manifest, sort_keys=False))
        lock["sources"][family_id] = {
            "provider": "google-fonts", "family": family, "repository": REPOSITORY,
            "revision": REVISION, "directory": upstream_dir,
            "files": {"normal": {"source": source_name, "sha256": sha256(source)}},
            "license": {"id": license_id, "file": license_path.name, "sha256": sha256(license_path)},
        }
    args.lockfile.write_text(yaml.safe_dump(lock, sort_keys=False))


if __name__ == "__main__":
    main()
