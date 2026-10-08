"""Canonical extraction and revision tests for the offline byte probe."""
import hashlib
import pathlib
import tempfile
import unittest
from http.server import BaseHTTPRequestHandler, HTTPServer
from threading import Thread
from urllib.error import HTTPError
from urllib.request import Request, urlopen

from probe import CanonicalRanges, projection


def document(text, header="Global header"):
    return ('<!doctype html><html lang="en"><head><title>Example</title>'
            '<link rel="canonical" href="https://example.com/a/">'
            '<script src="/route.js" defer></script></head>'
            '<body class="article"><header>' + header + '</header>'
            '<div id="view-transition-page"><aside>Sidebar</aside><main>'
            '<div>' + text + '</div></main></div>'
            '<footer class="site-footer">Route state</footer></body></html>')


class ProbeTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = pathlib.Path(self.temp.name)
        (self.root / 'route.js').write_text('example()')

    def project(self, text="Article", header="Global header"):
        return projection(document(text, header), self.root)

    def test_exact_region_and_canonical_metadata(self):
        original = document('A &amp; B')
        artifact, _, _, _ = self.project('A &amp; B')
        source = CanonicalRanges(original)
        source.feed(original)
        target = CanonicalRanges(artifact)
        target.feed(artifact)
        self.assertEqual(original[source.start:source.end], artifact[target.start:target.end])
        self.assertEqual(source.title, target.title)
        self.assertEqual(source.canonical, target.canonical)
        self.assertEqual(source.bodyattrs, target.bodyattrs)
        self.assertEqual(source.htmlattrs, {k: v for k, v in target.htmlattrs.items() if not k.startswith('data-nav-')})
        self.assertIn(hashlib.sha256(b'example()').hexdigest(), artifact)

    def test_route_locality_and_shell_separation(self):
        before = self.project()
        other = self.project('Unrelated route')
        self.assertEqual(before, self.project())
        changed = self.project('Changed article')
        self.assertNotEqual(before[1], changed[1])
        self.assertEqual(before[2], changed[2])
        self.assertEqual(other, self.project('Unrelated route'))
        shell_change = self.project(header='New global structure')
        self.assertNotEqual(before[2], shell_change[2])

    def test_asset_change_changes_route_revision(self):
        before = self.project()
        (self.root / 'route.js').write_text('newExample()')
        after = self.project()
        self.assertNotEqual(before[1], after[1])
        self.assertEqual(before[2], after[2])

    def test_missing_region_fails(self):
        with self.assertRaises(ValueError):
            projection(document('Article').replace('view-transition-page', 'unknown'), self.root)

    def test_real_http_etag_validation(self):
        # Illustrates a hosting adapter, not a header provided by static files.
        state = {'projection': self.project()}

        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                artifact, revision, _, _ = state['projection']
                etag = '"' + revision + '"'
                self.send_response(304 if self.headers.get('If-None-Match') == etag else 200)
                self.send_header('ETag', etag)
                self.send_header('Cache-Control', 'no-cache')
                self.end_headers()
                if self.headers.get('If-None-Match') != etag:
                    self.wfile.write(artifact.encode())

            def log_message(self, *args):
                pass

        server = HTTPServer(('127.0.0.1', 0), Handler)
        thread = Thread(target=server.serve_forever, daemon=True)
        thread.start()
        try:
            url = f'http://127.0.0.1:{server.server_port}/a/_nav.html'
            with urlopen(url) as response:
                etag = response.headers['ETag']
                self.assertEqual(response.status, 200)
                self.assertEqual(response.read().decode(), state['projection'][0])
            with self.assertRaises(HTTPError) as unchanged:
                urlopen(Request(url, headers={'If-None-Match': etag}))
            self.assertEqual(unchanged.exception.code, 304)
            unchanged.exception.close()
            for change in ['content', 'asset', 'shell']:
                if change == 'content':
                    state['projection'] = self.project('New article')
                elif change == 'asset':
                    (self.root / 'route.js').write_text('newExample()')
                    state['projection'] = self.project('New article')
                else:
                    state['projection'] = self.project('New article', 'New header')
                with urlopen(Request(url, headers={'If-None-Match': etag})) as response:
                    self.assertEqual(response.status, 200)
                    self.assertNotEqual(response.headers['ETag'], etag)
                    etag = response.headers['ETag']
        finally:
            server.shutdown()
            server.server_close()
            thread.join()


if __name__ == '__main__':
    unittest.main()
