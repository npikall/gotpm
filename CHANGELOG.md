# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

<!-- insertion marker -->
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
