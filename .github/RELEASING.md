# Releasing go-hexcore

Releases use stable `vMAJOR.MINOR.PATCH` tags, a GitHub Release, and a generated
changelog. Never move or delete a published tag.

## Publish a release

1. Merge the intended changes into `develop` and wait for CI.
2. Open **Actions → Prepare release → Run workflow**. Run it from `develop`
   with source `develop`.
3. Review the generated `release/vX.Y.Z` PR into `master`. It freezes the source
   commit and adds `.release.json` and the changelog. Approve its workflows if
   GitHub asks.
4. Check the version, diff, release notes, and CI, then select **Create a merge
   commit**. This merge authorizes publication.
5. After successful push CI for that exact `master` commit, **Publish release**
   creates the tag and GitHub Release, checks the Go proxy, and opens
   `sync/vX.Y.Z` into `develop`.
6. Merge the synchronization PR with **Create a merge commit** before preparing
   another release. Resolve any conflicts without dropping the tagged commit's
   ancestry.

Normal contribution PRs use squash merges. Release and synchronization PRs
require merge commits. Both branches require passing CI, resolved conversations,
and a manual merge; direct pushes and auto-merge are disabled.

If a release candidate needs changes, close its PR, delete its release branch,
merge the corrections normally, and prepare again. Do not modify a reviewed
candidate in place.

## Version calculation

Automation reads non-merge commits since the last release. The highest impact
determines one bump:

| Commit | Bump |
| --- | --- |
| Any type with `!`, `BREAKING CHANGE:`, or `BREAKING-CHANGE:` | Major |
| `feat:` | Minor |
| `fix:` or `perf:` | Patch |
| Other types without a breaking change | None |

Documentation, CI, and test-only changes wait for the next releasable code
change. The initial release is always v1.0.0. Prerelease and build-metadata tags
are not supported.

The baseline is the highest stable tag reachable from `master`. Its tip must
already be tagged before preparing the next release; stray version tags block
preparation. Automation checks ancestry, the module path, changelog, reviewed
merge tree, and CI on the exact commit.

For v2+, update the Go module path and imports to the matching major suffix
before preparing a release. Automation does not perform that migration.

## Publish a hotfix

Create `hotfix/name` from current `master`, commit and push the patch, then run
**Prepare release** from `develop` with source `hotfix/name`. The calculated
bump must be a patch. Pending feature work on `develop` is excluded.

After publication, merge the synchronization PR. Close older release candidates,
delete their release branches, and prepare again from the synchronized `develop`.

## Recover a failed publication

Read the **Publish release** log, fix the cause, and rerun the failed job. You
can also run the workflow from `develop` with the full master merge SHA in
`commit`; an empty input selects current `master`.

Successful push CI for that SHA is still required. Matching tags and GitHub
Releases are reused; conflicting tags are rejected. If proxy indexing fails
after publication, rerun rather than changing the tag.

Preparation and publication share a concurrency group. Wait for the active run;
GitHub retains at most one pending run in that group.

## Repository setup

Actions must be allowed to create pull requests. Workflows use `GITHUB_TOKEN`;
no separate PAT is needed. Only preparation and publication receive the write
permissions they need. Tag updates and deletion are blocked, including for
automation. Create release tags through the publication workflow.

[repository-settings.json](repository-settings.json) records the intended
settings and rulesets; editing the file does not apply them to GitHub.

## Publish the docs

The documentation build is part of required CI. PRs and `develop` pushes only
validate it. A `master` push deploys that commit's artifact after successful CI.
Configure Pages to use Actions; only the deployment job gets Pages write and
OIDC permissions.

After deployment, check the [site](https://2chevskii.github.io/go-hexcore/), a
direct nested page, search, and assets. The site follows the current release;
there are no archived versions.

## Check automation locally

Use Node 24, Git, and actionlint:

```sh
node --test .github/scripts/*.test.mjs
actionlint
```

Tests use temporary Git repositories and mocked GitHub responses. CI and release
automation make no HexCore API calls.
