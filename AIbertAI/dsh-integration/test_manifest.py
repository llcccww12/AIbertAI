import contextlib
import importlib.util
import io
import json
import os
from pathlib import Path
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location("dsh_manifest", Path(__file__).with_name("manifest.py"))
m = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(m)


class Args:
    pass


class ManifestTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name) / "dsh"
        (self.root / "bin").mkdir(parents=True)
        (self.root / "app/apps/cli/lib").mkdir(parents=True)
        (self.root / "bin/node").write_bytes(b"node fixture")
        (self.root / m.ENTRY).write_bytes(b"sdk fixture")
        self.lock = Path(self.tmp.name) / "pnpm-lock.yaml"
        self.lock.write_text("synthetic lockfile")
        self.patch = Path(__file__).parent / "patches/experimental-invocation-id.patch"
        self.args = Args()
        self.args.root = str(self.root)
        self.args.patch = str(self.patch)
        self.args.lockfile = str(self.lock)
        self.args.archive_sha256 = m.SOURCE_ARCHIVE_SHA256
        self.args.node_version = "v25.9.0"
        self.args.target = "darwin-arm64"

    def create(self):
        with contextlib.redirect_stdout(io.StringIO()) as out:
            m.create(self.args)
        self.args.manifest_sha256 = out.getvalue().strip()
        return self.args.manifest_sha256

    def verify(self):
        with contextlib.redirect_stdout(io.StringIO()):
            m.verify(self.args)

    def test_round_trip_and_tampering(self):
        (self.root / "app/alias").symlink_to("apps/cli/lib/bin.js")
        self.create()
        self.verify()
        (self.root / m.ENTRY).write_bytes(b"changed")
        with self.assertRaisesRegex(ValueError, "inventory mismatch"):
            self.verify()

    def test_manifest_requires_external_pin(self):
        self.create()
        payload = json.loads((self.root / m.MANIFEST).read_text())
        payload["target"] = "linux-amd64"
        (self.root / m.MANIFEST).write_text(json.dumps(payload))
        with self.assertRaisesRegex(ValueError, "digest mismatch"):
            self.verify()

    def test_added_file_and_escaping_symlink_rejected(self):
        self.create()
        (self.root / "extra").write_text("added")
        with self.assertRaisesRegex(ValueError, "inventory mismatch"):
            self.verify()
        (self.root / "extra").unlink()
        (self.root / "escape").symlink_to("../../outside")
        with self.assertRaises((ValueError, OSError)):
            self.verify()

    def test_local_configuration_cannot_be_staged(self):
        (self.root / "app/.npmrc").write_text("fixture")
        with self.assertRaisesRegex(ValueError, "local environment"):
            self.create()

    def test_wrong_source_and_patch_are_rejected(self):
        self.args.archive_sha256 = "0" * 64
        with self.assertRaisesRegex(ValueError, "source archive"):
            self.create()
        self.args.archive_sha256 = m.SOURCE_ARCHIVE_SHA256
        self.args.patch = str(self.lock)
        with self.assertRaisesRegex(ValueError, "patch digest"):
            self.create()


if __name__ == "__main__":
    unittest.main()
