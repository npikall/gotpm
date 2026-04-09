# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.5] - 2026-04-09

### 🚀 Features

- add self command to inspect the binarys build in more detail
- add loading of manifest
- add resolving of destination dir
- add resolve of destination and copy files
- add editable flag to install command
- read input from stdin
- add default values to build tags

### 🐛 Bug Fixes

- return errors instead of logging

### 🚜 Refactor

- embed git version and build info
- default logger
- update colors and styles
- clean up and add tests
- move internal/list into cmd and add tests
- move helper.go into cmd/internal package
- rename internal package import
- move internal package into cmd/internal
- update self representation

### 🧪 Testing

- loading of manifest
- add testing to os data-dir resolving
- resolving of destination

### ◀️ Revert

- remove version command as version flag works now

### 💼 Other

- move makefile functionality to justfile
- refactor the install command
- refactor install command
- update all dependencies
- merge pull request #10 from npikall/clean-up
- fix release recipe

## [0.3.4] - 2026-02-21

### 🚀 Features

- *(internal)* add bump method to packageinfo

### 🐛 Bug Fixes

- typo in error message
- bumping without actually changing the version

### 📚 Documentation

- add go report badge

### 🚜 Refactor

- move bump into files

### 🧪 Testing

- bumping of packageinfo

### 💼 Other

- *(release)* bump version and log the changes

## [0.3.3] - 2026-02-20

### 🚀 Features

- *(cmd)* add spinner to update command
- *(cmd)* add spinner to install command

### 🚜 Refactor

- *(cmd)* add examples to all docstrings
- clean code and extract functions
- move spinner into helpers and clean up
- reduce nested function calls

### 💼 Other

- add workflow_dispatch trigger to docs workflow
- updated git description for build
- *(release)* update changelog

## [0.3.2] - 2026-02-19

### 🚀 Features

- add show package version flag
- usse debug keyword instead of verbose
- add show-next flag to bump cmd
- add dry-run flag to install cmd
- *(internal)* adding requests functionality
- *(internal)* add comparison of two versions
- *(cmd)* add update command
- *(cmd)* add func to update import statements in file
- *(cmd)* add optional indentation of the typst.toml file
- *(cmd)* add update command
- *(cmd)* request latest vresions asynchronously

### 🐛 Bug Fixes

- long description text
- *(cmd)* embed version
- *(cmd)* uninstall all versions from a single namespace

### 📚 Documentation

- add gif thumbnail
- update config

### 🚜 Refactor

- harmonize logging
- *(cmd)* clearer docstrings
- *(cmd)* harmonize color scheme
- *(cmd)* clearer debug messages
- *(cmd)* update docstring

### 🧪 Testing

- *(cmd)* add test for func to update import statements in a file

### 💼 Other

- go mod tidy
- add github issue templates
- update license year
- go mod tidy
- merge pull request #4 from npikall/dev

improvements and bug fixes
- *(changelog)* update to latest version
- trigger docs only when files change

## [0.3.1] - 2026-01-17

### 🚀 Features

- add init command
- use more explicit regex to validate version

### 🐛 Bug Fixes

- regex pattern with groups
- no version available string
- init creates new directory

### 📚 Documentation

- add thumbnail image
- use monospace font in thumbnail
- add documentation for gh-pages
- move assets

### 🚜 Refactor

- add pre-styled logger and verbosity
- sorted functions
- moved functions, increased readability

### 🧪 Testing

- add semver validation test
- rename test

### 💼 Other

- tidy module
- go mod tidy
- update gitignore for potential docs
- update changelog

## [0.3.0] - 2026-01-14

### 🚀 Features

- add more complex logic to uninstall
- add path argument to install cmd
- add bump command
- [**breaking**] add version command to display gotpm's version

### 📚 Documentation

- update changelog

### 🚜 Refactor

- add examples and explanations to commands
- rename version module to bump
- handle errors with cobra or charmbracelet/fang

### ◀️ Revert

- remove self command

### 💼 Other

- add python version
- add dirty flag to git describe
- refactor git tag description

## [0.2.1] - 2026-01-14

### 🚀 Features

- add ascii art to help command

### 📚 Documentation

- update changelog
- update changelog

### 🚜 Refactor

- use styling from charmbracelet/fang

### ◀️ Revert

- remove uv from target changelog
- attempt of makeing goreleaser work

### 💼 Other

- tidy module
- add properly configured goreleaser
- run goreleaser in github actions
- update git-changelog creation
- add release-notes.md to gitignore for ci to work
- add python version
- move release-notes creation to goreleaser
- pipe release-notes to goreleaser
- create release-notes first and use in goreleaser

## [0.2.0] - 2026-01-12

### 🚀 Features

- *(cmd)* add locate command
- *(cmd)* add editable flag
- *(cmd)* add self command

### 🐛 Bug Fixes

- styling for printing to stdout

### 📚 Documentation

- use bash instead of console
- fix ci badge link in readme.md
- fix typo

### 🚜 Refactor

- remove copyright

### ◀️ Revert

- remove goreleaser in favor of more manual release process

### 💼 Other

- update title in release
- update ldflags and inject version
- v0.2.0

## [0.1.0] - 2026-01-10

### 🚀 Features

- add colored printing functions
- add cobra commands
- add regex validation
- *(version)* add short flag
- add install command
- add some functions
- add colored cobra
- add gitignore
- update colors
- add working install command
- *(cmd)* add uninstall function
- *(cmd)* add bump flag
- *(cmd)* add list command
- *(cmd)* add list command

### 🐛 Bug Fixes

- write toml error

### 📚 Documentation

- add contributing guide
- add changelog skeleton
- update changelog

### 🚜 Refactor

- rename utils to echo
- *(system)* rename to gettypstpathroot
- move function around
- move test into internal
- moved internal/echo into cmd/helpers
- change coloring
- remove worker struct
- remove unused variable
- extract issemver function

### ◀️ Revert

- *(system)* remove unused function

### 💼 Other

- initial commit
- update files
- *(go.mod)* run go mod tidy
- *(test)* use assertion library
- update files
- clean up, use must
- go mod tidy
- *(makefile)* update annotations
- clean up and refactor styling
- add release workflow
- update files
- update release workflow, use git-changelog

[0.3.5]: https://github.com/npikall/gotpm/compare/v0.3.4..0.3.5
[0.3.4]: https://github.com/npikall/gotpm/compare/v0.3.3..v0.3.4
[0.3.3]: https://github.com/npikall/gotpm/compare/v0.3.2..v0.3.3
[0.3.2]: https://github.com/npikall/gotpm/compare/v0.3.1..v0.3.2
[0.3.1]: https://github.com/npikall/gotpm/compare/v0.3.0..v0.3.1
[0.3.0]: https://github.com/npikall/gotpm/compare/v0.2.1..v0.3.0
[0.2.1]: https://github.com/npikall/gotpm/compare/v0.2.0..v0.2.1
[0.2.0]: https://github.com/npikall/gotpm/compare/v0.1.0..v0.2.0
[0.1.0]: https://github.com/npikall/gotpm/tree/v0.1.0

<!-- generated by git-cliff -->
