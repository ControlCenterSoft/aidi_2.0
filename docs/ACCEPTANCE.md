> Canonical source: AIDI v2.0.0 — Acceptance Matrix и ПМИ (APPROVED)
> Source: https://docs.google.com/document/d/1Ii9ZFYkQfi-FzVSGhH-VpaMK6RM2lqwuYuzBO_7gLPg/edit?usp=drivesdk
> Imported: 2026-09-26
> Status: APPROVED

﻿AIDI v2.0.0 — Acceptance Matrix и ПМИ
Status: APPROVED · Mandatory acceptance baseline
1. Общие правила приёмки
* Каждый Acceptance Criterion привязан к exact version, имеет Evidence и воспроизводимый PASS/FAIL.
* LLM-ответ, ручное утверждение разработчика или визуальный зелёный status без evidence не считаются приёмкой.
* Waiver возможен только по human/policy authority; отдельные security/integrity gates могут быть non-waivable.
* Production Release запрещён при failed mandatory ПМИ или unresolved Blocking Defect.
2. Installation
* AC-INST-001: clean supported system → полный installer → Installation Health Gate PASS.
* AC-INST-002: после install существует admin/admin; штатная эксплуатация до смены password невозможна.
* AC-INST-003: Initial Control Node является полноценным узлом AIDI и предоставляет Assessment/Deployment.
* AC-INST-004: offline bundle позволяет установить AIDI без Internet/Cloud AI.
3. Infrastructure
* AC-INF-001: Assessment получает фактические CPU/RAM/GPU/VRAM/storage/hypervisor capabilities.
* AC-INF-002: All-in-One/Distributed/HA рекомендуются только при достаточных ресурсах и реальных failure domains.
* AC-INF-003: при AUTO AIDI сама создаёт VM, устанавливает Agent и квалифицирует Node без ручного per-VM setup.
* AC-INF-004: Proxmox VE provider qualification PASS.
* AC-INF-005: Hyper-V provider qualification PASS.
* AC-INF-006: Node Agent enrollment/machine identity PASS.
* AC-INF-007: Runner не получает Node Agent administrative privileges.
* AC-INF-008: temporary/opportunistic Node корректно attach/drain/reconcile.
4. Local AI
* AC-AI-001: обязательный end-to-end lifecycle работает при CLOUD AI = DISABLED.
* AC-AI-002: Assistant работает на qualified local model.
* AC-AI-003: workflow запрашивает capability, Router выбирает deployment.
* AC-AI-004: Model не используется до qualification.
* AC-AI-005: при AUTO разрешённая model автоматически download→integrity/license→qualify→deploy.
* AC-AI-006: потеря primary local model имеет local fallback path.
5. Cloud AI
* AC-CLOUD-001: отсутствие всех Cloud Providers не блокирует AIDI.
* AC-CLOUD-002: effective cloud permission определяется policy hierarchy.
* AC-CLOUD-003: recommendation модели не предоставляет authorization.
* AC-CLOUD-004: Provider Wizard сохраняет credentials только через Secret Store.
* AC-CLOUD-005: paid synthetic test показывает provider/model/usage/cost/latency и не использует Project data.
* AC-CLOUD-006: Data/Regulatory Policy технически блокирует запрещённую передачу.
6. Identity & Access
* AC-ID-001: Local Identity работает независимо от AD.
* AC-ID-002: external directory authentication не сохраняет внешний password.
* AC-ID-003: Directory Group → AIDI Role mapping PASS.
* AC-ID-004: local break-glass сохраняет administrative recovery при отказе IdP.
* AC-ID-005: External Client получает только разрешённые commercial Projects.
* AC-ID-006: expiry прекращает access и сохраняет historical Audit attribution.
7. Invitations
* AC-INV-001: invite принимает exact login/email.
* AC-INV-002: UI/API не перечисляет глобальный directory.
* AC-INV-003: до ACCEPT ProjectMembership отсутствует.
* AC-INV-004: Invitation lifecycle доступен через Public API/Portal.
8. Email & Password Recovery
* AC-MAIL-001: SMTP configuration и test delivery PASS.
* AC-MAIL-002: Local Identity reset использует one-time expiring token.
* AC-MAIL-003: reset response не позволяет account enumeration.
* AC-MAIL-004: External IdP password AIDI не изменяет.
* AC-MAIL-005: Client Portal User не получает AIDI recovery email напрямую.
* AC-MAIL-006: internal project notifications управляются preferences/policy.
9. Specification Engine
* AC-SPEC-001: Project можно создать из одной пользовательской формулировки «Я хочу…».
* AC-SPEC-002: Discovery анализирует доступный контекст до вопросов.
* AC-SPEC-003: значимые решения становятся structured Requirements/Decisions.
* AC-SPEC-004: critical conflict блокирует Approval.
* AC-SPEC-005: mandatory Feasibility завершён до Approval.
* AC-SPEC-006: Product Runtime Profile определён до Approval.
* AC-SPEC-007: итоговое ТЗ содержит технологии конечного продукта.
* AC-SPEC-008: для installable product сформирован installation guide.
* AC-SPEC-009: Approval связан с exact Specification version.
10. Existing Product
* AC-LEG-001: внешний Git repository импортируется с exact revision.
* AC-LEG-002: создаётся local canonical private repository.
* AC-LEG-003: до planning выполнен AS-IS analysis.
11. Architecture & Planning
* AC-PLAN-001: Architecture выводится из approved Requirements.
* AC-PLAN-002: default — минимальная достаточная сложность.
* AC-PLAN-003: explicit customer architectural constraint проверяется feasibility и учитывается.
* AC-PLAN-004: Requirements распределены по Releases/Features.
* AC-PLAN-005: Tasks имеют explicit dependencies.
* AC-PLAN-006: plan учитывает actual resources/capabilities.
* AC-PLAN-007: изменения условий вызывают controlled replanning.
12. Autonomous Development
* AC-EXEC-001: Task и Attempt отдельны.
* AC-EXEC-002: Attempt привязан к exact source revision.
* AC-EXEC-003: generated code не меняет canonical branch напрямую.
* AC-EXEC-004: model statement «готово» не закрывает Task.
* AC-EXEC-005: independent tasks могут идти параллельно без source corruption.
* AC-EXEC-006: ChangeSet проходит validation перед merge.
13. Recovery
* AC-REC-001: failure classes различаются Task/Project/Infrastructure/Cloud/Internal.
* AC-REC-002: deterministic failure не повторяется бесконечно без strategy change.
* AC-REC-003: Root Cause evidence-based.
* AC-REC-004: successful recovery автоматически resumes workflow.
* AC-REC-005: существенный Recovery заканчивается Prevention Review.
* AC-REC-006: planned restart не создаёт false Warning.
* AC-REC-007: MaintenanceIntent overrun создаёт Problem.
* AC-REC-008: Cloud failure не останавливает eligible local execution.
14. Self-development
* AC-SYS-001: собственная разработка представлена AIDI_SYSTEM_PROJECT.
* AC-SYS-002: direct production self-edit запрещён.
* AC-SYS-003: self-update проходит canary.
* AC-SYS-004: bad candidate автоматически rollback к Last Known Good.
* AC-SYS-005: System Resource Reserve/quotas enforceable.
15. Canonical State & Concurrency
* AC-STATE-001: объект имеет один authoritative current state.
* AC-STATE-002: significant transition фиксирует State+Event атомарно.
* AC-STATE-003: Outbox сохраняет event при messaging outage.
* AC-STATE-004: duplicate command/event не создаёт duplicate side effect.
* AC-STATE-005: stale revision отклоняется.
* AC-STATE-006: stale owner/fencing token не может писать canonical state.
* AC-STATE-007: projection rebuild не теряет authoritative data.
16. Resource Management
* AC-RES-001: capacity подтверждается live telemetry/freshness.
* AC-RES-002: double allocation предотвращён reservation/lease.
* AC-RES-003: provisioning учитывает effective Role/User/Project policy.
* AC-RES-004: AUTO VM provisioning выполняется AIDI.
* AC-RES-005: REQUEST_ADMIN создаёт ActionRequest вместо скрытого failure.
* AC-RES-006: новый Node/Runner проходит representative qualification.
17. Security
* AC-SEC-001: User A не читает Project B.
* AC-SEC-002: RAG не смешивает private Projects.
* AC-SEC-003: Secrets не попадают в prompts/logs.
* AC-SEC-004: privileged tool action проходит Tool Broker authorization/policy.
* AC-SEC-005: prompt injection не получает system authority.
* AC-SEC-006: Assistant видит не больше текущего User.
* AC-SEC-007: temporary/untrusted Node не получает запрещённые secrets/data.
18. CRM / Client Portal
* AC-CRM-001: CRM не имеет direct DB access.
* AC-CRM-002: Portal→CRM→AIDI Project lifecycle PASS.
* AC-CRM-003: client approval exact Specification version передаётся AIDI.
* AC-CRM-004: AIDI не является коммерческим pricing/budget authority.
* AC-CRM-005: technical Change Impact рассчитывается reproducibly.
* AC-CRM-006: commercial threshold применяет CRM.
* AC-CRM-007: CRM outage не останавливает независимую engineering development.
19. Storage
* AC-STO-001: AIDI storage независим от HM.DM file storage.
* AC-STO-002: cross-project file access запрещён.
* AC-STO-003: critical files/artifacts имеют checksums.
* AC-STO-004: Released Artifact нельзя тихо заменить.
* AC-STO-005: external access только expiring grants.
* AC-STO-006: temporary storage outage reconciled/resumed.
20. Observability
* AC-OBS-001: correlation ID восстанавливает execution path.
* AC-OBS-002: один root failure не создаёт notification storm.
* AC-OBS-003: Super Admin видит system notifications/actions и delivery status.
* AC-OBS-004: secret redaction PASS.
* AC-OBS-005: Problem→Attempt→Trace→Logs→Recovery доступно из admin workflow.
21. Backup / Restore / DR
* AC-DR-001: Backup имеет Manifest/checksums.
* AC-DR-002: Backup без successful Restore Test не считается полностью validated.
* AC-DR-003: AIDI восстанавливается на другое hardware.
* AC-DR-004: full installation loss восстанавливается без обязательного Cloud AI.
* AC-DR-005: pre-disaster RUNNING Attempt reconciled, а не дублируется.
22. Update
* AC-UPD-001: существует qualified Last Known Good.
* AC-UPD-002: critical update проверяется как candidate/canary до promotion.
* AC-UPD-003: failed candidate автоматически rollback.
* AC-UPD-004: failed Node Agent update не лишает management access.
23. Performance & HA
* AC-HA-001: полноценная AIDI работает All-in-One.
* AC-HA-002: несколько VM на одном physical host не считаются полноценным HA.
* AC-HA-003: user workload не вытесняет Control Plane Reserve.
* AC-HA-004: network partition не создаёт двух authoritative writers.
* AC-HA-005: HA profile выполняет controlled failover.
* AC-HA-006: потеря части compute capacity приводит к graceful degradation.
24. Licensing / Supply Chain
* AC-SC-001: Unknown-license component не проходит AUTO admission.
* AC-SC-002: changed package digest detected.
* AC-SC-003: AIDI Release имеет exact-version SBOM.
* AC-SC-004: known vulnerability получает impact assessment.
* AC-SC-005: license change вызывает re-admission.
* AC-SC-006: self-developed AIDI artifact проходит те же supply-chain gates.
25. Regulatory / Retention
* AC-RU-001: Project получает Regulatory Applicability Assessment.
* AC-RU-002: personal-data applicability активирует соответствующие policies.
* AC-RU-003: residency policy enforceable.
* AC-RU-004: разрешённый Cloud Provider всё равно не получает запрещённые regulatory data.
* AC-RU-005: ambiguity → LEGAL_REVIEW_REQUIRED.
* AC-RET-001: Hidden Project не удаляется физически немедленно.
* AC-RET-002: archived Project имеет explicit retention deadline.
* AC-RET-003: HOLD блокирует purge.
* AC-RET-004: derived RAG/search/extracted data следует lifecycle source.
* AC-RET-005: ошибка удаления одной managed copy → PURGE_INCOMPLETE.
* AC-RET-006: retention shortening показывает impact до применения.
26. UI / Assistant
* AC-UI-001: ru-RU default.
* AC-UI-002: EN переключается без изменения canonical IDs.
* AC-UI-003: timezone пользователя применяется к display, internal timestamps UTC.
* AC-UI-004: human-readable dates/times используются как основной UX.
* AC-UI-005: основные USER flows работают на mobile.
* AC-UI-006: один canonical object отображает согласованный state во всех projections.
* AC-ASST-001: Assistant работает после qualification local LLM.
* AC-ASST-002: Super Admin может глобально отключить Assistant.
* AC-ASST-003: Assistant context ограничен effective user permissions.
* AC-ASST-004: запрещённый Tool operation невозможен.
* AC-ASST-005: significant outcomes превращаются в canonical entities.
27. ПМИ
* AC-PMI-001: ПМИ — versioned human-readable + machine-readable artifact.
* AC-PMI-002: automatable steps исполняет AIDI.
* AC-PMI-003: manual steps выполняются через Guided Tester Mode.
* AC-PMI-004: каждый critical step имеет Evidence.
* AC-PMI-005: после испытаний автоматически создаётся Протокол испытаний.
* AC-PMI-006: production release невозможен при failed mandatory ПМИ.
28. Full Local-Only E2E
Clean install → admin password change → Cloud OFF → local LLM → Infrastructure Assessment → distributed deployment if possible → user → «Я хочу…» → Discovery → Specification → Feasibility → Approval → Architecture → Planning → Autonomous Development → Local Review → Tests → injected failure → Detect/RootCause/Recovery/Prevention → Auto Resume → Build → Clean Install Product → RC → Acceptance → Released Artifact.
* PASS: ни один обязательный AI-step не требует Cloud AI.
* PASS: штатная recoverable failure не требует ручного администрирования.
29. Definition of Done AIDI 2.0.0
* Все mandatory requirements implemented and evidenced.
* Все mandatory ПМИ suites PASS.
* Release SBOM, signatures/checksums, verified installer и Protocol of Tests существуют.
* Unresolved Blocking Defects = 0.
* Release Artifact Set immutable.