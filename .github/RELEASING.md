# Releasing go-hexcore

The first release is **v1.0.0**. Releases are stable SemVer Git tags, with a
matching GitHub Release and generated changelog. This library ships source;
there are no binary packages to upload. Never move or delete a published tag.

## Branches and reviews

`develop` is the default branch and receives normal contribution PRs. Both
`develop` and `master` require passing `CI`, resolved review conversations,
and a PR. Direct pushes, force pushes, and branch deletion are prohibited,
including for administrators. The maintainer's deliberate Merge action is the
approval gate; zero mandatory approving reviews allows a sole maintainer to
merge their own PRs. Auto-merge is disabled.

Squash normal PRs into `develop`, using their Conventional Commit titles and
preserving breaking-change footers in their bodies. Prefer `!` in the title
for breaking changes. Merge release PRs into `master` with **Create a merge
commit**; the master ruleset enforces this method. Synchronization PRs into
`develop` also require **Create a merge commit** to preserve ancestry.

Normal work uses `feature/*`, `feat/*`, `fix/*`, `docs/*`, `ci/*`, `test/*`,
`refactor/*`, `perf/*`, `build/*`, `chore/*`, `revert/*`, or `style/*` branches.
Version tags use `vMAJOR.MINOR.PATCH`; prerelease and build-metadata tags are
not supported by this release channel.

## Prepare and publish

1. Merge the intended contribution PRs into `develop` and ensure its CI passes.
2. Open **Actions → Prepare release → Run workflow**, select `develop` as
   the workflow branch, and leave the source input as `develop`.
3. Automation calculates the next version and opens a `release/vX.Y.Z` PR
   into `master`. It freezes the source commit and adds exactly one generated
   commit containing `.release.json` and `CHANGELOG.md`. No tag is created.
4. In the bot-created PR, select **Approve workflows to run** if GitHub asks.
   The workflows use `GITHUB_TOKEN`, so no PAT or GitHub App credential is
   required. The repository setting allowing Actions to create PRs must be on;
   the automation never approves reviews or merges PRs itself.
5. Review the diff, calculated version, release notes, and CI. Select **Create
   a merge commit** when ready to publish.
6. After successful push CI on that exact master merge commit, **Publish
   release** validates the candidate again, creates an annotated tag and
   GitHub Release, verifies the Go module proxy, and opens `sync/vX.Y.Z`
   into `develop`.
7. Approve pending CI runs and merge the synchronization PR with **Create
   a merge commit** before preparing the next release. Resolve merge conflicts
   on the sync branch while retaining the tagged commit as an ancestor.

Development can continue while a release PR is reviewed. Its source is frozen.
For changes to the candidate, close its PR, delete its disposable release
branch, merge corrections through the normal contribution process, and prepare
again. The preparation workflow never silently replaces a reviewed candidate.

## Version calculation

After v1.0.0, all included non-merge commits since the last release are parsed.
Normal squash commits provide one change per contribution. The highest impact
determines a single bump:

| Commit | Bump |
| --- | --- |
| Any type with `!`, `BREAKING CHANGE:`, or `BREAKING-CHANGE:` | Major |
| `feat:` | Minor |
| `fix:` or `perf:` | Patch |
| Other types without a breaking change | None |

A docs/CI/test-only change set does not publish a new version. Such changes
appear in the next release's changelog. The bootstrap release includes all
existing history and is always v1.0.0.

The baseline is the numerically highest stable version tag reachable from
master. Master must be tagged at its current tip before another release can
be prepared. Unexpected version tags elsewhere block releases. PR validation
recalculates the version, source ancestry, module path, and exact changelog;
publication additionally verifies the reviewed merge tree and successful CI.

For v2 and later, migrate the Go module path and imports to the matching major
suffix before preparing the release. A calculated major bump fails until this
is done; release automation does not rewrite the public API or imports.

## Hotfixes

Branch `hotfix/name` from current `master`, commit the patch, and push that
work branch. Run **Prepare release** from `develop`, with the source input set
to `hotfix/name`. It prepares the same reviewed `release/vX.Y.Z` PR into master,
using only that hotfix source. Hotfixes must calculate a patch bump.

After publication, merge the synchronization PR into develop. Any older release
candidate is now stale: close its PR, delete its release branch, synchronize
develop, and prepare again. Feature work already on develop is excluded from
the hotfix release.

## Recovery and safeguards

Preparation and publication share a concurrency group and never cancel a
running release operation. GitHub keeps at most one pending run in that group;
wait for the active operation rather than queueing multiple manual requests.

If publication fails, inspect the **Publish release** log. Fix the cause and
rerun the failed job, or manually run the workflow from `develop` with the full
master merge SHA in its `commit` input. An empty input selects current master.
Successful push CI on that exact SHA is still required. A matching existing
tag or GitHub Release is reused, so failures after tag creation are recoverable.
A conflicting tag is always an error. A proxy indexing failure occurs after
publication and is recovered by rerunning, never by changing the tag.

Version-tag updates and deletion are blocked by a ruleset with no bypass actors,
including the automation itself. Create tags through the publication workflow.
GitHub does not permit its built-in Actions integration as a tag-creation-rule
bypass actor for this repository. Restricting creation exclusively to a bot
would require a separately installed GitHub App; this setup uses GITHUB_TOKEN.
The desired repository settings and rulesets are recorded in
[repository-settings.json](repository-settings.json); changing that file does
not itself apply GitHub settings. Workflow permissions remain read-only by
default; only preparation and publication receive the required write scopes.

All CI and release operations are offline with respect to HexCore. No live
credentials, paid resources, or Terraform provider releases are involved.

## Local automation checks

Use Node.js 24 and Git to run the dependency-free release tests:

```sh
node --test .github/scripts/*.test.mjs
actionlint .github/workflows/*.yml
```

Tests use disposable local Git repositories and mocked GitHub responses,
including interrupted-publication recovery. They do not create remote tags,
branches, PRs, or releases.
