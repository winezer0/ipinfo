import hashlib
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import fetch_test_assets as assets


class FetchTests(unittest.TestCase):
    def test_existing_file_requires_checksum(self):
        with tempfile.TemporaryDirectory() as directory:
            target = Path(directory) / "test.xdb"
            target.write_bytes(b"database")
            entry = {"name": target.name, "sha256": hashlib.sha256(b"database").hexdigest()}
            with patch.object(assets.urllib.request, "urlopen") as request:
                self.assertIn("verified", assets.fetch(entry, target.parent))
                request.assert_not_called()
            target.write_bytes(b"corrupt")
            self.assertFalse(assets.matches(target, entry))

    def test_bad_download_does_not_replace_existing_file(self):
        import io
        with tempfile.TemporaryDirectory() as directory:
            target = Path(directory) / "test.xdb"
            target.write_bytes(b"old")
            entry = {"name": target.name, "url": "https://example.invalid/test", "sha256": hashlib.sha256(b"expected").hexdigest()}
            with patch.object(assets.urllib.request, "urlopen", side_effect=lambda *a, **k: io.BytesIO(b"wrong")), patch.object(assets.time, "sleep"):
                with self.assertRaisesRegex(ValueError, "checksum mismatch"):
                    assets.fetch(entry, target.parent)
            self.assertEqual(target.read_bytes(), b"old")
            self.assertEqual(list(target.parent.glob("*.part")), [])

    def test_rejects_escaping_filename(self):
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaises(ValueError):
                assets.fetch({"name": "../escape"}, Path(directory))


if __name__ == "__main__":
    unittest.main()
