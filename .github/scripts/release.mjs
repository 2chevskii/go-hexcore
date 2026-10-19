import { execFileSync } from 'node:child_process';

export const initialVersion = 'v1.0.0';
export const changelogHeader = '# Changelog\n\n';
export const conventionalTitle = /^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([^()\r\n]+\))?!?: \S.*$/;

function requireThat(condition, message) {
  if (!condition) throw new Error(message);
}

// This release channel deliberately accepts canonical stable versions only.
export function parseVersion(value) {
  const match = /^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/.exec(value);
  requireThat(match, `Expected a stable version such as v1.2.3, got ${value}`);
  return match.slice(1).map(BigInt);
}

export function compareVersions(left, right) {
  const a = parseVersion(left);
  const b = parseVersion(right);
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return a[i] > b[i] ? 1 : -1;
  }
  return 0;
}

export function parseCommit(message) {
  const [title, ...body] = message.trim().split('\n');
  requireThat(conventionalTitle.test(title), `Non-conventional commit: ${title}`);
  const match = /^(\w+)(?:\([^()]+\))?(!)?: (.+)$/.exec(title);
  const breaking = Boolean(match[2]) || body.some(line => /^BREAKING[ -]CHANGE: \S/.test(line));
  const bump = breaking ? 'major' : match[1] === 'feat' ? 'minor'
    : ['fix', 'perf'].includes(match[1]) ? 'patch' : null;
  return { title, type: match[1], breaking, bump };
}

export function nextVersion(previous, commits) {
  if (previous === null) return initialVersion;
  const priority = { major: 3, minor: 2, patch: 1 };
  const bump = commits.reduce((result, commit) =>
    (priority[commit.bump] || 0) > (priority[result] || 0) ? commit.bump : result, null);
  requireThat(bump, 'No releasable changes: use feat, fix, perf, or an explicit breaking change.');
  let [major, minor, patch] = parseVersion(previous);
  if (bump === 'major') { major++; minor = 0n; patch = 0n; }
  if (bump === 'minor') { minor++; patch = 0n; }
  if (bump === 'patch') patch++;
  return `v${major}.${minor}.${patch}`;
}

export function git(args, cwd = process.cwd()) {
  return execFileSync('git', args, { cwd, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trimEnd();
}

export function isAncestor(ancestor, descendant, cwd) {
  try { git(['merge-base', '--is-ancestor', ancestor, descendant], cwd); return true; }
  catch (error) { if (error.status === 1) return false; throw error; }
}

export function readAt(sha, path, cwd, optional = false) {
  // Missing files are optional; other Git failures must remain visible.
  const files = git(['ls-tree', '--name-only', sha, '--', path], cwd);
  if (!files) {
    requireThat(optional, `${path} is missing at ${sha}`);
    return null;
  }
  return execFileSync('git', ['show', `${sha}:${path}`], { cwd, encoding: 'utf8' });
}

export function versionTags(cwd) {
  return git(['tag', '--list', 'v[0-9]*'], cwd).split('\n').filter(Boolean).map(version => {
    parseVersion(version);
    return { version, sha: git(['rev-parse', `${version}^{commit}`], cwd) };
  });
}

export function latestAt(sha, tags, cwd) {
  return tags.filter(tag => isAncestor(tag.sha, sha, cwd))
    .sort((a, b) => compareVersions(b.version, a.version))[0] || null;
}

export function assertTagState(tags, baseSha, candidate, cwd) {
  for (const tag of tags) {
    if (candidate && tag.version === candidate.version) {
      requireThat(tag.sha === candidate.sha, `${tag.version} already points to another commit.`);
    } else {
      requireThat(isAncestor(tag.sha, baseSha, cwd),
        `Unexpected version tag ${tag.version} outside the release baseline; reconcile tags first.`);
    }
  }
}

export function releasePlan({ baseSha, sourceSha, sourceBranch, date }, cwd) {
  requireThat(/^[0-9a-f]{40}$/.test(baseSha) && /^[0-9a-f]{40}$/.test(sourceSha), 'Expected full commit SHAs.');
  requireThat(sourceBranch === 'develop' || /^hotfix\/[a-zA-Z0-9][a-zA-Z0-9._/-]*$/.test(sourceBranch),
    'Source must be develop or a hotfix/* branch.');
  requireThat(/^\d{4}-\d{2}-\d{2}$/.test(date) && new Date(`${date}T00:00:00Z`).toISOString().startsWith(date),
    'Expected a valid release date.');
  requireThat(isAncestor(baseSha, sourceSha, cwd),
    'Source must include current master. Merge the synchronization PR before preparing another release.');
  const previous = latestAt(baseSha, versionTags(cwd), cwd);
  requireThat(previous === null || previous.sha === baseSha,
    'master contains an unpublished release. Recover publication before preparing another one.');
  const range = previous ? `${baseSha}..${sourceSha}` : sourceSha;
  const hashes = git(['log', '--reverse', '--no-merges', '--format=%H', range], cwd).split('\n').filter(Boolean);
  const commits = hashes.map(sha => ({ sha, ...parseCommit(git(['show', '-s', '--format=%B', sha], cwd)) }));
  const version = nextVersion(previous?.version || null, commits);
  if (sourceBranch.startsWith('hotfix/')) {
    requireThat(previous, 'Publish v1.0.0 from develop before preparing hotfixes.');
    const [major, minor, patch] = parseVersion(previous.version);
    requireThat(version === `v${major}.${minor}.${patch + 1n}`, 'Hotfixes must contain patch-level changes only.');
  }
  const [major] = parseVersion(version);
  const modulePath = `github.com/2chevskii/go-hexcore${major > 1n ? `/v${major}` : ''}`;
  const module = /^module\s+(\S+)\s*$/m.exec(readAt(sourceSha, 'go.mod', cwd))?.[1];
  requireThat(module === modulePath, `${version} requires module ${modulePath}; migrate go.mod and imports first.`);
  const metadata = { schema: 1, version, previousVersion: previous?.version || null, baseSha, sourceSha, sourceBranch, date };
  const repository = 'https://github.com/2chevskii/go-hexcore';
  const link = previous ? `${repository}/compare/${previous.version}...${version}` : `${repository}/releases/tag/${version}`;
  const sections = [
    ['Breaking changes', commits.filter(commit => commit.breaking)],
    ['Features', commits.filter(commit => commit.type === 'feat' && !commit.breaking)],
    ['Fixes and performance', commits.filter(commit => ['fix', 'perf'].includes(commit.type) && !commit.breaking)],
    ['Other changes', commits.filter(commit => !['feat', 'fix', 'perf'].includes(commit.type) && !commit.breaking)],
  ];
  const notes = sections.filter(([, items]) => items.length).map(([heading, items]) =>
    `### ${heading}\n\n${items.map(commit => `- ${commit.title} ([${commit.sha.slice(0, 7)}](${repository}/commit/${commit.sha}))`).join('\n')}`
  ).join('\n\n');
  const oldChangelog = readAt(sourceSha, 'CHANGELOG.md', cwd, true) || changelogHeader;
  requireThat(oldChangelog.startsWith(changelogHeader), 'CHANGELOG.md must start with "# Changelog" and a blank line.');
  const changelog = `${changelogHeader}## [${version}](${link}) - ${date}\n\n${notes}\n\n${oldChangelog.slice(changelogHeader.length)}`;
  return { metadata, commits, notes, changelog, metadataText: `${JSON.stringify(metadata, null, 2)}\n` };
}

export function validateCandidate({ headSha, baseSha, branch, title }, cwd) {
  const metadataText = readAt(headSha, '.release.json', cwd);
  const metadata = JSON.parse(metadataText);
  requireThat(metadata.baseSha === baseSha, 'Release baseline is stale. Close this PR and prepare a fresh release.');
  requireThat(branch === `release/${metadata.version}`, 'Release branch and version disagree.');
  requireThat(title === `chore(release): ${metadata.version}`, 'Release PR title must match its version.');
  requireThat(git(['show', '-s', '--format=%P', headSha], cwd) === metadata.sourceSha,
    'Release candidate must be one generated metadata commit on top of its frozen source.');
  const changed = git(['diff-tree', '--no-commit-id', '--name-only', '-r', headSha], cwd).split('\n').sort();
  requireThat(JSON.stringify(changed) === JSON.stringify(['.release.json', 'CHANGELOG.md']),
    'The generated release commit may change only .release.json and CHANGELOG.md.');
  const plan = releasePlan(metadata, cwd);
  requireThat(metadataText === plan.metadataText, 'Release metadata differs from the calculated version and source.');
  requireThat(readAt(headSha, 'CHANGELOG.md', cwd) === plan.changelog, 'Release changelog differs from the included commits.');
  return plan;
}

export function validatePullRequest(event, cwd) {
  const pr = event.pull_request;
  requireThat(pr, 'Expected a pull_request event.');
  requireThat(conventionalTitle.test(pr.title), 'Use a Conventional Commit PR title.');
  const ownBranch = pr.head.repo.full_name === event.repository.full_name;
  if (pr.base.ref === 'master') {
    requireThat(ownBranch && pr.head.ref.startsWith('release/'), 'master accepts prepared release PRs from this repository only.');
    const baseSha = git(['rev-parse', 'origin/master'], cwd);
    requireThat(pr.base.sha === baseSha, 'PR checks must run against current master.');
    const plan = validateCandidate({ headSha: pr.head.sha, baseSha, branch: pr.head.ref, title: pr.title }, cwd);
    assertTagState(versionTags(cwd), baseSha, null, cwd);
    return plan;
  }
  requireThat(pr.base.ref === 'develop', 'PRs must target develop, or use a prepared release PR into master.');
  if (pr.head.ref.startsWith('sync/')) {
    requireThat(ownBranch, 'Synchronization branches must belong to this repository.');
    const version = pr.head.ref.slice('sync/'.length);
    parseVersion(version);
    const taggedSha = git(['rev-parse', `${version}^{commit}`], cwd);
    requireThat(isAncestor(taggedSha, pr.head.sha, cwd) &&
      isAncestor(taggedSha, 'origin/master', cwd), 'Synchronization must preserve the tagged master commit.');
    requireThat(pr.title === `chore(sync): merge ${version} into develop`, 'Unexpected synchronization PR title.');
  } else {
    requireThat(/^(feature|feat|fix|docs|ci|test|refactor|perf|build|chore|revert|style)\/.+/.test(pr.head.ref),
      'Use a contribution branch such as feature/name, fix/name, or docs/name. Hotfixes use Prepare release.');
  }
  return null;
}
