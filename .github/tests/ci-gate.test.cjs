const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const {requireSuccessfulCI, resolveTag, automaticTarget, releaseTarget, dockerTarget} = require('../ci-gate.cjs');

const SHA = 'a'.repeat(40);
const OTHER_SHA = 'b'.repeat(40);
const TAG_SHA = 'c'.repeat(40);
const repo = {owner: 'owner', repo: 'wallet'};
const repository = {full_name: 'owner/wallet'};

function run(overrides = {}) {
  return {
    id: 100, workflow_id: 7, path: '.github/workflows/ci.yaml', head_sha: SHA,
    head_branch: 'main', event: 'push', status: 'completed', conclusion: 'success',
    repository, head_repository: repository, ...overrides,
  };
}

function fixture(options = {}) {
  const runs = options.runs ?? [run()];
  const listWorkflowRuns = () => assert.fail('Use pagination for workflow runs');
  const github = {
    rest: {
      actions: {
        listWorkflowRuns,
        getWorkflow: async args => {
          assert.deepEqual(args, {...repo, workflow_id: 'ci.yaml'});
          if (options.apiError) throw new Error('API unavailable');
          return {data: {id: 7, path: '.github/workflows/ci.yaml', state: 'active', ...options.workflow}};
        },
        getWorkflowRun: async args => {
          assert.equal(args.owner, repo.owner);
          assert.equal(args.repo, repo.repo);
          const current = options.current ?? runs.find(item => item.id === args.run_id);
          assert.ok(current);
          return {data: current};
        },
      },
      git: {
        getRef: async args => {
          assert.equal(args.owner, repo.owner);
          assert.equal(args.repo, repo.repo);
          if (args.ref.startsWith('tags/')) {
            if (options.missingTag) throw new Error('Tag not found');
            return {data: {object: options.tagObject ?? {type: 'commit', sha: SHA}}};
          }
          assert.ok(['heads/main', 'heads/develop'].includes(args.ref));
          return {data: {object: {type: 'commit', sha: options.branchSha ?? SHA}}};
        },
        getTag: async args => {
          assert.deepEqual(args, {...repo, tag_sha: TAG_SHA});
          return {data: {object: options.annotatedObject ?? {type: 'commit', sha: SHA}}};
        },
      },
    },
    paginate: async (method, args) => {
      assert.equal(method, listWorkflowRuns);
      assert.deepEqual(args, {
        ...repo, workflow_id: 7, head_sha: SHA, branch: options.branch ?? 'main', per_page: 100,
      });
      if (options.paginationError) throw new Error('Pagination unavailable');
      return runs;
    },
  };
  return github;
}

function automatic(overrides = {}) {
  return {repo, eventName: 'workflow_run', sha: OTHER_SHA, ref: 'refs/heads/main', payload: {workflow_run: run()}, ...overrides};
}

function dispatch(overrides = {}) {
  return {repo, eventName: 'workflow_dispatch', sha: OTHER_SHA, ref: 'refs/heads/main', payload: {}, ...overrides};
}

function tagPush(overrides = {}) {
  return {repo, eventName: 'push', sha: SHA, ref: 'refs/tags/v1.2.3', payload: {}, ...overrides};
}

test('accepts the exact successful CI workflow and re-reads its current attempt', async () => {
  assert.equal((await requireSuccessfulCI(fixture(), repo, SHA, 'main')).id, 100);
});

for (const [name, overrides] of Object.entries({
  failed: {conclusion: 'failure'}, cancelled: {conclusion: 'cancelled'},
  neutral: {conclusion: 'neutral'}, skipped: {conclusion: 'skipped'},
  pending: {status: 'in_progress', conclusion: null}, queued: {status: 'queued', conclusion: null},
  otherSHA: {head_sha: OTHER_SHA}, otherBranch: {head_branch: 'feature/unapproved'},
  pullRequest: {event: 'pull_request'}, otherWorkflow: {workflow_id: 8},
  otherPath: {path: '.github/workflows/go.yaml'},
  fork: {head_repository: {full_name: 'fork/wallet'}},
  otherRepository: {repository: {full_name: 'other/wallet'}},
})) {
  test(`rejects ${name} CI`, async () => {
    await assert.rejects(requireSuccessfulCI(fixture({runs: [run(overrides)]}), repo, SHA, 'main'));
  });
}

for (const options of [
  {runs: []}, {apiError: true}, {paginationError: true},
  {workflow: {state: 'disabled_manually'}}, {workflow: {path: '.github/workflows/go.yaml'}},
]) {
  test(`fails closed for missing or unavailable CI: ${JSON.stringify(options)}`, async () => {
    await assert.rejects(requireSuccessfulCI(fixture(options), repo, SHA, 'main'));
  });
}

test('does not choose an older success over a newer failure or pending run', async () => {
  for (const status of [{conclusion: 'failure'}, {status: 'queued', conclusion: null}]) {
    await assert.rejects(requireSuccessfulCI(fixture({runs: [run(), run({id: 101, ...status})]}), repo, SHA, 'main'));
  }
});

test('rejects a failed re-run even if the list still describes its earlier success', async () => {
  await assert.rejects(requireSuccessfulCI(fixture({current: run({run_attempt: 2, conclusion: 'failure'})}), repo, SHA, 'main'));
});

test('accepts a successful manual CI on the same trusted branch and SHA', async () => {
  await requireSuccessfulCI(fixture({runs: [run({event: 'workflow_dispatch'})]}), repo, SHA, 'main');
});

test('a newer failed manual CI blocks publication despite an earlier push success', async () => {
  await assert.rejects(requireSuccessfulCI(fixture({runs: [run(), run({id: 101, event: 'workflow_dispatch', conclusion: 'failure'})]}), repo, SHA, 'main'));
});

test('automatic publication uses the CI SHA, not the workflow_run default-branch SHA', async () => {
  assert.deepEqual(await automaticTarget(fixture(), automatic()), {sha: SHA, branch: 'main', tag: '', publish: true});
});

test('automatic publication skips an approved commit after its branch advances', async () => {
  const target = await automaticTarget(fixture({branchSha: OTHER_SHA}), automatic());
  assert.equal(target.publish, false);
  assert.equal(target.sha, SHA);
});

for (const overrides of [
  {event: 'pull_request'}, {event: 'workflow_dispatch'}, {conclusion: 'failure'},
  {status: 'queued'}, {head_branch: 'develop'}, {head_repository: {full_name: 'fork/wallet'}},
]) {
  test(`rejects untrusted automatic trigger: ${JSON.stringify(overrides)}`, async () => {
    await assert.rejects(automaticTarget(fixture(), automatic({payload: {workflow_run: run(overrides)}})));
  });
}

test('rejects a completed event for an older run', async () => {
  await assert.rejects(automaticTarget(fixture({runs: [run(), run({id: 101})]}), automatic()));
});

test('resolves lightweight and annotated version tags', async () => {
  assert.equal(await resolveTag(fixture(), repo, 'v1.2.3'), SHA);
  assert.equal(await resolveTag(fixture({tagObject: {type: 'tag', sha: TAG_SHA}}), repo, 'v1.2.3-rc.1'), SHA);
});

for (const tag of ['main', '../main', 'v1.2.3\nsha=evil', 'v1.2.3;echo x', '', undefined]) {
  test(`rejects unsafe tag ${JSON.stringify(tag)}`, async () => {
    await assert.rejects(resolveTag(fixture(), repo, tag));
  });
}

test('fails closed for absent, non-commit or cyclic tags', async () => {
  for (const options of [
    {missingTag: true}, {tagObject: {type: 'tree', sha: SHA}},
    {tagObject: {type: 'commit', sha: 'invalid'}},
    {tagObject: {type: 'tag', sha: TAG_SHA}, annotatedObject: {type: 'tag', sha: TAG_SHA}},
  ]) {
    await assert.rejects(resolveTag(fixture(options), repo, 'v1.2.3'));
  }
});

test('manual release from main builds the approved tag SHA, not the dispatch SHA', async () => {
  assert.deepEqual(await releaseTarget(fixture(), dispatch(), 'v1.2.3'), {sha: SHA, tag: 'v1.2.3'});
});

test('tag push release requires the exact triggering SHA', async () => {
  assert.deepEqual(await releaseTarget(fixture(), tagPush(), 'v1.2.3'), {sha: SHA, tag: 'v1.2.3'});
  await assert.rejects(releaseTarget(fixture(), tagPush({sha: OTHER_SHA}), 'v1.2.3'));
});

test('manual release from an untrusted ref or without successful CI cannot publish', async () => {
  await assert.rejects(releaseTarget(fixture(), dispatch({ref: 'refs/heads/feature'}), 'v1.2.3'));
  await assert.rejects(releaseTarget(fixture({runs: []}), dispatch(), 'v1.2.3'));
  await assert.rejects(releaseTarget(fixture({runs: [run({conclusion: 'failure'})]}), tagPush(), 'v1.2.3'));
});

test('Docker supports approved main and develop pushes without confusing their SHAs', async () => {
  assert.equal((await dockerTarget(fixture(), automatic())).publish, true);
  const development = run({head_branch: 'develop'});
  const target = await dockerTarget(fixture({runs: [development], branch: 'develop'}), automatic({payload: {workflow_run: development}}));
  assert.deepEqual(target, {sha: SHA, branch: 'develop', tag: '', publish: true});
});

test('Docker manual and tag paths require CI and immutable target commits', async () => {
  assert.equal((await dockerTarget(fixture(), dispatch({sha: SHA}))).publish, true);
  assert.equal((await dockerTarget(fixture({branchSha: OTHER_SHA}), dispatch({sha: SHA}))).publish, false);
  assert.deepEqual(await dockerTarget(fixture(), tagPush()), {sha: SHA, branch: '', tag: 'v1.2.3', publish: true});
  assert.equal((await dockerTarget(fixture(), tagPush({eventName: 'workflow_dispatch'}))).publish, true);
  await assert.rejects(dockerTarget(fixture(), tagPush({sha: OTHER_SHA})));
  await assert.rejects(dockerTarget(fixture({runs: []}), tagPush()));
  await assert.rejects(dockerTarget(fixture(), dispatch({ref: 'refs/heads/feature'})));
  await assert.rejects(dockerTarget(fixture(), dispatch({eventName: 'pull_request'})));
  await assert.rejects(dockerTarget(fixture(), dispatch({eventName: 'push'})));
});

function workflow(name) {
  return fs.readFileSync(path.join(__dirname, '..', 'workflows', name), 'utf8');
}

function job(source, name) {
  const match = source.match(new RegExp(`^  ${name}:\\n[\\s\\S]*?(?=^  [a-z][a-z0-9-]*:\\n|(?![\\s\\S]))`, 'm'));
  assert.ok(match, `missing job ${name}`);
  return match[0];
}

test('versioning only follows trusted completed CI and pins checkout and commit lookup', () => {
  const source = workflow('version-bump.yml');
  assert.match(source, /on:\n  workflow_run:\n    workflows: \[CI\]\n    types: \[completed\]\n    branches: \[main\]/);
  const guard = job(source, 'verify-ci');
  for (const condition of ["conclusion == 'success'", "event == 'push'", "head_branch == 'main'", 'head_repository.full_name == github.repository']) {
    assert.ok(guard.includes(condition));
  }
  const bump = job(source, 'bump');
  assert.ok(bump.includes('needs: verify-ci'));
  assert.ok(bump.includes("if: needs.verify-ci.outputs.publish == 'true'"));
  assert.ok(bump.includes('ref: ${{ needs.verify-ci.outputs.sha }}'));
  assert.ok(bump.includes('commits/${RELEASE_SHA}/pulls'));
  assert.ok(bump.includes('gh workflow run release.yaml --ref main -f tag="$NEW_TAG"'));
  assert.ok(!bump.includes('GITHUB_SHA'));
});

test('release gates creation and pins every build and metadata to the approved SHA', () => {
  const source = workflow('release.yaml');
  const create = job(source, 'create-release');
  assert.ok(create.includes('needs: verify-ci'));
  assert.ok(create.includes('ref: ${{ needs.verify-ci.outputs.sha }}'));
  assert.ok(create.includes('sha: ${{ needs.verify-ci.outputs.sha }}'));
  for (const name of ['build-and-upload', 'build-and-upload-darwin-arm64-metal', 'docker-build']) {
    const build = job(source, name);
    assert.ok(build.includes('needs: create-release'));
    assert.ok(build.includes('ref: ${{ needs.create-release.outputs.sha }}'));
  }
  assert.ok(!source.includes('${{ github.sha }}'));
  assert.ok(job(source, 'notify').includes('if: success()'));
});

test('Docker PRs remain build-only while every publication uses gate outputs', () => {
  const source = workflow('docker.yaml');
  assert.match(source, /workflow_run:\n    workflows: \[CI\]\n    types: \[completed\]\n    branches: \[main, develop\]/);
  assert.match(source, /  push:\n    tags:/);
  const guard = job(source, 'verify-ci');
  assert.ok(guard.includes("github.event_name != 'pull_request'"));
  const build = job(source, 'build');
  assert.ok(build.includes('needs: verify-ci'));
  assert.ok(build.includes("github.event_name == 'pull_request' || (needs.verify-ci.result == 'success' && needs.verify-ci.outputs.publish == 'true')"));
  assert.ok(build.includes("push: ${{ needs.verify-ci.outputs.publish == 'true' }}"));
  assert.ok(build.includes('ref: ${{ needs.verify-ci.outputs.sha || github.sha }}'));
  assert.ok(build.includes('bloco-vgen@${{ steps.build.outputs.digest }}'));
  assert.ok(job(source, 'multi-arch-test').includes('IMAGE_REF: ${{ needs.build.outputs.image }}'));
  assert.ok(job(source, 'cleanup').includes("needs.verify-ci.outputs.branch == 'main'"));
});

test('all guards run trusted-main code with read-only tokens before any publication', () => {
  for (const name of ['version-bump.yml', 'release.yaml', 'docker.yaml']) {
    const guard = job(workflow(name), 'verify-ci');
    assert.ok(guard.includes('ref: main'));
    assert.ok(guard.includes('persist-credentials: false'));
    assert.ok(guard.includes('contents: read'));
    assert.ok(guard.includes('actions: read'));
    assert.doesNotMatch(guard, /(?:contents|actions|packages): write/);
  }
  assert.ok(workflow('ci.yaml').includes('node --test .github/tests/ci-gate.test.cjs'));
});
