// semantic-release versions the module from the conventional commit messages (ADR-015). Its tag,
// vX.Y.Z, is the Go module's version: the Go module proxy fetches it from GitHub. The release writes
// its version into version.go, and @semantic-release/git commits it, so that the tagged commit
// carries it.
export default {
  branches: ["main", { name: "beta", prerelease: true }, { name: "alpha", prerelease: true }],
  plugins: [
    "@semantic-release/commit-analyzer",
    "@semantic-release/release-notes-generator",
    ["@semantic-release/changelog", { changelogFile: "CHANGELOG.md" }],
    [
      "@semantic-release/exec",
      {
        prepareCmd:
          "printf '// DO NOT EDIT.\\n// Managed via automated CI/CD in release.config.js.\\n\\npackage smarter\\n\\n// Version is the SDK'\"'\"'s version.\\nconst Version = \"%s\"\\n' '${nextRelease.version}' > version.go",
      },
    ],
    "@semantic-release/github",
    [
      "@semantic-release/git",
      {
        assets: ["CHANGELOG.md", "version.go"],
        message: "chore(release): ${nextRelease.version} [skip ci]\n\n${nextRelease.notes}",
      },
    ],
  ],
};
