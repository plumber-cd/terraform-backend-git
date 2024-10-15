# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.7] - 2024-10-15

### Added

- New `--amend` flag to use commit amend + force push when updating state to maintain repo size (thanks @rucciva!)

## [0.1.6] - 2023-09-21

### Fixed

- Fix incorrect state file path on Windows (#40)

## [0.1.5] - 2023-07-18

### Added

- Added GCP KMS option for sops encryption (#38) (thanks [@shaggyone](https://github.com/shaggyone)!)

## [0.1.4] - 2023-03-16

### Added

- Add some additional debug output for SOPS decryption

## [0.1.3] - 2023-02-15

### Fixed

- Fixed #31 (thanks [@joaquinrovira](https://github.com/joaquinrovira)!)

### Updated

- Go to 1.19
- Debian image to `bullseye`
- Various Go dependencies (except for go-git due to backward incompatible changes in <https://github.com/go-git/go-git/pull/548>)

## [0.1.2] - 2022-05-15

### Added

- Added `Dockerfile`
- Publish image to `ghcr.io`

## [0.1.0] - 2022-05-15

### Added

- Implemented TLS mode
- New state file encryption provider using `sops`. Currently integrated with PGP, AWS KMS and Hashicorp Vault.
- Introduced `--dir` option under `git` backend - now current working directory can be changed dynamically

### Changed

- Existing AES256 state file encryption is no longer recommended.

## [0.0.18] - 2022-04-30

### Changed

- Updated Go to 1.18 and all dependencies

### Fixed

- `ERROR: You're using an RSA key with SHA-1, which is no longer allowed. Please use a newer client or a different key type.`

## [0.0.17] - 2022-01-15

### Changed

- Use cross-platform detection for SSH-agent, now supports Pageant on Windows [#21](https://github.com/plumber-cd/terraform-backend-git/pull/21) (Authored-by: [@blaubaer](https://github.com/blaubaer))
- Updated dependencies, fixed CVE-2020-16845
- Updated to use Go 1.17, and Ubuntu 20.04 builder
- Updated Alpine 3.15
- Build `arm64` version of binaries for Mac and Linux; stop building `386` for Mac

## [0.0.16] - 2021-02-08

### Added

- GitHub Action (Authored-by: [@mambax](https://github.com/mambax))

### Fixed

- GitHub deprecated `set-env`; replaced with <https://docs.github.com/en/actions/reference/workflow-commands-for-github-actions#environment-files>

## [0.0.14] - 2020-05-30

### Added

- HTTP Basic Authentication

## [0.0.13] - 2020-05-30

### Added

- `terraform-backend-git version` command

## [0.0.12] - 2020-04-18

### Added

- Git storage: support `StrictHostKeyChecking=no`

## [0.0.11] - 2020-04-18

### Fix

- Git storage: SSH Agent auth type was crashing the backend

## [0.0.10] - 2020-04-17

### Fix

- If `git.state` contained elements of relative path (i.e. `foo/./bar` or `foo//bar`) - now correctly handle this scenario

## [0.0.8] - 2020-04-17

### Fix

- Git storage: `GIT_TOKEN` was used instead of `GITHUB_TOKEN` env variable

## [0.0.6] - 2020-04-15

### Fix

- If host user did not had a display name, commit author was empty

## [0.0.5] - 2020-04-15

### Fix

- Do not print an error message if config file was not found

## [0.0.4] - 2020-04-15

### Added

- Implemented config files for wrapper mode

## [0.0.3] - 2020-04-12

### Added

- Implemented wrapper mode (#3)

## [0.0.2] - 2020-04-12

### Added

- Backend side encryption (#2)

## [0.0.1] - 2020-04-12

### Added

- Initial implementation
