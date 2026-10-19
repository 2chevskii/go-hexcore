import { appendFileSync, readFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';
import { execFileSync } from 'node:child_process';
import {
  assertTagState, git, isAncestor, parseVersion, readAt, releasePlan,
  validateCandidate, validatePullRequest, versionTags,
} from './release.mjs';

function requireThat(condition, message) {
  if (!condition) throw new Error(message);
}

export class GitHub {
  constructor(repository, token, request = fetch) {
    requireThat(repository === '2chevskii/go-hexcore', 'Release automation is restricted to 2chevskii/go-hexcore.');
    requireThat(token, 'GH_TOKEN is required.');
    this.repository = repository;
    this.token = token;
    this.request = request;
  }

  async api(path, { method = 'GET', body, optional = false } = {}) {
    const response = await this.request(`https://api.github.com/repos/${this.repository}/${path}`, {
      method,
      headers: {
        Authorization: `Bearer ${this.token}`,
        Accept: 'application/vnd.github+json',
        'X-GitHub-Api-Version': '2022-11-28',
        'Content-Type': 'application/json',
      },
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: AbortSignal.timeout(30_000),
    });
    if (optional && response.status === 404) return null;
    if (response.status === 204) return null;
    const data = await response.json();
    requireThat(response.ok, `GitHub ${method} ${path}: HTTP ${response.status}: ${data.message || 'request failed'}`);
    return data;
  }

  async branch(name) {
    return this.api(`git/ref/heads/${encodeURIComponent(name)}`, { optional: true });
  }

  async createBranch(name, sha) {
    const existing = await this.branch(name);
    if (existing) {
      requireThat(existing.object.sha === sha, `${name} already exists at another commit.`);
      return;
    }
    await this.api('git/refs', { method: 'POST', body: { ref: `refs/heads/${name}`, sha } });
  }

  async ensurePR(branch, base, title, body) {
    const owner = this.repository.split('/')[0];
    const query = new URLSearchParams({ state: 'open', head: `${owner}:${branch}`, base });
    const existing = await this.api(`pulls?${query}`);
    requireThat(existing.length <= 1, `Multiple open PRs for ${branch}.`);
    if (existing.length) {
      requireThat(existing[0].title === title, 'Existing PR title differs from the generated title.');
      return existing[0];
    }
    // A superseded candidate may have used the same version branch name.
    // Create a fresh PR rather than reopening an old review of another SHA.
    return this.api('pulls', { method: 'POST', body: { head: branch, base, title, body } });
  }
}

function refresh() {
  git(['fetch', '--prune', '--tags', 'origin', '+refs/heads/*:refs/remotes/origin/*']);
}

function summarize(text) {
  console.log(text);
  if (process.env.GITHUB_STEP_SUMMARY) appendFileSync(process.env.GITHUB_STEP_SUMMARY, `${text}\n`);
}

function requireDefaultBranch() {
  requireThat(process.env.GITHUB_REF === 'refs/heads/develop', 'Run this workflow from the default develop branch.');
}

export async function prepare(github, sourceBranch, { refreshRepository = refresh, report = summarize } = {}) {
  requireDefaultBranch();
  refreshRepository();
  requireThat(sourceBranch === 'develop' || /^hotfix\/[a-zA-Z0-9][a-zA-Z0-9._/-]*$/.test(sourceBranch),
    'Source must be develop or hotfix/name.');
  const baseSha = git(['rev-parse', 'origin/master']);
  const sourceSha = git(['rev-parse', `refs/remotes/origin/${sourceBranch}^{commit}`]);
  assertTagState(versionTags(), baseSha);
  let plan = releasePlan({ baseSha, sourceSha, sourceBranch, date: new Date().toISOString().slice(0, 10) });
  const branch = `release/${plan.metadata.version}`;
  const title = `chore(release): ${plan.metadata.version}`;
  const existing = await github.branch(branch);
  if (existing) {
    plan = validateCandidate({ headSha: existing.object.sha, baseSha, branch, title });
    requireThat(plan.metadata.sourceSha === sourceSha && plan.metadata.sourceBranch === sourceBranch,
      'This version already has a frozen candidate. Merge it, or close its PR and delete its release branch before preparing again.');
  } else {
    const source = await github.api(`git/commits/${sourceSha}`);
    const tree = await github.api('git/trees', { method: 'POST', body: {
      base_tree: source.tree.sha,
      tree: [
        { path: '.release.json', mode: '100644', type: 'blob', content: plan.metadataText },
        { path: 'CHANGELOG.md', mode: '100644', type: 'blob', content: plan.changelog },
      ],
    } });
    const commit = await github.api('git/commits', { method: 'POST', body: {
      message: title, tree: tree.sha, parents: [sourceSha],
    } });
    await github.createBranch(branch, commit.sha);
  }
  const { version, previousVersion } = plan.metadata;
  const pr = await github.ensurePR(branch, 'master', title, [
    `Release **${version}** from frozen \`${sourceBranch}\` commit \`${sourceSha}\`.`,
    `Previous release: ${previousVersion || 'none; bootstrap release is v1.0.0'}.`,
    plan.notes,
    'Review the full diff and CI results, then use **Create a merge commit** to publish. Squash and rebase merges are prohibited on master.',
    'If CI is awaiting approval, select **Approve workflows to run** in this PR. Preparation uses GITHUB_TOKEN.',
    'If master changes during review, close this PR, delete its release branch, merge the synchronization PR, and prepare again.',
  ].join('\n\n'));
  report(`Prepared ${version}: ${pr.html_url}\n\nNo tag has been created. Merge the PR when ready to publish.`);
  return pr;
}

export function publicationPlan(sha, pr, cwd) {
  requireThat(/^[0-9a-f]{40}$/.test(sha), 'Expected a full publication commit SHA.');
  requireThat(isAncestor(sha, 'origin/master', cwd), 'Publication commit is not on master.');
  requireThat(pr.merged_at && pr.base.ref === 'master' && pr.merge_commit_sha === sha &&
    pr.head.repo?.full_name === '2chevskii/go-hexcore', 'Commit must come from a merged release PR in this repository.');
  const parents = git(['show', '-s', '--format=%P', sha], cwd).split(' ');
  requireThat(parents.length === 2 && parents[1] === pr.head.sha, 'Release must use a merge commit preserving the candidate.');
  const plan = validateCandidate({ headSha: parents[1], baseSha: parents[0], branch: pr.head.ref, title: pr.title }, cwd);
  requireThat(git(['rev-parse', `${sha}^{tree}`], cwd) === git(['rev-parse', `${parents[1]}^{tree}`], cwd),
    'Merged release tree differs from the reviewed candidate.');
  assertTagState(versionTags(cwd), parents[0], { version: plan.metadata.version, sha }, cwd);
  return plan;
}

export async function ensureTag(github, version, sha) {
  parseVersion(version);
  const existing = await github.api(`git/ref/tags/${version}`, { optional: true });
  if (existing) {
    let object = existing.object;
    while (object.type === 'tag') object = (await github.api(`git/tags/${object.sha}`)).object;
    requireThat(object.type === 'commit' && object.sha === sha, `${version} already points to another commit. Never move a published tag.`);
    return;
  }
  const tag = await github.api('git/tags', { method: 'POST', body: {
    tag: version, message: `Release ${version}`, object: sha, type: 'commit',
  } });
  await github.api('git/refs', { method: 'POST', body: { ref: `refs/tags/${version}`, sha: tag.sha } });
}

export function verifyGoProxy(modulePath, version) {
  execFileSync('go', ['list', '-m', '-json', `${modulePath}@${version}`], {
    encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'],
    env: { ...process.env, GOPROXY: 'https://proxy.golang.org', GOSUMDB: 'sum.golang.org', GOPRIVATE: '', GONOPROXY: '', GONOSUMDB: '' },
    timeout: 120_000,
  });
}

export async function publish(github, sha, { refreshRepository = refresh, verifyProxy = verifyGoProxy, report = summarize } = {}) {
  requireThat(/^[0-9a-f]{40}$/.test(sha), 'Expected a full publication commit SHA.');
  refreshRepository();
  const associated = await github.api(`commits/${sha}/pulls?per_page=100`);
  const matches = associated.filter(pr => pr.merged_at && pr.merge_commit_sha === sha && pr.base.ref === 'master');
  requireThat(matches.length === 1, 'Expected exactly one merged release PR for this commit.');
  const plan = publicationPlan(sha, matches[0]);
  const runs = await github.api(`actions/workflows/ci.yml/runs?${new URLSearchParams({ head_sha: sha, event: 'push', per_page: '100' })}`);
  requireThat(runs.workflow_runs.some(run => run.head_sha === sha && run.head_branch === 'master' && run.conclusion === 'success'),
    'Successful push CI on this exact master commit is required before publication. Wait for CI or rerun CI first.');
  const { version } = plan.metadata;
  await ensureTag(github, version, sha);
  const existingRelease = await github.api(`releases/tags/${version}`, { optional: true });
  if (existingRelease) {
    requireThat(!existingRelease.draft && !existingRelease.prerelease, 'Existing GitHub release must be a published stable release.');
  } else {
    await github.api('releases', { method: 'POST', body: {
      tag_name: version, target_commitish: sha, name: version,
      body: plan.notes, draft: false, prerelease: false, make_latest: 'true',
    } });
  }
  const developSha = git(['rev-parse', 'origin/develop']);
  if (!isAncestor(sha, developSha)) {
    const branch = `sync/${version}`;
    await github.createBranch(branch, sha);
    const sync = await github.ensurePR(branch, 'develop', `chore(sync): merge ${version} into develop`,
      `Bring the tagged master commit into develop, including release metadata and hotfixes.\n\nUse **Create a merge commit** to preserve ancestry. Do not squash or rebase.\n\nApprove pending bot-created CI runs if GitHub requests it. Resolve conflicts in a separate sync branch while preserving this tagged commit as an ancestor.`);
    report(`Synchronization PR: ${sync.html_url}\n\nMerge it with a merge commit before preparing another release.`);
  }
  // The tag is already public. A proxy failure is recovered by rerunning this
  // workflow, never by moving or deleting the tag.
  const [major] = parseVersion(version);
  const modulePath = `github.com/2chevskii/go-hexcore${major > 1n ? `/v${major}` : ''}`;
  await verifyProxy(modulePath, version);
  report(`Published ${version} at ${sha}.\n\nhttps://github.com/${github.repository}/releases/tag/${version}\n\nGo module proxy verification passed.`);
  return plan;
}

async function main() {
  const command = process.argv[2];
  if (command === 'check-pr') {
    refresh();
    validatePullRequest(JSON.parse(readFileSync(process.env.GITHUB_EVENT_PATH, 'utf8')));
    console.log('Branch and release policy checks passed.');
    return;
  }
  requireDefaultBranch();
  const github = new GitHub(process.env.GITHUB_REPOSITORY, process.env.GH_TOKEN);
  if (command === 'prepare') return prepare(github, process.env.RELEASE_SOURCE || 'develop');
  if (command === 'publish') {
    const event = JSON.parse(readFileSync(process.env.GITHUB_EVENT_PATH, 'utf8'));
    const sha = event.workflow_run?.head_sha || process.env.RELEASE_COMMIT || git(['rev-parse', 'origin/master']);
    return publish(github, sha);
  }
  throw new Error(`Unknown command: ${command}`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main().catch(error => { console.error(error.message); process.exitCode = 1; });
}
