
# Feed view contract

## Built-in views

The canonical built-in feed presentation identifiers are `default`, `simple`, and `calendar`.
When no view list is configured, all three are available. `default` is mandatory; `simple` and
`calendar` may be removed globally through `feed_defaults.views` or overridden per feed with
`feeds[].views`.

## Rendering and generation

View controls MUST only link to enabled presentations. Calendar assets and its hidden source
payload MUST NOT be emitted when `calendar` is disabled. Simple HTML MUST NOT be generated when
`simple` is disabled, even if the `simple_html` output-format flag is otherwise enabled. Feed
view availability participates in incremental output hashing so configuration changes rebuild
affected feeds.

## Validation

Explicit view lists MUST contain `default`. Unknown view identifiers MUST fail validation with
the supported identifiers in the diagnostic. Omitted lists inherit; explicitly empty lists are
invalid rather than meaning "inherit".
