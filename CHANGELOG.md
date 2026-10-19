# Changelog

## [v1.0.0](https://github.com/2chevskii/go-hexcore/releases/tag/v1.0.0) - 2026-10-19

### Features

- feat(models): add typed resources and wire normalization ([c2852cd](https://github.com/2chevskii/go-hexcore/commit/c2852cd7ad02f9d6d434f27fb55c384b8a7dee82))
- feat(client): add HTTP transport and read services ([6c9fee4](https://github.com/2chevskii/go-hexcore/commit/6c9fee46515118c437407c300e160795e092a89c))
- feat(vds): add server provisioning and lifecycle actions ([b9b5bea](https://github.com/2chevskii/go-hexcore/commit/b9b5beaf57a6cf13728f6ecc5a32820c18623897))
- feat(disks): add disk provisioning and attachment actions ([7c88f76](https://github.com/2chevskii/go-hexcore/commit/7c88f7697b77d79c72770aac10de28895d98afb3))
- feat(actions): add action history and cancellable polling ([0e7f5b0](https://github.com/2chevskii/go-hexcore/commit/0e7f5b002ae930edce002b9ce67c4cc6f5417deb))
- feat(release): add reviewed stable module releases (#1) ([7c0bb7a](https://github.com/2chevskii/go-hexcore/commit/7c0bb7aead0d9783c43897c638019fb7ad3b117a))

### Fixes and performance

- fix(client): support live response envelopes and nested errors ([d604c6d](https://github.com/2chevskii/go-hexcore/commit/d604c6de2cc3a61e452c2899e343d84cf0c93c6f))
- fix(actions): accept asynchronous submission acknowledgements ([e4c023e](https://github.com/2chevskii/go-hexcore/commit/e4c023ee2425f80b0575e32231ba0f6d58289588))

### Other changes

- chore: initialize Go module and development tooling ([9eda2cd](https://github.com/2chevskii/go-hexcore/commit/9eda2cdc52445ebe42b178a39633419c26a11005))
- test(client): cover all documented API operations ([e645d61](https://github.com/2chevskii/go-hexcore/commit/e645d61a65afd443d350ebf9f986eacf961eb749))
- test(integration): add opt-in live API checks ([a558667](https://github.com/2chevskii/go-hexcore/commit/a558667b698ce7071c3a78d38b1984d30acdc0f6))
- docs: document client usage and API compatibility gaps ([55a338a](https://github.com/2chevskii/go-hexcore/commit/55a338a1815e0a63fa2eb8d0638bf8fa7a2b5b04))
- test(integration): isolate VDS checks and harden cleanup ([fd3f395](https://github.com/2chevskii/go-hexcore/commit/fd3f39552b8a3794c186cfdae11bfd2279023b34))
- test(client): cover live contracts and cleanup guarantees ([c073f44](https://github.com/2chevskii/go-hexcore/commit/c073f4419afda010aa2574c3242312800b72fa89))
- docs: record phase two verification and API blockers ([2510473](https://github.com/2chevskii/go-hexcore/commit/25104737629adc8e333486730b3d2661ae835218))
- docs: license the library under MIT ([db0e321](https://github.com/2chevskii/go-hexcore/commit/db0e321abd8e3ac9825d9ea53a93ca95a2e2e3ec))
- docs: add community guides and contribution templates ([12cb49d](https://github.com/2chevskii/go-hexcore/commit/12cb49d950429ef7aec14be6985025e8dd99ece9))
- ci: add offline checks for master and pull requests ([c84b046](https://github.com/2chevskii/go-hexcore/commit/c84b046ae8328fda905d746c7572d19e40d7a20a))
- ci: run checks on Linux with stable Go only ([35bc41b](https://github.com/2chevskii/go-hexcore/commit/35bc41b96ee427045dfa41ad98318430d7ae9271))

