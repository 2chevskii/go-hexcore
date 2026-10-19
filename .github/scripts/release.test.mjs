import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdtempSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';
import {
  assertTagState, changelogHeader, compareVersions, git, isAncestor, nextVersion,
  parseCommit, parseVersion, releasePlan, validateCandidate, validatePullRequest, versionTags,
} from './release.mjs';
import { ensureTag, GitHub, publicationPlan, publish } from './release-workflow.mjs';

const repository = '2chevskii/go-hexcore';

function fixture(t, version = null) {
  const cwd = mkdtempSync(join(tmpdir(), 'hexcore-release-'));
  t.after(() => rmSync(cwd, { recursive: true, force: true }));
  const run = (...args) => git(args, cwd);
  run('init', '-b', 'master');
  run('config', 'user.name', 'Release tests');
  run('config', 'user.email', 'tests@example.invalid');
  run('config', 'commit.gpgsign', 'false');
  run('config', 'core.autocrlf', 'false');
  const write = (path, text) => writeFileSync(join(cwd, path), text);
  const commit = (message, file = 'changes.txt') => {
    write(file, `${message}\n${run('rev-parse', '--verify', 'HEAD')}\n`);
    run('add', '.'); run('commit', '-m', message);
    return run('rev-parse', 'HEAD');
  };
  write('go.mod', `module github.com/${repository}\n\ngo 1.22\n`);
  write('CHANGELOG.md', changelogHeader);
  run('add', '.'); run('commit', '-m', 'feat: initial client');
  if (version) run('tag', version);
  const baseSha = run('rev-parse', 'HEAD');
  run('update-ref', 'refs/remotes/origin/master', baseSha);
  run('switch', '-c', 'develop');
  run('update-ref', 'refs/remotes/origin/develop', baseSha);
  const plan = (sourceBranch = 'develop') => releasePlan({
    baseSha, sourceSha: run('rev-parse', 'HEAD'), sourceBranch, date: '2026-10-19',
  }, cwd);
  const candidate = (prepared = plan()) => {
    const branch = `release/${prepared.metadata.version}`;
    run('update-ref', 'refs/remotes/origin/develop', prepared.metadata.sourceSha);
    run('switch', '-c', branch);
    write('.release.json', prepared.metadataText);
    write('CHANGELOG.md', prepared.changelog);
    run('add', '.'); run('commit', '-m', `chore(release): ${prepared.metadata.version}`);
    const headSha = run('rev-parse', 'HEAD');
    return { headSha, baseSha, branch, title: `chore(release): ${prepared.metadata.version}` };
  };
  const merge = candidateInfo => {
    run('switch', 'master'); run('merge', '--no-ff', candidateInfo.branch, '-m', candidateInfo.title);
    const sha = run('rev-parse', 'HEAD');
    run('update-ref', 'refs/remotes/origin/master', sha);
    const pr = {
      merged_at: '2026-10-19T00:00:00Z', merge_commit_sha: sha,
      title: candidateInfo.title, base: { ref: 'master' },
      head: { ref: candidateInfo.branch, sha: candidateInfo.headSha, repo: { full_name: repository } },
    };
    return { sha, pr };
  };
  return { cwd, run, write, commit, plan, candidate, merge, baseSha };
}

test('version ordering is numeric and does not lose integer precision', () => {
  assert.equal(compareVersions('v1.10.0', 'v1.9.99'), 1);
  assert.equal(compareVersions('v10.0.0', 'v2.99.99'), 1);
  assert.equal(compareVersions('v1.0.0', 'v1.0.0'), 0);
  assert.equal(nextVersion('v1.0.9007199254740992', [parseCommit('fix: a bug')]), 'v1.0.9007199254740993');
  for (const invalid of ['1.2.3', 'v01.2.3', 'v1.2', 'v1.2.3-rc.1', 'v1.2.3+build']) {
    assert.throws(() => parseVersion(invalid), /stable version/);
  }
});

test('the highest commit impact determines one bump; bootstrap is always v1.0.0', () => {
  assert.equal(nextVersion(null, [parseCommit('feat!: initial API')]), 'v1.0.0');
  assert.equal(nextVersion('v1.9.7', ['fix: a', 'perf: b', 'feat: c', 'fix: d'].map(parseCommit)), 'v1.10.0');
  assert.equal(nextVersion('v1.9.7', ['feat: a', 'refactor!: b'].map(parseCommit)), 'v2.0.0');
  assert.equal(nextVersion('v1.9.7', [parseCommit('fix: a\n\nBREAKING CHANGE: callers must migrate')]), 'v2.0.0');
  assert.equal(nextVersion('v1.9.7', [parseCommit('chore: a\n\nBREAKING-CHANGE: callers must migrate')]), 'v2.0.0');
  assert.equal(nextVersion('v1.9.7', [parseCommit('perf(client): avoid allocations')]), 'v1.9.8');
  assert.throws(() => nextVersion('v1.9.7', [parseCommit('docs: explain setup')]), /No releasable/);
  assert.throws(() => parseCommit('Merge pull request #1'), /Non-conventional/);
});

test('a bootstrap candidate contains all original history and validates deterministically', t => {
  const f = fixture(t);
  f.commit('ci: introduce release automation');
  const plan = f.plan();
  assert.equal(plan.metadata.version, 'v1.0.0');
  assert.equal(plan.commits.length, 2);
  assert.match(plan.changelog, /initial client/);
  const candidate = f.candidate(plan);
  assert.deepEqual(validateCandidate(candidate, f.cwd).metadata, plan.metadata);
  assert.throws(() => validateCandidate({ ...candidate, title: 'chore(release): v9.0.0' }, f.cwd), /title/);
  assert.throws(() => validateCandidate({ ...candidate, branch: 'release/v9.0.0' }, f.cwd), /branch/);
});

test('release metadata cannot hide extra code or invented changelog entries', t => {
  const f = fixture(t, 'v1.0.0');
  f.commit('fix: preserve cancellation');
  const candidate = f.candidate();
  f.write('surprise.go', 'package surprise\n');
  f.run('add', '.'); f.run('commit', '--amend', '--no-edit');
  assert.throws(() => validateCandidate({ ...candidate, headSha: f.run('rev-parse', 'HEAD') }, f.cwd), /may change only/);
  f.run('rm', 'surprise.go');
  f.write('CHANGELOG.md', '# Changelog\n\nInvented release notes\n');
  f.run('add', '.'); f.run('commit', '--amend', '--no-edit');
  assert.throws(() => validateCandidate({ ...candidate, headSha: f.run('rev-parse', 'HEAD') }, f.cwd), /changelog differs/);
});

test('version inflation and stale release baselines are rejected', t => {
  const f = fixture(t, 'v1.0.0');
  f.commit('fix: preserve cancellation');
  const plan = f.plan();
  const inflated = { ...plan, metadataText: plan.metadataText.replace('v1.0.1', 'v1.9.0') };
  const candidate = f.candidate(inflated);
  assert.throws(() => validateCandidate({ ...candidate, branch: 'release/v1.9.0', title: 'chore(release): v1.9.0' }, f.cwd), /metadata differs/);
  assert.throws(() => validateCandidate({ ...candidate, baseSha: candidate.headSha }, f.cwd), /baseline is stale/);
});

test('major releases require the matching Go module path', t => {
  const f = fixture(t, 'v1.0.0');
  f.commit('refactor!: replace the public client');
  assert.throws(() => f.plan(), /requires module github.com\/2chevskii\/go-hexcore\/v2/);
  f.write('go.mod', `module github.com/${repository}/v2\n\ngo 1.22\n`);
  f.run('add', '.'); f.run('commit', '-m', 'refactor: migrate imports and module path');
  assert.equal(f.plan().metadata.version, 'v2.0.0');
});

test('hotfixes must be based on current master and must produce only a patch release', t => {
  const f = fixture(t, 'v1.2.3');
  f.commit('fix: repair an urgent regression');
  assert.equal(f.plan('hotfix/regression').metadata.version, 'v1.2.4');
  f.commit('feat: unrelated feature');
  assert.throws(() => f.plan('hotfix/regression'), /patch-level/);
  f.run('switch', 'master');
  const newerMaster = f.commit('fix: another urgent change');
  f.run('tag', 'v1.2.4');
  f.run('switch', 'develop');
  assert.throws(() => releasePlan({ ...f.plan().metadata, baseSha: newerMaster }, f.cwd), /include current master/);
});

test('unpublished master commits and stray version tags block preparation', t => {
  const f = fixture(t, 'v1.0.0');
  const sourceSha = f.commit('fix: repair a regression');
  assert.throws(() => releasePlan({ baseSha: sourceSha, sourceSha, sourceBranch: 'develop', date: '2026-10-19' }, f.cwd), /unpublished release/);
  f.run('tag', 'v9.0.0');
  assert.throws(() => assertTagState(versionTags(f.cwd), f.baseSha, null, f.cwd), /outside the release baseline/);
});

test('publication accepts only the reviewed merge tree and immutable matching tags', t => {
  const f = fixture(t, 'v1.0.0');
  f.commit('fix: repair a regression');
  const candidate = f.candidate();
  const { sha, pr } = f.merge(candidate);
  assert.equal(publicationPlan(sha, pr, f.cwd).metadata.version, 'v1.0.1');
  f.run('tag', '-a', 'v1.0.1', '-m', 'Release v1.0.1');
  assert.equal(publicationPlan(sha, pr, f.cwd).metadata.version, 'v1.0.1');
  assert.throws(() => publicationPlan(sha, { ...pr, head: { ...pr.head, sha: f.baseSha } }, f.cwd), /merge commit/);
  f.write('unexpected.txt', 'unreviewed\n');
  f.run('add', '.'); f.run('commit', '--amend', '--no-edit');
  const altered = f.run('rev-parse', 'HEAD');
  f.run('update-ref', 'refs/remotes/origin/master', altered);
  assert.throws(() => publicationPlan(altered, { ...pr, merge_commit_sha: altered }, f.cwd), /tree differs/);
});

test('PR policy routes ordinary work into develop and accepts frozen releases into master', t => {
  const f = fixture(t);
  const source = f.commit('fix: repair a regression');
  const event = {
    repository: { full_name: repository },
    pull_request: {
      title: 'fix: repair a regression', base: { ref: 'develop', sha: f.baseSha },
      head: { ref: 'fix/regression', sha: source, repo: { full_name: 'contributor/fork' } },
    },
  };
  assert.equal(validatePullRequest(event, f.cwd), null);
  assert.throws(() => validatePullRequest({ ...event, pull_request: { ...event.pull_request, base: { ref: 'master' } } }, f.cwd), /prepared release/);
  const candidate = f.candidate();
  event.pull_request = {
    title: candidate.title, base: { ref: 'master', sha: f.baseSha },
    head: { ref: candidate.branch, sha: candidate.headSha, repo: { full_name: repository } },
  };
  assert.equal(validatePullRequest(event, f.cwd).metadata.version, 'v1.0.0');
});

test('synchronization retains ancestry and permits conflict resolution merges', t => {
  const f = fixture(t, 'v1.0.0');
  f.commit('fix: repair a regression');
  const { sha } = f.merge(f.candidate());
  f.run('tag', 'v1.0.1');
  f.run('switch', 'develop');
  f.commit('feat: next development change', 'next.txt');
  f.run('switch', '-c', 'sync/v1.0.1', sha);
  f.run('merge', '--no-ff', 'develop', '-m', 'chore(sync): resolve development merge');
  const event = {
    repository: { full_name: repository },
    pull_request: {
      title: 'chore(sync): merge v1.0.1 into develop', base: { ref: 'develop' },
      head: { ref: 'sync/v1.0.1', sha: f.run('rev-parse', 'HEAD'), repo: { full_name: repository } },
    },
  };
  assert.equal(validatePullRequest(event, f.cwd), null);
  assert.ok(isAncestor(sha, event.pull_request.head.sha, f.cwd));
});

test('a hotfix excludes pending features and synchronization avoids releasing the fix twice', t => {
  const f = fixture(t, 'v1.0.0');
  f.commit('feat: planned feature', 'feature.txt');
  f.run('switch', '-c', 'hotfix/urgent', 'master');
  const hotfixSha = f.commit('fix: urgent production regression', 'hotfix.txt');
  const hotfixPlan = releasePlan({ baseSha: f.baseSha, sourceSha: hotfixSha,
    sourceBranch: 'hotfix/urgent', date: '2026-10-19' }, f.cwd);
  assert.equal(hotfixPlan.metadata.version, 'v1.0.1');
  assert.equal(hotfixPlan.commits.length, 1);
  const { sha } = f.merge(f.candidate(hotfixPlan));
  f.run('tag', 'v1.0.1');
  f.run('switch', 'develop');
  f.run('merge', '--no-ff', 'master', '-m', 'chore(sync): merge v1.0.1 into develop');
  const next = releasePlan({ baseSha: sha, sourceSha: f.run('rev-parse', 'HEAD'),
    sourceBranch: 'develop', date: '2026-10-19' }, f.cwd);
  assert.equal(next.metadata.version, 'v1.1.0');
  assert.deepEqual(next.commits.map(commit => commit.title), ['feat: planned feature']);
});

test('publication cannot write a tag without successful CI on the exact master SHA', async t => {
  const previousCwd = process.cwd();
  t.after(() => process.chdir(previousCwd));
  const f = fixture(t);
  f.commit('ci: add release automation');
  const { sha, pr } = f.merge(f.candidate());
  process.chdir(f.cwd);
  const github = { api: async (path, options = {}) => {
    assert.equal(options.method, undefined, 'No write should occur before the CI gate');
    if (path.startsWith('commits/')) return [pr];
    if (path.startsWith('actions/')) return { workflow_runs: [
      { head_sha: 'f'.repeat(40), head_branch: 'master', conclusion: 'success' },
      { head_sha: sha, head_branch: 'develop', conclusion: 'success' },
      { head_sha: sha, head_branch: 'master', conclusion: 'failure' },
    ] };
    throw new Error(`Unexpected API call: ${path}`);
  } };
  await assert.rejects(publish(github, sha, { refreshRepository: () => {} }), /Successful push CI on this exact/);
});

test('tag recovery accepts matching annotated tags and never overwrites conflicting ones', async () => {
  const requests = [];
  const sha = 'a'.repeat(40);
  const github = { api: async (path, options) => {
    requests.push({ path, options });
    if (path.startsWith('git/ref/')) return { object: { type: 'tag', sha: 'annotated' } };
    return { object: { type: 'commit', sha } };
  } };
  await ensureTag(github, 'v1.0.0', sha);
  assert.equal(requests.length, 2);
  assert.ok(requests.every(request => !request.options?.method));
  await assert.rejects(ensureTag(github, 'v1.0.0', 'b'.repeat(40)), /another commit/);
});

test('a failed GitHub release can resume after its tag was created without recreating the tag', async t => {
  const previousCwd = process.cwd();
  t.after(() => process.chdir(previousCwd));
  const f = fixture(t);
  f.commit('ci: add release automation');
  const { sha, pr } = f.merge(f.candidate());
  process.chdir(f.cwd);
  let annotated = null;
  let release = null;
  let failRelease = true;
  let tagWrites = 0;
  let proxyChecks = 0;
  const github = {
    repository,
    api: async (path, options = {}) => {
      if (path.startsWith('commits/')) return [pr];
      if (path.startsWith('actions/')) return { workflow_runs: [{ head_sha: sha, head_branch: 'master', conclusion: 'success' }] };
      if (path === 'git/ref/tags/v1.0.0') return annotated && { object: { type: 'tag', sha: annotated } };
      if (path.startsWith('git/tags/')) return { object: { type: 'commit', sha } };
      if (path === 'git/tags') {
        annotated = execFileSync('git', ['mktag'], { cwd: f.cwd, encoding: 'utf8', input:
          `object ${sha}\ntype commit\ntag v1.0.0\ntagger Tests <tests@example.invalid> 1792368000 +0000\n\nRelease v1.0.0\n` }).trim();
        return { sha: annotated };
      }
      if (path === 'git/refs') {
        tagWrites++;
        f.run('update-ref', options.body.ref, options.body.sha);
        return {};
      }
      if (path === 'releases/tags/v1.0.0') return release;
      if (path === 'releases') {
        if (failRelease) { failRelease = false; throw new Error('Simulated interruption after tag creation'); }
        release = { draft: false, prerelease: false };
        return release;
      }
      throw new Error(`Unexpected API call: ${path}`);
    },
    createBranch: async () => {},
    ensurePR: async () => ({ html_url: 'https://example.invalid/sync' }),
  };
  const options = { refreshRepository: () => {}, report: () => {}, verifyProxy: async (modulePath, version) => {
    assert.equal(modulePath, `github.com/${repository}`); assert.equal(version, 'v1.0.0'); proxyChecks++;
  } };
  await assert.rejects(publish(github, sha, options), /Simulated interruption/);
  await publish(github, sha, options);
  await publish(github, sha, options);
  assert.equal(tagWrites, 1);
  assert.equal(proxyChecks, 2);
});

test('GitHub API handles optional missing resources and does not treat errors as absence', async () => {
  const response = status => ({ status, ok: status < 400, json: async () => ({ message: 'failure' }) });
  const missing = new GitHub(repository, 'test-token', async () => response(404));
  assert.equal(await missing.api('git/ref/tags/v1.0.0', { optional: true }), null);
  const denied = new GitHub(repository, 'test-token', async () => response(403));
  await assert.rejects(denied.api('git/ref/tags/v1.0.0', { optional: true }), /HTTP 403/);
});
