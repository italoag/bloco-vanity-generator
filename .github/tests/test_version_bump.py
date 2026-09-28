import os
from pathlib import Path
import subprocess
import tempfile
import textwrap
import unittest


WORKFLOW = Path(__file__).resolve().parents[1] / "workflows" / "version-bump.yml"


def step_script(name):
    lines = WORKFLOW.read_text().splitlines()
    start = lines.index(f"      - name: {name}") + 1
    for index in range(start, len(lines)):
        if lines[index] == "        run: |":
            body = []
            for line in lines[index + 1:]:
                if line and not line.startswith("          "):
                    break
                body.append(line)
            return textwrap.dedent("\n".join(body))
        if lines[index].startswith("      - name:"):
            break
    raise AssertionError(f"Missing run script for {name}")


class VersionSelectionTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.repo = Path(self.temp.name)
        self.env = {
            **os.environ,
            "GIT_AUTHOR_NAME": "Version Test",
            "GIT_AUTHOR_EMAIL": "version-test@example.invalid",
            "GIT_COMMITTER_NAME": "Version Test",
            "GIT_COMMITTER_EMAIL": "version-test@example.invalid",
            "GITHUB_OUTPUT": str(self.repo / "outputs"),
        }
        self.git("init", "--quiet")
        self.git("commit", "--quiet", "--allow-empty", "-m", "initial")

    def git(self, *args):
        return subprocess.run(
            ["git", *args], cwd=self.repo, env=self.env,
            check=True, text=True, capture_output=True,
        ).stdout.strip()

    def selected_tag(self):
        subprocess.run(
            ["bash", "-e", "-o", "pipefail", "-c", step_script("Get last tag")],
            cwd=self.repo, env=self.env, check=True, text=True, capture_output=True,
        )
        return (self.repo / "outputs").read_text().strip().removeprefix("tag=")

    def test_empty_repository_defaults_to_zero(self):
        self.assertEqual(self.selected_tag(), "v0.0.0")

    def test_nearest_tag_is_not_highest_version(self):
        for version in ("v0.3.0", "v0.4.0", "v0.5.0", "v0.6.0"):
            self.git("tag", version)
        self.git("commit", "--quiet", "--allow-empty", "-m", "later")
        self.git("tag", "-a", "v0.2.2", "-m", "latest published release")
        self.assertEqual(self.git("describe", "--tags", "--abbrev=0"), "v0.2.2")
        self.assertEqual(self.selected_tag(), "v0.6.0")

    def test_versions_are_sorted_numerically(self):
        for version in ("v0.9.9", "v0.10.0", "v0.10.2", "v0.10.10"):
            self.git("tag", version)
        self.assertEqual(self.selected_tag(), "v0.10.10")

    def test_only_strict_stable_tags_count(self):
        for version in ("v1.2.3", "v9.0.0-rc.1", "v9.0.0+build", "v9.0.0.1", "v09.0.0", "v9.00.0", "v9.0.00", "release-9.0.0"):
            self.git("tag", version)
        self.assertEqual(self.selected_tag(), "v1.2.3")

    def test_nonstable_tags_only_default_to_zero(self):
        self.git("tag", "v1.0.0-rc.1")
        self.assertEqual(self.selected_tag(), "v0.0.0")

    def test_unreachable_tag_reserves_its_version(self):
        first = self.git("rev-parse", "HEAD")
        self.git("commit", "--quiet", "--allow-empty", "-m", "other history")
        self.git("tag", "v2.0.0")
        self.git("checkout", "--quiet", "--detach", first)
        self.git("tag", "v1.0.0")
        self.assertEqual(self.selected_tag(), "v2.0.0")


class TagPublicationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.sha = "a" * 40
        self.calls = self.root / "git-calls"
        self.output = self.root / "outputs"
        self.env = {
            **os.environ,
            "PATH": f"{self.root}{os.pathsep}{os.environ['PATH']}",
            "GITHUB_REPOSITORY": "owner/wallet",
            "GITHUB_OUTPUT": str(self.output),
            "RELEASE_SHA": self.sha,
            "CURRENT_SHA": self.sha,
            "NEW_TAG": "v1.2.3",
            "BUMP": "patch",
            "REASON": "fixture",
            "GIT_CALLS": str(self.calls),
            "API_FAIL": "0",
            "PUSH_FAIL": "0",
        }
        scripts = {
            "gh": '#!/bin/sh\n[ "$API_FAIL" = "0" ] || exit 1\n'
                  '[ "$1" = "api" ] && [ "$2" = "repos/owner/wallet/git/ref/heads/main" ] || exit 2\n'
                  'printf "%s\\n" "$CURRENT_SHA"\n',
            "git": '#!/bin/sh\nprintf "%s\\n" "$*" >> "$GIT_CALLS"\n'
                   'if [ "$1" = "push" ] && [ "$PUSH_FAIL" = "1" ]; then exit 1; fi\n',
        }
        for name, content in scripts.items():
            executable = self.root / name
            executable.write_text(content)
            executable.chmod(0o700)

    def execute(self):
        return subprocess.run(
            ["bash", "-e", "-o", "pipefail", "-c", step_script("Create tag")],
            cwd=self.root, env=self.env, text=True, capture_output=True,
        )

    def test_unchanged_branch_creates_and_pushes_tag(self):
        result = self.execute()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.calls.read_text().splitlines(), [
            "tag -a v1.2.3 -m Release v1.2.3 (patch): fixture", "push origin v1.2.3",
        ])
        self.assertEqual(self.output.read_text(), "created=true\n")

    def test_branch_advanced_does_not_write_tag_or_dispatch(self):
        self.env["CURRENT_SHA"] = "b" * 40
        self.assertEqual(self.execute().returncode, 0)
        self.assertFalse(self.calls.exists())
        self.assertFalse(self.output.exists())

    def test_api_failure_does_not_write_tag(self):
        self.env["API_FAIL"] = "1"
        self.assertNotEqual(self.execute().returncode, 0)
        self.assertFalse(self.calls.exists())
        self.assertFalse(self.output.exists())

    def test_failed_push_does_not_enable_release_dispatch(self):
        self.env["PUSH_FAIL"] = "1"
        self.assertNotEqual(self.execute().returncode, 0)
        self.assertIn("push origin v1.2.3", self.calls.read_text())
        self.assertFalse(self.output.exists())


if __name__ == "__main__":
    unittest.main()
