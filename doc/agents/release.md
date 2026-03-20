# Release Process

## 1. Update Release Notes

Edit `Release.md` in the project root with the changes for this version:

```markdown
## Features
* ...

## Improvements
* ...

## Fixes
* ...
```

This file is used by GoReleaser as the GitHub Release body.

## 2. Bump Version

Update the version string in `pkg/util/version/version.go`:

```go
var version = "0.X.0"
```

Commit and push to `dev`:

```bash
git add pkg/util/version/version.go Release.md
git commit -m "bump version to vX.Y.Z"
git push origin dev
```

## 3. Merge dev → master

Create a PR from `dev` to `master`:

```bash
gh pr create --base master --head dev --title "bump version"
```

Wait for CI to pass, then merge using **merge commit** (not squash).

## 4. Tag the Release

```bash
git checkout master
git pull origin master
git tag -a vX.Y.Z -m "bump version"
git push origin vX.Y.Z
```

## 5. Trigger GoReleaser

Manually trigger the `goreleaser` workflow in GitHub Actions:

```bash
gh workflow run goreleaser --ref master
```

GoReleaser will:
1. Run `package.sh` to cross-compile all platforms and create archives
2. Create a GitHub Release with all packages, using `Release.md` as release notes

## Key Files

| File | Purpose |
|------|---------|
| `pkg/util/version/version.go` | Version string |
| `Release.md` | Release notes (read by GoReleaser) |
| `.goreleaser.yml` | GoReleaser config |
| `package.sh` | Cross-compile and packaging script |
| `.github/workflows/goreleaser.yml` | GitHub Actions workflow (manual trigger) |

## Versioning

- Minor release: `v0.X.0`
- Patch release: `v0.X.Y` (e.g., `v0.62.1`)
