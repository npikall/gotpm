# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

<!-- insertion marker -->
## [v0.3.3](https://github.com/npikall/gotpm/releases/tag/v0.3.3) - 2026-02-20

<small>[Compare with v0.3.2](https://github.com/npikall/gotpm/compare/v0.3.2...v0.3.3)</small>

### Features

- add spinner to install command ([0c5c05a](https://github.com/npikall/gotpm/commit/0c5c05a96879be4ff807362e6641c95916bfe9e1) by npikall).
- add spinner to update command ([a01632a](https://github.com/npikall/gotpm/commit/a01632a48f1338e30a4863174c320804b888dac9) by npikall).

### Code Refactoring

- reduce nested function calls ([89e34bf](https://github.com/npikall/gotpm/commit/89e34bff255f824125dd0c9b8a11160b65b659a3) by npikall).
- move spinner into helpers and clean up ([41d88f1](https://github.com/npikall/gotpm/commit/41d88f10f4ebd24f42b970533a3dc54c7d44c658) by npikall).
- clean code and extract functions ([27109f0](https://github.com/npikall/gotpm/commit/27109f0dfa26ec4f22e6ad98b9a60c8c925b8e6d) by Nikolas Pikall).

## [v0.3.2](https://github.com/npikall/gotpm/releases/tag/v0.3.2) - 2026-02-19

<small>[Compare with v0.3.1](https://github.com/npikall/gotpm/compare/v0.3.1...v0.3.2)</small>

### Features

- request latest vresions asynchronously ([febd7d8](https://github.com/npikall/gotpm/commit/febd7d809224b98db3106c5feda982544317559b) by npikall).
- add update command ([d498b7b](https://github.com/npikall/gotpm/commit/d498b7b967fea6dfaec1e28f81102fe21a589234) by npikall).
- add optional indentation of the typst.toml file ([55edc63](https://github.com/npikall/gotpm/commit/55edc63b12f3f1081901581702af40f71fca8e80) by npikall).
- add func to update import statements in file ([631c0b8](https://github.com/npikall/gotpm/commit/631c0b81c19629e0df4146e87b55932e0d1a6420) by npikall).
- add comparison of two versions ([eec2d66](https://github.com/npikall/gotpm/commit/eec2d66fd8a08b8d75d5a26023a47896746cc9f8) by npikall).
- adding requests functionality ([06d7022](https://github.com/npikall/gotpm/commit/06d7022aa2b1a35e6f0cca5199771d0aae1538eb) by npikall).
- add dry-run flag to install cmd ([4e48328](https://github.com/npikall/gotpm/commit/4e4832808bb5db630d20d6d1b49ffed6b9d0073e) by npikall).
- add show-next flag to bump cmd ([d3de9b7](https://github.com/npikall/gotpm/commit/d3de9b7479fa05b95e3f5d5142ef606bd9cf01e8) by npikall).
- usse debug keyword instead of verbose ([05a2e18](https://github.com/npikall/gotpm/commit/05a2e18b461687cf0c1ff7721d4fcd09bfb3ba4e) by npikall).
- add show package version flag ([b6f5f45](https://github.com/npikall/gotpm/commit/b6f5f457259b19e707576e2f9a4177a0e00a5627) by Nikolas Pikall).

### Bug Fixes

- uninstall all versions from a single namespace ([ac70e18](https://github.com/npikall/gotpm/commit/ac70e182752f41174f2009456a36f334ec0db384) by npikall).
- embed version ([446a4e2](https://github.com/npikall/gotpm/commit/446a4e2415a83ff789ecd2664f7a9b0c7f41c6a9) by npikall).
- long description text ([f9c2272](https://github.com/npikall/gotpm/commit/f9c22724dc71c438ea1ccea2423ade58944d66c4) by npikall).

### Code Refactoring

- clearer debug messages ([4c7733d](https://github.com/npikall/gotpm/commit/4c7733df6530e02a9d12bcc64a8360e0be33e1c6) by npikall).

## [v0.3.1](https://github.com/npikall/gotpm/releases/tag/v0.3.1) - 2026-01-17

<small>[Compare with v0.3.0](https://github.com/npikall/gotpm/compare/v0.3.0...v0.3.1)</small>

### Features

- use more explicit regex to validate version ([e09dff6](https://github.com/npikall/gotpm/commit/e09dff64d5c58e4f21d40184107b7eaa11d534d9) by Nikolas Pikall).
- add init command ([55b748a](https://github.com/npikall/gotpm/commit/55b748ae624747fca3f16be83cfb1e839f9d0bed) by npikall).

### Bug Fixes

- init creates new directory ([b911203](https://github.com/npikall/gotpm/commit/b91120305f24a13d70e10f0fc6d154a27256f6ac) by Nikolas Pikall).
- no version available string ([8f64131](https://github.com/npikall/gotpm/commit/8f64131b1d8d27705ef9a639b0f181dd54d7b2d3) by Nikolas Pikall).
- regex pattern with groups ([80bb263](https://github.com/npikall/gotpm/commit/80bb2630f5cfb1a763c12cb9db3c1fd47c11e685) by Nikolas Pikall).

### Code Refactoring

- moved functions, increased readability ([a90587c](https://github.com/npikall/gotpm/commit/a90587c37fb9d3697a67bfd979ef4ae96d4a52f0) by npikall).
- sorted functions ([0039a47](https://github.com/npikall/gotpm/commit/0039a47c0b2145d67d66b1d2b142660149534d2c) by npikall).

## [v0.3.0](https://github.com/npikall/gotpm/releases/tag/v0.3.0) - 2026-01-14

<small>[Compare with v0.2.1](https://github.com/npikall/gotpm/compare/v0.2.1...v0.3.0)</small>

### Features

- add bump command ([1033463](https://github.com/npikall/gotpm/commit/10334637ff6dc1f8d12340ebc66e5e66af00742b) by npikall).
- add path argument to install cmd ([325a6ba](https://github.com/npikall/gotpm/commit/325a6badd1d88191255616d522592ed2546d92e0) by npikall).
- add more complex logic to uninstall ([e8cad74](https://github.com/npikall/gotpm/commit/e8cad747890b94a4520f18219573db30295c4954) by npikall).

### Reverts

- remove self command ([30162d8](https://github.com/npikall/gotpm/commit/30162d80d03271a0a313ad13458a694a471b9529) by npikall).

### Code Refactoring

- handle errors with cobra or charmbracelet/fang ([6ef9752](https://github.com/npikall/gotpm/commit/6ef97526012f49097f271b2271e9c9a7ede958a8) by npikall).
- rename version module to bump ([5c3664d](https://github.com/npikall/gotpm/commit/5c3664d8fafd6d44afcc5246e208a307ae49bafa) by npikall).

## [v0.2.1](https://github.com/npikall/gotpm/releases/tag/v0.2.1) - 2026-01-14

<small>[Compare with v0.2.0](https://github.com/npikall/gotpm/compare/v0.2.0...v0.2.1)</small>

### Features

- add ascii art to help command ([2013b6f](https://github.com/npikall/gotpm/commit/2013b6f21271b3695256463e06fcd4c4f9b208eb) by npikall).

### Reverts

- attempt of makeing goreleaser work ([5b54f02](https://github.com/npikall/gotpm/commit/5b54f02fdcab10092a26f549d82fef5b086c897a) by npikall).

### Code Refactoring

- use styling from charmbracelet/fang ([9257fa7](https://github.com/npikall/gotpm/commit/9257fa7a27610b84fb64ad94d18412019c3ea3bf) by npikall).

## [v0.2.0](https://github.com/npikall/gotpm/releases/tag/v0.2.0) - 2026-01-12

<small>[Compare with v0.1.0](https://github.com/npikall/gotpm/compare/v0.1.0...v0.2.0)</small>

### Features

- add self command ([f0a4f06](https://github.com/npikall/gotpm/commit/f0a4f069bb4360faa419156983162d628cf83d32) by Nikolas Pikall).
- add editable flag ([8d26502](https://github.com/npikall/gotpm/commit/8d26502e63f7aeeffc9677a5fd8d1404c1db3463) by Nikolas Pikall).
- add locate command ([3dcd794](https://github.com/npikall/gotpm/commit/3dcd794521a8b21375e4b986fe9e483f5522a60d) by Nikolas Pikall).

### Bug Fixes

- styling for printing to stdout ([36db6e7](https://github.com/npikall/gotpm/commit/36db6e78ab39d9eeeb74398026191d940df7f759) by Nikolas Pikall).

### Reverts

- remove goreleaser in favor of more manual release process ([ecdb393](https://github.com/npikall/gotpm/commit/ecdb3937dcc7a2d4980fbff958b80ecba55a705f) by Nikolas Pikall).

## [v0.1.0](https://github.com/npikall/gotpm/releases/tag/v0.1.0) - 2026-01-10

<small>[Compare with first commit](https://github.com/npikall/gotpm/compare/9a21129a9635fe4c78f65106ba936c8b6cf6b9d9...v0.1.0)</small>

### Features

- add list command ([f73e272](https://github.com/npikall/gotpm/commit/f73e27287d8f3a9578fc2b01792cb44987ce7864) by Nikolas Pikall).
- add bump flag ([0755832](https://github.com/npikall/gotpm/commit/0755832f7b0112c1f923a6f2daa1ae47ea0296b6) by Nikolas Pikall).
- add uninstall function ([587e5ef](https://github.com/npikall/gotpm/commit/587e5ef3b53aeec63a3a09986691fb5db13cace6) by Nikolas Pikall).
- add working install command ([64ecbac](https://github.com/npikall/gotpm/commit/64ecbac295938e7b426343c73b74759dbad06100) by npikall).
- update colors ([7e2ac6f](https://github.com/npikall/gotpm/commit/7e2ac6f2912ea2f128b95fb933c8de430dbdd251) by npikall).
- add gitignore ([19a978c](https://github.com/npikall/gotpm/commit/19a978c726ac93eecc09908288675bb92d73821c) by Nikolas Pikall).
- add colored cobra ([6232381](https://github.com/npikall/gotpm/commit/6232381f33ced19f7608c646ec7423d359fcca3f) by Nikolas Pikall).
- add some functions ([8c758ae](https://github.com/npikall/gotpm/commit/8c758aeda783dcdbefbda1e89bd0ccc8e28d0a51) by Nikolas Pikall).
- add install command ([110dca0](https://github.com/npikall/gotpm/commit/110dca0b13cdaf0fd3b09b20d01abfad30354070) by Nikolas Pikall).
- add short flag ([70f0ef2](https://github.com/npikall/gotpm/commit/70f0ef29b8853559e48ca95ee26b27ec1eac9b19) by Nikolas Pikall).
- add regex validation ([d3bd306](https://github.com/npikall/gotpm/commit/d3bd30643774c1b16f28c51ab686ed936cbf1065) by Nikolas Pikall).
- add cobra commands ([f4e67ff](https://github.com/npikall/gotpm/commit/f4e67ff4536a93454c2b4e41eb9f07a960e7f308) by Nikolas Pikall).
- add colored printing functions ([1536f30](https://github.com/npikall/gotpm/commit/1536f3048e19909cdcefa6c1b8708e4ac7e43a55) by Nikolas Pikall).

### Bug Fixes

- write toml error ([9adc796](https://github.com/npikall/gotpm/commit/9adc79662dc465ce79a1ddd4dbac98712a25b8f4) by Nikolas Pikall).

### Code Refactoring

- extract isSemVer function ([e032e31](https://github.com/npikall/gotpm/commit/e032e31cf63710d8859af0ba38dbed8ffb4b15f9) by Nikolas Pikall).
- remove unused variable ([683cdfc](https://github.com/npikall/gotpm/commit/683cdfc64bf4e77f15b840b174f387a546f73b03) by Nikolas Pikall).
- remove worker struct ([e17eb32](https://github.com/npikall/gotpm/commit/e17eb329bbccbdacaf1be6c575ea45394a224b73) by Nikolas Pikall).
- change coloring ([cce3b19](https://github.com/npikall/gotpm/commit/cce3b191699869ab0d5ac85ef7d695ab04fb970e) by Nikolas Pikall).
- moved internal/echo into cmd/helpers ([a62e7aa](https://github.com/npikall/gotpm/commit/a62e7aa567c0f8b0982cae8b718df9601801f48c) by npikall).
- move test into internal ([42be599](https://github.com/npikall/gotpm/commit/42be59992dcc5c206414fa890b07b4fb8a4b3510) by Nikolas Pikall).
- move function around ([e3ff9ca](https://github.com/npikall/gotpm/commit/e3ff9ca528ba765810b815d29ad17cee9abbe653) by Nikolas Pikall).
- rename to GetTypstPathRoot ([27c001f](https://github.com/npikall/gotpm/commit/27c001f50266794727207c093606ff320d295e8a) by Nikolas Pikall).
- rename utils to echo ([3f79adf](https://github.com/npikall/gotpm/commit/3f79adf66990005a6609cbc7fd97c5a6fb879f23) by Nikolas Pikall).
