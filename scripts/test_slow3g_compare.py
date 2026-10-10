"""Check network asset-response validation without starting Chromium."""

import unittest

from slow3g_compare import asset_response_problem


class AssetResponseTests(unittest.TestCase):
    def test_valid_assets(self):
        for kind, mime in (
            ("Stylesheet", "text/css"),
            ("Stylesheet", "text/css; charset=utf-8"),
            ("Script", "application/javascript"),
            ("Script", "text/javascript"),
            ("Image", "image/svg+xml"),
            ("Image", "image/jpeg"),
            ("Font", "font/woff2"),
            ("Font", "application/octet-stream"),
        ):
            with self.subTest(kind=kind, mime=mime):
                self.assertIsNone(asset_response_problem(kind, mime, 200))

    def test_html_is_not_a_valid_asset(self):
        for kind in ("Stylesheet", "Script", "Image", "Font"):
            with self.subTest(kind=kind):
                self.assertIsNotNone(asset_response_problem(kind, "text/html", 200))

    def test_missing_assets_are_rejected(self):
        self.assertEqual(asset_response_problem("Stylesheet", "text/css", 404), "HTTP 404")

    def test_html_document_is_not_a_subresource(self):
        self.assertIsNone(asset_response_problem("Document", "text/html", 200))


if __name__ == "__main__":
    unittest.main()
