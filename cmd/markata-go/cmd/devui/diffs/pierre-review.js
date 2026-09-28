import { FileDiff, parseDiffFromFile, setLanguageOverride } from '@pierre/diffs';

const mounted = new WeakMap();

/**
 * Render a Markata source-fix preview with Pierre Diffs.
 *
 * The caller owns the surrounding review/apply controls. This adapter is only
 * responsible for visualizing the exact before/after text returned by the
 * existing preview API, so it cannot broaden or change the approved edit set.
 *
 * Serve fixes currently target Markdown source. Force that language explicitly
 * instead of relying on the filename so the production asset can eventually
 * vendor only the Markdown grammar plus the Pierre light/dark themes.
 */
export function renderPierreDiff(host, { path, before, after, diffStyle = 'split' }) {
  cleanupPierreDiff(host);

  const name = path || 'source.md';
  const fileDiff = parseDiffFromFile(
    { name, contents: before ?? '' },
    { name, contents: after ?? '' },
  );
  setLanguageOverride(fileDiff, 'markdown');

  const view = new FileDiff({
    diffStyle,
    theme: { light: 'pierre-light', dark: 'pierre-dark' },
  });

  view.render({ fileContainer: host, fileDiff });
  mounted.set(host, view);
  return view;
}

export function cleanupPierreDiff(host) {
  const view = mounted.get(host);
  if (!view) return;
  view.cleanUp();
  mounted.delete(host);
}
