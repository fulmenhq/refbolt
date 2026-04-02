# Release Checklist

Standard checklist for refbolt releases to ensure consistency and quality.

## Pre-Release Phase

### Version Planning

- [ ] Feature briefs completed in productbook
- [ ] All planned features implemented and tested
- [ ] Breaking changes documented
- [ ] Version number decided (semantic versioning: MAJOR.MINOR.PATCH)

### Code Quality

- [ ] All tests passing: `make test`
- [ ] Code formatted: `make fmt`
- [ ] Lint checks clean: `make lint`
- [ ] Application builds: `make build`
- [ ] Manual smoke tests completed:
  - [ ] `./bin/refbolt version`
  - [ ] `./bin/refbolt sync --help`
  - [ ] `./bin/refbolt sync --all` (with test config)

### Documentation

- [ ] `README.md` reviewed and updated
- [ ] `docs/development.md` reflects any workflow changes
- [ ] `docs/providers/README.md` updated for new providers
- [ ] `docs/ARCHITECTURE.md` updated for structural changes

### Dependencies

- [ ] `go.mod` dependencies reviewed
- [ ] `go mod tidy` executed
- [ ] No security vulnerabilities in dependencies
- [ ] License audit clean: `make license-audit`

## Release Preparation

### Version Updates

- [ ] Update VERSION file: `make version-set VERSION=x.y.z`
- [ ] Search for hardcoded version references
- [ ] Update docker/container image tags (if applicable)

### Git Hygiene

- [ ] All changes committed
- [ ] Commit messages follow attribution standard (see AGENTS.md)
- [ ] No uncommitted changes: `git status` clean

### Final Validation

- [ ] Fresh clone test: Clone repo fresh, run `make build && make test`
- [ ] CI passes on main (format-check + build-test)

## Release Execution

### Environment Variables

The following variables must be set before signing. Store them in a credentials file outside the repo (e.g., a shell script you source) to keep signing keys out of version control.

| Variable               | Purpose                          | Example                                                  |
| ---------------------- | -------------------------------- | -------------------------------------------------------- |
| `REFBOLT_GPG_HOMEDIR`  | GnuPG home directory for signing | `~/vault/fulmenhq-gpg`                                   |
| `REFBOLT_PGP_KEY_ID`   | PGP signing key fingerprint      | `448A539320A397AF!`                                      |
| `REFBOLT_MINISIGN_KEY` | Path to minisign private key     | `~/vault/fulmenhq-minisign/fulmenhq-release-signing.key` |
| `REFBOLT_MINISIGN_PUB` | Path to minisign public key      | `~/vault/fulmenhq-minisign/fulmenhq-release-signing.pub` |
| `REFBOLT_VERSION_TAG`  | Release tag for this release     | `v0.2.0`                                                 |

The first four are stable across releases. `REFBOLT_VERSION_TAG` is set per release so the credentials file does not need to change.

```bash
# Source your credentials file, then set the release tag
source <your-credentials-file>
export REFBOLT_VERSION_TAG=v<version>
```

### Tag and Push

Create the tag and push it first — this triggers the release workflow which builds artifacts and creates a draft GitHub Release.

- [ ] Create annotated git tag: `git tag -a v<version> -m "Release v<version>"`
- [ ] Verify tag: `git tag -v v<version>`
- [ ] Push commits: `git push origin main`
- [ ] Push tag: `git push origin v<version>`
- [ ] Wait for release workflow to complete (check GitHub Actions)
- [ ] Verify draft GitHub Release appears with CI-built artifacts

### Sign and Upload Provenance

After the draft release is created by CI, download its artifacts locally for signing. Follow the Fulmen "manifest-only" provenance pattern: sign checksum manifests (not individual binaries), ship public keys with the release.

- [ ] (Recommended) Download CI-built artifacts and (re)generate manifests:

  ```bash
  make release-clean
  make release-download
  make release-checksums
  make release-verify-checksums
  ```

  This path uses GitHub Release artifacts built in CI and avoids local build drift. `RELEASE_TAG` is read from `REFBOLT_VERSION_TAG` automatically.

- [ ] (Alternative) Build artifacts locally (if CI build is unavailable):

  ```bash
  make release-build
  make release-verify-checksums
  ```

- [ ] Sign manifests (minisign required; PGP optional):

  ```bash
  # Ensure GPG can prompt for passphrase in this terminal
  export GPG_TTY="$(tty)"
  gpg-connect-agent updatestartuptty /bye

  make release-sign
  ```

- [ ] Export public keys: `make release-export-keys`
- [ ] Verify exported keys are public-only: `make release-verify-keys`
- [ ] Copy release notes: `make release-notes`
- [ ] Upload provenance assets: `make release-upload`
  - For fully manual release (no CI artifacts): `make release-upload-all`

### Publish

- [ ] Review the draft release on GitHub (verify artifacts, notes, signatures)
- [ ] Publish the release

### Homebrew

Update the formula in `fulmenhq/homebrew-tap` (sibling repo at `../homebrew-tap`):

- [ ] Update `Formula/refbolt.rb`: version, URLs, and SHA256 checksums from `dist/release/SHA256SUMS`
- [ ] Run `make precommit` in the tap repo to validate (RuboCop + goneat)
- [ ] Commit and push to `homebrew-tap/main`
- [ ] Verify: `brew update && brew upgrade refbolt && refbolt version`

### Scoop (Windows)

Update the manifest in `fulmenhq/scoop-bucket` (sibling repo at `../scoop-bucket`):

- [ ] Update `bucket/refbolt.json`: version, URLs, and SHA256 checksums (both `64bit` and `arm64`)
- [ ] Run `make precommit` in the scoop-bucket repo to validate
- [ ] Commit and push to `scoop-bucket/main`
- [ ] Verify: `scoop update && scoop install fulmenhq/refbolt && refbolt version`

### Docker Images

- [ ] Build and test local images:
  - [ ] `make docker-build` (CLI image)
  - [ ] `make docker-build-runner` (runner/cron image)
- [ ] Tag and push to ghcr.io (when container registry is set up)

## Post-Release

### Verification

- [ ] `go install github.com/fulmenhq/refbolt@v<version>` works
- [ ] `brew install fulmenhq/tap/refbolt` installs correct version
- [ ] Docker images run correctly
- [ ] Monitor GitHub issues for release-related bugs

### Housekeeping

- [ ] Archive completed feature briefs in productbook
- [ ] Plan next version features

## Version-Specific Notes

### For Major Releases (x.0.0)

- [ ] Breaking changes documented with upgrade guide
- [ ] Provider config migration guide (if schema changed)

### For Minor Releases (0.x.0)

- [ ] New providers documented in `docs/providers/README.md`
- [ ] New fetch strategies documented in `docs/development.md`

### For Patch Releases (0.0.x)

- [ ] Bug fixes documented with issue references
- [ ] Regression tests added for fixed bugs

## Emergency Hotfix Process

- [ ] Critical bug or security issue identified
- [ ] Hotfix branch created: `hotfix/v<version>`
- [ ] Minimal fix implemented and tested
- [ ] Version bumped (patch level)
- [ ] Tag pushed immediately after merge
- [ ] Root cause analysis documented
