# Deterministic SVG profile v1

Compiled assets use UTF-8, no BOM, explicit basic geometry, stable attribute
ordering, and fixed decimal precision. Canonical assets may contain only local
geometry and masks. Scripts, event handlers, `foreignObject`, animation,
external resources, runtime fonts, and arbitrary filters or turbulence are not
allowed. Custom artwork is sanitized and ingested before it enters a bundle.
