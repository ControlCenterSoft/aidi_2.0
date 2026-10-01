#!/usr/bin/env bash
set -euo pipefail

REPO="${REPO:?}"
CURRENT="${GITHUB_RUN_ID:?}"
OUT="${GITHUB_OUTPUT:?}"
kick=false
fingerprint=""
outcome="noop"

emit() {
  echo "kick_core=$kick" >> "$OUT"
  echo "fingerprint=$fingerprint" >> "$OUT"
  echo "outcome=$outcome" >> "$OUT"
}

active="$(gh run list --repo "$REPO" --limit 100 --json databaseId,status |
  jq --argjson current "$CURRENT" '[.[] | select((.status=="queued" or .status=="in_progress") and .databaseId != $current)] | length')"
if [[ "$active" -gt 0 ]]; then
  outcome="other_run_active"
  emit
  exit 0
fi

prs="$(gh pr list --repo "$REPO" --state open --limit 100 --json number,headRefName |
  jq '[.[] | select(.headRefName | startswith("automation/"))]')"
count="$(jq 'length' <<<"$prs")"

if [[ "$count" -gt 1 ]]; then
  fingerprint="multiple_open_automation_prs"
  outcome="blocked_multiple_prs"
  emit
  exit 0
fi

if [[ "$count" -eq 1 ]]; then
  pr="$(jq -r '.[0].number' <<<"$prs")"
  head="$(gh pr view "$pr" --repo "$REPO" --json headRefName --jq '.headRefName')"
  sha="$(gh pr view "$pr" --repo "$REPO" --json headRefOid --jq '.headRefOid')"
  run="$(gh run list --repo "$REPO" --workflow ci.yml --branch "$head" --event workflow_dispatch --limit 20 --json databaseId,headSha,status,conclusion |
    jq --arg sha "$sha" '[.[] | select(.headSha==$sha)] | first // {}')"
  run_id="$(jq -r '.databaseId // empty' <<<"$run")"
  if [[ -z "$run_id" ]]; then
    kick=true
    outcome="missing_exact_head_ci"
    emit
    exit 0
  fi

  status="$(jq -r '.status // empty' <<<"$run")"
  conclusion="$(jq -r '.conclusion // empty' <<<"$run")"
  if [[ "$status" == "queued" || "$status" == "in_progress" ]]; then
    outcome="exact_head_ci_active"
    emit
    exit 0
  fi
  if [[ "$conclusion" == "success" ]]; then
    kick=true
    outcome="exact_head_ci_green"
    emit
    exit 0
  fi

  jobs="$(gh api "repos/$REPO/actions/runs/$run_id/jobs?filter=latest&per_page=100")"
  failed="$(jq '[.jobs[] | select(.conclusion=="failure" or .conclusion=="cancelled" or .conclusion=="timed_out")]' <<<"$jobs")"
  failed_count="$(jq 'length' <<<"$failed")"
  if [[ "$failed_count" -ne 1 ]]; then
    fingerprint="ci_multiple_or_ambiguous_failed_jobs"
    outcome="blocked_ci_not_single_job"
    emit
    exit 0
  fi

  failed_job="$(jq -r '.[0].id' <<<"$failed")"
  log="$(gh run view "$run_id" --repo "$REPO" --log-failed 2>&1 || true)"
  if grep -Eiq 'hosted runner|lost communication|connection reset|connection timed out|TLS|temporary failure|service unavailable|failed to download action|rate limit|502|503|504|network is unreachable|no space left on device' <<<"$log"; then
    gh api --method POST "repos/$REPO/actions/jobs/$failed_job/rerun" >/dev/null
    set +e
    gh run watch "$run_id" --repo "$REPO" --exit-status
    rc=$?
    set -e
    if [[ "$rc" -eq 0 ]]; then
      kick=true
      outcome="transient_ci_single_job_recovered"
    else
      fingerprint="transient_ci_rerun_failed"
      outcome="blocked_transient_rerun_failed"
    fi
    emit
    exit 0
  fi

  fingerprint="non_transient_exact_head_ci_failure"
  outcome="blocked_non_transient_ci"
  emit
  exit 0
fi

issues="$(gh issue list --repo "$REPO" --state open --limit 100 --json number,author |
  jq '[.[] | select(.author.login=="github-actions[bot]")]')"
branches="$(gh api --paginate --slurp "repos/$REPO/branches?per_page=100" | jq 'add')"
candidate_issue=""
candidate_branch=""
candidate_count=0

while IFS= read -r issue; do
  [[ -n "$issue" ]] || continue
  matches="$(jq --arg p "automation/core-$issue-" '[.[] | select(.name | startswith($p))] | map(.name)' <<<"$branches")"
  if [[ "$(jq 'length' <<<"$matches")" -eq 1 ]]; then
    branch="$(jq -r '.[0]' <<<"$matches")"
    existing="$(gh pr list --repo "$REPO" --state all --head "$branch" --limit 10 --json number | jq 'length')"
    if [[ "$existing" -eq 0 ]]; then
      candidate_count=$((candidate_count+1))
      candidate_issue="$issue"
      candidate_branch="$branch"
    fi
  fi
done < <(jq -r '.[].number' <<<"$issues")

if [[ "$candidate_count" -eq 1 ]]; then
  title="$(gh issue view "$candidate_issue" --repo "$REPO" --json title --jq '.title')"
  gh pr create --repo "$REPO" --draft --base main --head "$candidate_branch" --title "$title" --body "Automated GitHub-only recovery PR.

Closes #$candidate_issue

Recovery: metadata-only draft PR recreated for the existing stranded automation branch. No product source was modified by GitHub Doctor." >/dev/null
  kick=true
  outcome="orphan_issue_branch_pr_recovered"
  emit
  exit 0
fi

if [[ "$candidate_count" -gt 1 ]]; then
  fingerprint="multiple_orphan_issue_branch_candidates"
  outcome="blocked_ambiguous_orphans"
  emit
  exit 0
fi

kick=true
fingerprint="core_failure_without_durable_pr"
outcome="retry_core_once"
emit
