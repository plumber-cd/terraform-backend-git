# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
