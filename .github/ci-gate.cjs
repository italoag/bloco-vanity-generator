const SHA = /^[0-9a-f]{40}$/;
const TAG = /^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/;
const CI_PATH = '.github/workflows/ci.yaml';

function repositoryName(repo) {
  return `${repo.owner}/${repo.repo}`.toLowerCase();
}

async function requireSuccessfulCI(github, repo, sha, branch, expectedRunId) {
  if (!SHA.test(sha) || !['main', 'develop'].includes(branch)) {
    throw new Error('Invalid CI target');
  }
  const {data: workflow} = await github.rest.actions.getWorkflow({...repo, workflow_id: 'ci.yaml'});
  if (workflow.path !== CI_PATH || workflow.state !== 'active') {
    throw new Error('Expected CI workflow is not active');
  }
  const matches = run => run.head_sha === sha && run.head_branch === branch &&
    ['push', 'workflow_dispatch'].includes(run.event) && run.workflow_id === workflow.id &&
    [CI_PATH, `${CI_PATH}@refs/heads/${branch}`].includes(run.path) &&
    run.repository?.full_name?.toLowerCase() === repositoryName(repo) &&
    run.head_repository?.full_name?.toLowerCase() === repositoryName(repo);
  const runs = await github.paginate(github.rest.actions.listWorkflowRuns, {
    ...repo, workflow_id: workflow.id, branch, head_sha: sha, per_page: 100,
  });
  const candidates = runs.filter(matches).sort((a, b) => b.id - a.id);
  const latest = candidates[0];
  const trigger = expectedRunId === undefined ? latest : candidates.find(run => run.id === expectedRunId);
  if (!latest || !trigger) {
    throw new Error(`No matching CI approval for ${sha} on ${branch}`);
  }
  let approved;
  for (const id of new Set([latest.id, trigger.id])) {
    const {data: current} = await github.rest.actions.getWorkflowRun({...repo, run_id: id});
    if (!matches(current) || current.id !== id || current.status !== 'completed' || current.conclusion !== 'success') {
      throw new Error(`CI approval has not succeeded for ${sha} on ${branch}`);
    }
    if (id === latest.id) approved = current;
  }
  return approved;
}

async function resolveTag(github, repo, tag) {
  if (!TAG.test(tag)) {
    throw new Error('Expected a version tag, not a branch or arbitrary ref');
  }
  const {data: ref} = await github.rest.git.getRef({...repo, ref: `tags/${tag}`});
  let object = ref.object;
  for (let depth = 0; object.type === 'tag' && depth < 8; depth++) {
    if (!SHA.test(object.sha)) {
      throw new Error('Invalid annotated tag object');
    }
    const {data: annotated} = await github.rest.git.getTag({...repo, tag_sha: object.sha});
    object = annotated.object;
  }
  if (object.type !== 'commit' || !SHA.test(object.sha)) {
    throw new Error('Tag does not resolve to a commit');
  }
  return object.sha;
}

async function branchTarget(github, repo, sha, branch, expectedRunId) {
  await requireSuccessfulCI(github, repo, sha, branch, expectedRunId);
  const {data: ref} = await github.rest.git.getRef({...repo, ref: `heads/${branch}`});
  return {sha, branch, tag: '', publish: ref.object.type === 'commit' && ref.object.sha === sha};
}

async function automaticTarget(github, context, branches = ['main']) {
  const run = context.payload.workflow_run;
  if (context.eventName !== 'workflow_run' || !run || !['push', 'workflow_dispatch'].includes(run.event) ||
      run.status !== 'completed' || run.conclusion !== 'success' || !branches.includes(run.head_branch) ||
      run.head_repository?.full_name?.toLowerCase() !== repositoryName(context.repo)) {
    throw new Error('Not an approved same-repository branch CI completion');
  }
  return branchTarget(github, context.repo, run.head_sha, run.head_branch, run.id);
}

async function releaseTarget(github, context, tag) {
  const manual = context.eventName === 'workflow_dispatch' && context.ref === 'refs/heads/main';
  const pushedTag = context.eventName === 'push' && context.ref === `refs/tags/${tag}`;
  if (!manual && !pushedTag) {
    throw new Error('Dispatch releases from main or push a version tag');
  }
  const sha = await resolveTag(github, context.repo, tag);
  if (pushedTag && sha !== context.sha) {
    throw new Error('Tag no longer matches the triggering commit');
  }
  await requireSuccessfulCI(github, context.repo, sha, 'main');
  return {sha, tag};
}

async function dockerTarget(github, context) {
  if (context.eventName === 'workflow_run') {
    return automaticTarget(github, context, ['main', 'develop']);
  }
  if (context.ref.startsWith('refs/tags/') && ['push', 'workflow_dispatch'].includes(context.eventName)) {
    const tag = context.ref.slice('refs/tags/'.length);
    const sha = await resolveTag(github, context.repo, tag);
    if (sha !== context.sha) {
      throw new Error('Tag no longer matches the triggering commit');
    }
    await requireSuccessfulCI(github, context.repo, sha, 'main');
    return {sha, branch: '', tag, publish: true};
  }
  if (context.eventName === 'workflow_dispatch' && ['refs/heads/main', 'refs/heads/develop'].includes(context.ref)) {
    return branchTarget(github, context.repo, context.sha, context.ref.slice('refs/heads/'.length));
  }
  throw new Error('Unsupported Docker publication event');
}

module.exports = {requireSuccessfulCI, resolveTag, automaticTarget, releaseTarget, dockerTarget};
