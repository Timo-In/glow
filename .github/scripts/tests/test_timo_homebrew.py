import subprocess
import tempfile
import unittest
from pathlib import Path


SCRIPT = Path(__file__).resolve().parents[1] / "timo-generate-homebrew-formula.sh"


class HomebrewFormulaTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.version = "3.0.0-timo.2"
        self.tag = "v" + self.version
        self.checksums = self.root / "checksums.txt"
        self.output = self.root / "Formula" / "glow-timo.rb"
        self.artifacts = [
            f"glow-timo_{self.version}_{platform}_{arch}.tar.gz"
            for platform in ("Darwin", "Linux")
            for arch in ("arm64", "x86_64")
        ]
        self.checksums.write_text(
            "".join(f"{str(i) * 64}  {name}\n" for i, name in enumerate(self.artifacts, 1))
        )

    def generate(self, version=None, tag=None):
        return subprocess.run(
            [str(SCRIPT), version or self.version, tag or self.tag, str(self.checksums), str(self.output)],
            capture_output=True,
            text=True,
        )

    def test_generates_all_platforms_with_matching_checksums(self):
        result = self.generate()
        self.assertEqual(result.returncode, 0, result.stderr)
        formula = self.output.read_text()
        for i, name in enumerate(self.artifacts, 1):
            self.assertIn(f"/releases/download/{self.tag}/{name}", formula)
            self.assertIn(f'sha256 "{str(i) * 64}"', formula)
        self.assertIn(f'version "{self.version}"', formula)
        self.assertIn('bin.install "glow"', formula)

    def test_rejects_nonfork_versions(self):
        for version in ("3.0.0", "3.0.0-timo", "3.0.0-timo.2/unsafe"):
            with self.subTest(version=version):
                self.assertNotEqual(self.generate(version=version).returncode, 0)
        self.assertFalse(self.output.exists())

    def test_rejects_tag_version_mismatch(self):
        self.assertNotEqual(self.generate(tag="v3.0.0-timo.1").returncode, 0)
        self.assertFalse(self.output.exists())

    def test_rejects_missing_checksums_file(self):
        self.checksums.unlink()
        self.assertNotEqual(self.generate().returncode, 0)
        self.assertFalse(self.output.exists())

    def test_rejects_missing_platform_checksum(self):
        lines = self.checksums.read_text().splitlines(keepends=True)
        self.checksums.write_text("".join(lines[:-1]))
        self.assertNotEqual(self.generate().returncode, 0)
        self.assertFalse(self.output.exists())

    def test_rejects_invalid_or_duplicate_checksums(self):
        original = self.checksums.read_text()
        self.checksums.write_text(original.replace("1" * 64, "bad-checksum"))
        self.assertNotEqual(self.generate().returncode, 0)
        self.checksums.write_text(original + original.splitlines(keepends=True)[0])
        self.assertNotEqual(self.generate().returncode, 0)
        self.assertFalse(self.output.exists())


if __name__ == "__main__":
    unittest.main()
