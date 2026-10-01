#!/usr/bin/env bash
set -euo pipefail

REPO="${REPO:?}"
CURRENT="${GITHUB_RUN_ID:?}"
OUT="${GITHUB_OUTPUT:?}"
LEASE_OWNER="doctor-$CURRENT"
kick=false
fingerprint=""
outcome="noop"
lease_acquired=false

lease_json="$(jq -n --arg owner "$LEASE_OWNER" --arg started "$(date -u +%Y-%m-%dT%H:%M:%SZ)" --arg until "$(date -u -d '+4 minutes' +%Y-%m-%dT%H:%M:%SZ)" '{owner:$owner,started_at:$started,lease_until:$until,task_role:"AIDI GitHub Doctor"}')"
lease_encoded="$(printf '%s\n' "$lease_json" | base64 -w0)"
if gh api --method PUT "repos/$REPO/contents/.automation/lease.active" -f message="chore(automation): acquire $LEASE_OWNER lease" -f content="$lease_encoded" -f branch="automation-control" >/dev/null 2>&1; then
  lease_acquired=true
else
  exit 0
fi

release_lease() {
  [[ "$lease_acquired" == "true" ]] || return 0
  current="$(gh api "repos/$REPO/contents/.automation/lease.active?ref=automation-control" 2>/dev/null || true)"
  [[ -n "$current" ]] || { lease_acquired=false; return 0; }
  sha="$(jq -r '.sha' <<<"$current")"
  body="$(jq -r '.content' <<<"$current" | base64 -d)"
  owner="$(jq -r '.owner // empty' <<<"$body")"
  [[ "$owner" == "$LEASE_OWNER" ]] || { lease_acquired=false; return 0; }
  gh api --method DELETE "repos/$REPO/contents/.automation/lease.active" -f message="chore(automation): release $LEASE_OWNER lease" -f sha="$sha" -f branch="automation-control" >/dev/null
  lease_acquired=false
}
trap release_lease EXIT

update_circuit() {
  state_path=".automation/doctor/state.json"
  old_fp=""; old_repeats=0; old_sha=""
  if current="$(gh api "repos/$REPO/contents/$state_path?ref=automation-control" 2>/dev/null)"; then
    old_sha="$(jq -r '.sha' <<<"$current")"
    body="$(jq -r '.content' <<<"$current" | base64 -d)"
    old_fp="$(jq -r '.last_fingerprint // empty' <<<"$body")"
    old_repeats="$(jq -r '.repeat_count // 0' <<<"$body")"
  fi
  repeats=0; circuit=false
  if [[ -n "$fingerprint" ]]; then
    if [[ "$fingerprint" == "$old_fp" ]]; then repeats=$((old_repeats+1)); else repeats=1; fi
    if [[ "$repeats" -ge 3 ]]; then circuit=true; fi
  fi
  state="$(jq -n --arg fp "$fingerprint" --arg outcome "$outcome" --argjson repeats "$repeats" --argjson circuit "$circuit" --arg updated "$(date -u +%Y-%m-%dT%H:%M:%SZ)" '{last_fingerprint:$fp,repeat_count:$repeats,circuit_open:$circuit,last_outcome:$outcome,updated_at:$updated}')"
  encoded="$(printf '%s\n' "$state" | base64 -w0)"
  if [[ -n "$old_sha" ]]; then
    gh api --method PUT "repos/$REPO/contents/$state_path" -f message="chore(automation): update GitHub Doctor state" -f content="$encoded" -f sha="$old_sha" -f branch="automation-control" >/dev/null
  else
    gh api --method PUT "repos/$REPO/contents/$state_path" -f message="chore(automation): create GitHub Doctor state" -f content="$encoded" -f branch="automation-control" >/dev/null
  fi
  printf '%s' "$circuit"
}

emit() {
  echo "kick_core=$kick" >> "$OUT"
  echo "fingerprint=$fingerprint" >> "$OUT"
  echo "outcome=$outcome" >> "$OUT"
  circuit="$(update_circuit)"
  release_lease
  if [[ "$kick" == "true" && "$circuit" != "true" ]]; then
    path=".automation/kicks/doctor-$CURRENT.trigger"
    encoded="$(printf 'GitHub Doctor recovery completed; resume autonomous core.\n' | base64 -w0)"
    gh api --method PUT "repos/$REPO/contents/$path" -f message="chore(automation): resume core after Doctor recovery" -f content="$encoded" -f branch="automation-control" >/dev/null
  fi
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
