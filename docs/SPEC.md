> Canonical source: AIDI v2.0.0 — Полное техническое задание (APPROVED)
> Source: https://docs.google.com/document/d/1Yns_yXpw6y72aKKtsY8OBTHjZKwDeHXlLW5fpy7gUZk/edit?usp=drivesdk
> Imported: 2026-09-26
> Status: APPROVED

﻿AIDI v2.0.0 — Полное техническое задание
Утверждённая baseline-редакция · Status: APPROVED · Готовность: 100%
0. Статус документа и управление baseline
Настоящий документ является утверждённым техническим заданием AIDI v2.0.0 и нормативным источником требований для дальнейшей реализации. Функциональная архитектура, Technology Binding & Defaults, Acceptance Matrix и модель ПМИ согласованы. Любое изменение требований после фиксации baseline выполняется только через SPEC Change Request с версионированием, анализом влияния и аудитом.
* Текущая AIDI до cutover не считается реализацией AIDI 2.0.0 и может использоваться только как источник опыта, тестовых сценариев и антипримеров.
* AIDI Seed полностью исключён. Не существует отдельной урезанной Seed-редакции, Seed resource profile или bootstrap self-development продукта.
* Первый устанавливаемый сервер называется Initial Control Node / Первый управляющий узел и является полноценным узлом полной AIDI 2.0.0.
* После установки полной AIDI разрешены безопасные self-repair и self-development через AIDI_SYSTEM_PROJECT, canary, Guardian и rollback.
1. Назначение и границы продукта
AIDI 2.0.0 — локальная автономная многопользовательская платформа полного цикла разработки программного обеспечения. Пользователь формулирует желаемый результат, а AIDI организует инженерный процесс: Discovery → ТЗ → Feasibility → Approval → Architecture → Planning → Development → Testing → Review → Recovery → Build → Acceptance → Release → Support/Evolution.
* Основная ценность: автономная разработка без обязательной зависимости от облачных AI/SaaS и без обязательного ручного администрирования штатных восстанавливаемых ситуаций.
* CRM и Client Portal могут быть тесно интегрированы с AIDI, но не являются обязательными компонентами ядра AIDI.
* AIDI поддерживает несколько пользователей и несколько независимых development pipelines с изоляцией данных, ресурсов и прав.
* Для обычного пользователя основной сценарий начинается с фразы «Я хочу…»; AIDI помогает сформулировать и утвердить ТЗ, затем ведёт процесс до результата.
2. Глобальные принципы
2.1 Local First
* Все обязательные AI-функции должны иметь полностью локальный путь: Specification, architecture, planning, coding, review, debugging, testing, recovery и release.
* Интернет разрешён как источник документации, исходного кода, пакетов, обновлений, security/regulatory intelligence и research.
* Cloud AI — optional partner, а не prerequisite. Его отсутствие, отказ, quota/rate limit или исчерпание бюджета не должны блокировать базовый локальный lifecycle.
2.2 Durable/event-driven execution
* Оркестрация строится на durable workflows и событиях, а не на наборе cron-скриптов и не на таймерах как источнике состояния.
* Каноническое состояние отделено от UI, кэшей, Kanban, CRM projections, search и RAG.
* Task и Attempt — разные сущности. RUNNING допустим только при реальном executor ownership/lease.
2.3 Evidence-driven engineering
* LLM-фраза «готово» никогда не является достаточным основанием для DONE.
* Task/Feature/Release закрываются только при наличии проверяемых evidence: tests, validators, review, build, CI, acceptance и иных обязательных gates.
2.4 Explainability and safety
* Значимые решения должны быть объяснимыми: почему выбрана модель, ресурс, topology, policy, recovery strategy или архитектурное решение.
* Неизвестное состояние обозначается UNKNOWN; запрещено изображать ложный RUNNING/HEALTHY.
* Critical commands idempotent; side effects контролируются; optimistic concurrency, fencing и reconciliation обязательны.
3. Пользователи, Workspace, Project и Identity
3.1 Identity
* Поддерживаются Local Identity, Microsoft Active Directory, Samba AD, LDAP/LDAPS, FreeIPA и OIDC-compatible identity providers.
* После clean install создаётся local admin/admin. Система остаётся UNINITIALIZED до обязательной смены пароля при первом входе.
* Local administrative identity сохраняется как break-glass даже при использовании внешнего IdP.
* Directory groups могут маппиться на AIDI roles; несколько IdP могут существовать одновременно.
3.2 Roles and membership
* Глобальная роль SUPER_ADMIN имеет полный системный административный контур. Обычные пользователи не получают системные полномочия автоматически.
* Project roles: PROJECT_OWNER, PRODUCT_OWNER, CONTRIBUTOR, VIEWER, CLIENT, PROJECT_MANAGER.
* Workspace — самостоятельная сущность. Workspace membership и Project membership независимы: участие в Workspace не означает автоматического доступа ко всем Projects.
* Project invitation создаётся только по exact login/email. Глобальный user directory/autocomplete не раскрывается. Доступ появляется только после ACCEPT; поддерживаются DECLINE, EXPIRE и REVOKE.
* External Client Account ограничен конкретными Projects и contract lifetime; окончание доступа не удаляет исторические approvals/audit.
3.3 Service identities
* Node Agent, CRM, providers и automation используют отдельные service/workload identities, а не пользовательские аккаунты.
* Workload credentials по возможности short-lived; least privilege обязателен.
4. Каноническая модель данных и состояния
4.1 Основная иерархия
Installation → Identity/User → Workspace → Project → Specification → Requirement → Release → Feature → Task → Workflow → Attempt → ChangeSet/Verification/Evidence/Artifact. Дополнительные first-class сущности: Decision, Approval, Risk, ChangeRequest, Problem, RecoveryCase, Policy, Resource, Event/Audit, OperationalKnowledge.
4.2 Canonical state
* Используется current transactional canonical state + append-only Event Journal; pure event sourcing не требуется.
* Каждый значимый переход фиксирует state и Event атомарно. Event содержит ID, type, object/revision, timestamp, actor, correlation, causation и schema version.
* Transactional Outbox предотвращает потерю committed events при недоступности message bus; Inbox/dedup предотвращает повторные side effects.
* Object revisions и optimistic concurrency обязательны; last-write-wins для критического state запрещён.
* Workers, LLM, Runner и integrations не пишут canonical DB напрямую; переходы выполняются authoritative controllers через валидируемые commands.
* Derived projections, search indexes, RAG и caches полностью rebuildable и не являются source of truth.
4.3 Workflow ownership
* Workflow/Attempt state persisted, а не RAM-only. Lease и fencing token являются first-class; просроченный owner не может записать результат после потери ownership.
* Lease expiry переводит систему в reconciliation, а не автоматически в FAILED/retry.
* Replay workflow не должен повторно выполнять внешний side effect без idempotency/side-effect record.
5. User Lifecycle и Specification Engine
5.1 Project lifecycle
Идея → Discovery → Requirements → ТЗ → Feasibility → Approval → Architecture → Planning → Releases/Features/Tasks → autonomous development → verification/recovery/build → Release Candidate → Acceptance → Delivery → Support/Evolution.
5.2 Adaptive discovery
* AIDI сначала анализирует всё доступное описание и контекст, затем задаёт только необходимые вопросы.
* Используется структурированная модель Known / Unknown / Assumed / Conflict / Risk / DecisionRequired / Derived.
* Пользователя не заставляют выбирать инженерные детали, которые AIDI способна безопасно определить самостоятельно.
* Значимые сообщения Chat/Conversation преобразуются в Requirement/Decision/Approval/ChangeRequest и не остаются только в истории чата.
5.3 Requirements quality
* Requirements должны быть версионируемыми, трассируемыми к источнику и acceptance criteria, проверяемыми и достаточно однозначными.
* Обязательны обнаружение duplicate/conflict/ambiguity, derived engineering requirements, out-of-scope и quality gates.
* Critical unresolved conflict блокирует READY_FOR_APPROVAL.
5.4 Mandatory Feasibility Analysis
* Feasibility выполняется до Approval и является обязательным gate, а не справочным отчётом.
* Проверяется техническая реализуемость, APIs/SDK/dependencies, local development/testing/deployment/installability, licensing, security/regulatory risks.
* Отдельно рассчитываются Development Resource Profile и Product Runtime Profile; Minimum и Recommended runtime requirements должны попасть в итоговое ТЗ.
5.5 Итоговое ТЗ создаваемого продукта
* Итоговая Specification содержит назначение, actors/scenarios, functional/non-functional requirements, data, integrations, security, performance, deployment, runtime, constraints, acceptance, risks и out-of-scope.
* Обязательно указываются технологии и состав конечного продукта, поддерживаемые OS/platform/runtime, Minimum/Recommended resources и installation/update architecture.
* Для устанавливаемого продукта создаётся базовая инструкция установки ориентировочно 1–3 страницы либо больше, если продукт объективно сложнее.
* Approval всегда относится к exact Specification version. После approval изменения выполняются только как Change Request.
6. Existing Product Mode
* AIDI принимает existing source, docs, APIs, DB/schema, tests, screenshots, backlog и другие материалы, в том числе из внешних Git providers.
* Внешний repository клонируется/импортируется в локальный private canonical repository AIDI с фиксацией exact revision. Дальнейшая разработка по умолчанию выполняется локально.
* Исключение — продолжение прямой работы во внешнем repository — разрешается только соответствующей manager/admin policy.
* Для existing product обязательны AS-IS analysis и TARGET architecture/migration plan.
* Large binary files хранятся через Object Storage, а не в Git.
7. Product Architecture и Planning
7.1 Product architecture
* Product Architecture отделяется от внутреннего AIDI Development Execution Plan.
* По умолчанию выбирается минимальная достаточная сложность, если заказчик явно не требует иного.
* По умолчанию предпочтителен free/open-source stack. Платная технология допустима по явному требованию заказчика после оценки необходимости, альтернатив, trial/free editions и требований лицензий development/runtime.
* Фиксируются ADR, component/data/integration/security/deployment/install/update architecture.
* Для внутренних AIDI projects полноценная update/rollback модель обязательна. Для коммерческих проектов техническая updateability описывается всегда, а коммерческие support obligations определяет договор/CRM.
7.2 Release planning
* Planning hierarchy: Product → Release/Milestone → Feature → Task → branch/ChangeSet → CI/verification → integration → DONE.
* Requirement traceability сохраняется до Feature/Task/Test/Evidence.
* Dependency Graph, critical path, parallelism и readiness вычисляются явно.
* Планирование resource-aware и capability-aware; допускается динамическая декомпозиция Task и controlled replanning.
* Schedule forecast включает optimistic / expected / conservative dates и confidence; изменение ресурсов/dependencies вызывает пересчёт.
8. Autonomous Execution
8.1 Attempt isolation
* Task ≠ Attempt. Каждый Attempt получает exact source SHA/revision, isolated workspace, Runner/Node/model/resources, lease, logs, error/evidence/artifacts.
* Generated code не изменяет canonical branch напрямую. Результат оформляется ChangeSet и проходит verification/integration queue.
* Source checkout integrity проверяется до и после выполнения; stale source приводит к rebase/regenerate/obsolete handling.
8.2 Coding loop
Understand → Inspect → Plan → Implement → deterministic checks → Tests → Self-review → ChangeSet.
* Model выбирается по capability/history/resources, а не по жёстко заданному имени.
* Aider/no-edit, syntax/parse/build failures и другие детерминированные ошибки классифицируются и не повторяются вслепую.
* Documentation обновляется в процессе разработки и является частью deliverable.
8.3 Verification and review
* Перед integration обязательны deterministic validators, applicable tests и independent review context, где это возможно.
* Используются unit/component/integration/contract/system/E2E/security/performance/install/update/recovery tests по применимости.
* Retry bounded; duplicate execution/side effects protected; secrets least-privileged; sandbox/network policies enforceable.
8.4 Software and Service Catalog
* AIDI может применять сторонние бесплатные инструменты на любых инженерных этапах, если они проходят source/license/security/integrity/compatibility qualification.
* Catalog показывает назначение простым языком, version, license, status/health, resource needs, dependencies и recommendation.
* Provision/update policy: AUTO / REQUEST_ADMIN / DENY. Если tool становится runtime dependency клиентского продукта, он переносится в Product Architecture/BOM/install docs.
9. Recovery, Self-Healing и Self-Repair
9.1 Failure model
* Recovery Engine — first-class subsystem, а не overlay retry.
* Failure classes: Task, Project, Infrastructure, Network/Internet, Hardware/Storage/Hypervisor/Power, External Service, Cloud AI, AIDI Platform, AIDI Internal Logic.
* FaultSignal отделён от RootCause. Сигналы коррелируются, дедуплицируются и fingerprinted; severity S0–S4.
* Recoverability classes: AUTO_SAFE, AUTO_GUARDED, APPROVAL_REQUIRED, MANUAL_EXTERNAL.
9.2 Evidence-first recovery
* Recovery: Detect → Diagnose → Evidence → Root Cause → Repair → Verify → Prevent → Resume.
* NO_EFFECT — отдельная диагностика. Повтор без изменившейся стратегии/input запрещён.
* Retry имеет budget, backoff, mutation/diversification и loop detection.
* Zombie attempts, stale leases, obsolete WIP/reservations, duplicate execution и source-integrity drift автоматически reconciled.
9.3 Planned disruptions
* Restart/reboot/failover, инициированный AIDI, оформляется MaintenanceIntent. В пределах ожидаемой scope/duration это не Warning.
* Если planned operation выходит за срок/границы или сервис не восстанавливается, создаётся Problem.
9.4 Infrastructure and cloud recovery
* AIDI может restart/failover/reprovision/replace resource/config rollback; потеря Internet не останавливает local work.
* Cloud outage/rate/quota/budget/auth/model removed переключает workflow на локальный путь, если он допустим, и создаёт нужное уведомление/ActionRequest.
9.5 Self-development
* AIDI_SYSTEM_PROJECT представляет собственный продукт AIDI. Прямое редактирование production core запрещено.
* System Requirement → implementation → verification → applicable ПМИ → build → canary → stability window → promote/rollback.
* Recovery Guardian / Update Supervisor — минимальный независимый компонент без зависимости от main LLM/planner; умеет health/watchdog, safe mode, rollback и Last Known Good.
* System Resource Reserve позволяет гарантировать CPU/RAM/GPU/Runner ресурсы для critical recovery и ограниченного self-development. Anti-starvation обязателен.
* Recovery modes: CONTINUE_DEVELOPMENT и RECOVERY_DOMINANT.
* Operational Knowledge хранит Problem → Evidence → RootCause → Repair → Verification → Outcome → Prevention; это внешняя память, не скрытые веса модели.
10. Infrastructure & Resource Management
10.1 Provider abstraction and topology
* Abstractions: Compute, Virtualization, Container, Storage, Network, ModelRuntime Provider. Single-node, distributed и HA profiles поддерживаются штатно.
* Первым устанавливается полный Initial Control Node. Super Admin запускает Infrastructure Assessment; AIDI обнаруживает доступные providers/resources и рекомендует ALL-IN-ONE ONLY / DISTRIBUTED AVAILABLE / DISTRIBUTED RECOMMENDED / HA CAPABLE.
* Если ресурсов достаточно, предлагаются MINIMUM/RECOMMENDED/HA topology; после approval AIDI сама создаёт VM group, ставит Node Agent, назначает roles и квалифицирует cluster.
* Логические roles могут co-locate: CONTROL_PLANE, CORE_DATA, SCM, OBJECT_STORAGE, MODEL_RUNTIME, EXECUTION_RUNNER, OBSERVABILITY, BACKUP, SECRETS_SECURITY, RECOVERY_GUARDIAN.
10.2 Node Agent
* Node Agent отделён от Runner и управляет machine identity, inventory, heartbeat, role/service lifecycle, config, telemetry, maintenance и recovery commands.
* Generated code/Runner не получает Agent privileges. Enrollment: one-time token → cryptographic/mTLS machine identity.
* Agent update защищён локальным supervisor/watchdog и rollback; обновление не должно self-disconnect управляемый Node.
10.3 Virtualization providers
* Обязательные adapters: Proxmox VE и Microsoft Hyper-V. Архитектура расширяема на KVM/libvirt, VMware и другие providers.
* При policy=AUTO и достаточной capacity AIDI самостоятельно создаёт VM, устанавливает Agent/Runner, конфигурирует и квалифицирует её.
10.4 Resource management
* Inventory динамически включает CPU/RAM/GPU/VRAM/PCIe/storage/network; данные имеют freshness, stale capacity не считается доступной.
* Capacity ledger: total / reserved / allocated / available. Reservation и Allocation Lease предотвращают phantom/double allocation.
* Resource Pools, capability tags, quotas, weighted fairness, idle borrowing, anti-starvation и system reserve обязательны.
* Provisioning policy наследуется System → Role → User → Workspace → Project и для resource class принимает AUTO / REQUEST_ADMIN / DENY.
10.5 Temporary/opportunistic resources
* Temporary/preemptible Node может быть доступен online-based, resource-free-based, schedule-based, manual или combined.
* На него отправляются только подходящие repeatable/noncritical tasks; owner workload защищён. Исчезновение Node приводит к reconciliation/reschedule, возвращение — к health/qualification перед повторным использованием.
10.6 GPU
* GPU/VRAM — first-class resources: model residency, admission, reservation, temperature/power/error health.
* AIDI не выполняет unsafe overclock/voltage как штатную автоматизацию; thermal/power policy служит защите стабильности.
11. LLM / Model Management
11.1 Model abstractions
* Различаются ModelArtifact, Variant, Runtime, Deployment, Instance, Profile и RoutingProfile.
* Workflow запрашивает capability: GENERAL_REASONING, CODING_FAST, CODING_HIGH, CODE_REVIEW, DEBUGGING, LONG_CONTEXT, SPECIFICATION, EMBEDDING, RERANKING, VISION, STRUCTURED_EXTRACTION.
* Model Catalog содержит source, exact revision/checksum, license, resource profile, qualification, benchmarks, health, update state и deployments.
11.2 Qualification and routing
* Публичный benchmark не является достаточным основанием. Используются local AIDI evaluation suites и Project micro-evals.
* Qualification даёт verdict per capability. Router учитывает quality, historical success, latency, resource pressure, availability, cost и risk.
* Profiles: FAST / BALANCED / HIGH_QUALITY. Поддерживаются multimodel/fallback и независимый reviewer.
11.3 Cloud AI Policy
* Cloud modes: DISABLED / CONSULT_ONLY / FALLBACK / ALLOWED. Effective policy зависит от system role, user, workspace, project, task и data classification.
* Local Router может рекомендовать provider/model по capability/quality/cost/privacy/history, но recommendation не обходится без authorization.
* При Cloud DENY технический внешний вызов невозможен.
11.4 Context and RAG
* Context Manager версионирует prompt templates, управляет context budget, provenance и trust classification.
* RAG изолирован по tenant/project; embeddings/reranking являются derived data и rebuildable.
* Operational memory хранится снаружи модели. Fine-tuning optional и допускается только на governed dataset.
11.5 Model placement
* Учитываются VRAM/residency/admission/eviction/pinning/CPU/multi-GPU. Warm resident model может иметь placement advantage, но policy/quality имеют приоритет.
* Health/quarantine/update выполняются side-by-side/canary/rollback.
12. Security
12.1 Authorization model
Identity → Authentication → Role → Policy → Authorization → Action → Audit.
* RBAC дополняется policy engine; explicit deny overrides allow.
* Project/workspace isolation enforceable backend-side; UI hiding не является security control.
* Service identities, workload identities и tool permissions следуют least privilege.
12.2 Secrets and data classification
* Secrets хранятся через Secret Store/SecretReference, инъецируются только разрешённым workload и не попадают в logs/prompts.
* Data classes: PUBLIC, INTERNAL, CONFIDENTIAL, SECRET. SECRET запрещён к отправке внешнему model/cloud.
* Storage/repo/RAG/runtime/workspace isolation, sandbox, network profiles, production boundary и control-plane isolation обязательны.
12.3 Prompt Injection Defense
* Untrusted content включает chat, CRM/Portal, files, repositories, issue/PR, docs, web, RAG и tool outputs.
* Untrusted content является данными, а не governing instruction: оно не может менять policy, повышать privileges, получать secrets, вызывать privileged tools или отключать gates.
* Защита строится не на одном detector, а на provenance/trust classification, context isolation, Tool Broker, least privilege, network/sandbox и independent authorization.
* Suspected injection фиксируется как security event; Acceptance включает prompt-injection test suite.
12.4 API/session security
* API scopes, idempotency, optimistic concurrency, sessions, CSRF/browser protection, rate limiting, last-admin protection и certificate lifecycle обязательны.
* MFA capability предусматривается архитектурно; break-glass отделён от обычного recovery.
12.5 Supply chain
* Package/model/source integrity, signatures, SBOM, provenance и dependency pinning обязательны для release и self-development.
13. Public API, CRM и Client Portal
13.1 Boundaries
* AIDI полностью функционирует без CRM/Portal. Единственная интеграционная граница — versioned Public Application API; прямой доступ внешних систем к AIDI DB запрещён.
* Commands отделены от Queries; поддерживаются idempotency command IDs, correlation, optimistic concurrency, structured errors и rate limits.
* Service identity дополняется delegated end-user attribution.
13.2 Commercial lifecycle
Client Portal → CRM → AIDI Project → Discovery → Specification → Feasibility/Resources/Schedule → CRM commercial review → Client decision → START или CANCEL.
* CRM authoritative для leads/deals/price/budget/contracts/commercial thresholds. AIDI authoritative для engineering state.
* CANCEL скрывает Project из client projection, переводит его в controlled archive/retention lifecycle, но не удаляет данные мгновенно.
* Functional questions могут идти напрямую через Portal; commercial/license questions маршрутизируются Manager/CRM.
13.3 Change Request
* AIDI вычисляет reproducible technical change impact: Requirements/Features/Rework/Schedule/Resources/Risks и Technical Change %.
* CRM применяет коммерческий threshold/budget workflow и возвращает approval/rejection.
13.4 Events/webhooks
* Event delivery tolerant к at-least-once: unique IDs, retry, dead-letter, ordering metadata, schema versions, query fallback/resync.
* Недоступность CRM не должна останавливать независимую engineering development.
13.5 Files
* External systems получают temporary upload/download grants, а не permanent storage credentials; files проходят security/trust checks.
14. Storage, Files, Artifacts и Data
14.1 Storage boundary
* AIDI использует отдельное object/file storage и не зависит от HM.DM family storage.
* Storage Provider abstraction обязательна; reference backend — SeaweedFS OSS, но архитектура не связывается с конкретным backend.
14.2 File/Object model
* FileObject содержит immutable ID, associations, filename/MIME/size/checksum, uploader/source, classification, retention, state и provenance.
* Multipart uploads, associations, access isolation, security scanning и archive safety обязательны.
* Workspace storage non-authoritative и воспроизводим; release artifacts/evidence имеют повышенную immutability/protection.
14.3 Quotas and pressure
* Поддерживаются quotas, soft/hard limits, system reserve и disk-pressure cleanup order. Resource pressure не имеет права обходить retention/hold/protected classes.
* Backup не равен replication; integrity scrubbing/encryption и storage health контролируются отдельно.
14.4 Data lifecycle metadata
* Data objects содержат retention class, retention start event, retain-until, applicable policies, hold status и purge history.
15. Observability, Audit, Notifications и Email
15.1 Observability
* Events, Audit, Logs, Metrics, Traces и Notifications — разные сущности.
* Correlation/trace/causation обязательны; structured logs redact secrets; AI provenance фиксируется без скрытого chain-of-thought.
* Metrics покрывают infrastructure/GPU/model/cloud/task/attempt/recovery/quality/project/fairness/queue/services/storage/API.
* Alerting кореллирует root causes, suppresses planned disruptions, дедуплицирует и контролирует fatigue.
* Notification Center отделён от Action Center. Super Admin видит system notifications/actions, destinations и delivery state.
15.2 Audit
* Audit immutable и хранит actor/action/target/before-after/time/source/correlation. Self-development, tools, infrastructure, model, cloud и security actions аудируются.
15.3 Email
* AIDI имеет outbound SMTP notifications для внутренних AIDI users и password recovery только для Local Identity.
* AD/LDAP/FreeIPA users могут получать notifications, но password восстанавливается authoritative IdP.
* Client Portal users по умолчанию не получают прямые AIDI email и не используют AIDI password reset; коммуникацию ведёт CRM/Portal.
* Local reset требует verified email, neutral anti-enumeration response, one-time expiring cryptographic token, raw token not stored, rate limiting, audit и session revocation policy.
* SMTP credentials хранятся в Secret Store; delivery async/retry; SMTP failure не является Project failure.
16. Installation, Update, Backup и Disaster Recovery
16.1 Installation
* Официальный install package устанавливает полную AIDI на clean supported system; modes: interactive, unattended и restore.
* Preflight проверяет OS/arch/CPU/RAM/storage/ports/privileges. После установки выполняется Health Gate.
* Поддерживается offline bundle: после подготовки пакета Internet для установки не требуется.
* После Initial Control Node дальнейшие managed role servers устанавливаются централизованно AIDI, без ручного per-server setup.
16.2 Update
Build → Verify → Backup → Candidate → Canary → Health → Switch → Stability → Promote; failure → Rollback → Last Known Good.
* No blind auto-update. Migrations versioned; expand-migrate-contract; rolling multi-node updates where applicable.
* Node Agent/Guardian update не должен лишать AIDI management channel.
16.3 Backup
* Backup scope: canonical state, specs/projects/events/audit/policies/id mappings/config, repositories, artifact metadata/objects, encrypted secrets, recovery metadata.
* Backup has manifest/checksums, consistency, encryption, retention, health, validation and restore tests. Backup без успешной restore validation не считается полностью подтверждённым.
16.4 DR
* Поддерживаются full restore, project/repository/file/config restore, clean-server restore и different-hardware disaster recovery.
* После DR running Attempts reconciled; система не создаёт дубли только потому, что прежний executor исчез.
* DR должен работать без обязательного Cloud AI.
17. UI / UX
17.1 USER UI
* Responsive desktop/tablet/mobile. Default locale ru-RU, English optional; timezone configurable, canonical timestamps UTC.
* Primary navigation: Projects, Action Center, Notifications, Files, Profile.
* Project navigation: Overview, ТЗ, Plan, Kanban, Releases, Results, Files, History, Participants.
* UI не показывает fake progress или декоративный green; status отражает canonical lifecycle/health.
17.2 SUPER ADMIN UI
* Overview, Projects, Users & Access, Infrastructure, AI Models, Services, Storage, Recovery, Security, Integrations, Backup & Updates, Observability, System Development, Settings.
* Infrastructure wizard проводит assessment/topology/deploy. Cloud Provider wizard: credentials → connectivity → model discovery → policy → verify.
* Кнопка «Проверить платным запросом» использует synthetic minimal prompt, заранее показывает provider/model/expected minimal spend и после запроса actual/estimated cost, latency, usage и verdict.
17.3 AIDI Orb Assistant
* AIDI Orb доступен после qualification local LLM, имеет modes Explain / Analyze / Help / Suggest / Execute.
* Effective Assistant access = current identity + role + project membership + policy + data classification. Никаких скрытых admin rights.
* Все actions Assistant проходят Tool Broker. Super Admin может disable globally/by role/user/workspace/project; backend также обязан deny, а не только скрывать UI.
* Assistant не является отдельным source of truth или скрытой памятью; значимые решения должны материализоваться в canonical entities.
18. Testing, Verification, ПМИ и Definition of Done
18.1 Test model
* Test, Verification, Validation, Qualification, Gate и Acceptance — разные понятия и сохраняют exact version/evidence.
* Test levels: unit, component, integration, contract, system, E2E, security, performance, install, update, recovery, acceptance.
* Generated tests считаются untrusted до deterministic validation/review. Test execution изолирован по environments/data/secrets.
* Result states: NOT_RUN, RUNNING, PASSED, FAILED, SKIPPED, INCONCLUSIVE, ERROR; failure классифицируется.
* Flaky tests выявляются, не превращаются автоматически в PASS. Regression selection зависит от change/risk/dependency; full regression доступен.
18.2 Mandatory security/recovery tests
* Обязательны Prompt Injection, cross-project/tenant, RBAC, Cloud/Data Policy, Tool Broker, secrets, infrastructure and model qualification tests.
* Обязательны install/upgrade/rollback/migration/backup/restore/DR/recovery/fault injection tests.
18.3 ПМИ
* ПМИ — обязательный versioned first-class artifact, одновременно human-readable и machine-readable.
* Каждый test case содержит ID/title/goal/requirements/preconditions/infrastructure/test data/steps/expected per step/auto-manual/evidence/pass-fail/recovery.
* AIDI автоматически выполняет все automatable steps; Guided Tester Mode даёт точные ручные инструкции и собирает evidence.
* После выполнения автоматически формируется Протокол испытаний. Production Release невозможен без mandatory ПМИ pass.
18.4 Mandatory local-only E2E
Clean install → admin password change → local LLM → cloud off → infrastructure assessment/deployment → user → «Я хочу…» → Discovery → ТЗ → Feasibility → Approval → Architecture → Plan → Development → Local Review → Tests → injected failure → Root Cause/Recovery/Prevention → Auto Resume → Build → Clean Install продукта → Release Candidate → Acceptance → Released Artifact.
* PASS: ни один обязательный AI-step не использует Cloud; штатная recoverable ошибка не требует ручного администрирования.
19. Licensing, Regulatory Compliance, Retention и Data Lifecycle
19.1 Component governance
* Component lifecycle: Discover → identity/source → license → security → integrity → compatibility → policy → qualify → admit → monitor.
* Exact versions/digests обязательны. Unknown/conflicting license не проходит AUTO admission. License changes требуют re-admission.
* Dependency/transitive licensing, model/dataset governance, SBOM/provenance, vulnerabilities и obligations tracked.
* AIDI Core не должен иметь required commercial license. Коммерческие product components допускаются только по явному customer requirement с оценкой alternatives/licensing impact.
19.2 Russian Regulatory Profile
* Regulatory Registry versioned и хранит effective dates/status. Applicability Engine оценивает customer/location/data/industry/AI/CII/signature/trade-secret factors.
* При неоднозначности создаётся LEGAL_REVIEW_REQUIRED; система не обещает blanket compliance без applicability evidence.
* Personal data tagging/residency и Cloud AI cross-border data gate enforceable технически. Regulatory changes проходят impact analysis.
19.3 Retention lifecycle
CREATE → ACTIVE → INACTIVE → ARCHIVE → RETENTION → PURGE_ELIGIBLE → PURGE, либо HOLD.
* Access lifecycle и Data lifecycle разделены. Hidden ≠ Deleted; окончание client access не означает удаление данных.
* Retention Classes минимум: TEMPORARY, WORKSPACE, CACHE, PROJECT_ACTIVE, PROJECT_ARCHIVE, CONTRACTUAL, RELEASE, AUDIT, SECURITY, RECOVERY, BACKUP, SYSTEM_CONFIGURATION, MODEL, LEGAL_HOLD.
* Effective retention: SYSTEM → REGULATORY → CONTRACT → WORKSPACE → PROJECT → DATA CLASS → OBJECT TYPE; mandatory более строгая policy имеет приоритет.
* Retention start event фиксируется явно; Cancelled Project архивируется, получает deadline и физически удаляется только после checks.
* Purge запрещён при Hold, active contractual/regulatory requirement, protected release/evidence или open incident.
* Project purge не обязан удалять Audit. Derived copies — previews, extracted text, embeddings, RAG/search index — следуют lifecycle source.
* Raw AI prompts/responses не хранятся бессрочно по умолчанию; canonical Requirements/Decisions живут независимо от raw conversation.
* Multi-provider purge отслеживается до полного результата; ошибка удаления одной копии → PURGE_INCOMPLETE, а не success.
* Policy shortening требует impact preview; LLM не имеет самостоятельного purge authority.
20. Acceptance Matrix — обязательные категории
20.1 Installation & infrastructure
* AC: Clean install полной AIDI на поддерживаемой системе проходит Health Gate.
* AC: admin/admin существует только как bootstrap и требует mandatory password change до эксплуатации.
* AC: Initial Control Node является полноценной AIDI и способен выполнять assessment/deployment.
* AC: Offline installation из подготовленного bundle не требует Internet/Cloud AI.
* AC: Proxmox VE и Hyper-V providers проходят qualification; при AUTO AIDI сама создаёт VM и подключает Node Agent.
* AC: Runner отделён от Node Agent privileges; temporary Node корректно attach/drain/reconcile.
20.2 Local/Cloud AI
* AC: Полный обязательный lifecycle проходит при CLOUD AI = DISABLED.
* AC: Model Router выбирает qualified deployment по capability, а не hard-coded model name.
* AC: новая модель не используется до qualification; при AUTO разрешённая model может быть downloaded/qualified/deployed автоматически.
* AC: local model failure имеет local fallback path; Cloud recommendation не даёт authorization.
* AC: Cloud Provider wizard и paid synthetic test работают без project data; запрещённый data class не отправляется во внешний endpoint.
20.3 Identity & access
* AC: Local Identity работает независимо от AD; AD/LDAP/FreeIPA/OIDC integration не хранит внешний password.
* AC: Break-glass local admin работает при отказе external IdP; last available Super Admin нельзя случайно удалить/disable.
* AC: Project invitation использует exact login/email, не перечисляет users, доступ создаётся только после ACCEPT.
* AC: Local password reset одноразовый/expiring/anti-enumeration; external IdP passwords AIDI не сбрасывает; client Portal users исключены из AIDI recovery.
20.4 Specification & planning
* AC: Project создаётся из «Я хочу…»; Discovery adaptive; critical conflict блокирует Approval.
* AC: до Approval завершены Feasibility, Development Resource Profile и Product Runtime Profile.
* AC: итоговое ТЗ содержит product technologies и installation guide; Approval связан с exact version.
* AC: existing repository import фиксирует exact revision и создаёт local canonical repository; AS-IS анализ обязателен.
* AC: plan содержит Releases/Features/Tasks/dependencies и учитывает реальные resources; replanning controlled.
20.5 Execution & recovery
* AC: Task/Attempt separation, exact source, isolated change and integration verification доказаны.
* AC: duplicate command/side-effect protected; stale source и fencing ownership корректно обрабатываются.
* AC: failure классифицируется, Root Cause evidence-based, blind retry отсутствует, successful recovery автоматически resumes workflow.
* AC: planned restart не создаёт false alert; failed MaintenanceIntent становится Problem.
* AC: Cloud outage не останавливает eligible local work.
20.6 Self-development / canonical state
* AC: собственный код AIDI меняется только через AIDI_SYSTEM_PROJECT; production direct self-edit запрещён.
* AC: canary/rollback/Last Known Good и System Resource Reserve работают.
* AC: current state + Event atomic; Outbox/Inbox/idempotency; stale revision rejected; stale owner fencing rejected; projections rebuildable.
20.7 Security / integration / storage
* AC: Project/RAG/secrets isolation и Tool Broker authorization проходят negative tests.
* AC: prompt injection из chat/file/web/RAG/tool output не получает system authority.
* AC: CRM работает только через API, commercial/engineering boundaries разделены, CRM outage не останавливает engineering.
* AC: AIDI storage независим, checksums enforced, Released Artifact immutable, external file access только временными grants.
20.8 Observability / Backup / HA / compliance
* AC: correlation позволяет пройти Problem → Attempt → Trace → Logs → Recovery; secrets redacted.
* AC: valid backup имеет manifest/checksums и Restore Test; clean/different-hardware restore works; running attempt reconciled.
* AC: HA honest — несколько VM на одном host не считаются полноценным HA; split brain предотвращён fencing/quorum.
* AC: unknown license не auto-admitted; exact SBOM/signature/provenance формируются; regulatory applicability/residency/cloud-data rules enforceable.
* AC: Hidden ≠ Deleted, Hold blocks purge, derived data follows purge lifecycle, incomplete provider deletion remains PURGE_INCOMPLETE.
20.9 ПМИ / final acceptance
* AC: ПМИ versioned human+machine readable; automatable steps исполняет AIDI, manual steps ведёт Guided Tester Mode; Protocol generated.
* AC: Production Release невозможен при failed mandatory ПМИ, unresolved Blocking Defects или отсутствии release SBOM/verified installer.
21. Technology Binding & Defaults
21.1 Базовый стек
* Server OS — Debian 13.
* AIDI Core — Go; AI/ML workers — Python.
* Frontend — React + TypeScript + Vite.
* Canonical DB — PostgreSQL 16; DB HA — Patroni + etcd.
* Durable Workflows — Temporal; Event Bus — NATS JetStream.
* Local SCM — Gitea. Forgejo остаётся поддерживаемым import/optional provider, но не default.
* Object Storage — SeaweedFS OSS; Vector/RAG — Qdrant.
* Secrets/Internal PKI — OpenBao.
* Container runtime — containerd; Builds — BuildKit; reverse proxy/TLS endpoint — Caddy.
* Primary high-throughput local LLM runtime — vLLM; universal local fallback — llama.cpp; Ollama — optional adapter.
* Telemetry — OpenTelemetry Collector; Metrics — Prometheus; Tracing — Jaeger; technical logs/search — OpenSearch.
* Supply chain — Syft, Trivy, Cosign, OpenSSF Scorecard; OSV when Internet available.
* Mandatory virtualization adapters — Proxmox VE + Microsoft Hyper-V.
* Public API — REST/JSON + OpenAPI 3.1; Internal API — gRPC/Protobuf + mTLS; realtime UI — SSE; email — SMTP.
* Kubernetes, Cloud AI/DB, GitHub/GitLab/Forgejo, Grafana и SaaS monitoring/secrets не являются обязательными dependencies AIDI Core.
21.2 Default policies
* Clean install: LOCAL_FIRST, Cloud Providers не настроены и не активируются автоматически.
* Default Data Cloud Policy: PUBLIC may be permitted; INTERNAL explicit policy; CONFIDENTIAL LOCAL_ONLY; SECRET never to external model/cloud.
* Default repository policy: AUTO_LOCAL → private Gitea repository.
* Resource default: SUPER_ADMIN temporary VM/container AUTO within limits; USER temporary container AUTO, new VM REQUEST_ADMIN unless overridden; CLIENT no infrastructure authority.
* Control Plane reserve default: не менее 15% CPU и 20% RAM соответствующего Control Node либо эквивалентный hard reserve.
* Self-development default execution weight: 10% после production install; Critical S4 Recovery в RECOVERY_DOMINANT может использовать до 80% allocatable capacity, не затрагивая Control Plane Reserve.
* Attempt heartbeat default 15 s; Lease TTL default 60 s; expiry → reconciliation.
* Transient retry default до 3 attempts с exponential backoff; deterministic retry только после существенного изменения strategy/input.
* ProjectInvitation TTL default 7 days; Local Password Reset Token TTL default 30 minutes.
* Successful Attempt workspace default retention 24 h; failed/recovery workspace 7 days; Recovery Hold может продлить.
* Cancelled commercial Project default retention 30 days, если System/Contract/Regulatory policy не требует иного.
* Initial operational retention defaults: debug logs 14 days; completed notifications 90 days; Recovery Evidence 90 days; Audit 365 days; raw high-frequency metrics 7 days; aggregated metrics up to 365 days. Это эксплуатационные, не юридические сроки.
* Backup baseline: PITR/WAL где настроено, daily backups, 7 daily + 4 weekly + 6 monthly recovery points; production restore drill не реже одного раза в месяц.
21.3 Hardware and SLO defaults
* Minimum control-only node: 4 vCPU, 8 GB RAM, 80 GB SSD.
* Functional minimum All-in-One: 8 vCPU, 32 GB RAM, 300 GB SSD/NVMe; local AI может работать CPU-only с ограниченной производительностью.
* Recommended All-in-One: 16+ CPU threads, 64 GB RAM, 500 GB+ NVMe, GPU ≥12 GB VRAM; более крупные coding models требуют больше ресурсов.
* Compact distributed reference: Control 4 vCPU/8 GB; Data/Core 8 vCPU/16 GB; AI/Execution 8+ vCPU/32+ GB + GPU; planner может перераспределять roles.
* HA требует минимум 3 независимых failure domains для quorum services; при их отсутствии UI не показывает HA READY.
* Initial technical SLOs on Recommended profile: Public API p95 <300 ms; обычный UI read p95 <500 ms; scheduling после появления подходящего resource p95 <2 s; internal Action/Notification propagation p95 <5 s; heartbeat-managed fault detection target <60 s.
* HA Control Plane internal availability objective: 99.9%; это engineering objective, а не клиентский SLA.
21.4 Technology governance
* Production запрещает unpinned latest. Перед implementation фиксируется Technology Lock Manifest: exact versions, OCI digests, binary hashes, licenses, compatibility, SBOM и admission decisions.
* Ни один компонент не считается permanent forever; license/security/abandonment/compatibility change может инициировать controlled architecture migration.
22. Definition of Done AIDI v2.0.0
* Все Mandatory Requirements реализованы и трассируются к evidence.
* Local-only E2E, Security, Prompt Injection, Recovery, Infrastructure, Update/Rollback, Backup/DR, Persistence/Concurrency, API/CRM, Retention и applicable Regulatory ПМИ — PASS.
* Release SBOM сформирована, installer verified, Protocol of Tests сформирован, unresolved Blocking Defects отсутствуют.
* Release Artifact Set immutable и подписан/проверяем.
* Только после выполнения этих условий RC может стать AIDI 2.0.0 PRODUCTION RELEASE.
23. Реализация и cutover baseline
* Implementation roadmap: A Foundation → B Identity & Platform → C Infrastructure → D AI Platform → E Specification & Planning → F Autonomous Development → G Recovery & Self-Healing → H Self-Development & Update → I CRM/Client Integration → J Storage/Supply Chain/Compliance → K HA/DR/Hardening → RC Final Acceptance.
* Текущая legacy-разработка до cutover маркируется LEGACY PRE-2.0 и не смешивается с новой AIDI 2.0 execution queue.
* Controlled cutover: FREEZE → safe checkpoint → backup/restore point → legacy isolation → import SPEC/roadmap/backlog/gates/dependencies → integrity validation → activate Release A only → stabilization.
* При критической аномалии до стабильности выполняется rollback к pre-cutover restore point.
Конец утверждённого baseline AIDI v2.0.0.