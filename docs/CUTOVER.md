> Canonical source: AIDI v2.0.0 — Controlled Cutover Plan (APPROVED)
> Source: https://docs.google.com/document/d/1Ud1BNCHx76qUBkM7FuOcJRvEWnUe8yIpM280t_5VCic/edit?usp=drivesdk
> Imported: 2026-09-26
> Status: APPROVED

﻿AIDI v2.0.0 — Controlled Cutover Plan
Freeze current development → import AIDI 2.0 baseline → activate Release A
1. Цель
Загрузить утверждённый AIDI 2.0.0 baseline и полный A–K/RC backlog в текущую AIDI без смешивания с legacy autonomous development и с возможностью безопасного rollback.
2. Phase 0 — Preconditions
* Master backlog validation PASS; administrative channel доступен; current AIDI state и exact source revisions зафиксированы.
* Restore procedure и backup targets определены до любого destructive change.
3. Phase 1 — Development Freeze
* Block creation of new non-cutover/legacy development tasks.
* Stop autonomous/developer/shadow/supervisor scheduling loops, не отключая management/DB/SCM/tunnels, необходимые для backup/import.
* Running Attempts довести до safe checkpoint либо controlled pause/cancel/reconciliation.
* Freeze gate: new_legacy_work = 0.
4. Phase 2 — Backup / Restore Point
* Snapshot/backup canonical DB, repositories/SCM, configs/policies, required artifacts and exact source SHA.
* Create hypervisor restore point where available.
* Validate logical backups and restore metadata. Import запрещён до Backup Gate PASS.
5. Phase 3 — Legacy Isolation
* Mark pre-2.0 projects/issues/tasks LEGACY PRE-2.0.
* Remove legacy work from active 2.0 execution queue while preserving history/evidence.
* Cutover does not delete legacy data.
6. Phase 4 — Import order
* 1) SPEC 2.0.0 baseline metadata.
* 2) Technology Binding & Defaults.
* 3) Acceptance Matrix / ПМИ baseline.
* 4) Release roadmap A–K/RC.
* 5) Milestones and labels/types/priorities.
* 6) 1470 implementation Issues.
* 7) 231 Gate Issues.
* 8) Hard dependency edges and release prerequisite gates.
* All import writes must be idempotent.
7. Phase 5 — Integrity Validation
* Expected release/issue/gate counts match manifest.
* Duplicate stable keys = 0; unresolved hard dependencies = 0; dependency cycles = 0.
* Legacy issues are not READY in 2.0 queue.
* Only Release A dependency-ready work may become READY; B–RC blocked.
8. Phase 6 — Activation
* Set AIDI 2.0 roadmap as authoritative development plan.
* Enable Release A queue only; re-enable scheduler/planner under 2.0 policy.
* Observe first cycle and verify no legacy task is spawned.
9. Phase 7 — Stabilization
* Monitor source revisions, leases, runners, recovery and queue correctness during first cycles.
* Any critical integrity/security/recovery anomaly before stability triggers rollback.
10. Rollback
* Re-freeze; disable 2.0 work creation; restore pre-cutover snapshot/backup; validate old state; record failed-cutover evidence; diagnose before retry.
11. Cutover PASS
* Backup gate PASS.
* Import validation PASS.
* Only Release A dependency-ready work can start.
* Legacy work isolated.
* First 2.0 scheduler cycle correct.
* No critical integrity/security/recovery anomaly.