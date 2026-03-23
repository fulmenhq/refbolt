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

Before signing, source the CI/CD signing credentials:

```bash
source ~/devsecops/vars/fulmenhq-refbolt-cicd.sh
export REFBOLT_VERSION_TAG=v<version>
```

The CI/CD vars file provides four stable variables:

| Variable               | Purpose                          |
| ---------------------- | -------------------------------- |
| `REFBOLT_GPG_HOMEDIR`  | GnuPG home directory for signing |
| `REFBOLT_PGP_KEY_ID`   | PGP signing key fingerprint      |
| `REFBOLT_MINISIGN_KEY` | Path to minisign private key     |
| `REFBOLT_MINISIGN_PUB` | Path to minisign public key      |

`REFBOLT_VERSION_TAG` is set separately so the vars file remains stable across releases.

### Release Artifacts & Signing

Follow the Fulmen "manifest-only" provenance pattern:

- Generate SHA256 + SHA512 manifests
- Sign manifests with minisign (primary) and optionally PGP
- Ship trust anchors (public keys) with the release

- [ ] (Recommended) Download CI-built artifacts and (re)generate manifests:

  ```bash
  make release-clean
  make release-download RELEASE_TAG=$REFBOLT_VERSION_TAG
  make release-checksums
  make release-verify-checksums
  ```

  This path uses GitHub Release artifacts built in CI and avoids local build drift.

- [ ] (Alternative) Build artifacts locally:

  ```bash
  make release-build
  make release-verify-checksums
  ```

- [ ] Sign manifests (minisign required; PGP optional):

  ```bash
  # Ensure GPG can prompt for passphrase in this terminal
  export GPG_TTY="$(tty)"
  gpg-connect-agent updatestartuptty /bye

  make release-sign RELEASE_TAG=$REFBOLT_VERSION_TAG
  ```

- [ ] Export public keys: `make release-export-keys`
- [ ] Verify exported keys are public-only: `make release-verify-keys`
- [ ] Copy release notes: `make release-notes RELEASE_TAG=$REFBOLT_VERSION_TAG`
- [ ] Upload provenance assets: `make release-upload RELEASE_TAG=$REFBOLT_VERSION_TAG`
  - For fully manual release (no CI artifacts): `make release-upload-all RELEASE_TAG=$REFBOLT_VERSION_TAG`

### Tagging

- [ ] Create annotated git tag: `git tag -a v<version> -m "Release v<version>"`
- [ ] Verify tag: `git tag -v v<version>`
- [ ] Push commits: `git push origin main`
- [ ] Push tag: `git push origin v<version>`
- [ ] Verify GitHub release appears (draft — review and publish)

### Docker Images

- [ ] Build and test local images:
  - [ ] `make docker-build` (CLI image)
  - [ ] `make docker-build-runner` (runner/cron image)
- [ ] Tag and push to ghcr.io (when container registry is set up)

## Post-Release

### Verification

- [ ] `go install github.com/fulmenhq/refbolt@v<version>` works
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
